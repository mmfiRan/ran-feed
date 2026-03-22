// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"
	"ran-feed/pkg/utils"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFavoriteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteLogic {
	return &UserFavoriteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFavoriteLogic) UserFavorite(req *types.UserFavoriteFeedReq) (resp *types.UserFavoriteFeedRes, err error) {

	var userId *int64
	id, err := utils.GetContextUserId(l.ctx)
	if err == nil {
		userId = &id
	}

	rpcResp, err := l.svcCtx.FeedRpc.UserFavoriteFeed(l.ctx, &content.UserFavoriteFeedReq{
		ViewerId: userId,
		UserId:   *req.UserId,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.UserFavoriteFeedItem, 0, len(rpcResp.Items))
	for _, it := range rpcResp.Items {
		if it == nil {
			continue
		}
		items = append(items, types.UserFavoriteFeedItem{
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

	return &types.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: rpcResp.NextCursor,
		HasMore:    rpcResp.HasMore,
	}, nil
}
