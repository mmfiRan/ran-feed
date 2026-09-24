package search_reindex

import (
	"context"
	"fmt"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/logic/indexer"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
)

const HandlerName = "search.reindex"

// scanBatchSize 全量重建单页游标大小
const scanBatchSize = 500

// SearchReindexJob 全量以 content/user 域为真相源经 RPC 投影重建索引
// 采用别名切换:新物理索引灌满后原子切别名再删旧索引 期间旧索引持续可读 且天然无孤儿文档
type SearchReindexJob struct {
	svc *svc.ServiceContext
	logx.Logger
}

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &SearchReindexJob{
		svc:    svcCtx,
		Logger: logx.WithContext(ctx),
	}
	executor.RegisterTask(HandlerName, job.Run)
}

func (j *SearchReindexJob) Run(ctx context.Context, _ xxljob.TriggerParam) (string, error) {
	contentTotal, err := es.RebuildIndex(ctx, j.svc.ES, es.IndexContent, j.loadContent)
	if err != nil {
		return "", err
	}
	userTotal, err := es.RebuildIndex(ctx, j.svc.ES, es.IndexUser, j.loadUser)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ok content=%d user=%d", contentTotal, userTotal), nil
}

// loadContent 游标经 content-rpc 投影扫可索引内容 分批 bulk 灌入指定物理索引
// 出错上抛由 RebuildIndex 清理新索引并保留旧索引
func (j *SearchReindexJob) loadContent(ctx context.Context, index string) (int, error) {
	var cursor int64
	var total int
	for {
		res, err := j.svc.ContentRpc.ListContentForIndex(ctx, &content.ListContentForIndexReq{Cursor: cursor, Limit: scanBatchSize})
		if err != nil {
			return total, err
		}
		if len(res.Items) == 0 {
			break
		}

		items := make([]es.IndexItem, 0, len(res.Items))
		for _, it := range res.Items {
			items = append(items, indexer.ContentIndexItemToItem(it))
		}
		failed, err := es.BulkUpsert(ctx, j.svc.ES, index, items)
		if err != nil {
			return total, err
		}
		if failed > 0 {
			j.Errorf("内容重建部分写入失败 batch=%d failed=%d", len(res.Items), failed)
		}
		total += len(res.Items) - failed

		cursor = res.Items[len(res.Items)-1].ContentId
		if len(res.Items) < scanBatchSize {
			break
		}
	}
	return total, nil
}

// loadUser 游标经 user-rpc 投影扫可索引用户 分批 bulk 灌入指定物理索引
func (j *SearchReindexJob) loadUser(ctx context.Context, index string) (int, error) {
	var cursor int64
	var total int
	for {
		res, err := j.svc.UserRpc.ListUserForIndex(ctx, &user.ListUserForIndexReq{Cursor: cursor, Limit: scanBatchSize})
		if err != nil {
			return total, err
		}
		if len(res.Items) == 0 {
			break
		}

		items := make([]es.IndexItem, 0, len(res.Items))
		for _, it := range res.Items {
			items = append(items, indexer.UserIndexItemToItem(it))
		}
		failed, err := es.BulkUpsert(ctx, j.svc.ES, index, items)
		if err != nil {
			return total, err
		}
		if failed > 0 {
			j.Errorf("用户重建部分写入失败 batch=%d failed=%d", len(res.Items), failed)
		}
		total += len(res.Items) - failed

		cursor = res.Items[len(res.Items)-1].UserId
		if len(res.Items) < scanBatchSize {
			break
		}
	}
	return total, nil
}
