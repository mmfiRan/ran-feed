package userservicelogic

import (
	"context"
	"time"

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

	u, err := l.userRepo.GetByMobile(mobile)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if u == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	if !utils.CheckPassword(u.PasswordHash, password+u.PasswordSalt) {
		return nil, errorx.NewMsg("密码错误")
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
