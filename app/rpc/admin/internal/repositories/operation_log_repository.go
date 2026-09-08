package repositories

import (
	"context"
	"time"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/app/rpc/admin/internal/types"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type OperationLogRepository interface {
	WithTx(tx *query.Query) OperationLogRepository
	Create(row *model.RanFeedOperationLog) (int64, error)
	// Page 按条件分页查审计日志 id 倒序 返回列表与总数
	Page(filter types.OperationLogFilter, offset, limit int) ([]*model.RanFeedOperationLog, int64, error)
}

type operationLogRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
}

func NewOperationLogRepository(ctx context.Context, db *orm.DB) OperationLogRepository {
	return &operationLogRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *operationLogRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *operationLogRepositoryImpl) WithTx(tx *query.Query) OperationLogRepository {
	return &operationLogRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// Create 落一条操作审计日志 返回ID
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

// Page 按条件分页查审计日志 id 倒序 复用 gen FindByPage 末页不满免 COUNT
func (r *operationLogRepositoryImpl) Page(filter types.OperationLogFilter, offset, limit int) ([]*model.RanFeedOperationLog, int64, error) {
	q := query.Q.RanFeedOperationLog
	do := q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0))
	if filter.AdminID > 0 {
		do = do.Where(q.AdminID.Eq(filter.AdminID))
	}
	if filter.Action != "" {
		do = do.Where(q.Action.Eq(filter.Action))
	}
	if filter.Status > 0 {
		do = do.Where(q.Status.Eq(filter.Status))
	}
	if filter.StartMillis > 0 {
		do = do.Where(q.CreatedAt.Gte(time.UnixMilli(filter.StartMillis)))
	}
	if filter.EndMillis > 0 {
		do = do.Where(q.CreatedAt.Lte(time.UnixMilli(filter.EndMillis)))
	}
	return do.Order(q.ID.Desc()).FindByPage(offset, limit)
}
