// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"context"

	"ran-feed/app/admin/internal/common/utils"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"

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

func (l *GetContentDetailLogic) GetContentDetail(req *types.AdminContentDetailReq) (resp *types.AdminContentDetailRes, err error) {
	if req.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminGetContentDetail(l.ctx, &content.AdminGetContentDetailReq{
		ContentId: req.ContentId,
	})
	if err != nil {
		return nil, err
	}

	d := rpcRes.GetDetail()
	return &types.AdminContentDetailRes{
		Detail: types.AdminContentDetailData{
			ContentId:      d.GetContentId(),
			ContentType:    utils.ToEnumValue(d.GetContentType()),
			Status:         utils.ToEnumValue(d.GetStatus()),
			Visibility:     utils.ToEnumValue(d.GetVisibility()),
			AuthorId:       d.GetAuthorId(),
			Title:          d.GetTitle(),
			Description:    d.GetDescription(),
			CoverUrl:       d.GetCoverUrl(),
			ArticleContent: d.GetArticleContent(),
			VideoUrl:       d.GetVideoUrl(),
			VideoDuration:  d.GetVideoDuration(),
			LikeCount:      d.GetLikeCount(),
			FavoriteCount:  d.GetFavoriteCount(),
			CommentCount:   d.GetCommentCount(),
			PublishedAt:    d.GetPublishedAt().AsTime().UnixMilli(),
			CreatedAt:      d.GetCreatedAt().AsTime().UnixMilli(),
			UpdatedAt:      d.GetUpdatedAt().AsTime().UnixMilli(),
		},
	}, nil
}
