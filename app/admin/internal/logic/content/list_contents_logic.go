// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"context"

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
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	// 各筛选项 0 表示不限 转成可选指针
	if req.Status > 0 {
		s := content.ContentStatus(req.Status)
		in.Status = &s
	}
	if req.ContentType > 0 {
		t := content.ContentType(req.ContentType)
		in.ContentType = &t
	}
	if req.AuthorId > 0 {
		a := req.AuthorId
		in.AuthorId = &a
	}

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminListContents(l.ctx, in)
	if err != nil {
		return nil, err
	}

	items := make([]types.AdminContentListItem, 0, len(rpcRes.GetItems()))
	for _, it := range rpcRes.GetItems() {
		items = append(items, types.AdminContentListItem{
			ContentId:     it.GetContentId(),
			ContentType:   int32(it.GetContentType()),
			Status:        int32(it.GetStatus()),
			Visibility:    int32(it.GetVisibility()),
			AuthorId:      it.GetAuthorId(),
			Title:         it.GetTitle(),
			LikeCount:     it.GetLikeCount(),
			FavoriteCount: it.GetFavoriteCount(),
			CommentCount:  it.GetCommentCount(),
			PublishedAt:   it.GetPublishedAt(),
			CreatedAt:     it.GetCreatedAt(),
		})
	}

	return &types.AdminContentListRes{
		Items: items,
		PageQueryResp: types.PageQueryResp{
			Page:     rpcRes.GetPage(),
			PageSize: rpcRes.GetPageSize(),
			Total:    rpcRes.GetTotal(),
		},
	}, nil
}
