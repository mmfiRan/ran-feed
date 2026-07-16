package notificationservicelogic

import (
	"context"

	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkAllReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkAllReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllReadLogic {
	return &MarkAllReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MarkAllReadLogic) MarkAllRead(in *notification.MarkAllReadReq) (*notification.MarkAllReadRes, error) {
	// todo: add your logic here and delete this line

	return &notification.MarkAllReadRes{}, nil
}
