package notificationservicelogic

import (
	"time"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/notification"
)

// buildNotificationItem
func buildNotificationItem(row *model.RanFeedNotification) *notification.NotificationItem {
	if row == nil {
		return nil
	}
	return &notification.NotificationItem{
		Id:          row.ID,
		RecipientId: row.RecipientID,
		ActorId:     row.ActorID,
		NotifyType:  notification.NotifyType(row.NotifyType),
		AggCount:    int64(row.AggCount),
		ContentId:   row.ContentID,
		CommentId:   row.CommentID,
		Snippet:     row.Snippet,
		IsRead:      row.IsRead == 1,
		UpdatedAt:   row.UpdatedAt.UnixMilli(),
	}
}

// splitOverFetch 处理 over-fetch+1 结果 切出真实页 + next cursor + has_more
func splitOverFetch(rows []*model.RanFeedNotification, pageSize int) ([]*model.RanFeedNotification, bool, int64, int64) {
	if len(rows) == 0 || pageSize <= 0 {
		return nil, false, 0, 0
	}
	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}
	if !hasMore {
		return rows, false, 0, 0
	}
	last := rows[len(rows)-1]
	return rows, true, last.UpdatedAt.UnixMilli(), last.ID
}

// parseCursor 复合游标解析 cursorUpdatedAt=0 视首页返
func parseCursor(cursorUpdatedAtMillis, cursorID int64) (time.Time, int64) {
	if cursorUpdatedAtMillis <= 0 {
		return time.Time{}, 0
	}
	return time.UnixMilli(cursorUpdatedAtMillis), cursorID
}
