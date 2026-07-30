package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"

	contentpb "ran-feed/app/rpc/content/content"
	notifypb "ran-feed/app/rpc/notification/notification"
	userpb "ran-feed/app/rpc/user/user"
)

func TestEncodeCursor(t *testing.T) {
	assert.Equal(t, "", encodeCursor(0, 0), "0 视为首页返空串")
	assert.Equal(t, "", encodeCursor(100, 0), "id 为 0 视为无游标")
	assert.Equal(t, "", encodeCursor(0, 5), "ts 为 0 视为无游标")
	assert.Equal(t, "1720000000000:42", encodeCursor(1720000000000, 42))
}

func TestDecodeCursor(t *testing.T) {
	ts, id := decodeCursor("")
	assert.Equal(t, int64(0), ts)
	assert.Equal(t, int64(0), id)

	ts, id = decodeCursor("  ")
	assert.Equal(t, int64(0), ts)
	assert.Equal(t, int64(0), id)

	ts, id = decodeCursor("1720000000000:42")
	assert.Equal(t, int64(1720000000000), ts)
	assert.Equal(t, int64(42), id)

	// 编解码往返
	round := decode(t, encodeCursor(1234567890, 99))
	assert.Equal(t, int64(1234567890), round.ts)
	assert.Equal(t, int64(99), round.id)

	// 非法格式返首页 而非报错(防翻页死循环)
	for _, bad := range []string{"garbage", "abc:def", "100:xyz", "xyz:100", "only-one-part", "100:200:300"} {
		ts, id = decodeCursor(bad)
		if bad == "100:200:300" {
			// SplitN(N=2) 时后段是 "200:300" 解析为 int 失败 → 0
			assert.Equal(t, int64(0), ts, "bad cursor %q ts", bad)
			assert.Equal(t, int64(0), id, "bad cursor %q id", bad)
			continue
		}
		assert.Equal(t, int64(0), ts, "bad cursor %q ts", bad)
		assert.Equal(t, int64(0), id, "bad cursor %q id", bad)
	}
}

type cursorPair struct{ ts, id int64 }

func decode(t *testing.T, s string) cursorPair {
	t.Helper()
	ts, id := decodeCursor(s)
	return cursorPair{ts, id}
}

func TestParseIDs(t *testing.T) {
	assert.Nil(t, parseIDs(nil))
	assert.Nil(t, parseIDs([]string{}))

	// 正常
	got := parseIDs([]string{"100", "200"})
	assert.Equal(t, []int64{100, 200}, got)

	// 空/非法/负数过滤
	got = parseIDs([]string{"", "  ", "abc", "-1", "0", "100"})
	assert.Equal(t, []int64{100}, got)

	// 去重保序
	got = parseIDs([]string{"100", "200", "100", "300", "200"})
	assert.Equal(t, []int64{100, 200, 300}, got)

	// 前后空白容错
	got = parseIDs([]string{"  100  ", "200"})
	assert.Equal(t, []int64{100, 200}, got)
}

func TestNilSafeInt64(t *testing.T) {
	assert.Nil(t, nilSafeInt64(0), "0 → nil")
	assert.Nil(t, nilSafeInt64(-1), "负 → nil")
	assert.NotNil(t, nilSafeInt64(1))
	assert.Equal(t, int64(42), *nilSafeInt64(42))
}

func TestNilSafeString(t *testing.T) {
	assert.Nil(t, nilSafeString(""), "空串 → nil")
	assert.NotNil(t, nilSafeString("hello"))
	assert.Equal(t, "hello", *nilSafeString("hello"))
}

func TestCollectRefIDs(t *testing.T) {
	// 空/nil 输入
	actors, contents := collectRefIDs(nil)
	assert.Nil(t, actors)
	assert.Nil(t, contents)

	// 常规 + nil 项 + 关注类 content_id=0 + 去重
	items := []*notifypb.NotificationItem{
		{ActorId: 100, ContentId: 500},
		nil,
		{ActorId: 200, ContentId: 500}, // content 重复
		{ActorId: 100, ContentId: 600}, // actor 重复
		{ActorId: 300, ContentId: 0},   // 关注类 content 跳过
	}
	actors, contents = collectRefIDs(items)
	assert.Equal(t, []int64{100, 200, 300}, actors, "保序去重")
	assert.Equal(t, []int64{500, 600}, contents, "content=0 跳过 保序去重")
}

func TestAssembleNotificationItems(t *testing.T) {
	commentID := int64(999)
	snippet := "点赞"
	items := []*notifypb.NotificationItem{
		{
			Id: 1, ActorId: 100, NotifyType: notifypb.NotifyType_LIKE_FAVORITE,
			AggCount: 3, ContentId: 500, CommentId: 0, IsRead: false, UpdatedAt: 1720000000000,
		},
		{
			Id: 2, ActorId: 200, NotifyType: notifypb.NotifyType_COMMENT_REPLY,
			AggCount: 1, ContentId: 500, CommentId: commentID, Snippet: snippet, IsRead: true, UpdatedAt: 1720000001000,
		},
		{
			Id: 3, ActorId: 300, NotifyType: notifypb.NotifyType_FOLLOW,
			AggCount: 1, ContentId: 0, IsRead: false, UpdatedAt: 1720000002000,
		},
		{
			Id: 4, ActorId: 400, NotifyType: notifypb.NotifyType_LIKE_FAVORITE,
			AggCount: 1, ContentId: 700, IsRead: false, UpdatedAt: 1720000003000, // content 已删
		},
		nil,
	}
	userMap := map[int64]*userpb.UserInfo{
		100: {UserId: 100, Nickname: "小明", Avatar: "u100.png"},
		200: {UserId: 200, Nickname: "小红", Avatar: "u200.png"},
		// actor 300 缺失
		400: {UserId: 400, Nickname: "老四", Avatar: "u400.png"},
	}
	contentMap := map[int64]*contentpb.ContentItem{
		500: {ContentId: 500, Title: "标题500", CoverUrl: "c500.jpg"},
		// content 700 已删
	}

	out := assembleNotificationItems(items, userMap, contentMap)
	assert.Len(t, out, 4, "nil 应跳过")

	// item1 LIKE_FAVORITE 完整
	assert.Equal(t, int64(1), out[0].Id)
	assert.Equal(t, int32(notifypb.NotifyType_LIKE_FAVORITE), out[0].Type)
	assert.Equal(t, "小明", out[0].Actor.Nickname)
	assert.NotNil(t, out[0].Content)
	assert.Equal(t, "标题500", out[0].Content.Title)
	assert.Nil(t, out[0].CommentId, "LIKE_FAVORITE CommentId=0 → nil")
	assert.Nil(t, out[0].Snippet, "LIKE_FAVORITE 无 snippet → nil")

	// item2 COMMENT_REPLY 带 snippet
	assert.NotNil(t, out[1].CommentId)
	assert.Equal(t, int64(999), *out[1].CommentId)
	assert.NotNil(t, out[1].Snippet)
	assert.Equal(t, "点赞", *out[1].Snippet)
	assert.True(t, out[1].IsRead)

	// item3 FOLLOW actor 300 缺失但保留通知 content 恒 nil commentId/snippet 均为 nil
	assert.Equal(t, int64(300), out[2].Actor.UserId, "actor 缺失仍带 UserId")
	assert.Equal(t, "", out[2].Actor.Nickname, "找不到的 actor 昵称空")
	assert.Nil(t, out[2].Content, "关注类 content 恒 nil")
	assert.Nil(t, out[2].CommentId, "关注类无 comment")
	assert.Nil(t, out[2].Snippet, "关注类无 snippet")

	// item4 LIKE_FAVORITE 但 content 已删 Content 置 nil 保留通知
	assert.Equal(t, int64(4), out[3].Id)
	assert.Nil(t, out[3].Content, "content 找不到应置 nil 但保留通知")
	assert.Equal(t, "老四", out[3].Actor.Nickname)
}