// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetAdminStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetAdminStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAdminStatusLogic {
	return &SetAdminStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetAdminStatusLogic) SetAdminStatus(req *types.AdminUserStatusReq) (resp *types.AdminUserStatusRes, err error) {
	status, ok := mapStatusAction(req.Action)
	if !ok {
		return nil, errorx.NewMsg("不支持的操作")
	}

	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	_, err = l.svcCtx.AdminRpc.SetAdminStatus(l.ctx, &admin.SetAdminStatusReq{
		Id:         req.Id,
		Status:     status,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}

	// 禁用即踢下线
	if status == admin.AdminStatus_ADMIN_DISABLED {
		kickAdminSession(l.ctx, l.svcCtx, req.Id)
	}
	return &types.AdminUserStatusRes{}, nil
}
