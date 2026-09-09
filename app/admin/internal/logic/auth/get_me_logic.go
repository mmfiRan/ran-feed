// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/commonpb"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type GetMeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeLogic {
	return &GetMeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMeLogic) GetMe() (resp *types.AdminMeRes, err error) {
	adminID := utils.GetContextAdminIdWithDefault(l.ctx)
	if adminID <= 0 {
		return nil, consts.ErrAdminNotLogin
	}

	var (
		adminRes    *admin.GetAdminRes
		permissions []string
	)

	err = mr.Finish(
		func() error {
			res, err := l.svcCtx.AdminAuthRpc.GetAdmin(l.ctx, &admin.GetAdminReq{
				AdminId: adminID,
			})
			if err != nil {
				return err
			}
			if res == nil {
				return errorx.NewMsg("管理员不存在")
			}
			adminRes = res
			return nil
		},
		func() error {
			permRes, pErr := l.svcCtx.AdminAuthRpc.ListAdminPermissions(l.ctx, &admin.ListAdminPermissionsReq{
				AdminId: adminID,
			})
			if pErr != nil {
				return err
			}
			if permRes != nil {
				permissions = permRes.GetCodes()
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.AdminMeRes{
		AdminInfo: types.AdminInfo{
			AdminId:  adminRes.GetAdminId(),
			Username: adminRes.GetUsername(),
			Nickname: adminRes.GetNickname(),
			Status:   toEnumValue(adminRes.GetStatus()),
		},
		Permissions: permissions,
	}, nil
}

func toEnumValue(v *commonpb.EnumValue) types.EnumValue {
	if v == nil {
		return types.EnumValue{}
	}
	return types.EnumValue{
		Code:    v.GetCode(),
		Name:    v.GetName(),
		Message: v.GetMessage(),
	}
}
