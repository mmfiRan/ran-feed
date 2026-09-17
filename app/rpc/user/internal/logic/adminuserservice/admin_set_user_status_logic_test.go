package adminuserservicelogic

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/user/internal/common/consts/redis"
	"ran-feed/app/rpc/user/internal/common/utils/session"
	"ran-feed/app/rpc/user/internal/do"
	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/internal/entity/query"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
)

func TestValidateUserStatusTransition(t *testing.T) {
	const canceled = user.UserStatus_USER_STATUS_CANCELLED // 注销

	tests := []struct {
		name     string
		cur      user.UserStatus
		target   user.UserStatus
		wantNoop bool
		wantErr  bool
	}{
		{"封禁 正常->禁用", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_DISABLED, false, false},
		{"恢复 禁用->正常", user.UserStatus_USER_STATUS_DISABLED, user.UserStatus_USER_STATUS_ACTIVE, false, false},
		{"封禁 幂等 已禁用->禁用", user.UserStatus_USER_STATUS_DISABLED, user.UserStatus_USER_STATUS_DISABLED, true, false},
		{"恢复 幂等 已正常->正常", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_ACTIVE, true, false},
		{"封禁 注销不可封禁", canceled, user.UserStatus_USER_STATUS_DISABLED, false, true},
		{"恢复 注销不可复活", canceled, user.UserStatus_USER_STATUS_ACTIVE, false, true},
		{"恢复 正常不可恢复", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_ACTIVE, true, false},
		{"不支持的目标态 UNSPECIFIED", user.UserStatus_USER_STATUS_ACTIVE, user.UserStatus_USER_STATUS_UNSPECIFIED, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noop, err := validateUserStatusTransition(tt.cur, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantNoop, noop)
		})
	}
}

type mockUserRepo struct {
	row        *model.RanFeedUser
	getErr     error
	updateErr  error
	getCalls   int
	updateCall int
	updatedTo  int32
}

func (m *mockUserRepo) WithTx(*query.Query) repositories.UserRepository { return m }
func (m *mockUserRepo) GetByMobile(string) (*do.UserDO, error)          { return nil, nil }
func (m *mockUserRepo) GetByID(int64) (*do.UserDO, error)               { return nil, nil }
func (m *mockUserRepo) BatchGetByIDs([]int64) (map[int64]*do.UserDO, error) {
	return nil, nil
}
func (m *mockUserRepo) BatchGetActiveForIndex([]int64) (map[int64]*model.RanFeedUser, error) {
	return nil, nil
}
func (m *mockUserRepo) ScanActiveForIndex(int64, int) ([]*model.RanFeedUser, error) {
	return nil, nil
}
func (m *mockUserRepo) Create(*do.UserDO) (int64, error) { return 0, nil }
func (m *mockUserRepo) AdminPageUsers(*int32, *string, *string, int, int) ([]*model.RanFeedUser, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) AdminGetByID(int64) (*model.RanFeedUser, error) {
	m.getCalls++
	return m.row, m.getErr
}
func (m *mockUserRepo) AdminUpdateStatus(_ int64, status int32, _ int64) (int64, error) {
	m.updateCall++
	m.updatedTo = status
	return 1, m.updateErr
}

func newTestLogic(t *testing.T, repo *mockUserRepo) (*AdminSetUserStatusLogic, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	return &AdminSetUserStatusLogic{
		ctx:      ctx,
		svcCtx:   &svc.ServiceContext{Redis: r},
		userRepo: repo,
		Logger:   logx.WithContext(ctx),
	}, mr
}

func TestAdminSetUserStatus(t *testing.T) {
	active := user.UserStatus_USER_STATUS_ACTIVE
	disabled := user.UserStatus_USER_STATUS_DISABLED

	tests := []struct {
		name         string
		row          *model.RanFeedUser
		req          *user.AdminSetUserStatusReq
		wantErr      string
		wantCode     int32
		wantName     string
		wantMessage  string
		wantUpdates  int
		wantGetCalls int
	}{
		{
			name:         "封禁返回 20 禁用",
			row:          &model.RanFeedUser{ID: 7, Status: int32(active)},
			req:          &user.AdminSetUserStatusReq{UserId: 7, Status: disabled, OperatorId: 1},
			wantCode:     20,
			wantName:     "DISABLED",
			wantMessage:  "禁用",
			wantUpdates:  1,
			wantGetCalls: 1,
		},
		{
			name:         "恢复返回 10 正常",
			row:          &model.RanFeedUser{ID: 7, Status: int32(disabled)},
			req:          &user.AdminSetUserStatusReq{UserId: 7, Status: active, OperatorId: 1},
			wantCode:     10,
			wantName:     "ACTIVE",
			wantMessage:  "正常",
			wantUpdates:  1,
			wantGetCalls: 1,
		},
		{
			name:         "已是目标态 幂等不落库 仍回显状态",
			row:          &model.RanFeedUser{ID: 7, Status: int32(disabled)},
			req:          &user.AdminSetUserStatusReq{UserId: 7, Status: disabled, OperatorId: 1},
			wantCode:     20,
			wantName:     "DISABLED",
			wantMessage:  "禁用",
			wantUpdates:  0,
			wantGetCalls: 1,
		},
		{
			name:         "用户不存在",
			row:          nil,
			req:          &user.AdminSetUserStatusReq{UserId: 7, Status: disabled, OperatorId: 1},
			wantErr:      "用户不存在",
			wantGetCalls: 1,
		},
		{
			name:         "注销账号不可恢复",
			row:          &model.RanFeedUser{ID: 7, Status: int32(user.UserStatus_USER_STATUS_CANCELLED)},
			req:          &user.AdminSetUserStatusReq{UserId: 7, Status: active, OperatorId: 1},
			wantErr:      "仅已封禁用户可恢复",
			wantGetCalls: 1,
		},
		{
			name:         "user_id 非法 参数错误",
			req:          &user.AdminSetUserStatusReq{UserId: 0, Status: disabled, OperatorId: 1},
			wantErr:      "参数错误",
			wantGetCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepo{row: tt.row}
			logic, _ := newTestLogic(t, repo)

			resp, err := logic.AdminSetUserStatus(tt.req)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp.GetStatus())
				assert.Equal(t, tt.wantCode, resp.GetStatus().GetCode())
				assert.Equal(t, tt.wantName, resp.GetStatus().GetName())
				assert.Equal(t, tt.wantMessage, resp.GetStatus().GetMessage())
			}
			assert.Equal(t, tt.wantUpdates, repo.updateCall)
			assert.Equal(t, tt.wantGetCalls, repo.getCalls)
		})
	}
}

func TestAdminSetUserStatusDisableKicksSession(t *testing.T) {
	ctx := context.Background()
	const userID = int64(7)
	token := session.NewSessionToken()

	repo := &mockUserRepo{row: &model.RanFeedUser{ID: userID, Status: int32(user.UserStatus_USER_STATUS_ACTIVE)}}
	logic, mr := newTestLogic(t, repo)

	require.NoError(t, session.SaveSession(ctx, logic.svcCtx.Redis, userID, token, time.Hour))
	tokenKey := rediskey.BuildUserSessionKey(token)
	userKey := rediskey.BuildUserSessionUserKey(userID)
	require.True(t, mr.Exists(tokenKey))
	require.True(t, mr.Exists(userKey))

	resp, err := logic.AdminSetUserStatus(&user.AdminSetUserStatusReq{
		UserId:     userID,
		Status:     user.UserStatus_USER_STATUS_DISABLED,
		OperatorId: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(20), resp.GetStatus().GetCode())

	assert.False(t, mr.Exists(tokenKey), "踢下线后 token 会话应被删除")
	assert.False(t, mr.Exists(userKey), "踢下线后用户反向索引应被删除")
}
