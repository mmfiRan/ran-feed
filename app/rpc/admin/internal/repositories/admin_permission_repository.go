package repositories

import (
	"context"

	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminPermissionRepository interface {
	ListCodesByIDs(ids []int64) ([]string, error)
}

type adminPermissionRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewAdminPermissionRepository(ctx context.Context, db *orm.DB) AdminPermissionRepository {
	return &adminPermissionRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// ListCodesByIDs 按权限点ID集合取 code 集合
func (r *adminPermissionRepositoryImpl) ListCodesByIDs(ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := query.Q
	rows, err := q.RanFeedAdminPermission.WithContext(r.ctx).
		Select(q.RanFeedAdminPermission.Code).
		Where(q.RanFeedAdminPermission.ID.In(ids...)).
		Where(q.RanFeedAdminPermission.IsDeleted.Eq(0)).
		Find()
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.Code != "" {
			codes = append(codes, row.Code)
		}
	}
	return codes, nil
}
