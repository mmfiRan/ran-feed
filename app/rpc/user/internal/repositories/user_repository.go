package repositories

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"ran-feed/app/rpc/user/internal/do"
	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/internal/entity/query"
	"ran-feed/pkg/orm"
)

const (
	UserStatusActive int32 = 10
)

type UserRepository interface {
	WithTx(tx *query.Query) UserRepository
	// GetByMobile 根据手机号查询用户
	GetByMobile(mobile string) (*do.UserDO, error)
	// GetByID 根据用户ID查询用户
	GetByID(userID int64) (*do.UserDO, error)
	// BatchGetByIDs 批量根据用户ID查询
	BatchGetByIDs(userIDs []int64) (map[int64]*do.UserDO, error)
	// Create 创建用户
	Create(userDO *do.UserDO) (int64, error)
}

type userRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewUserRepository(ctx context.Context, db *orm.DB) UserRepository {
	return &userRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *userRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *userRepositoryImpl) WithTx(tx *query.Query) UserRepository {
	return &userRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *userRepositoryImpl) GetByMobile(mobile string) (*do.UserDO, error) {
	q := r.getQuery()

	row, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.Mobile.Eq(mobile)).
		Where(q.RanFeedUser.IsDeleted.Eq(0)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &do.UserDO{
		ID:           row.ID,
		Username:     row.Username,
		Nickname:     row.Nickname,
		Avatar:       row.Avatar,
		Bio:          row.Bio,
		Mobile:       row.Mobile,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		PasswordSalt: row.PasswordSalt,
		Gender:       row.Gender,
		Birthday:     row.Birthday,
		Status:       row.Status,
		CreatedBy:    row.CreatedBy,
		UpdatedBy:    row.UpdatedBy,
	}, nil
}

func (r *userRepositoryImpl) GetByID(userID int64) (*do.UserDO, error) {
	if userID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.ID.Eq(userID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	return &do.UserDO{
		ID:           row.ID,
		Username:     row.Username,
		Nickname:     row.Nickname,
		Avatar:       row.Avatar,
		Bio:          row.Bio,
		Mobile:       row.Mobile,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		PasswordSalt: row.PasswordSalt,
		Gender:       row.Gender,
		Birthday:     row.Birthday,
		Status:       row.Status,
		CreatedBy:    row.CreatedBy,
		UpdatedBy:    row.UpdatedBy,
	}, nil
}

func (r *userRepositoryImpl) BatchGetByIDs(userIDs []int64) (map[int64]*do.UserDO, error) {
	if len(userIDs) == 0 {
		return map[int64]*do.UserDO{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.ID.In(userIDs...)).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*do.UserDO, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ID] = &do.UserDO{
			ID:           row.ID,
			Username:     row.Username,
			Nickname:     row.Nickname,
			Avatar:       row.Avatar,
			Bio:          row.Bio,
			Mobile:       row.Mobile,
			Email:        row.Email,
			PasswordHash: row.PasswordHash,
			PasswordSalt: row.PasswordSalt,
			Gender:       row.Gender,
			Birthday:     row.Birthday,
			Status:       row.Status,
			CreatedBy:    row.CreatedBy,
			UpdatedBy:    row.UpdatedBy,
		}
	}
	return res, nil
}

func (r *userRepositoryImpl) Create(userDO *do.UserDO) (int64, error) {
	q := r.getQuery()

	row := &model.RanFeedUser{
		Username:     userDO.Username,
		Nickname:     userDO.Nickname,
		Avatar:       userDO.Avatar,
		Bio:          userDO.Bio,
		Mobile:       userDO.Mobile,
		Email:        userDO.Email,
		PasswordHash: userDO.PasswordHash,
		PasswordSalt: userDO.PasswordSalt,
		Gender:       userDO.Gender,
		Birthday:     userDO.Birthday,
		Status:       userDO.Status,
	}

	if err := q.RanFeedUser.WithContext(r.ctx).Create(row); err != nil {
		return 0, err
	}

	return row.ID, nil
}
