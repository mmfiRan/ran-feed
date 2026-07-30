// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	notifypb "ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

type MarkAllNotificationReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkAllNotificationReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllNotificationReadLogic {
	return &MarkAllNotificationReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// MarkAllNotificationRead 全部标记已读 recipient 强制取当前登录
func (l *MarkAllNotificationReadLogic) MarkAllNotificationRead() (resp *types.MarkAllNotificationReadRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}
	rpcResp, err := l.svcCtx.NotificationRpc.MarkAllRead(l.ctx, &notifypb.MarkAllReadReq{RecipientId: userID})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("全部已读失败"))
	}
	return &types.MarkAllNotificationReadRes{Affected: rpcResp.Affected}, nil
}
