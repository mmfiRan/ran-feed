// Package feedprojector 把内容生命周期事件投影到 feed 各存储
// 消息消费链路与对账补跑链路共用同一份投影语义 避免两条路径各自演进
package feedprojector

import (
	"context"

	"ran-feed/app/rpc/content/internal/common/component/contentcache"
	"ran-feed/app/rpc/content/internal/common/component/feedpub"
	"ran-feed/app/rpc/content/internal/repositories"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event/contentevent"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Projector 内容事件到 feed 存储的投影
type Projector struct {
	redis         *redis.Redis
	contentRepo   repositories.ContentRepository
	contentCache  *contentcache.Cache
	feedPublisher *feedpub.Publisher
}

func New(
	redisClient *redis.Redis,
	contentRepo repositories.ContentRepository,
	contentCache *contentcache.Cache,
	feedPublisher *feedpub.Publisher,
) *Projector {
	return &Projector{
		redis:         redisClient,
		contentRepo:   contentRepo,
		contentCache:  contentCache,
		feedPublisher: feedPublisher,
	}
}

// Apply 按事件类型分发 未覆盖的类型直接忽略
func (p *Projector) Apply(ctx context.Context, evt *contentevent.ContentEvent) error {
	switch evt.EventType {
	case contentenums.EventTypePublished, contentenums.EventTypeRestored:
		return p.syncPublished(ctx, evt)
	case contentenums.EventTypeDeleted, contentenums.EventTypeTakenDown:
		return p.cleanup(ctx, evt.ContentID, evt.AuthorID)
	default:
		return nil
	}
}
