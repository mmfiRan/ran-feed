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
	if in == nil || in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	row, err := l.userRepo.AdminGetByID(in.UserId)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	cur := user.UserStatus(row.Status)
	noop, err := validateUserStatusTransition(cur, in.Status)
	if err != nil {
		return nil, err
	}
	// 当前已是目标态 幂等直接返回
	if noop {
		return &user.AdminSetUserStatusRes{}, nil
	}

	if _, err = l.userRepo.AdminUpdateStatus(in.UserId, int32(in.Status), in.OperatorId); err != nil {
		return nil, err
	}

	// 封禁后踢下线 失败只记日志不阻断
	if in.Status == user.UserStatus_USER_STATUS_DISABLED {
		if err = session.RemoveByUserID(l.ctx, l.svcCtx.Redis, in.UserId); err != nil {
			l.Errorf("踢下线失败 userId=%d err=%v", in.UserId, err)
		}
	}

	return &user.AdminSetUserStatusRes{}, nil
}

// validateUserStatusTransition 校验封禁/恢复状态机 返回 noop 表示当前已是目标态无需落库
//   - 封禁 DISABLED 仅允许从 ACTIVE 转入
//   - 恢复 ACTIVE 仅允许从 DISABLED 转入
//
// 杜绝把注销账号 status 30 直接改回正常态
func validateUserStatusTransition(cur, target user.UserStatus) (noop bool, err error) {
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
