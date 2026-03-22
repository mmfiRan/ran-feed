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

type LikeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikeLogic) Like(req *types.LikeReq) (resp *types.LikeRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	scene, err := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *req.Scene)
	if err != nil {
		return nil, errorx.NewMsg("场景参数错误")
	}

	_, err = l.svcCtx.LikeRpc.Like(l.ctx, &interaction.LikeReq{
		UserId:        userID,
		ContentId:     *req.ContentId,
		ContentUserId: *req.ContentUserId,
		Scene:         scene,
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}
