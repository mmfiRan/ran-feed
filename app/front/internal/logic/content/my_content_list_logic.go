// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package content

import (
	"context"

	frontutils "ran-feed/app/front/internal/common/utils"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type MyContentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMyContentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MyContentListLogic {
	return &MyContentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MyContentListLogic) MyContentList(req *types.MyContentListReq) (resp *types.MyContentListRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.MyContentList(l.ctx, &content.MyContentListReq{
		UserId:      userID,
		Status:      utils.CastPtr[content.ContentStatus](req.Status),
		ContentType: utils.CastPtr[content.ContentType](req.ContentType),
		Cursor:      utils.Deref(req.Cursor),
		PageSize:    req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.MyContentItem, 0, len(rpcResp.Items))
	for _, it := range rpcResp.Items {
		if it == nil {
			continue
		}
		items = append(items, types.MyContentItem{
			ContentId:     it.ContentId,
			ContentType:   frontutils.ToEnumValue(it.ContentType),
			Status:        frontutils.ToEnumValue(it.Status),
			Visibility:    frontutils.ToEnumValue(it.Visibility),
			Title:         it.Title,
			CoverUrl:      it.CoverUrl,
			CreatedAt:     tsUnix(it.CreatedAt),
			PublishedAt:   tsUnix(it.PublishedAt),
			RejectReason:  it.RejectReason,
			LikeCount:     it.LikeCount,
			FavoriteCount: it.FavoriteCount,
			CommentCount:  it.CommentCount,
		})
	}

	return &types.MyContentListRes{
		Items:      items,
		NextCursor: rpcResp.NextCursor,
		HasMore:    rpcResp.HasMore,
	}, nil
}
