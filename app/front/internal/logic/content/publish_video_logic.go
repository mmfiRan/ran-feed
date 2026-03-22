// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishVideoLogic {
	return &PublishVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishVideoLogic) PublishVideo(req *types.PublishVideoReq) (resp *types.PublishVideoRes, err error) {

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}
	rpcResp, err := l.svcCtx.ContentRpc.PublishVideo(l.ctx, &content.VideoPublishReq{
		UserId:      userID,
		Title:       *req.Title,
		Description: req.Description,
		VideoUrl:    *req.VideoUrl,
		CoverUrl:    *req.CoverUrl,
		Duration:    *req.Duration,
		Visibility:  content.Visibility(*req.Visibility),
	})
	if err != nil {
		return nil, err
	}
	return &types.PublishVideoRes{
		ContentId: rpcResp.ContentId,
	}, nil

}
