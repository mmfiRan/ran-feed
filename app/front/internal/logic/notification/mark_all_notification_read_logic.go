// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package notification

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
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

func (l *MarkAllNotificationReadLogic) MarkAllNotificationRead() (resp *types.MarkAllNotificationReadRes, err error) {
	// todo: add your logic here and delete this line

	return
}
