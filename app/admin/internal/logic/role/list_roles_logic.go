// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package role

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRolesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRolesLogic {
	return &ListRolesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListRolesLogic) ListRoles(req *types.AdminRoleListReq) (resp *types.AdminRoleListRes, err error) {
	rpcRes, err := l.svcCtx.AdminRpc.ListRoles(l.ctx, &admin.ListRolesReq{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminRoleItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminRoleItem{
			Id:        it.GetId(),
			Code:      it.GetCode(),
			Name:      it.GetName(),
			Remark:    it.GetRemark(),
			CreatedAt: it.GetCreatedAt(),
		})
	}
	return &types.AdminRoleListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    rpcRes.GetTotal(),
		},
	}, nil
}
