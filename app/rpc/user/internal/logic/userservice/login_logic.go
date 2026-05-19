package userservicelogic

import (
	"context"
	"fmt"
	"time"

	"ran-feed/app/rpc/user/internal/common/utils/ratelimit"
	"ran-feed/app/rpc/user/internal/common/utils/session"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	mobile := in.GetMobile()
	password := in.GetPassword()

	rlCfg := l.svcCtx.Config.LoginRateLimit
	if check, rlErr := ratelimit.CheckLogin(l.ctx, l.svcCtx.Redis, rlCfg, mobile); rlErr != nil {
		return nil, errorx.Wrap(l.ctx, rlErr, errorx.NewMsg("登录限频检查失败"))
	} else if !check.Allowed {
		return nil, errorx.NewMsg(fmt.Sprintf("登录尝试过于频繁，请 %d 秒后重试", check.RetryAfter))
	}

	u, err := l.userRepo.GetByMobile(mobile)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if u == nil || !utils.CheckPassword(u.PasswordHash, password+u.PasswordSalt) {
		if _, recErr := ratelimit.RecordLoginFailure(l.ctx, l.svcCtx.Redis, rlCfg, mobile); recErr != nil {
			logx.WithContext(l.ctx).Errorf("记录登录失败计数异常 mobile=%s err=%v", mobile, recErr)
		}
		return nil, errorx.NewMsg("手机号或密码错误")
	}

	if u.Status != int32(user.UserStatus_USER_STATUS_ACTIVE) {
		return nil, errorx.NewMsg("账号已被禁用")
	}

	if clrErr := ratelimit.ClearLoginFailures(l.ctx, l.svcCtx.Redis, rlCfg, mobile); clrErr != nil {
		logx.WithContext(l.ctx).Errorf("清理登录失败计数异常 mobile=%s err=%v", mobile, clrErr)
	}

	sessionTTL := session.GetSessionTTL(l.svcCtx.Config)
	token := session.NewSessionToken()
	if err = session.SaveSession(l.ctx, l.svcCtx.Redis, u.ID, token, sessionTTL); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存登录态失败"))
	}

	return &user.LoginRes{
		UserId:    u.ID,
		Token:     token,
		ExpiredAt: time.Now().Add(sessionTTL).Unix(),
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
	}, nil
}
