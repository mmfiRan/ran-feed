// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/utils"

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
	status := admin.AdminStatus(req.Status)

	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)
	_, err = l.svcCtx.AdminUserRpc.SetAdminStatus(l.ctx, &admin.SetAdminStatusReq{
		Id:         req.Id,
		Status:     status,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminUserStatusRes{}, nil
}
