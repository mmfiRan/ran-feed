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
	// BatchGetActiveForIndex 增量回源 取可索引用户(正常+未删除) 附建索引所需原始字段
	BatchGetActiveForIndex(userIDs []int64) (map[int64]*model.RanFeedUser, error)
	// ScanActiveForIndex 全量重建 按 id 升序游标扫可索引用户
	ScanActiveForIndex(cursorID int64, limit int) ([]*model.RanFeedUser, error)
	// Create 创建用户
	Create(userDO *do.UserDO) (int64, error)
	// AdminListUsers 后台管理列表用户
	AdminListUsers(status int32, keyword string, offset, limit int) ([]*model.RanFeedUser, error)
	// AdminCountUsers 后台管理用户总数
	AdminCountUsers(status int32, keyword string) (int64, error)
	// AdminGetByID 后台管理获取用户详情 任意状态
	AdminGetByID(userID int64) (*model.RanFeedUser, error)
	// AdminUpdateStatus 后台管理更新用户状态
	AdminUpdateStatus(userID int64, status int32, updatedBy int64) (int64, error)
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
		Where(q.RanFeedUser.IsDeleted.Eq(0)).
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
		Where(q.RanFeedUser.IsDeleted.Eq(0)).
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

// BatchGetActiveForIndex 建索引批量取正常未删除用户 只选建索引所需字段(含 updated_at 作 version)
func (r *userRepositoryImpl) BatchGetActiveForIndex(userIDs []int64) (map[int64]*model.RanFeedUser, error) {
	if len(userIDs) == 0 {
		return map[int64]*model.RanFeedUser{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedUser.WithContext(r.ctx).
		Select(q.RanFeedUser.ID, q.RanFeedUser.Nickname, q.RanFeedUser.Bio, q.RanFeedUser.Username, q.RanFeedUser.Status, q.RanFeedUser.UpdatedAt).
		Where(q.RanFeedUser.ID.In(userIDs...)).
		Where(q.RanFeedUser.Status.Eq(UserStatusActive)).
		Where(q.RanFeedUser.IsDeleted.Eq(0)).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedUser, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ID] = row
	}
	return res, nil
}

// ScanActiveForIndex 全量重建 按 id 升序 keyset 游标扫正常未删除用户
func (r *userRepositoryImpl) ScanActiveForIndex(cursorID int64, limit int) ([]*model.RanFeedUser, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedUser.WithContext(r.ctx).
		Select(q.RanFeedUser.ID, q.RanFeedUser.Nickname, q.RanFeedUser.Bio, q.RanFeedUser.Username, q.RanFeedUser.Status, q.RanFeedUser.UpdatedAt).
		Where(q.RanFeedUser.Status.Eq(UserStatusActive)).
		Where(q.RanFeedUser.IsDeleted.Eq(0))

	if cursorID > 0 {
		doQuery = doQuery.Where(q.RanFeedUser.ID.Gt(cursorID))
	}

	return doQuery.Order(q.RanFeedUser.ID).Limit(limit).Find()
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

// adminUserQuery 后台管理用户查询共享筛选
func (r *userRepositoryImpl) adminUserQuery(status int32, keyword string) query.IRanFeedUserDo {
	q := r.getQuery()
	doQuery := q.RanFeedUser.WithContext(r.ctx).Where(q.RanFeedUser.IsDeleted.Eq(0))

	if status > 0 {
		doQuery = doQuery.Where(q.RanFeedUser.Status.Eq(status))
	}

	// keyword 命中 username nickname mobile 任一 子 DO 作为分组条件 OR 被括号包住不破坏外层软删与状态过滤
	if keyword != "" {
		kw := "%" + keyword + "%"
		keywordGroup := q.RanFeedUser.WithContext(r.ctx).
			Where(q.RanFeedUser.Username.Like(kw)).
			Or(q.RanFeedUser.Nickname.Like(kw)).
			Or(q.RanFeedUser.Mobile.Like(kw))
		doQuery = doQuery.Where(keywordGroup)
	}

	return doQuery
}

func (r *userRepositoryImpl) AdminListUsers(status int32, keyword string, offset, limit int) ([]*model.RanFeedUser, error) {
	return r.adminUserQuery(status, keyword).Order(r.getQuery().RanFeedUser.ID.Desc()).Offset(offset).Limit(limit).Find()
}

func (r *userRepositoryImpl) AdminCountUsers(status int32, keyword string) (int64, error) {
	return r.adminUserQuery(status, keyword).Count()
}

func (r *userRepositoryImpl) AdminGetByID(userID int64) (*model.RanFeedUser, error) {
	if userID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.ID.Eq(userID)).
		Where(q.RanFeedUser.IsDeleted.Eq(0)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// AdminUpdateStatus 更新用户状态 updatedBy 为管理员 id 与 C 端用户 id 不同域 真实操作审计走 admin AuditMiddleware
func (r *userRepositoryImpl) AdminUpdateStatus(userID int64, status int32, updatedBy int64) (int64, error) {
	q := r.getQuery()
	result, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.ID.Eq(userID)).
		Where(q.RanFeedUser.IsDeleted.Eq(0)).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_by": updatedBy,
		})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}
