package notificationservicelogic

import (
	"context"

	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListNotificationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListNotificationsLogic {
	return &ListNotificationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListNotificationsLogic) ListNotifications(in *notification.ListNotificationsReq) (*notification.ListNotificationsRes, error) {
	// todo: add your logic here and delete this line

	return &notification.ListNotificationsRes{}, nil
}
