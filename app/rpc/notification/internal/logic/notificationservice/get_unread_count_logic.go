package notificationservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/errorx"
)

type GetUnreadCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
}

func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcCtx.MysqlDb),
	}
}

// GetUnreadCount 未读数(DB 真相 走 idx_recipient_unread)
func (l *GetUnreadCountLogic) GetUnreadCount(in *notification.GetUnreadCountReq) (*notification.GetUnreadCountRes, error) {
	if in == nil || in.RecipientId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	total, err := l.notifyRepo.CountUnread(in.RecipientId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询未读数失败"))
	}
	return &notification.GetUnreadCountRes{Total: total}, nil
}
