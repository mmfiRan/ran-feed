package repositories

import (
	"context"

	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/pkg/enum"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type VideoRepository interface {
	GetByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedVideo, error)
}

type videoRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewVideoRepository(ctx context.Context, db *orm.DB) VideoRepository {
	return &videoRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// GetByContentIDs 按 content_id 批量取未删除视频 供组装内容文档取标题 视频无简介正文
func (r *videoRepositoryImpl) GetByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedVideo, error) {
	res := make(map[int64]*model.RanFeedVideo, len(contentIDs))
	if len(contentIDs) == 0 {
		return res, nil
	}

	q := query.Q
	rows, err := q.RanFeedVideo.WithContext(r.ctx).
		Where(q.RanFeedVideo.IsDeleted.Eq(enum.NotDeleted.Int32())).
		Where(q.RanFeedVideo.ContentID.In(contentIDs...)).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ContentID] = row
	}
	return res, nil
}
