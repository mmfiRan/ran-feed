package notificationservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/errorx"
)

type MarkAllReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
}

func NewMarkAllReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllReadLogic {
	return &MarkAllReadLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcCtx.MysqlDb),
	}
}

// MarkAllRead 全部标已读 recipient 入 where 只翻未读避免重复计数
func (l *MarkAllReadLogic) MarkAllRead(in *notification.MarkAllReadReq) (*notification.MarkAllReadRes, error) {
	if in == nil || in.RecipientId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	affected, err := l.notifyRepo.MarkAllRead(in.RecipientId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("全部已读失败"))
	}
	return &notification.MarkAllReadRes{Affected: affected}, nil
}
