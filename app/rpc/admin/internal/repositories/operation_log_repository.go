package repositories

import (
	"context"
	"time"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

// OperationLogFilter 审计日志查询条件 零值即不限 时间为毫秒
type OperationLogFilter struct {
	AdminID     int64
	Action      string
	TargetType  string
	StartMillis int64
	EndMillis   int64
}

type OperationLogRepository interface {
	Create(row *model.RanFeedOperationLog) (int64, error)
	// List 按条件分页查审计日志 id 倒序
	List(filter OperationLogFilter, offset, limit int) ([]*model.RanFeedOperationLog, error)
	// Count 按条件统计总数
	Count(filter OperationLogFilter) (int64, error)
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

// buildQuery 按条件组装查询 软删过滤加可选筛选加时间区间
func (r *operationLogRepositoryImpl) buildQuery(filter OperationLogFilter) query.IRanFeedOperationLogDo {
	q := query.Q.RanFeedOperationLog
	do := q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0))
	if filter.AdminID > 0 {
		do = do.Where(q.AdminID.Eq(filter.AdminID))
	}
	if filter.Action != "" {
		do = do.Where(q.Action.Eq(filter.Action))
	}
	if filter.TargetType != "" {
		do = do.Where(q.TargetType.Eq(filter.TargetType))
	}
	if filter.StartMillis > 0 {
		do = do.Where(q.CreatedAt.Gte(time.UnixMilli(filter.StartMillis)))
	}
	if filter.EndMillis > 0 {
		do = do.Where(q.CreatedAt.Lte(time.UnixMilli(filter.EndMillis)))
	}
	return do
}

// List 按条件分页查审计日志 id 倒序
func (r *operationLogRepositoryImpl) List(filter OperationLogFilter, offset, limit int) ([]*model.RanFeedOperationLog, error) {
	q := query.Q.RanFeedOperationLog
	return r.buildQuery(filter).Order(q.ID.Desc()).Offset(offset).Limit(limit).Find()
}

// Count 按条件统计总数
func (r *operationLogRepositoryImpl) Count(filter OperationLogFilter) (int64, error) {
	return r.buildQuery(filter).Count()
}
