// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetContentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetContentDetailLogic {
	return &GetContentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetContentDetailLogic) GetContentDetail(req *types.GetContentDetailReq) (resp *types.GetContentDetailRes, err error) {
	var viewerID *int64
	if id := utils.GetContextUserIdWithDefault(l.ctx); id > 0 {
		viewerID = &id
	}

	rpcResp, err := l.svcCtx.ContentRpc.GetContentDetail(l.ctx, &content.GetContentDetailReq{
		ContentId: *req.ContentId,
		ViewerId:  viewerID,
	})
	if err != nil {
		return nil, err
	}

	resp = &types.GetContentDetailRes{}
	if rpcResp == nil || rpcResp.Detail == nil {
		return resp, nil
	}

	resp.Detail = types.ContentDetail{
		ContentId:         rpcResp.Detail.ContentId,
		ContentType:       int32(rpcResp.Detail.ContentType),
		AuthorId:          rpcResp.Detail.AuthorId,
		AuthorName:        rpcResp.Detail.AuthorName,
		AuthorAvatar:      rpcResp.Detail.AuthorAvatar,
		Title:             rpcResp.Detail.Title,
		Description:       rpcResp.Detail.Description,
		CoverUrl:          rpcResp.Detail.CoverUrl,
		ArticleContent:    rpcResp.Detail.ArticleContent,
		VideoUrl:          rpcResp.Detail.VideoUrl,
		VideoDuration:     rpcResp.Detail.VideoDuration,
		PublishedAt:       rpcResp.Detail.PublishedAt,
		LikeCount:         rpcResp.Detail.LikeCount,
		FavoriteCount:     rpcResp.Detail.FavoriteCount,
		CommentCount:      rpcResp.Detail.CommentCount,
		IsLiked:           rpcResp.Detail.IsLiked,
		IsFavorited:       rpcResp.Detail.IsFavorited,
		IsFollowingAuthor: rpcResp.Detail.IsFollowingAuthor,
	}

	return resp, nil
}
