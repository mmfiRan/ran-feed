package repositories

import (
	"context"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type OperationLogRepository interface {
	Create(row *model.RanFeedOperationLog) (int64, error)
}

type operationLogRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewOperationLogRepository(ctx context.Context, db *orm.DB) OperationLogRepository {
	return &operationLogRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// Create 落一条操作审计日志 返回自增ID
func (r *operationLogRepositoryImpl) Create(row *model.RanFeedOperationLog) (int64, error) {
	if row == nil {
		return 0, nil
	}
	q := query.Q
	if err := q.RanFeedOperationLog.WithContext(r.ctx).Create(row); err != nil {
		return 0, err
	}
	return row.ID, nil
}
