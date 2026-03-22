// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/transform"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLikeInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryLikeInfoLogic) QueryLikeInfo(req *types.QueryLikeInfoReq) (resp *types.QueryLikeInfoRes, err error) {
	// 解析scene参数
	scene, err := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *req.Scene)
	if err != nil {
		return nil, errorx.NewMsg("场景参数错误")
	}

	uid := utils.GetContextUserIdWithDefault(l.ctx)

	rpcResp, err := l.svcCtx.LikeRpc.QueryLikeInfo(l.ctx, &interaction.QueryLikeInfoReq{
		UserId:    uid,
		ContentId: *req.ContentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryLikeInfoRes{
		LikeCount: rpcResp.LikeCount,
		IsLiked:   rpcResp.IsLiked,
		ContentId: rpcResp.ContentId,
		Scene:     rpcResp.Scene.String(),
	}, nil
}
