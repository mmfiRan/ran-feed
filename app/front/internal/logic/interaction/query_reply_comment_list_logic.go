// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/interaction/interaction"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryReplyCommentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryReplyCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryReplyCommentListLogic {
	return &QueryReplyCommentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryReplyCommentListLogic) QueryReplyCommentList(req *types.QueryReplyCommentListReq) (resp *types.QueryReplyCommentListRes, err error) {

	rpcResp, err := l.svcCtx.CommentRpc.QueryReplyList(l.ctx, &interaction.QueryReplyListReq{
		RootId:   *req.CommentId,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	comments := make([]*types.CommentItem, 0, len(rpcResp.Replies))
	for _, c := range rpcResp.Replies {
		if c == nil {
			continue
		}
		comments = append(comments, &types.CommentItem{
			CommentId:     c.CommentId,
			ContentId:     c.ContentId,
			UserId:        c.UserId,
			ReplyToUserId: c.ReplyToUserId,
			ParentId:      c.ParentId,
			RootId:        c.RootId,
			Comment:       c.Comment,
			CreatedAt:     c.CreatedAt,
			Status:        c.Status,
			UserName:      c.UserName,
			UserAvatar:    c.UserAvatar,
			ReplyCount:    c.ReplyCount,
		})
	}

	return &types.QueryReplyCommentListRes{
		Comments:   comments,
		NextCursor: rpcResp.NextCursor,
		HasMore:    rpcResp.HasMore,
	}, nil
}
