package notificationservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/errorx"
)

type MarkReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
}

func NewMarkReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkReadLogic {
	return &MarkReadLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcCtx.MysqlDb),
	}
}

// MarkRead 按 ids 标已读 recipient 入 where 由 repo 侧防越权 空 ids 直返 affected=0
func (l *MarkReadLogic) MarkRead(in *notification.MarkReadReq) (*notification.MarkReadRes, error) {
	if in == nil || in.RecipientId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if len(in.Ids) == 0 {
		return &notification.MarkReadRes{Affected: 0}, nil
	}
	affected, err := l.notifyRepo.MarkRead(in.RecipientId, in.Ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("标记已读失败"))
	}
	return &notification.MarkReadRes{Affected: affected}, nil
}
