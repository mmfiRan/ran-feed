// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	notifypb "ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

type UnreadCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnreadCountLogic {
	return &UnreadCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UnreadCount 未读数 recipient 强制取当前登录用户防越权查他人
func (l *UnreadCountLogic) UnreadCount() (resp *types.UnreadCountRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, errorx.NewMsg("未登录")
	}
	rpcResp, err := l.svcCtx.NotificationRpc.GetUnreadCount(l.ctx, &notifypb.GetUnreadCountReq{RecipientId: userID})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询未读数失败"))
	}
	return &types.UnreadCountRes{Total: rpcResp.Total}, nil
}
