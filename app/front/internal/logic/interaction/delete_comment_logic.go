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
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCommentLogic {
	return &DeleteCommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCommentLogic) DeleteComment(req *types.DeleteCommentReq) (resp *types.DeleteCommentRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	if req == nil || req.CommentId == nil || *req.CommentId <= 0 {
		return nil, errorx.NewMsg("评论ID不能为空")
	}

	scene := interaction.Scene_SCENE_UNKNOWN
	if req.Scene != nil && *req.Scene != "" {
		parsed, parseErr := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *req.Scene)
		if parseErr != nil {
			return nil, errorx.NewMsg("场景参数错误")
		}
		scene = parsed
	}

	rpcReq := &interaction.DeleteCommentReq{
		UserId:    userID,
		CommentId: *req.CommentId,
		Scene:     scene,
	}
	if req.ContentId != nil {
		rpcReq.ContentId = *req.ContentId
	}
	if req.RootId != nil {
		rpcReq.RootId = req.RootId
	}
	if req.ParentId != nil {
		rpcReq.ParentId = req.ParentId
	}

	_, err = l.svcCtx.CommentRpc.DeleteComment(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	return &types.DeleteCommentRes{}, nil
}
