// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"
	"ran-feed/app/admin/internal/common/utils"

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
	rpcRes, err := l.svcCtx.AdminUserRpc.GetAdminDetail(l.ctx, &admin.GetAdminDetailReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}

	adminLists := rpcRes.GetAdmin()
	return &types.AdminUserDetailRes{
		Admin: types.AdminUserItem{
			Id:        adminLists.GetId(),
			Username:  adminLists.GetUsername(),
			Nickname:  adminLists.GetNickname(),
			Status:    utils.ToEnumValue(adminLists.GetStatus()),
			RoleCodes: adminLists.GetRoleCodes(),
			CreatedAt: adminLists.GetCreatedAt(),
		},
		RoleIds: rpcRes.GetRoleIds(),
	}, nil
}
