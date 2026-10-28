package sse

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	red "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/pkg/event/notificationevent"
	"ran-feed/pkg/rediskey"
)

func TestConnManager_AddRemove(t *testing.T) {
	m := NewConnManager()
	assert.Equal(t, 0, m.UserCount())

	done := make(chan struct{})
	c1 := NewConn(done)
	c2 := NewConn(done)

	m.Add(100, c1)
	m.Add(100, c2)
	m.Add(200, c1)
	assert.Equal(t, 2, m.UserCount())
	assert.Equal(t, 2, m.ConnCount(100))
	assert.Equal(t, 1, m.ConnCount(200))

	// 参数守卫
	m.Add(0, c1)
	m.Add(300, nil)
	assert.Equal(t, 2, m.UserCount(), "非法输入不入库")

	// Remove 一个 c1 只从 100 移除,不影响 200
	m.Remove(100, c1)
	assert.Equal(t, 1, m.ConnCount(100))
	assert.Equal(t, 1, m.ConnCount(200))

	// 移光 100 下最后一个,应从 map 清 user 项
	m.Remove(100, c2)
	assert.Equal(t, 0, m.ConnCount(100))
	assert.Equal(t, 1, m.UserCount())

	// 重复 Remove 幂等
	m.Remove(100, c2)
	m.Remove(999, c1)
	assert.Equal(t, 1, m.UserCount())
}

func TestConnManager_Broadcast(t *testing.T) {
	m := NewConnManager()
	done := make(chan struct{})
	c1 := NewConn(done)
	c2 := NewConn(done)
	c3 := NewConn(done)
	m.Add(100, c1)
	m.Add(100, c2)
	m.Add(200, c3)

	delivered, dropped := m.Broadcast(100, []byte("hi"))
	assert.Equal(t, 2, delivered)
	assert.Equal(t, 0, dropped)

	// 收方向验证
	assert.Equal(t, "hi", string(<-c1.Send))
	assert.Equal(t, "hi", string(<-c2.Send))

	// 广播另一 user 不串
	delivered, dropped = m.Broadcast(200, []byte("y"))
	assert.Equal(t, 1, delivered)
	assert.Equal(t, 0, dropped)
	assert.Equal(t, "y", string(<-c3.Send))

	// 广播到不存在 user 全 0
	delivered, dropped = m.Broadcast(999, []byte("x"))
	assert.Equal(t, 0, delivered)
	assert.Equal(t, 0, dropped)
}

func TestConnManager_Broadcast_慢连接丢帧(t *testing.T) {
	m := NewConnManager()
	done := make(chan struct{})
	slow := NewConn(done)
	m.Add(100, slow)

	// 塞满 send 缓冲(DefaultSendBuffer=8)
	for i := 0; i < DefaultSendBuffer; i++ {
		select {
		case slow.Send <- []byte("fill"):
		default:
			t.Fatalf("填缓冲阶段不应满 i=%d", i)
		}
	}

	// 再广播应全部 dropped(缓冲满 select default)
	delivered, dropped := m.Broadcast(100, []byte("late"))
	assert.Equal(t, 0, delivered)
	assert.Equal(t, 1, dropped, "慢连接缓冲满应 drop 不阻塞广播")
}

func TestConnManager_Broadcast_参数守卫(t *testing.T) {
	m := NewConnManager()
	m.Add(100, NewConn(nil))

	d, dr := m.Broadcast(0, []byte("x"))
	assert.Equal(t, 0, d)
	assert.Equal(t, 0, dr)

	d, dr = m.Broadcast(100, nil)
	assert.Equal(t, 0, d)
	assert.Equal(t, 0, dr)

	d, dr = m.Broadcast(100, []byte{})
	assert.Equal(t, 0, d)
	assert.Equal(t, 0, dr)
}

func TestDecodePayload(t *testing.T) {
	// 空
	_, err := DecodePayload(nil)
	assert.Error(t, err)
	_, err = DecodePayload([]byte{})
	assert.Error(t, err)

	// 非法 JSON
	_, err = DecodePayload([]byte("garbage"))
	assert.Error(t, err)

	// 字段缺失 recipient=0 拒
	_, err = DecodePayload([]byte(`{"unread":5}`))
	assert.Error(t, err)
	_, err = DecodePayload([]byte(`{"recipient_id":0,"unread":5}`))
	assert.Error(t, err)
	_, err = DecodePayload([]byte(`{"recipient_id":-1,"unread":5}`))
	assert.Error(t, err)

	// 正常
	msg, err := DecodePayload([]byte(`{"recipient_id":100,"unread":3}`))
	require.NoError(t, err)
	assert.Equal(t, int64(100), msg.RecipientID)
	assert.Equal(t, int64(3), msg.Unread)

	// unread=0 允许(全部已读广播归零)
	msg, err = DecodePayload([]byte(`{"recipient_id":100,"unread":0}`))
	require.NoError(t, err)
	assert.Equal(t, int64(0), msg.Unread)
}

func TestEncodeMessage(t *testing.T) {
	raw, err := EncodeMessage(notificationevent.Payload{RecipientID: 1, Unread: 7})
	require.NoError(t, err)
	var frame SSEFrame
	require.NoError(t, json.Unmarshal(raw, &frame))
	assert.Equal(t, "notify", frame.Type)
	assert.Equal(t, int64(7), frame.Unread)
	// recipient 不入前端帧(避免向 A 泄漏 B 的 recipient_id)
	assert.NotContains(t, string(raw), "recipient_id")
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate([]byte("abc"), 10))
	assert.Equal(t, "abcde", truncate([]byte("abcde"), 5))
	got := truncate([]byte("abcdefghij"), 3)
	assert.Equal(t, "abc...(7 more bytes)", got)
}

func TestNotifyTransformer(t *testing.T) {
	userIDs, frame, err := NotifyTransformer([]byte(`{"recipient_id":100,"unread":3}`))
	require.NoError(t, err)
	assert.Equal(t, []int64{100}, userIDs)

	var f SSEFrame
	require.NoError(t, json.Unmarshal(frame, &f))
	assert.Equal(t, "notify", f.Type)
	assert.Equal(t, int64(3), f.Unread)

	_, _, err = NotifyTransformer([]byte("garbage"))
	assert.Error(t, err)
}

func TestPubSubHandle(t *testing.T) {
	m := NewConnManager()
	done := make(chan struct{})
	c1 := NewConn(done)
	c2 := NewConn(done)
	m.Add(100, c1)
	m.Add(200, c2)
	p := &PubSub{cm: m, tf: NotifyTransformer}

	p.handle([]byte(`{"recipient_id":100,"unread":2}`))
	select {
	case got := <-c1.Send:
		var f SSEFrame
		require.NoError(t, json.Unmarshal(got, &f))
		assert.Equal(t, int64(2), f.Unread)
	default:
		t.Fatal("期望 100 收到通知帧")
	}

	// 非法载荷丢弃,不 panic 也不投递
	p.handle([]byte("garbage"))
	assert.Equal(t, 0, len(c1.Send))
	assert.Equal(t, 0, len(c2.Send))

	// 多接收者逐用户投递
	p.tf = func([]byte) ([]int64, []byte, error) {
		return []int64{100, 200}, []byte("multi"), nil
	}
	p.handle([]byte("x"))
	assert.Equal(t, "multi", string(<-c1.Send))
	assert.Equal(t, "multi", string(<-c2.Send))

	// 无接收者不投递
	p.tf = func([]byte) ([]int64, []byte, error) {
		return nil, []byte("none"), nil
	}
	p.handle([]byte("x"))
	assert.Equal(t, 0, len(c1.Send))
	assert.Equal(t, 0, len(c2.Send))
}

func TestPubSub_订阅投递(t *testing.T) {
	mr := miniredis.RunT(t)

	m := NewConnManager()
	conn := NewConn(make(chan struct{}))
	m.Add(100, conn)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ps := NewPubSub(redis.RedisConf{Host: mr.Addr()}, rediskey.NotifyPushChannel, m, NotifyTransformer)
	ps.Start(ctx)

	pub := red.NewClient(&red.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = pub.Close() })

	payload := `{"recipient_id":100,"unread":5}`
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("3s 内未收到 SSE 帧")
		case <-tick.C:
			// 订阅建立是异步的,首帧到达前反复发布
			require.NoError(t, pub.Publish(ctx, rediskey.NotifyPushChannel, payload).Err())
		case got := <-conn.Send:
			var f SSEFrame
			require.NoError(t, json.Unmarshal(got, &f))
			assert.Equal(t, "notify", f.Type)
			assert.Equal(t, int64(5), f.Unread)
			return
		}
	}
}

func TestIMTransformer(t *testing.T) {
	raw := []byte(`{"user_ids":[100,200],"frame":{"type":"message","seq":"42"}}`)
	userIDs, frame, err := IMTransformer(raw)
	require.NoError(t, err)
	assert.Equal(t, []int64{100, 200}, userIDs)

	var f map[string]any
	require.NoError(t, json.Unmarshal(frame, &f))
	assert.Equal(t, "message", f["type"])
	assert.Equal(t, "42", f["seq"])

	for _, bad := range []string{
		"",
		"garbage",
		`{"user_ids":[],"frame":{"type":"message"}}`,
		`{"user_ids":[1],"frame":null}`,
		`{"user_ids":[0,-1],"frame":{"type":"message"}}`,
	} {
		_, _, err := IMTransformer([]byte(bad))
		assert.Error(t, err, bad)
	}
}
