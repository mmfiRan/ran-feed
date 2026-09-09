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
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
)

func newTestRegisterLogic(t *testing.T, repo repositories.UserRepository) (*RegisterLogic, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	return &RegisterLogic{
		ctx:      ctx,
		svcCtx:   &svc.ServiceContext{Config: config.Config{SessionTTL: 3600}, Redis: r},
		Logger:   logx.WithContext(ctx),
		userRepo: repo,
	}, mr
}

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return nil, nil },
		createFn:      func(_ *do.UserDO) (int64, error) { return 100, nil },
	}
	logic, _ := newTestRegisterLogic(t, repo)

	res, err := logic.Register(&user.RegisterReq{
		Mobile:   "13800138001",
		Password: "pass123",
		Nickname: "tester",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(100), res.UserId)
	assert.NotEmpty(t, res.Token)
	assert.Greater(t, res.ExpiredAt, int64(0))
}

func TestRegister_MobileAlreadyExists(t *testing.T) {
	existing := &do.UserDO{ID: 1, Mobile: "13800138001"}
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return existing, nil },
	}
	logic, _ := newTestRegisterLogic(t, repo)

	_, err := logic.Register(&user.RegisterReq{
		Mobile:   "13800138001",
		Password: "pass123",
	})
	assert.Error(t, err)
}

func TestRegister_NilRequest(t *testing.T) {
	logic, _ := newTestRegisterLogic(t, &mockUserRepository{})
	_, err := logic.Register(nil)
	assert.Error(t, err)
}

func TestRegister_DefaultNickname(t *testing.T) {
	var captured *do.UserDO
	repo := &mockUserRepository{
		getByMobileFn: func(_ string) (*do.UserDO, error) { return nil, nil },
		createFn: func(u *do.UserDO) (int64, error) {
			captured = u
			return 101, nil
		},
	}
	logic, _ := newTestRegisterLogic(t, repo)

	_, err := logic.Register(&user.RegisterReq{
		Mobile:   "13900139000",
		Password: "pass456",
		Nickname: "", // 空昵称应默认为手机号
	})
	require.NoError(t, err)
	assert.Equal(t, "13900139000", captured.Nickname)
}
