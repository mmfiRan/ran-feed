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

type ArticleRepository interface {
	WithTx(tx *query.Query) ArticleRepository
	CreateArticle(articleDO *do.ArticleDO) error
	DeleteByContentID(contentID int64) error
	GetByContentID(contentID int64) (*model.RanFeedArticle, error)
	BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error)
}

type ArticleRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
}

func NewArticleRepository(ctx context.Context, db *orm.DB) ArticleRepository {
	return &ArticleRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *ArticleRepositoryImpl) WithTx(tx *query.Query) ArticleRepository {
	return &ArticleRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *ArticleRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *ArticleRepositoryImpl) CreateArticle(articleDO *do.ArticleDO) error {
	q := r.getQuery()

	articleModel := &model.RanFeedArticle{
		ID:          articleDO.ID,
		ContentID:   articleDO.ContentID,
		Title:       articleDO.Title,
		Description: articleDO.Description,
		Cover:       articleDO.Cover,
		Content:     articleDO.Content,
	}

	return q.RanFeedArticle.WithContext(r.ctx).Create(articleModel)
}

func (r *ArticleRepositoryImpl) DeleteByContentID(contentID int64) error {
	if contentID <= 0 {
		return nil
	}

	q := r.getQuery()
	_, err := q.RanFeedArticle.WithContext(r.ctx).
		Where(q.RanFeedArticle.ContentID.Eq(contentID)).
		UpdateSimple(q.RanFeedArticle.IsDeleted.Value(1))
	return err
}

func (r *ArticleRepositoryImpl) GetByContentID(contentID int64) (*model.RanFeedArticle, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedArticle.WithContext(r.ctx).
		Where(q.RanFeedArticle.ContentID.Eq(contentID)).
		Where(q.RanFeedArticle.IsDeleted.Eq(0)).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *ArticleRepositoryImpl) BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedArticle{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedArticle.WithContext(r.ctx).
		Select(q.RanFeedArticle.ContentID, q.RanFeedArticle.Title, q.RanFeedArticle.Cover).
		Where(q.RanFeedArticle.ContentID.In(contentIDs...)).
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
