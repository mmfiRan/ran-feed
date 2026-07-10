// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserDetailLogic {
	return &GetUserDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserDetailLogic) GetUserDetail(req *types.CUserDetailReq) (resp *types.CUserDetailRes, err error) {
	rpcRes, err := l.svcCtx.UserAdminRpc.AdminGetUserDetail(l.ctx, &user.AdminGetUserDetailReq{
		UserId: req.UserId,
	})
	if err != nil {
		return nil, err
	}

	detail := rpcRes.GetDetail()
	return &types.CUserDetailRes{
		Detail: types.CUserDetailData{
			UserId:    detail.GetUserId(),
			Username:  detail.GetUsername(),
			Nickname:  detail.GetNickname(),
			Mobile:    detail.GetMobile(),
			Avatar:    detail.GetAvatar(),
			Status:    int32(detail.GetStatus()),
			Bio:       detail.GetBio(),
			Gender:    int32(detail.GetGender()),
			Email:     detail.GetEmail(),
			CreatedAt: detail.GetCreatedAt(),
			UpdatedAt: detail.GetUpdatedAt(),
		},
	}, nil
}
