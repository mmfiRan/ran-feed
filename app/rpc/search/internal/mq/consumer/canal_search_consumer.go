package consumer

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/logic/indexer"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/dedup"

	pkgconsts "ran-feed/pkg/consts"
)

const consumerName = "search.canal_consumer"

type CanalSearchConsumer struct {
	ctx context.Context
	svc *svc.ServiceContext
	logx.Logger
	dedupGate *dedup.Gate
}

func NewCanalSearchConsumer(ctx context.Context, svcContext *svc.ServiceContext) *CanalSearchConsumer {
	return &CanalSearchConsumer{
		ctx:       ctx,
		svc:       svcContext,
		Logger:    logx.WithContext(ctx),
		dedupGate: dedup.New(svcContext.MysqlDb.DB),
	}
}

// Consume 解析后按表路由 content/article/video 归内容文档 user 归用户文档 未监听表跳过
func (c *CanalSearchConsumer) Consume(ctx context.Context, key, val string) error {
	msg, err := canal.Parse(val)
	if err != nil {
		logc.Errorf(ctx, "解析 canal 消息失败 err=%v val=%s", err, val)
		return err
	}

	eventID := msg.EventID(val)
	switch msg.Table() {
	case consts.SourceTableContent, consts.SourceTableArticle, consts.SourceTableVideo:
		return c.handleContent(ctx, msg, eventID)
	case consts.SourceTableUser:
		return c.handleUser(ctx, msg, eventID)
	default:
		return nil
	}
}

// handleContent 取 content_id 回源 content-rpc 索引投影 返回的 upsert 缺席的判删
func (c *CanalSearchConsumer) handleContent(ctx context.Context, msg *canal.Message, eventID string) error {
	isContentTable := msg.Table() == consts.SourceTableContent
	ids, keys, err := c.collectPending(ctx, msg, eventID, func(row map[string]any) int64 {
		if isContentTable {
			id, _ := canal.ParseInt64(row["id"])
			return id
		}
		id, _ := canal.ParseInt64(row["content_id"])
		return id
	}, func(row, oldRow map[string]any) bool {
		// content 表热榜分钟级落库只改 hot_score 等列 与索引无关 跳过避免无谓回源与 ES 写
		return isContentTable && canal.OnlyIgnoredColumnsChanged(row, oldRow, pkgconsts.HotScoreOnlyColumns...)
	})
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	if len(ids) == 0 {
		c.markConsumed(ctx, keys)
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

	if err := c.writeES(ctx, es.IndexContent, items, deleteIDs, msg.UpdatedAt().UnixMilli()); err != nil {
		return err
	}
	c.markConsumed(ctx, keys)
	return nil
}

// handleUser 回源 user-rpc 索引投影 返回的 upsert 缺席的判删
func (c *CanalSearchConsumer) handleUser(ctx context.Context, msg *canal.Message, eventID string) error {
	ids, keys, err := c.collectPending(ctx, msg, eventID, func(row map[string]any) int64 {
		id, _ := canal.ParseInt64(row["id"])
		return id
	}, nil)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	if len(ids) == 0 {
		c.markConsumed(ctx, keys)
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

	if err := c.writeES(ctx, es.IndexUser, items, deleteIDs, msg.UpdatedAt().UnixMilli()); err != nil {
		return err
	}
	c.markConsumed(ctx, keys)
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

// collectPending 逐行查去重表收集未处理行 用 extract 取目标 id 去重
// 幂等键不在此落库 待 ES 写成功后再由 markConsumed 落 避免回源或写 ES 失败重试被去重吞掉
// skip 非空时先行过滤 命中的行既不落键也不回源 返回去重后的目标 id 与本次待消费的幂等键
func (c *CanalSearchConsumer) collectPending(ctx context.Context, msg *canal.Message, eventID string, extract func(map[string]any) int64, skip func(row, oldRow map[string]any) bool) ([]int64, []string, error) {
	table, op := msg.Table(), msg.Op()
	seen := make(map[int64]bool, len(msg.Data))
	ids := make([]int64, 0, len(msg.Data))
	keys := make([]string, 0, len(msg.Data))
	for i, row := range msg.Data {
		if row == nil {
			continue
		}
		if skip != nil && skip(row, msg.OldRow(i)) {
			continue
		}
		key := canal.RowEventID(eventID, table, op, row, i)
		exists, err := c.dedupGate.Exists(ctx, consumerName, key)
		if err != nil {
			return nil, nil, err
		}
		if exists {
			continue
		}
		keys = append(keys, key)
		if id := extract(row); id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, keys, nil
}

// markConsumed 写成功后落幂等键 best-effort 失败只记日志 重复消费幂等无害
func (c *CanalSearchConsumer) markConsumed(ctx context.Context, keys []string) {
	for _, key := range keys {
		if _, err := c.dedupGate.InsertIfAbsent(ctx, consumerName, key); err != nil {
			c.Errorf("插入去重表失败 key=%s err=%v", key, err)
		}
	}
}

// writeES upsert 与 delete 批量写 硬错误上抛触发重投 部分失败仅记日志靠重建 job 补
// 增量路径 upsert 与 delete 统一用 canal 事件 ts 作 version 单分区内单调 防 删-恢复 快速翻转被旧 tombstone 挡住
func (c *CanalSearchConsumer) writeES(ctx context.Context, index string, items []es.IndexItem, deleteIDs []int64, version int64) error {
	for i := range items {
		items[i].Version = version
	}
	if len(items) > 0 {
		if failed, err := es.BulkUpsert(ctx, c.svc.ES, index, items); err != nil {
			return err
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
			return err
		} else if failed > 0 {
			logc.Errorf(ctx, "增量 delete 部分失败 index=%s failed=%d", index, failed)
		}
	}
	return nil
}
