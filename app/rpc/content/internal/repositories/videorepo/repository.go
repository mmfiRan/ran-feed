package videorepo

import (
	"context"
	"errors"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/pkg/enums"

	"gorm.io/gorm"
)

var _ repositories.VideoRepository = (*Repository)(nil)

type Repository struct {
	tx *query.Query
}

func New() *Repository {
	return &Repository{}
}

func (r *Repository) WithTx(tx *query.Query) repositories.VideoRepository {
	return &Repository{
		tx: tx,
	}
}

func (r *Repository) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *Repository) CreateVideo(ctx context.Context, video *model.RanFeedVideo) error {
	q := r.getQuery()
	return q.RanFeedVideo.WithContext(ctx).Create(video)
}

func (r *Repository) UpdateByContentID(ctx context.Context, video *model.RanFeedVideo) error {
	if video.ContentID <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedVideo.WithContext(ctx).
		Where(q.RanFeedVideo.ContentID.Eq(video.ContentID)).
		Where(q.RanFeedVideo.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedVideo.Title.Value(video.Title),
			q.RanFeedVideo.OriginURL.Value(video.OriginURL),
			q.RanFeedVideo.CoverURL.Value(video.CoverURL),
			q.RanFeedVideo.Duration.Value(video.Duration),
		)
	return err
}

func (r *Repository) DeleteByContentID(ctx context.Context, contentID int64) error {
	q := r.getQuery()
	_, err := q.RanFeedVideo.WithContext(ctx).
		Where(q.RanFeedVideo.ContentID.Eq(contentID)).
		UpdateSimple(q.RanFeedVideo.IsDeleted.Value(enums.Deleted.Int32()))
	return err
}

func (r *Repository) GetByContentID(ctx context.Context, contentID int64) (*model.RanFeedVideo, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedVideo.WithContext(ctx).
		Where(q.RanFeedVideo.ContentID.Eq(contentID)).
		Where(q.RanFeedVideo.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *Repository) BatchGetBriefByContentIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedVideo, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedVideo{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedVideo.WithContext(ctx).
		Select(q.RanFeedVideo.ContentID, q.RanFeedVideo.Title, q.RanFeedVideo.CoverURL).
		Where(q.RanFeedVideo.ContentID.In(contentIDs...)).
		Where(q.RanFeedVideo.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
