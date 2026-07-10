// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetUserStatusLogic {
	return &SetUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetUserStatusLogic) SetUserStatus(req *types.CUserStatusReq) (resp *types.CUserStatusRes, err error) {
	status, err := mapUserStatusAction(req.Action)
	if err != nil {
		return nil, err
	}

	operatorID, ok := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	if !ok || operatorID <= 0 {
		return nil, errorx.NewMsg("操作者ID获取失败")
	}

	_, err = l.svcCtx.UserAdminRpc.AdminSetUserStatus(l.ctx, &user.AdminSetUserStatusReq{
		UserId:     req.UserId,
		Status:     status,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}

	return &types.CUserStatusRes{
		Status: int32(status),
	}, nil
}

// mapUserStatusAction 将 action 映射为 UserStatus
func mapUserStatusAction(action string) (user.UserStatus, error) {
	switch action {
	case "ban":
		return user.UserStatus_USER_STATUS_DISABLED, nil
	case "restore":
		return user.UserStatus_USER_STATUS_ACTIVE, nil
	default:
		return user.UserStatus_USER_STATUS_UNKNOWN, errorx.NewMsg("不支持的操作")
	}
}
