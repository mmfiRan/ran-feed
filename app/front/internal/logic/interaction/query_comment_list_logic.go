// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/transform"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryCommentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryCommentListLogic {
	return &QueryCommentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryCommentListLogic) QueryCommentList(req *types.QueryCommentListReq) (resp *types.QueryCommentListRes, err error) {
	if req == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if req.ContentId == nil || req.Scene == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	// 解析scene参数
	scene, err := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *req.Scene)
	if err != nil {
		return nil, errorx.NewMsg("场景参数错误")
	}

	rpcResp, err := l.svcCtx.CommentRpc.QueryCommentList(l.ctx, &interaction.QueryCommentListReq{
		ContentId: *req.ContentId,
		Scene:     scene,
		Cursor:    *req.Cursor,
		PageSize:  *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	comments := make([]*types.CommentItem, 0, len(rpcResp.Comments))
	for _, c := range rpcResp.Comments {
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

	return &types.QueryCommentListRes{
		Comments:   comments,
		NextCursor: rpcResp.NextCursor,
		HasMore:    rpcResp.HasMore,
	}, nil
}
