package userservicelogic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/user/internal/common/utils/session"
	"ran-feed/app/rpc/user/internal/do"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterReq) (*user.RegisterRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	mobile := in.GetMobile()
	password := in.GetPassword()
	nickname := in.GetNickname()
	avatar := in.GetAvatar()
	bio := in.GetBio()
	email := in.GetEmail()
	gender := in.GetGender()
	birthday := in.GetBirthday()
	if nickname == "" {
		nickname = mobile
	}

	// 生日默认截取到日
	birthdayTime := l.truncateToDate(time.Unix(birthday, 0))

	// 校验手机号是否已注册
	exist, err := l.userRepo.GetByMobile(mobile)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if exist != nil {
		return nil, errorx.NewMsg("手机号已注册")
	}

	// 生成密码哈希
	passwordSalt, err := l.newPasswordSalt()
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成密码盐失败"))
	}
	passwordHash, err := utils.HashPassword(password + passwordSalt)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("密码加密失败"))
	}

	// 创建用户
	userID, err := l.userRepo.Create(&do.UserDO{
		Username:     mobile,
		Nickname:     nickname,
		Avatar:       avatar,
		Bio:          bio,
		Mobile:       mobile,
		Email:        email,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		Birthday:     birthdayTime,
		Gender:       int32(gender),
		Status:       int32(user.UserStatus_USER_STATUS_ACTIVE),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建用户失败"))
	}

	// 生成并保存登录态 token
	sessionTTL := session.GetSessionTTL(l.svcCtx.Config)
	token := session.NewSessionToken()
	if err = session.SaveSession(l.ctx, l.svcCtx.Redis, userID, token, sessionTTL); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存登录态失败"))
	}

	return &user.RegisterRes{
		UserId:    userID,
		Token:     token,
		ExpiredAt: time.Now().Add(sessionTTL).Unix(),
	}, nil
}

func (l *RegisterLogic) newPasswordSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}

func (l *RegisterLogic) truncateToDate(t time.Time) *time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &day
}
