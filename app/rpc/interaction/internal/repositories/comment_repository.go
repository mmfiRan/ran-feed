package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/pkg/orm"
)

const commentStatusDeleted = 20

type CommentRepository interface {
	WithTx(tx *query.Query) CommentRepository
	Create(commentDO *do.CommentDO) (int64, error)
	GetByID(id int64) (*do.CommentDO, error)
	MarkDeleted(id int64, updatedBy int64) error
	DeleteByID(id int64) error
	HasReferences(id int64) (bool, error)
	ListRootByContentID(contentID int64, cursor int64, limit int) ([]*model.RanFeedComment, error)
	ListReplyByRootID(rootID int64, cursor int64, limit int) ([]*model.RanFeedComment, error)
	ListByIDs(ids []int64) ([]*model.RanFeedComment, error)
	BatchCountByParentIDs(parentIDs []int64) (map[int64]int64, error)
	BatchCountByRootIDs(rootIDs []int64) (map[int64]int64, error)
}

type commentRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewCommentRepository(ctx context.Context, db *orm.DB) CommentRepository {
	return &commentRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *commentRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *commentRepositoryImpl) WithTx(tx *query.Query) CommentRepository {
	return &commentRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *commentRepositoryImpl) Create(commentDO *do.CommentDO) (int64, error) {
	if commentDO == nil {
		return 0, nil
	}

	q := r.getQuery()
	row := &model.RanFeedComment{
		ContentID:     commentDO.ContentID,
		ContentUserID: commentDO.ContentUserID,
		UserID:        commentDO.UserID,
		ReplyToUserID: commentDO.ReplyToUserID,
		ParentID:      commentDO.ParentID,
		RootID:        commentDO.RootID,
		Comment:       commentDO.Comment,
		Status:        commentDO.Status,
		Version:       commentDO.Version,
		IsDeleted:     commentDO.IsDeleted,
		CreatedBy:     commentDO.CreatedBy,
		UpdatedBy:     commentDO.UpdatedBy,
	}
	if err := q.RanFeedComment.WithContext(r.ctx).Create(row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (r *commentRepositoryImpl) GetByID(id int64) (*do.CommentDO, error) {
	if id <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.ID.Eq(id)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &do.CommentDO{
		ID:            row.ID,
		ContentID:     row.ContentID,
		ContentUserID: row.ContentUserID,
		UserID:        row.UserID,
		ReplyToUserID: row.ReplyToUserID,
		ParentID:      row.ParentID,
		RootID:        row.RootID,
		Comment:       row.Comment,
		Status:        row.Status,
		Version:       row.Version,
		IsDeleted:     row.IsDeleted,
		CreatedBy:     row.CreatedBy,
		UpdatedBy:     row.UpdatedBy,
	}, nil
}

func (r *commentRepositoryImpl) MarkDeleted(id int64, updatedBy int64) error {
	if id <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.ID.Eq(id)).
		UpdateSimple(q.RanFeedComment.IsDeleted.Value(1))
	return err
}

func (r *commentRepositoryImpl) DeleteByID(id int64) error {
	if id <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.ID.Eq(id)).
		Delete()
	return err
}

func (r *commentRepositoryImpl) HasReferences(id int64) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.IsDeleted.Eq(0)).
		Where(
			q.RanFeedComment.ParentID.Eq(id),
		).
		Or(
			q.RanFeedComment.RootID.Eq(id),
		).
		Limit(1).
		Find()
	if err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func (r *commentRepositoryImpl) ListRootByContentID(contentID int64, cursor int64, limit int) ([]*model.RanFeedComment, error) {
	if contentID <= 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	q := r.getQuery()
	doQuery := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.ContentID.Eq(contentID)).
		Where(q.RanFeedComment.ParentID.Eq(0))

	if cursor > 0 {
		doQuery = doQuery.Where(q.RanFeedComment.ID.Lt(cursor))
	}

	rows, err := doQuery.
		Order(q.RanFeedComment.ID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *commentRepositoryImpl) ListByIDs(ids []int64) ([]*model.RanFeedComment, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.ID.In(ids...)).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *commentRepositoryImpl) ListReplyByRootID(rootID int64, cursor int64, limit int) ([]*model.RanFeedComment, error) {
	if rootID <= 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	q := r.getQuery()
	doQuery := q.RanFeedComment.WithContext(r.ctx).
		Where(q.RanFeedComment.RootID.Eq(rootID))

	if cursor > 0 {
		doQuery = doQuery.Where(q.RanFeedComment.ID.Lt(cursor))
	}

	rows, err := doQuery.
		Order(q.RanFeedComment.ID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *commentRepositoryImpl) BatchCountByParentIDs(parentIDs []int64) (map[int64]int64, error) {
	res := make(map[int64]int64)
	if len(parentIDs) == 0 {
		return res, nil
	}

	type countRow struct {
		ParentID int64 `gorm:"column:parent_id"`
		Cnt      int64 `gorm:"column:cnt"`
	}
	rows := make([]countRow, 0)
	q := r.getQuery()
	err := q.RanFeedComment.WithContext(r.ctx).
		Select(q.RanFeedComment.ParentID, q.RanFeedComment.ID.Count().As("cnt")).
		Where(q.RanFeedComment.IsDeleted.Eq(0)).
		Where(q.RanFeedComment.Status.Neq(commentStatusDeleted)).
		Where(q.RanFeedComment.ParentID.In(parentIDs...)).
		Group(q.RanFeedComment.ParentID).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		res[row.ParentID] = row.Cnt
	}
	return res, nil
}

func (r *commentRepositoryImpl) BatchCountByRootIDs(rootIDs []int64) (map[int64]int64, error) {
	res := make(map[int64]int64)
	if len(rootIDs) == 0 {
		return res, nil
	}

	type countRow struct {
		RootID int64 `gorm:"column:root_id"`
		Cnt    int64 `gorm:"column:cnt"`
	}
	rows := make([]countRow, 0)
	q := r.getQuery()
	err := q.RanFeedComment.WithContext(r.ctx).
		Select(q.RanFeedComment.RootID, q.RanFeedComment.ID.Count().As("cnt")).
		Where(q.RanFeedComment.IsDeleted.Eq(0)).
		Where(q.RanFeedComment.Status.Neq(commentStatusDeleted)).
		Where(q.RanFeedComment.RootID.In(rootIDs...)).
		Group(q.RanFeedComment.RootID).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		res[row.RootID] = row.Cnt
	}
	return res, nil
}
