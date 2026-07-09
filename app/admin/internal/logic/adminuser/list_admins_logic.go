// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAdminsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAdminsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminsLogic {
	return &ListAdminsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAdminsLogic) ListAdmins(req *types.AdminUserListReq) (resp *types.AdminUserListRes, err error) {
	rpcRes, err := l.svcCtx.AdminRpc.ListAdmins(l.ctx, &admin.ListAdminsReq{
		Status:   admin.AdminStatus(req.Status),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminUserItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminUserItem{
			Id:        it.GetId(),
			Username:  it.GetUsername(),
			Nickname:  it.GetNickname(),
			Status:    int32(it.GetStatus()),
			RoleCodes: it.GetRoleCodes(),
			CreatedAt: it.GetCreatedAt(),
		})
	}
	return &types.AdminUserListRes{Items: items, Total: rpcRes.GetTotal()}, nil
}
