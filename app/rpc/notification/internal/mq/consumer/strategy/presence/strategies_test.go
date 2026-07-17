package presence

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/notification"
)

func TestLikeStrategy_ExtractEvents(t *testing.T) {
	s := &likeStrategy{}
	ctx := context.Background()

	// INSERT 激活 产通知
	events := s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(200), "content_id": int64(500), "status": int64(10),
	}, nil)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, int64(200), e.RecipientID)
	assert.Equal(t, int64(100), e.ActorID)
	assert.Equal(t, int32(notification.NotifyType_LIKE_FAVORITE), e.NotifyType)
	assert.Equal(t, "LF:500", e.AggKey)
	assert.Equal(t, strategy.PersistAggregate, e.Action)
	assert.Equal(t, int64(500), e.ContentID)

	// 自我点赞过滤
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(100), "content_id": int64(500), "status": int64(10),
	}, nil)
	assert.Empty(t, events, "actor==recipient 应过滤")

	// 不活跃不产
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(200), "content_id": int64(500), "status": int64(20),
	}, nil)
	assert.Empty(t, events, "status 非 10 不产")

	// DELETE 不产
	events = s.ExtractEvents(ctx, "DELETE", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(200), "content_id": int64(500), "status": int64(10),
	}, nil)
	assert.Empty(t, events, "DELETE 不产通知")

	// 缺字段 不产
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"content_user_id": int64(200), "content_id": int64(500), "status": int64(10),
	}, nil)
	assert.Empty(t, events, "缺 user_id 不产")
}

func TestFavoriteStrategy_ExtractEvents_共享LF聚合(t *testing.T) {
	s := &favoriteStrategy{}
	ctx := context.Background()

	events := s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(200), "content_id": int64(500),
	}, nil)
	require.Len(t, events, 1)
	assert.Equal(t, "LF:500", events[0].AggKey, "favorite 与 like 共 LF:{content_id}")
	assert.Equal(t, int32(notification.NotifyType_LIKE_FAVORITE), events[0].NotifyType)
	assert.Equal(t, strategy.PersistAggregate, events[0].Action)

	// 自收藏过滤
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(100), "content_id": int64(500),
	}, nil)
	assert.Empty(t, events)

	// DELETE 不产(取消收藏无通知)
	events = s.ExtractEvents(ctx, "DELETE", map[string]interface{}{
		"user_id": int64(100), "content_user_id": int64(200), "content_id": int64(500),
	}, nil)
	assert.Empty(t, events)
}

func TestCommentStrategy_ExtractEvents_分流Recipient(t *testing.T) {
	s := &commentStrategy{}
	ctx := context.Background()

	// 顶评 parent_id=0 → 内容作者
	events := s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"id": int64(1000), "user_id": int64(100), "content_id": int64(500),
		"content_user_id": int64(200), "reply_to_user_id": int64(0),
		"parent_id": int64(0), "comment": "写得不错", "status": int64(10), "is_deleted": int64(0),
	}, nil)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, int64(200), e.RecipientID, "顶评应通知内容作者")
	assert.Equal(t, "CR:1000", e.AggKey)
	assert.Equal(t, strategy.PersistInsertOne, e.Action)
	assert.Equal(t, "写得不错", e.Snippet)
	assert.Equal(t, int64(500), e.ContentID)
	assert.Equal(t, int64(1000), e.CommentID)

	// 回复 parent_id>0 → 父评论作者
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"id": int64(1001), "user_id": int64(300), "content_id": int64(500),
		"content_user_id": int64(200), "reply_to_user_id": int64(400),
		"parent_id": int64(1000), "comment": "同意", "status": int64(10), "is_deleted": int64(0),
	}, nil)
	require.Len(t, events, 1)
	assert.Equal(t, int64(400), events[0].RecipientID, "回复应通知父评论作者")

	// 自评论过滤
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"id": int64(1002), "user_id": int64(200), "content_id": int64(500),
		"content_user_id": int64(200), "parent_id": int64(0),
		"comment": "自评", "status": int64(10), "is_deleted": int64(0),
	}, nil)
	assert.Empty(t, events, "作者评自己内容不产")

	// 软删不产
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"id": int64(1003), "user_id": int64(100), "content_id": int64(500),
		"content_user_id": int64(200), "parent_id": int64(0),
		"comment": "x", "status": int64(10), "is_deleted": int64(1),
	}, nil)
	assert.Empty(t, events)
}

func TestCommentStrategy_snippet按rune截断(t *testing.T) {
	s := &commentStrategy{}
	long := strings.Repeat("好评", 100) // 200 rune
	events := s.ExtractEvents(context.Background(), "INSERT", map[string]interface{}{
		"id": int64(1), "user_id": int64(100), "content_id": int64(1),
		"content_user_id": int64(200), "parent_id": int64(0),
		"comment": long, "status": int64(10), "is_deleted": int64(0),
	}, nil)
	require.Len(t, events, 1)
	runes := []rune(events[0].Snippet)
	assert.Equal(t, snippetMaxRunes, len(runes), "snippet 应按 rune 截到 140")
}

func TestFollowStrategy_ExtractEvents(t *testing.T) {
	s := &followStrategy{}
	ctx := context.Background()

	// 关注激活
	events := s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "follow_user_id": int64(200), "status": int64(10), "is_deleted": int64(0),
	}, nil)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, int64(200), e.RecipientID)
	assert.Equal(t, int64(100), e.ActorID)
	assert.Equal(t, "FO:100", e.AggKey, "FOLLOW 用 FO:{actor_id} 收敛取关重关")
	assert.Equal(t, strategy.PersistAggregate, e.Action)
	assert.Equal(t, int32(notification.NotifyType_FOLLOW), e.NotifyType)

	// UPDATE 复关(取关→再关注)应触发
	events = s.ExtractEvents(ctx, "UPDATE", map[string]interface{}{
		"user_id": int64(100), "follow_user_id": int64(200), "status": int64(10), "is_deleted": int64(0),
	}, map[string]interface{}{
		"status": int64(20),
	})
	require.Len(t, events, 1)

	// 自关过滤
	events = s.ExtractEvents(ctx, "INSERT", map[string]interface{}{
		"user_id": int64(100), "follow_user_id": int64(100), "status": int64(10), "is_deleted": int64(0),
	}, nil)
	assert.Empty(t, events)

	// 取关(active→inactive) 不产
	events = s.ExtractEvents(ctx, "UPDATE", map[string]interface{}{
		"user_id": int64(100), "follow_user_id": int64(200), "status": int64(20), "is_deleted": int64(0),
	}, map[string]interface{}{
		"status": int64(10),
	})
	assert.Empty(t, events)
}

func TestTruncateRunes(t *testing.T) {
	assert.Equal(t, "", truncateRunes("", 10))
	assert.Equal(t, "", truncateRunes("abc", 0))
	assert.Equal(t, "abc", truncateRunes("abc", 10))
	assert.Equal(t, "你好", truncateRunes("你好世界", 2))
	assert.Equal(t, "abcde", truncateRunes("abcdefg", 5))
}
