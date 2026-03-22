// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserPublishLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishLogic {
	return &UserPublishLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserPublishLogic) UserPublish(req *types.UserPublishFeedReq) (resp *types.UserPublishFeedRes, err error) {
	viewerID := req.ViewerId
	if ctxUserID := utils.GetContextUserIdWithDefault(l.ctx); ctxUserID > 0 {
		viewerID = &ctxUserID
	}

	rpcResp, err := l.svcCtx.FeedRpc.UserPublishFeed(l.ctx, &content.UserPublishFeedReq{
		AuthorId: *req.UserId,
		ViewerId: viewerID,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.UserPublishFeedItem, 0, len(rpcResp.Items))
	for _, it := range rpcResp.Items {
		if it == nil {
			continue
		}
		items = append(items, types.UserPublishFeedItem{
			ContentId:    it.ContentId,
			ContentType:  int32(it.ContentType),
			AuthorId:     it.AuthorId,
			AuthorName:   it.AuthorName,
			AuthorAvatar: it.AuthorAvatar,
			Title:        it.Title,
			CoverUrl:     it.CoverUrl,
			PublishedAt:  it.PublishedAt,
			IsLiked:      it.IsLiked,
			LikeCount:    it.LikeCount,
		})
	}

	return &types.UserPublishFeedRes{
		Items:      items,
		NextCursor: rpcResp.NextCursor,
		HasMore:    rpcResp.HasMore,
	}, nil
}
