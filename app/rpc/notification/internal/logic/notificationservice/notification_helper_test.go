package notificationservicelogic

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/notification"
)

func TestBuildNotificationItem(t *testing.T) {
	assert.Nil(t, buildNotificationItem(nil), "nil 应返 nil")

	now := time.Now()
	row := &model.RanFeedNotification{
		ID:          100,
		RecipientID: 200,
		ActorID:     300,
		NotifyType:  int32(notification.NotifyType_LIKE_FAVORITE),
		AggCount:    3,
		ContentID:   500,
		CommentID:   0,
		Snippet:     "",
		IsRead:      0,
		UpdatedAt:   now,
	}
	item := buildNotificationItem(row)
	assert.Equal(t, int64(100), item.Id)
	assert.Equal(t, int64(200), item.RecipientId)
	assert.Equal(t, int64(300), item.ActorId)
	assert.Equal(t, notification.NotifyType_LIKE_FAVORITE, item.NotifyType)
	assert.Equal(t, int64(3), item.AggCount)
	assert.Equal(t, int64(500), item.ContentId)
	assert.False(t, item.IsRead)
	assert.Equal(t, now.UnixMilli(), item.UpdatedAt)
}

func TestBuildNotificationItem_已读转Bool(t *testing.T) {
	row := &model.RanFeedNotification{ID: 1, IsRead: 1, UpdatedAt: time.Now()}
	assert.True(t, buildNotificationItem(row).IsRead, "IsRead=1 应转 true")
}

func TestSplitOverFetch_不足一页(t *testing.T) {
	rows := makeRows(3)
	items, hasMore, nextTime, nextID := splitOverFetch(rows, 5)
	assert.Len(t, items, 3)
	assert.False(t, hasMore, "不满 pageSize 应无下一页")
	assert.Equal(t, int64(0), nextTime)
	assert.Equal(t, int64(0), nextID)
}

func TestSplitOverFetch_满页无下一页(t *testing.T) {
	// repo 只返 pageSize 条(而非 pageSize+1)代表末页
	rows := makeRows(5)
	items, hasMore, nextTime, nextID := splitOverFetch(rows, 5)
	assert.Len(t, items, 5)
	assert.False(t, hasMore, "刚好一页无 overfetch 无下一页")
	assert.Equal(t, int64(0), nextTime)
	assert.Equal(t, int64(0), nextID)
}

func TestSplitOverFetch_有下一页(t *testing.T) {
	// over-fetch pageSize+1 条
	rows := makeRows(6)
	items, hasMore, nextTime, nextID := splitOverFetch(rows, 5)
	assert.Len(t, items, 5, "多出的一条应被截掉")
	assert.True(t, hasMore)
	// nextCursor 取第 5 条(rows[4])
	assert.Equal(t, rows[4].UpdatedAt.UnixMilli(), nextTime)
	assert.Equal(t, rows[4].ID, nextID)
}

func TestSplitOverFetch_空或无效pageSize(t *testing.T) {
	items, hasMore, nt, nid := splitOverFetch(nil, 5)
	assert.Nil(t, items)
	assert.False(t, hasMore)
	assert.Equal(t, int64(0), nt)
	assert.Equal(t, int64(0), nid)

	items, hasMore, _, _ = splitOverFetch(makeRows(2), 0)
	assert.Nil(t, items)
	assert.False(t, hasMore)

	items, hasMore, _, _ = splitOverFetch(makeRows(2), -5)
	assert.Nil(t, items)
	assert.False(t, hasMore)
}

func TestParseCursor(t *testing.T) {
	// 首页
	tm, id := parseCursor(0, 0)
	assert.True(t, tm.IsZero())
	assert.Equal(t, int64(0), id)

	// 负值也视为首页
	tm, id = parseCursor(-1, 100)
	assert.True(t, tm.IsZero())
	assert.Equal(t, int64(0), id)

	// 正常
	ms := time.Now().UnixMilli()
	tm, id = parseCursor(ms, 50)
	assert.Equal(t, ms, tm.UnixMilli())
	assert.Equal(t, int64(50), id)
}

// makeRows 造 n 条 updated_at 严格递减 id 递减
func makeRows(n int) []*model.RanFeedNotification {
	rows := make([]*model.RanFeedNotification, 0, n)
	base := time.Now().Add(-time.Hour)
	for i := n; i >= 1; i-- {
		rows = append(rows, &model.RanFeedNotification{
			ID:        int64(i),
			UpdatedAt: base.Add(time.Duration(i) * time.Second),
		})
	}
	return rows
}

// ---------- 参数守卫 ----------

func TestListNotifications_参数守卫(t *testing.T) {
	l := &ListNotificationsLogic{ctx: context.Background()}
	_, err := l.ListNotifications(nil)
	assert.Error(t, err)
	_, err = l.ListNotifications(&notification.ListNotificationsReq{RecipientId: 0})
	assert.Error(t, err)
	_, err = l.ListNotifications(&notification.ListNotificationsReq{RecipientId: -1})
	assert.Error(t, err)
}

func TestGetUnreadCount_参数守卫(t *testing.T) {
	l := &GetUnreadCountLogic{ctx: context.Background()}
	_, err := l.GetUnreadCount(nil)
	assert.Error(t, err)
	_, err = l.GetUnreadCount(&notification.GetUnreadCountReq{RecipientId: 0})
	assert.Error(t, err)
}

func TestMarkRead_参数守卫(t *testing.T) {
	l := &MarkReadLogic{ctx: context.Background()}
	_, err := l.MarkRead(nil)
	assert.Error(t, err)
	_, err = l.MarkRead(&notification.MarkReadReq{RecipientId: 0, Ids: []int64{1}})
	assert.Error(t, err)
}

func TestMarkRead_空ids直返(t *testing.T) {
	// ids 空时守卫内直返 affected=0 不调 repo 故可用零值 logic
	l := &MarkReadLogic{ctx: context.Background()}
	res, err := l.MarkRead(&notification.MarkReadReq{RecipientId: 1, Ids: nil})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), res.Affected)
	res, err = l.MarkRead(&notification.MarkReadReq{RecipientId: 1, Ids: []int64{}})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), res.Affected)
}

func TestMarkAllRead_参数守卫(t *testing.T) {
	l := &MarkAllReadLogic{ctx: context.Background()}
	_, err := l.MarkAllRead(nil)
	assert.Error(t, err)
	_, err = l.MarkAllRead(&notification.MarkAllReadReq{RecipientId: 0})
	assert.Error(t, err)
}
