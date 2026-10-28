package sse

import (
	"context"
	"strconv"
	"time"

	red "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/threading"
)

// Transformer 把 pubsub 原始载荷转为目标用户与 SSE 帧字节
// userIDs 为空表示无接收者,err 非空表示载荷非法由调用方丢弃并记日志
type Transformer func(raw []byte) (userIDs []int64, frame []byte, err error)

// PubSub 独立 go-redis client 订阅指定 channel 分发到 ConnManager
// channel 与帧转换由构造参数注入,notification 与 im 各自一个实例
type PubSub struct {
	client  *red.Client
	channel string
	cm      *ConnManager
	tf      Transformer
}

// NewPubSub 用 front-api.yaml 的 RedisConfig 建独立 go-redis client
// 目前仅支持 node 模式(与项目现有 RedisConfig 使用一致)
func NewPubSub(conf redis.RedisConf, channel string, cm *ConnManager, tf Transformer) *PubSub {
	dbNum := 0
	client := red.NewClient(&red.Options{
		Addr:     conf.Host,
		Username: conf.User,
		Password: conf.Pass,
		DB:       dbNum,
	})
	return &PubSub{client: client, channel: channel, cm: cm, tf: tf}
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

	// 订阅循环,出错重连避免瞬时网络抖动导致永久失联
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
	sub := p.client.Subscribe(ctx, p.channel)
	defer func() { _ = sub.Close() }()

	ch := sub.Channel()
	logx.Infof("sse pubsub 订阅启动 channel=%s", p.channel)

	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-ch:
			if !ok {
				logx.Errorf("sse pubsub channel 关闭 触发重连 channel=%s", p.channel)
				return
			}
			p.handle([]byte(m.Payload))
		}
	}
}

func (p *PubSub) handle(payload []byte) {
	userIDs, frame, err := p.tf(payload)
	if err != nil {
		logx.Errorf("sse pubsub 丢弃非法 payload channel=%s err=%v raw=%s", p.channel, err, truncate(payload, 128))
		return
	}
	if len(userIDs) == 0 || len(frame) == 0 {
		return
	}
	for _, userID := range userIDs {
		delivered, dropped := p.cm.Broadcast(userID, frame)
		if delivered == 0 && dropped == 0 {
			// 本 pod 未持有该 user 属正常情况(多 pod 每 pod 都收到 pubsub 但只有对应实例响应)
			continue
		}
		if dropped > 0 {
			logx.Infof("sse broadcast channel=%s userID=%d delivered=%d dropped=%d(慢连接)", p.channel, userID, delivered, dropped)
		}
	}
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(" + strconv.Itoa(len(b)-max) + " more bytes)"
}
