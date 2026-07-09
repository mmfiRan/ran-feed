// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package permission

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPermissionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPermissionsLogic {
	return &ListPermissionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListPermissionsLogic) ListPermissions(req *types.AdminPermissionListReq) (resp *types.AdminPermissionListRes, err error) {
	rpcRes, err := l.svcCtx.AdminRpc.ListPermissions(l.ctx, &admin.ListPermissionsReq{Module: req.Module})
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminPermissionItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminPermissionItem{
			Id:     it.GetId(),
			Code:   it.GetCode(),
			Name:   it.GetName(),
			Module: it.GetModule(),
		})
	}

	return &types.AdminPermissionListRes{Items: items}, nil
}
