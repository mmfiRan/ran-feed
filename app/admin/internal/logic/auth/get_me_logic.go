// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
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

	adminRes, err := l.svcCtx.AdminRpc.GetAdmin(l.ctx, &admin.GetAdminReq{AdminId: adminID})
	if err != nil {
		return nil, err
	}
	if adminRes == nil {
		return nil, errorx.NewMsg("管理员不存在")
	}

	var permissions []string
	if permRes, pErr := l.svcCtx.AdminRpc.ListAdminPermissions(l.ctx, &admin.ListAdminPermissionsReq{AdminId: adminID}); pErr != nil {
		l.Errorf("查询管理员权限失败 adminID=%d err=%v", adminID, pErr)
	} else if permRes != nil {
		permissions = permRes.GetCodes()
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

func toEnumValue(v *admin.EnumValue) types.EnumValue {
	if v == nil {
		return types.EnumValue{}
	}
	return types.EnumValue{
		Code:    v.GetCode(),
		Name:    v.GetName(),
		Message: v.GetMessage(),
	}
}
