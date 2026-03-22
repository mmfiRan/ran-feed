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

type BatchQueryLikeInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryLikeInfoLogic {
	return &BatchQueryLikeInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchQueryLikeInfoLogic) BatchQueryLikeInfo(req *types.BatchQueryLikeInfoReq) (resp *types.BatchQueryLikeInfoRes, err error) {
	userID := utils.GetContextUserIdWithDefault(l.ctx)
	likeInfos := make([]*interaction.LikeInfo, 0, len(req.LikeInfos))
	for _, reqInfo := range req.LikeInfos {
		scene, err := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *reqInfo.Scene)
		if err != nil {
			return nil, errorx.NewMsg("场景参数错误")
		}

		likeInfos = append(likeInfos, &interaction.LikeInfo{
			ContentId: *reqInfo.ContentId,
			Scene:     scene,
		})
	}
	rpcResp, err := l.svcCtx.LikeRpc.BatchQueryLikeInfo(l.ctx, &interaction.BatchQueryLikeInfoReq{
		UserId:    userID,
		LikeInfos: likeInfos,
	})
	if err != nil {
		return nil, err
	}
	likeInfoRes := make([]types.QueryLikeInfoRes, 0, len(rpcResp.LikeInfos))
	for _, rpcInfo := range rpcResp.LikeInfos {
		likeInfoRes = append(likeInfoRes, types.QueryLikeInfoRes{
			LikeCount: rpcInfo.LikeCount,
			IsLiked:   rpcInfo.IsLiked,
			ContentId: rpcInfo.ContentId,
			Scene:     rpcInfo.Scene.String(),
		})
	}
	return &types.BatchQueryLikeInfoRes{
		LikeInfos: likeInfoRes,
	}, nil
}
