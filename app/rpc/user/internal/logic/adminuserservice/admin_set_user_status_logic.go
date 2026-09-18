package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/common/utils"
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
	row, err := l.userRepo.AdminGetByID(in.UserId)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	target := utils.UserStatusValue(int32(in.Status))
	noop, err := l.validateUserStatusTransition(user.UserStatus(row.Status), in.Status)
	if err != nil {
		return nil, err
	}

	if noop {
		return &user.AdminSetUserStatusRes{
			Status: target,
		}, nil
	}

	if _, err = l.userRepo.AdminUpdateStatus(in.UserId, int32(in.Status), in.OperatorId); err != nil {
		return nil, err
	}

	// 封禁后踢下线
	if in.Status == user.UserStatus_USER_STATUS_DISABLED {
		if err = session.RemoveByUserID(l.ctx, l.svcCtx.Redis, in.UserId); err != nil {
			l.Errorf("踢下线失败 userId=%d err=%v", in.UserId, err)
		}
	}

	return &user.AdminSetUserStatusRes{
		Status: target,
	}, nil
}

// validateUserStatusTransition 校验封禁/恢复状态机
func (l *AdminSetUserStatusLogic) validateUserStatusTransition(cur, target user.UserStatus) (noop bool, err error) {
	if cur == target {
		return true, nil
	}
	switch target {
	case user.UserStatus_USER_STATUS_DISABLED:
		if cur != user.UserStatus_USER_STATUS_ACTIVE {
			return false, errorx.NewMsg("仅正常用户可封禁")
		}
	case user.UserStatus_USER_STATUS_ACTIVE:
		if cur != user.UserStatus_USER_STATUS_DISABLED {
			return false, errorx.NewMsg("仅已封禁用户可恢复")
		}
	default:
		return false, errorx.NewMsg("不支持的状态")
	}
	return false, nil
}
