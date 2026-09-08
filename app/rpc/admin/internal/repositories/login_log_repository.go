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

type LoginLogRepository interface {
	WithTx(tx *query.Query) LoginLogRepository
	Create(row *model.RanFeedLoginLog) (int64, error)
	// Page 按条件分页查登录日志 id 倒序 返回列表与总数
	Page(filter types.LoginLogFilter, offset, limit int) ([]*model.RanFeedLoginLog, int64, error)
}

type loginLogRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
}

func NewLoginLogRepository(ctx context.Context, db *orm.DB) LoginLogRepository {
	return &loginLogRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *loginLogRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *loginLogRepositoryImpl) WithTx(tx *query.Query) LoginLogRepository {
	return &loginLogRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// Create 落一条登录日志 返回ID
func (r *loginLogRepositoryImpl) Create(row *model.RanFeedLoginLog) (int64, error) {
	if row == nil {
		return 0, nil
	}
	if err := r.getQuery().RanFeedLoginLog.WithContext(r.ctx).Create(row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

// Page 按条件分页查登录日志 id 倒序 复用 gen FindByPage 末页不满免 COUNT
func (r *loginLogRepositoryImpl) Page(filter types.LoginLogFilter, offset, limit int) ([]*model.RanFeedLoginLog, int64, error) {
	q := r.getQuery().RanFeedLoginLog
	do := q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0))
	if filter.Username != "" {
		do = do.Where(q.Username.Like("%" + filter.Username + "%"))
	}
	if filter.IP != "" {
		do = do.Where(q.IP.Like("%" + filter.IP + "%"))
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
