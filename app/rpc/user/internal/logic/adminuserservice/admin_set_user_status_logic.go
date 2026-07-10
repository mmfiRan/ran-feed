package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/common/utils/session"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetUserStatusLogic struct {
	ctx      context.Context
	svcCtx   *svc.ServiceContext
	userRepo repositories.UserRepository
	logx.Logger
}

func NewAdminSetUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetUserStatusLogic {
	return &AdminSetUserStatusLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *AdminSetUserStatusLogic) AdminSetUserStatus(in *user.AdminSetUserStatusReq) (*user.AdminSetUserStatusRes, error) {
	if !isValidStatus(in.Status) {
		return nil, errorx.NewMsg("不支持的状态")
	}

	affected, err := l.userRepo.AdminUpdateStatus(in.UserId, int32(in.Status), in.OperatorId)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errorx.NewMsg("用户不存在")
	}

	if in.Status == user.UserStatus_USER_STATUS_DISABLED {
		if err := session.RemoveByUserID(l.ctx, l.svcCtx.Redis, in.UserId); err != nil {
			l.Errorf("踢下线失败 userId=%d err=%v", in.UserId, err)
		}
	}

	return &user.AdminSetUserStatusRes{}, nil
}

func isValidStatus(status user.UserStatus) bool {
	return status == user.UserStatus_USER_STATUS_ACTIVE || status == user.UserStatus_USER_STATUS_DISABLED
}
