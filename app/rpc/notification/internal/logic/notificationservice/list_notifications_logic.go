package notificationservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

type ListNotificationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
}

func NewListNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListNotificationsLogic {
	return &ListNotificationsLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcCtx.MysqlDb),
	}
}

// ListNotifications 收件箱复合游标列表
func (l *ListNotificationsLogic) ListNotifications(in *notification.ListNotificationsReq) (*notification.ListNotificationsRes, error) {
	if in == nil || in.RecipientId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	pageSize := utils.ClampPageSize(in.PageSize)
	cursorTime, cursorID := parseCursor(in.CursorUpdatedAt, in.CursorId)
	typeFilter := int32(in.TypeFilter)

	rows, err := l.notifyRepo.ListByRecipient(in.RecipientId, typeFilter, cursorTime, cursorID, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询通知列表失败"))
	}

	pageRows, hasMore, nextTime, nextID := splitOverFetch(rows, pageSize)
	items := make([]*notification.NotificationItem, 0, len(pageRows))
	for _, row := range pageRows {
		if item := buildNotificationItem(row); item != nil {
			items = append(items, item)
		}
	}

	return &notification.ListNotificationsRes{
		Items:               items,
		NextCursorUpdatedAt: nextTime,
		NextCursorId:        nextID,
		HasMore:             hasMore,
	}, nil
}
