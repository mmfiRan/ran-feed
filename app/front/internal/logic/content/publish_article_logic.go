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

type PublishArticleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishArticleLogic {
	return &PublishArticleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishArticleLogic) PublishArticle(req *types.PublishArticleReq) (resp *types.PublishArticleRes, err error) {

	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}
	rpcResp, err := l.svcCtx.ContentRpc.PublishArticle(l.ctx, &content.ArticlePublishReq{
		UserId:      userID,
		Title:       *req.Title,
		Description: req.Description,
		Cover:       *req.Cover,
		Content:     *req.Content,
		Visibility:  content.Visibility(*req.Visibility),
	})
	if err != nil {
		return nil, err
	}
	return &types.PublishArticleRes{
		ContentId: rpcResp.ContentId,
	}, nil
}
