package repositories

import (
	"context"
	"errors"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/enums"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ArticleRepository interface {
	WithTx(tx *query.Query) ArticleRepository
	CreateArticle(articleDO *do.ArticleDO) error
	UpdateByContentID(articleDO *do.ArticleDO) error
	DeleteByContentID(contentID int64) error
	GetByContentID(contentID int64) (*model.RanFeedArticle, error)
	BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error)
	BatchGetIndexByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error)
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

// UpdateByContentID 草稿编辑更新文章子表 按 content_id 定位
func (r *ArticleRepositoryImpl) UpdateByContentID(articleDO *do.ArticleDO) error {
	if articleDO.ContentID <= 0 {
		return nil
	}
	q := r.getQuery()
	description := ""
	if articleDO.Description != nil {
		description = *articleDO.Description
	}
	_, err := q.RanFeedArticle.WithContext(r.ctx).
		Where(q.RanFeedArticle.ContentID.Eq(articleDO.ContentID)).
		Where(q.RanFeedArticle.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedArticle.Title.Value(articleDO.Title),
			q.RanFeedArticle.Description.Value(description),
			q.RanFeedArticle.Cover.Value(articleDO.Cover),
			q.RanFeedArticle.Content.Value(articleDO.Content),
		)
	return err
}

func (r *ArticleRepositoryImpl) DeleteByContentID(contentID int64) error {
	q := r.getQuery()
	_, err := q.RanFeedArticle.WithContext(r.ctx).
		Where(q.RanFeedArticle.ContentID.Eq(contentID)).
		UpdateSimple(q.RanFeedArticle.IsDeleted.Value(enums.Deleted.Int32()))
	return err
}

func (r *ArticleRepositoryImpl) GetByContentID(contentID int64) (*model.RanFeedArticle, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedArticle.WithContext(r.ctx).
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

func (r *ArticleRepositoryImpl) BatchGetBriefByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedArticle{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedArticle.WithContext(r.ctx).
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

// BatchGetIndexByContentIDs 建索引取标题 摘要 正文 供搜索文档组装
func (r *ArticleRepositoryImpl) BatchGetIndexByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedArticle{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedArticle.WithContext(r.ctx).
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
