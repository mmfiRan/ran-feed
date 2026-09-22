// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

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

type SaveVideoDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSaveVideoDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveVideoDraftLogic {
	return &SaveVideoDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SaveVideoDraftLogic) SaveVideoDraft(req *types.SaveVideoDraftReq) (resp *types.SaveVideoDraftRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.SaveVideoDraft(l.ctx, &content.SaveVideoDraftReq{
		UserId:      userID,
		ContentId:   req.ContentId,
		Title:       utils.Deref(req.Title),
		Description: req.Description,
		VideoUrl:    utils.Deref(req.VideoUrl),
		CoverUrl:    utils.Deref(req.CoverUrl),
		Duration:    utils.Deref(req.Duration),
		Visibility:  visibilityOf(req.Visibility),
	})
	if err != nil {
		return nil, err
	}

	return &types.SaveVideoDraftRes{ContentId: rpcResp.ContentId}, nil
}
