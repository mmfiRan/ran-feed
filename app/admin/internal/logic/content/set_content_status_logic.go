// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetContentStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetContentStatusLogic {
	return &SetContentStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetContentStatusLogic) SetContentStatus(req *types.AdminContentStatusReq) (resp *types.AdminContentStatusRes, err error) {
	if req.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// action 语义映射到目标状态 下架->TAKEN_DOWN 恢复->PUBLISHED
	var target content.ContentStatus
	switch req.Action {
	case consts.ContentActionTakedown:
		target = content.ContentStatus_TAKEN_DOWN
	case consts.ContentActionRestore:
		target = content.ContentStatus_PUBLISHED
	default:
		return nil, errorx.NewMsg("不支持的操作")
	}

	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)

	rpcRes, err := l.svcCtx.ContentAdminRpc.AdminSetContentStatus(l.ctx, &content.AdminSetContentStatusReq{
		ContentId:  req.ContentId,
		Status:     target,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}

	return &types.AdminContentStatusRes{
		Status: int32(rpcRes.GetStatus()),
	}, nil
}