package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	red "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/threading"
)

// NotifyPushChannel 与 notification-rpc dispatch 侧对齐
const NotifyPushChannel = "notify:push"

// PubSubMessage notify:push channel 的信号 payload 与 notification-rpc dispatch 侧对齐
type PubSubMessage struct {
	RecipientID int64 `json:"recipient_id"`
	Unread      int64 `json:"unread"`
}

// SSEFrame 客户端事件帧 前端 EventSource 上按 type 分派
// type=notify 表示未读数变更 收到即走 /list 回拉
type SSEFrame struct {
	Type   string `json:"type"`
	Unread int64  `json:"unread"`
}

// EncodeMessage 把 pubsub 载荷 → SSE 帧序列化字节
func EncodeMessage(msg PubSubMessage) ([]byte, error) {
	return json.Marshal(SSEFrame{Type: "notify", Unread: msg.Unread})
}

// DecodePayload 解 pubsub 消息 空/非法 JSON/recipient<=0 返 error
func DecodePayload(raw []byte) (PubSubMessage, error) {
	if len(raw) == 0 {
		return PubSubMessage{}, fmt.Errorf("空载荷")
	}
	var msg PubSubMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return PubSubMessage{}, fmt.Errorf("解析 pubsub payload 失败: %w", err)
	}
	if msg.RecipientID <= 0 {
		return PubSubMessage{}, fmt.Errorf("非法 recipient_id=%d", msg.RecipientID)
	}
	return msg, nil
}

// PubSub 独立 go-redis client 订阅 notify:push 分发到 ConnManager
// 独立 client 是因为 go-zero core/stores/redis 未直接暴露 Subscribe 且订阅是长连接
// 与业务命令池分开更清晰
type PubSub struct {
	client *red.Client
	cm     *ConnManager
}

// NewPubSub 用 front-api.yaml 的 RedisConfig 建独立 go-redis client
// 目前仅支持 node 模式(与项目现有 RedisConfig 使用一致)
func NewPubSub(conf redis.RedisConf, cm *ConnManager) *PubSub {
	dbNum := 0
	client := red.NewClient(&red.Options{
		Addr:     conf.Host,
		Username: conf.User,
		Password: conf.Pass,
		DB:       dbNum,
	})
	return &PubSub{client: client, cm: cm}
}

// Start 起 goroutine 常驻订阅 ctx 取消则退出并关闭 client
func (p *PubSub) Start(ctx context.Context) {
	threading.GoSafe(func() {
		p.run(ctx)
	})
}

func (p *PubSub) run(ctx context.Context) {
	defer func() {
		if err := p.client.Close(); err != nil {
			logx.Errorf("sse pubsub client close err=%v", err)
		}
	}()

	// 订阅循环 出错重连避免瞬时网络抖动导致永久失联
	for {
		if ctx.Err() != nil {
			return
		}
		p.consume(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			// 断开后短暂退避再重连
		}
	}
}

func (p *PubSub) consume(ctx context.Context) {
	sub := p.client.Subscribe(ctx, NotifyPushChannel)
	defer func() { _ = sub.Close() }()

	ch := sub.Channel()
	logx.Infof("sse pubsub 订阅启动 channel=%s", NotifyPushChannel)

	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-ch:
			if !ok {
				logx.Errorf("sse pubsub channel 关闭 触发重连")
				return
			}
			p.handle([]byte(m.Payload))
		}
	}
}

func (p *PubSub) handle(payload []byte) {
	msg, err := DecodePayload(payload)
	if err != nil {
		logx.Errorf("sse pubsub 丢弃非法 payload err=%v raw=%s", err, truncate(payload, 128))
		return
	}
	frame, err := EncodeMessage(msg)
	if err != nil {
		logx.Errorf("sse pubsub 编码 frame 失败 err=%v recipient=%d", err, msg.RecipientID)
		return
	}
	delivered, dropped := p.cm.Broadcast(msg.RecipientID, frame)
	if delivered == 0 && dropped == 0 {
		// 本 pod 未持有该 user 属正常情况(多 pod 每 pod 都收到 pubsub 但只有对应实例响应)
		return
	}
	if dropped > 0 {
		logx.Infof("sse broadcast recipient=%d delivered=%d dropped=%d(慢连接)", msg.RecipientID, delivered, dropped)
	}
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(" + strconv.Itoa(len(b)-max) + " more bytes)"
}
