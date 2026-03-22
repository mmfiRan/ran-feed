// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/user/user"
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

func (l *GetMeLogic) GetMe() (resp *types.GetMeRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.UserRpc.GetMe(l.ctx, &user.GetMeReq{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	resp = &types.GetMeRes{
		UserInfo: types.UserInfo{},
	}
	if rpcResp.UserInfo != nil {
		resp.UserInfo = types.UserInfo{
			UserId:   rpcResp.UserInfo.UserId,
			Mobile:   rpcResp.UserInfo.Mobile,
			Nickname: rpcResp.UserInfo.Nickname,
			Avatar:   rpcResp.UserInfo.Avatar,
			Bio:      rpcResp.UserInfo.Bio,
			Gender:   int32(rpcResp.UserInfo.Gender),
			Status:   int32(rpcResp.UserInfo.Status),
		}
	}
	resp.FolloweeCount = rpcResp.FolloweeCount
	resp.FollowerCount = rpcResp.FollowerCount
	resp.LikeReceivedCount = rpcResp.LikeReceivedCount
	resp.FavoriteReceivedCount = rpcResp.FavoriteReceivedCount

	return resp, nil
}
