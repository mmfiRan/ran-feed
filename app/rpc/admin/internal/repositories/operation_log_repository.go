package repositories

import (
	"context"
	"ran-feed/pkg/enums"
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
	Page(filter types.OperationLogFilter, offset, limit int) ([]*types.OperationLogRow, int64, error)
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

// Page 按条件分页查审计日志 关联操作人用户名 id 倒序 复用 gen ScanByPage 末页不满免 COUNT
func (r *operationLogRepositoryImpl) Page(filter types.OperationLogFilter, offset, limit int) ([]*types.OperationLogRow, int64, error) {
	logT := r.getQuery().RanFeedOperationLog
	userT := r.getQuery().RanFeedAdminUser
	do := logT.WithContext(r.ctx).Where(logT.IsDeleted.Eq(enums.NotDeleted.Int32()))
	do = do.LeftJoin(userT, logT.AdminID.EqCol(userT.ID))
	do = do.Where(userT.IsDeleted.Eq(enums.NotDeleted.Int32()))
	if filter.Username != "" {
		do = do.Where(userT.Username.Like("%" + filter.Username + "%"))
	}
	if filter.Action != "" {
		do = do.Where(logT.Action.Like("%" + filter.Action + "%"))
	}
	if filter.Status > 0 {
		do = do.Where(logT.Status.Eq(filter.Status))
	}
	if filter.StartMillis > 0 {
		do = do.Where(logT.CreatedAt.Gte(time.UnixMilli(filter.StartMillis)))
	}
	if filter.EndMillis > 0 {
		do = do.Where(logT.CreatedAt.Lte(time.UnixMilli(filter.EndMillis)))
	}
	rows := make([]*types.OperationLogRow, 0)
	total, err := do.Select(logT.ALL, userT.Username).
		Order(logT.ID.Desc()).
		ScanByPage(&rows, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
