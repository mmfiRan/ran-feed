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

type CommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentLogic) Comment(req *types.CommentReq) (resp *types.CommentRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	scene, err := transform.ParseEnum[interaction.Scene](interaction.Scene_value, *req.Scene)
	if err != nil {
		return nil, errorx.NewMsg("场景参数错误")
	}

	rpcResp, err := l.svcCtx.CommentRpc.Comment(l.ctx, &interaction.CommentReq{
		UserId:        userID,
		ContentId:     *req.ContentId,
		Scene:         scene,
		Comment:       *req.Comment,
		ParentId:      *req.ParentId,
		RootId:        *req.RootId,
		ReplyToUserId: *req.ReplyToUserId,
		ContentUserId: *req.ContentUserId,
	})
	if err != nil {
		return nil, err
	}

	return &types.CommentRes{
		CommentId: rpcResp.CommentId,
	}, nil
}
