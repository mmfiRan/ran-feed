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

type MarkNotificationReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkNotificationReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkNotificationReadLogic {
	return &MarkNotificationReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// MarkNotificationRead 按 id 标记已读 recipient 强制取当前登录 防越权改他人通知
// ids 用 string(避免 int64 精度丢失)本地解析为 int64 过滤非法值 全非法直返 affected=0
func (l *MarkNotificationReadLogic) MarkNotificationRead(req *types.MarkNotificationReadReq) (resp *types.MarkNotificationReadRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil || userID <= 0 {
		return nil, consts.ErrUserNotLogin
	}
	ids := parseIDs(req.Ids)
	if len(ids) == 0 {
		return &types.MarkNotificationReadRes{Affected: 0}, nil
	}
	rpcResp, err := l.svcCtx.NotificationRpc.MarkRead(l.ctx, &notifypb.MarkReadReq{
		RecipientId: userID,
		Ids:         ids,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("标记已读失败"))
	}
	return &types.MarkNotificationReadRes{Affected: rpcResp.Affected}, nil
}
