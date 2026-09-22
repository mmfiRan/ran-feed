// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package content

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitContentLogic {
	return &SubmitContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitContentLogic) SubmitContent(req *types.SubmitContentReq) (resp *types.SubmitContentRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.SubmitContent(l.ctx, &content.SubmitContentReq{
		UserId:    userID,
		ContentId: req.ContentId,
	})
	if err != nil {
		return nil, err
	}

	return &types.SubmitContentRes{ContentId: rpcResp.ContentId}, nil
}
