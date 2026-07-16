package notificationservicelogic

import (
	"context"

	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkReadLogic {
	return &MarkReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MarkReadLogic) MarkRead(in *notification.MarkReadReq) (*notification.MarkReadRes, error) {
	// todo: add your logic here and delete this line

	return &notification.MarkReadRes{}, nil
}
