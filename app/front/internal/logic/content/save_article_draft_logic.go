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

type SaveArticleDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSaveArticleDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveArticleDraftLogic {
	return &SaveArticleDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SaveArticleDraftLogic) SaveArticleDraft(req *types.SaveArticleDraftReq) (resp *types.SaveArticleDraftRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.SaveArticleDraft(l.ctx, &content.SaveArticleDraftReq{
		UserId:      userID,
		ContentId:   req.ContentId,
		Title:       utils.Deref(req.Title),
		Description: req.Description,
		Cover:       utils.Deref(req.Cover),
		Content:     utils.Deref(req.Content),
		Visibility:  visibilityOf(req.Visibility),
	})
	if err != nil {
		return nil, err
	}

	return &types.SaveArticleDraftRes{ContentId: rpcResp.ContentId}, nil
}
