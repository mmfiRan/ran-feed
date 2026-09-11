// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"context"

	"ran-feed/app/admin/internal/common/utils"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/content/content"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListContentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListContentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListContentsLogic {
	return &ListContentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListContentsLogic) ListContents(req *types.AdminContentListReq) (resp *types.AdminContentListRes, err error) {

	in := &content.AdminListContentsReq{
		Page:        req.Page,
		PageSize:    req.PageSize,
		AuthorId:    req.AuthorId,
		Username:    req.Username,
		Status:      utils.CastPtr[content.ContentStatus](req.Status),
		ContentType: utils.CastPtr[content.ContentType](req.ContentType),
	}

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminListContents(l.ctx, in)
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminContentListItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminContentListItem{
			ContentId:     it.GetContentId(),
			ContentType:   utils.ToEnumValue(it.GetContentType()),
			Status:        utils.ToEnumValue(it.GetStatus()),
			Visibility:    utils.ToEnumValue(it.GetVisibility()),
			AuthorId:      it.GetAuthorId(),
			Title:         it.GetTitle(),
			LikeCount:     it.GetLikeCount(),
			FavoriteCount: it.GetFavoriteCount(),
			CommentCount:  it.GetCommentCount(),
			PublishedAt:   it.GetPublishedAt().AsTime().UnixMilli(),
			CreatedAt:     it.GetCreatedAt().AsTime().UnixMilli(),
		})
	}

	return &types.AdminContentListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    uint32(rpcRes.GetTotal()),
		},
	}, nil
}
