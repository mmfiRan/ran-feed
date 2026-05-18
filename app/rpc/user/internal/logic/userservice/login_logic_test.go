package userservicelogic

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/app/rpc/user/internal/config"
	"ran-feed/app/rpc/user/internal/do"
	"ran-feed/app/rpc/user/internal/entity/query"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/utils"
)

// mockUserRepository 实现 repositories.UserRepository，仅用于测试
type mockUserRepository struct {
	getByMobileFn  func(mobile string) (*do.UserDO, error)
	getByIDFn      func(userID int64) (*do.UserDO, error)
	batchGetByIDs  func(userIDs []int64) (map[int64]*do.UserDO, error)
	createFn       func(userDO *do.UserDO) (int64, error)
}

func (m *mockUserRepository) WithTx(_ *query.Query) repositories.UserRepository { return m }
func (m *mockUserRepository) GetByMobile(mobile string) (*do.UserDO, error) {
	return m.getByMobileFn(mobile)
}
func (m *mockUserRepository) GetByID(userID int64) (*do.UserDO, error) {
	return m.getByIDFn(userID)
}
func (m *mockUserRepository) BatchGetByIDs(userIDs []int64) (map[int64]*do.UserDO, error) {
	return m.batchGetByIDs(userIDs)
}
func (m *mockUserRepository) Create(userDO *do.UserDO) (int64, error) {
	return m.createFn(userDO)
}

func newTestLoginLogic(t *testing.T, repo repositories.UserRepository) (*LoginLogic, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	return &LoginLogic{
		ctx:      ctx,
		svcCtx:   &svc.ServiceContext{Config: config.Config{}, Redis: r},
		Logger:   logx.WithContext(ctx),
		userRepo: repo,
	}, mr
}

func testActiveUser(password string) *do.UserDO {
	salt := "testsalt"
	hash, _ := utils.HashPassword(password + salt)
	return &do.UserDO{
		ID:           1,
		Mobile:       "13800138000",
		PasswordHash: hash,
		PasswordSalt: salt,
		Status:       int32(user.UserStatus_USER_STATUS_ACTIVE),
		Nickname:     "tester",
		Avatar:       "",
	}
}

func TestLogin_Success(t *testing.T) {
	u := testActiveUser("correct-password")
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return u, nil },
	}
	logic, _ := newTestLoginLogic(t, repo)

	res, err := logic.Login(&user.LoginReq{
		Mobile:   "13800138000",
		Password: "correct-password",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.UserId)
	assert.NotEmpty(t, res.Token)
	assert.Greater(t, res.ExpiredAt, int64(0))
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return nil, nil },
	}
	logic, _ := newTestLoginLogic(t, repo)

	_, err := logic.Login(&user.LoginReq{Mobile: "13800138000", Password: "any"})
	assert.Error(t, err)
}

func TestLogin_WrongPassword(t *testing.T) {
	u := testActiveUser("correct-password")
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return u, nil },
	}
	logic, _ := newTestLoginLogic(t, repo)

	_, err := logic.Login(&user.LoginReq{Mobile: "13800138000", Password: "wrong-password"})
	assert.Error(t, err)
}

func TestLogin_DisabledAccount(t *testing.T) {
	u := testActiveUser("password")
	u.Status = int32(user.UserStatus_USER_STATUS_DISABLED)
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return u, nil },
	}
	logic, _ := newTestLoginLogic(t, repo)

	_, err := logic.Login(&user.LoginReq{Mobile: "13800138000", Password: "password"})
	assert.Error(t, err)
}

func TestLogin_NilRequest(t *testing.T) {
	logic, _ := newTestLoginLogic(t, &mockUserRepository{})
	_, err := logic.Login(nil)
	assert.Error(t, err)
}