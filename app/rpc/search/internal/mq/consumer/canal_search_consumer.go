package consumer

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/logic/indexer"
	"ran-feed/app/rpc/search/internal/repositories"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/user/user"
)

const consumerName = "search.canal_consumer"

type CanalSearchConsumer struct {
	ctx context.Context
	svc *svc.ServiceContext
	logx.Logger
	dedupRepo repositories.MqConsumeDedupRepository
}

func NewCanalSearchConsumer(ctx context.Context, svcContext *svc.ServiceContext) *CanalSearchConsumer {
	return &CanalSearchConsumer{
		ctx:       ctx,
		svc:       svcContext,
		Logger:    logx.WithContext(ctx),
		dedupRepo: repositories.NewMqConsumeDedupRepository(ctx, svcContext.MysqlDb),
	}
}

// Consume 解析后按表路由 content/article/video 归内容文档 user 归用户文档 未监听表跳过
func (c *CanalSearchConsumer) Consume(ctx context.Context, key, val string) error {
	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "解析 canal 消息失败 err=%v val=%s", err, val)
		return err
	}

	eventID := msg.eventID(val)
	switch msg.table() {
	case consts.SourceTableContent, consts.SourceTableArticle, consts.SourceTableVideo:
		return c.handleContent(ctx, msg, eventID)
	case consts.SourceTableUser:
		return c.handleUser(ctx, msg, eventID)
	default:
		return nil
	}
}

// handleContent 取 content_id 回源 content-rpc 索引投影 返回的 upsert 缺席的判删
func (c *CanalSearchConsumer) handleContent(ctx context.Context, msg canalMessage, eventID string) error {
	isContentTable := msg.table() == consts.SourceTableContent
	ids, err := c.dedupAndCollect(msg, eventID, func(row map[string]interface{}) int64 {
		if isContentTable {
			id, _ := parseInt64(row["id"])
			return id
		}
		id, _ := parseInt64(row["content_id"])
		return id
	})
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	res, err := c.svc.ContentRpc.BatchGetContentForIndex(ctx, &content.BatchGetContentForIndexReq{ContentIds: ids})
	if err != nil {
		return err
	}

	items := make([]es.IndexItem, 0, len(res.Items))
	present := make(map[int64]bool, len(res.Items))
	for _, it := range res.Items {
		present[it.ContentId] = true
		items = append(items, indexer.ContentIndexItemToItem(it))
	}
	deleteIDs := missingIDs(ids, present)

	c.writeES(ctx, es.IndexContent, items, deleteIDs, msg.updatedAt().UnixMilli())
	return nil
}

// handleUser 回源 user-rpc 索引投影 返回的 upsert 缺席的判删
func (c *CanalSearchConsumer) handleUser(ctx context.Context, msg canalMessage, eventID string) error {
	ids, err := c.dedupAndCollect(msg, eventID, func(row map[string]interface{}) int64 {
		id, _ := parseInt64(row["id"])
		return id
	})
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	res, err := c.svc.UserRpc.BatchGetUserForIndex(ctx, &user.BatchGetUserForIndexReq{UserIds: ids})
	if err != nil {
		return err
	}

	items := make([]es.IndexItem, 0, len(res.Items))
	present := make(map[int64]bool, len(res.Items))
	for _, it := range res.Items {
		present[it.UserId] = true
		items = append(items, indexer.UserIndexItemToItem(it))
	}
	deleteIDs := missingIDs(ids, present)

	c.writeES(ctx, es.IndexUser, items, deleteIDs, msg.updatedAt().UnixMilli())
	return nil
}

// missingIDs 请求了但源域投影未返回(不可索引)的 id 需从索引删除
func missingIDs(ids []int64, present map[int64]bool) []int64 {
	deleteIDs := make([]int64, 0)
	for _, id := range ids {
		if !present[id] {
			deleteIDs = append(deleteIDs, id)
		}
	}
	return deleteIDs
}

// dedupAndCollect 逐行幂等去重 用 extract 取目标 id 去重收集 dedup 出错上抛触发重试
func (c *CanalSearchConsumer) dedupAndCollect(msg canalMessage, eventID string, extract func(map[string]interface{}) int64) ([]int64, error) {
	table, op := msg.table(), msg.op()
	seen := make(map[int64]bool, len(msg.Data))
	ids := make([]int64, 0, len(msg.Data))
	for i, row := range msg.Data {
		if row == nil {
			continue
		}
		inserted, err := c.dedupRepo.InsertIfAbsent(consumerName, rowEventID(eventID, table, op, row, i))
		if err != nil {
			return nil, err
		}
		if !inserted {
			continue
		}
		if id := extract(row); id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// writeES upsert 与 delete 批量写 写失败非致命 仅 log 靠重建 job 补
// 增量路径 upsert 与 delete 统一用 canal 事件 ts 作 version 单分区内单调 防 删-恢复 快速翻转被旧 tombstone 挡住
func (c *CanalSearchConsumer) writeES(ctx context.Context, index string, items []es.IndexItem, deleteIDs []int64, version int64) {
	for i := range items {
		items[i].Version = version
	}
	if len(items) > 0 {
		if failed, err := es.BulkUpsert(ctx, c.svc.ES, index, items); err != nil {
			logc.Errorf(ctx, "增量 upsert 失败 index=%s err=%v", index, err)
		} else if failed > 0 {
			logc.Errorf(ctx, "增量 upsert 部分失败 index=%s failed=%d", index, failed)
		}
	}
	if len(deleteIDs) > 0 {
		refs := make([]es.DeleteRef, 0, len(deleteIDs))
		for _, id := range deleteIDs {
			refs = append(refs, es.DeleteRef{ID: strconv.FormatInt(id, 10), Version: version})
		}
		if failed, err := es.BulkDelete(ctx, c.svc.ES, index, refs); err != nil {
			logc.Errorf(ctx, "增量 delete 失败 index=%s err=%v", index, err)
		} else if failed > 0 {
			logc.Errorf(ctx, "增量 delete 部分失败 index=%s failed=%d", index, failed)
		}
	}
}
