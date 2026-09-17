// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"

	"ran-feed/app/admin/internal/common/utils"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	pkgutils "ran-feed/pkg/utils"

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
	operatorID, err := pkgutils.GetContextAdminId(l.ctx)
	if err != nil || operatorID <= 0 {
		return nil, errorx.NewMsg("操作者ID获取失败")
	}

	rpcRes, err := l.svcCtx.UserAdminRpc.AdminSetUserStatus(l.ctx, &user.AdminSetUserStatusReq{
		UserId:     req.UserId,
		Status:     user.UserStatus(req.Status),
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}

	return &types.CUserStatusRes{
		Status: utils.ToEnumValue(rpcRes.GetStatus()),
	}, nil
}
