// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnFollowUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnFollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnFollowUserLogic {
	return &UnFollowUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnFollowUserLogic) UnFollowUser(req *types.UnFollowUserReq) (resp *types.UnFollowUserRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.FollowRpc.UnfollowUser(l.ctx, &interaction.UnfollowUserReq{
		UserId:       userID,
		FollowUserId: *req.TargetUserId,
	})
	if err != nil {
		return nil, err
	}

	return &types.UnFollowUserRes{IsFollowed: rpcResp.IsFollowed}, nil
}
