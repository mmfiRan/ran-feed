package repositories

import (
	"context"
	"errors"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type VideoRepository interface {
	WithTx(tx *query.Query) VideoRepository
	CreateVideo(videoDO *do.VideoDO) error
	DeleteByContentID(contentID int64) error
	GetByContentID(contentID int64) (*model.RanFeedVideo, error)
	BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedVideo, error)
}

type VideoRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
}

func NewVideoRepository(ctx context.Context, db *orm.DB) VideoRepository {
	return &VideoRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *VideoRepositoryImpl) WithTx(tx *query.Query) VideoRepository {
	return &VideoRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *VideoRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *VideoRepositoryImpl) CreateVideo(videoDO *do.VideoDO) error {
	q := r.getQuery()

	videoModel := &model.RanFeedVideo{
		ID:              videoDO.ID,
		ContentID:       videoDO.ContentID,
		MediaID:         videoDO.MediaID,
		OriginURL:       videoDO.OriginURL,
		CoverURL:        videoDO.CoverURL,
		Duration:        videoDO.Duration,
		TranscodeStatus: videoDO.TranscodeStatus,
	}

	return q.RanFeedVideo.WithContext(r.ctx).Create(videoModel)
}

func (r *VideoRepositoryImpl) DeleteByContentID(contentID int64) error {
	if contentID <= 0 {
		return nil
	}

	q := r.getQuery()
	_, err := q.RanFeedVideo.WithContext(r.ctx).
		Where(q.RanFeedVideo.ContentID.Eq(contentID)).
		UpdateSimple(q.RanFeedVideo.IsDeleted.Value(1))
	return err
}

func (r *VideoRepositoryImpl) GetByContentID(contentID int64) (*model.RanFeedVideo, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedVideo.WithContext(r.ctx).
		Where(q.RanFeedVideo.ContentID.Eq(contentID)).
		Where(q.RanFeedVideo.IsDeleted.Eq(0)).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *VideoRepositoryImpl) BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedVideo, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedVideo{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedVideo.WithContext(r.ctx).
		Select(q.RanFeedVideo.ContentID, q.RanFeedVideo.Title, q.RanFeedVideo.CoverURL).
		Where(q.RanFeedVideo.ContentID.In(contentIDs...)).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedVideo, len(rows))
	for _, v := range rows {
		if v == nil {
			continue
		}
		res[v.ContentID] = v
	}
	return res, nil
}
