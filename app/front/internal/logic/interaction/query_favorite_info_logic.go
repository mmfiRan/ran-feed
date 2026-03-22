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

type QueryFavoriteInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryFavoriteInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteInfoLogic {
	return &QueryFavoriteInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryFavoriteInfoLogic) QueryFavoriteInfo(req *types.QueryFavoriteInfoReq) (resp *types.QueryFavoriteInfoRes, err error) {
	// 解析scene参数
	scene, err := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *req.Scene)
	if err != nil {
		return nil, errorx.NewMsg("场景参数错误")
	}

	uid := utils.GetContextUserIdWithDefault(l.ctx)

	rpcResp, err := l.svcCtx.FavoriteRpc.QueryFavoriteInfo(l.ctx, &interaction.QueryFavoriteInfoReq{
		UserId:    uid,
		ContentId: *req.ContentId,
		Scene:     scene,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryFavoriteInfoRes{
		FavoriteCount: rpcResp.FavoriteCount,
		IsFavorite:    rpcResp.IsFavorited,
		ContentId:     rpcResp.ContentId,
		Scene:         rpcResp.Scene.String(),
	}, nil
}
