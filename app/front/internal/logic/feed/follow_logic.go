// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/utils"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowLogic {
	return &FollowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowLogic) Follow(req *types.FollowFeedReq) (resp *types.FollowFeedRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, consts.ErrUserNotLogin
	}

	rpcResp, err := l.svcCtx.FeedRpc.FollowFeed(l.ctx, &content.FollowFeedReq{
		UserId:   userID,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.FollowFeedItem, 0, len(rpcResp.Items))
	for _, it := range rpcResp.Items {
		if it == nil {
			continue
		}
		items = append(items, types.FollowFeedItem{
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

	return &types.FollowFeedRes{
		Items:      items,
		NextCursor: rpcResp.NextCursor,
		HasMore:    rpcResp.HasMore,
	}, nil
}
