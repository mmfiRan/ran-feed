// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"
	"ran-feed/pkg/utils"
	"strconv"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecommendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRecommendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendLogic {
	return &RecommendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RecommendLogic) Recommend(req *types.RecommendFeedReq) (resp *types.RecommendFeedRes, err error) {
	var userID *int64
	if id := utils.GetContextUserIdWithDefault(l.ctx); id > 0 {
		userID = &id
	}

	rpcResp, err := l.svcCtx.FeedRpc.RecommendFeed(l.ctx, &content.RecommendFeedReq{
		UserId:     userID,
		Cursor:     *req.Cursor,
		PageSize:   *req.PageSize,
		SnapshotId: req.SnapshotId,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.RecommendFeedItem, 0, len(rpcResp.Items))
	for _, it := range rpcResp.Items {
		if it == nil {
			continue
		}
		items = append(items, types.RecommendFeedItem{
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

	nextCursor := ""
	if rpcResp.HasMore {
		nextCursor = strconv.FormatInt(rpcResp.NextCursor, 10)
	}

	return &types.RecommendFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    rpcResp.HasMore,
		SnapshotId: rpcResp.SnapshotId,
	}, nil
}
