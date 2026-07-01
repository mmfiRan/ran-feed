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

// SearchReindexJob 全量以 content/user 域为真相源经 RPC 投影重灌 ES 初始灌入与周期兜底漂移
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
	contentTotal, err := j.reindexContent(ctx)
	if err != nil {
		return "", err
	}
	userTotal, err := j.reindexUser(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ok content=%d user=%d", contentTotal, userTotal), nil
}

// reindexContent 游标经 content-rpc 投影扫可索引内容 分批 bulk 灌入
func (j *SearchReindexJob) reindexContent(ctx context.Context) (int, error) {
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
		failed, err := es.BulkUpsert(ctx, j.svc.ES, es.IndexContent, items)
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

// reindexUser 游标经 user-rpc 投影扫可索引用户 分批 bulk 灌入
func (j *SearchReindexJob) reindexUser(ctx context.Context) (int, error) {
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
		failed, err := es.BulkUpsert(ctx, j.svc.ES, es.IndexUser, items)
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
