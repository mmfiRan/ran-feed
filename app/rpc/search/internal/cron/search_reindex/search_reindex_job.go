package search_reindex

import (
	"context"
	"fmt"

	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/logic/indexer"
	"ran-feed/app/rpc/search/internal/repositories"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
)

const HandlerName = "search.reindex"

// scanBatchSize 全量扫表分批大小
const scanBatchSize = 500

// SearchReindexJob 全量扫 MySQL 以真相源重灌 ES 初始灌入与周期兜底漂移
type SearchReindexJob struct {
	svc         *svc.ServiceContext
	contentRepo repositories.ContentRepository
	userRepo    repositories.UserRepository
	assembler   *indexer.Assembler
	logx.Logger
}

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	job := &SearchReindexJob{
		svc:         svcCtx,
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		userRepo:    repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
		assembler: indexer.NewAssembler(
			repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
			repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
		),
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

// reindexContent 游标扫已发布公开内容 分批回源组装 bulk 灌入
func (j *SearchReindexJob) reindexContent(ctx context.Context) (int, error) {
	var cursor int64
	var total int
	for {
		rows, err := j.contentRepo.ScanPublishable(cursor, scanBatchSize)
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			break
		}

		items, err := j.assembler.AssembleContentDocs(rows)
		if err != nil {
			return total, err
		}
		failed, err := es.BulkUpsert(ctx, j.svc.ES, es.IndexContent, items)
		if err != nil {
			return total, err
		}
		if failed > 0 {
			j.Errorf("内容重建部分写入失败 batch=%d failed=%d", len(rows), failed)
		}
		total += len(rows) - failed

		cursor = rows[len(rows)-1].ID
		if len(rows) < scanBatchSize {
			break
		}
	}
	return total, nil
}

// reindexUser 游标扫正常用户 分批组装 bulk 灌入
func (j *SearchReindexJob) reindexUser(ctx context.Context) (int, error) {
	var cursor int64
	var total int
	for {
		rows, err := j.userRepo.ScanActive(cursor, scanBatchSize)
		if err != nil {
			return total, err
		}
		if len(rows) == 0 {
			break
		}

		items := indexer.AssembleUserDocs(rows)
		failed, err := es.BulkUpsert(ctx, j.svc.ES, es.IndexUser, items)
		if err != nil {
			return total, err
		}
		if failed > 0 {
			j.Errorf("用户重建部分写入失败 batch=%d failed=%d", len(rows), failed)
		}
		total += len(rows) - failed

		cursor = rows[len(rows)-1].ID
		if len(rows) < scanBatchSize {
			break
		}
	}
	return total, nil
}
