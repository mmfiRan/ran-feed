package articlerepo

import (
	"context"
	"errors"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/pkg/enums"

	"gorm.io/gorm"
)

var _ repositories.ArticleRepository = (*Repository)(nil)

type Repository struct {
	tx *query.Query
}

func New() *Repository {
	return &Repository{}
}

func (r *Repository) WithTx(tx *query.Query) repositories.ArticleRepository {
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

func (r *Repository) CreateArticle(ctx context.Context, article *model.RanFeedArticle) error {
	q := r.getQuery()
	return q.RanFeedArticle.WithContext(ctx).Create(article)
}

func (r *Repository) UpdateByContentID(ctx context.Context, article *model.RanFeedArticle) error {
	if article.ContentID <= 0 {
		return nil
	}
	q := r.getQuery()
	description := ""
	if article.Description != nil {
		description = *article.Description
	}
	_, err := q.RanFeedArticle.WithContext(ctx).
		Where(q.RanFeedArticle.ContentID.Eq(article.ContentID)).
		Where(q.RanFeedArticle.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedArticle.Title.Value(article.Title),
			q.RanFeedArticle.Description.Value(description),
			q.RanFeedArticle.Cover.Value(article.Cover),
			q.RanFeedArticle.Content.Value(article.Content),
		)
	return err
}

func (r *Repository) DeleteByContentID(ctx context.Context, contentID int64) error {
	q := r.getQuery()
	_, err := q.RanFeedArticle.WithContext(ctx).
		Where(q.RanFeedArticle.ContentID.Eq(contentID)).
		UpdateSimple(q.RanFeedArticle.IsDeleted.Value(enums.Deleted.Int32()))
	return err
}

func (r *Repository) GetByContentID(ctx context.Context, contentID int64) (*model.RanFeedArticle, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedArticle.WithContext(ctx).
		Where(q.RanFeedArticle.ContentID.Eq(contentID)).
		Where(q.RanFeedArticle.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *Repository) BatchGetBriefByContentIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedArticle, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedArticle{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedArticle.WithContext(ctx).
		Select(q.RanFeedArticle.ContentID, q.RanFeedArticle.Title, q.RanFeedArticle.Cover).
		Where(q.RanFeedArticle.ContentID.In(contentIDs...)).
		Where(q.RanFeedArticle.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedArticle, len(rows))
	for _, a := range rows {
		if a == nil {
			continue
		}
		res[a.ContentID] = a
	}
	return res, nil
}

func (r *Repository) BatchGetIndexByContentIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedArticle, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedArticle{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedArticle.WithContext(ctx).
		Select(q.RanFeedArticle.ContentID, q.RanFeedArticle.Title, q.RanFeedArticle.Description, q.RanFeedArticle.Content).
		Where(q.RanFeedArticle.ContentID.In(contentIDs...)).
		Where(q.RanFeedArticle.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedArticle, len(rows))
	for _, a := range rows {
		if a == nil {
			continue
		}
		res[a.ContentID] = a
	}
	return res, nil
}
