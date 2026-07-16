package notificationservicelogic

import (
	"context"

	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUnreadCountLogic) GetUnreadCount(in *notification.GetUnreadCountReq) (*notification.GetUnreadCountRes, error) {
	// todo: add your logic here and delete this line

	return &notification.GetUnreadCountRes{}, nil
}
