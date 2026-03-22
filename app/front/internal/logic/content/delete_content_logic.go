// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

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

type DeleteContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteContentLogic) DeleteContent(req *types.DeleteContentReq) (resp *types.DeleteContentRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户id失败"))
	}

	_, err = l.svcCtx.ContentRpc.DeleteContent(l.ctx, &content.DeleteContentReq{
		UserId:    userID,
		ContentId: req.ContentId,
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}
