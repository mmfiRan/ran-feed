package repositories

import (
	"context"

	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/pkg/enum"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentRepository interface {
	ScanPublishable(cursorID int64, limit int) ([]*model.RanFeedContent, error)
}

type contentRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewContentRepository(ctx context.Context, db *orm.DB) ContentRepository {
	return &contentRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// ScanPublishable 按 id 游标批量扫已发布公开未删除内容 供全量重建
func (r *contentRepositoryImpl) ScanPublishable(cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := query.Q
	rows, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.IsDeleted.Eq(enum.NotDeleted.Int32())).
		Where(q.RanFeedContent.Status.Eq(consts.ContentStatusPublished)).
		Where(q.RanFeedContent.Visibility.Eq(consts.ContentVisibilityPublic)).
		Where(q.RanFeedContent.ID.Gt(cursorID)).
		Order(q.RanFeedContent.ID).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}
