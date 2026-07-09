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

type GetAdminDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAdminDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminDetailLogic {
	return &GetAdminDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAdminDetailLogic) GetAdminDetail(req *types.AdminUserDetailReq) (resp *types.AdminUserDetailRes, err error) {
	rpcRes, err := l.svcCtx.AdminRpc.GetAdminDetail(l.ctx, &admin.GetAdminDetailReq{Id: req.Id})
	if err != nil {
		return nil, err
	}

	a := rpcRes.GetAdmin()
	return &types.AdminUserDetailRes{
		Admin: types.AdminUserItem{
			Id:        a.GetId(),
			Username:  a.GetUsername(),
			Nickname:  a.GetNickname(),
			Status:    int32(a.GetStatus()),
			RoleCodes: a.GetRoleCodes(),
			CreatedAt: a.GetCreatedAt(),
		},
		RoleIds: rpcRes.GetRoleIds(),
	}, nil
}
