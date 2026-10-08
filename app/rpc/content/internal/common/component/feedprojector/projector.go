// Package feedprojector 把内容生命周期事件投影到 feed 各存储
// 消息消费链路与对账补跑链路共用同一份投影语义 避免两条路径各自演进
// 它只做分发与编排 每份状态的写入都委托给对应的 owner 组件
package feedprojector

import (
	"context"

	"ran-feed/app/rpc/content/internal/common/component/contentcache"
	"ran-feed/app/rpc/content/internal/common/component/feedpub"
	"ran-feed/app/rpc/content/internal/common/component/hotfeed"
	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	"ran-feed/app/rpc/content/internal/repositories"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event/contentevent"
)

// Projector 内容事件到 feed 存储的投影
type Projector struct {
	contentRepo   repositories.ContentRepository
	contentCache  *contentcache.Cache
	hotFeed       *hotfeed.Feed
	publishBox    *publishbox.Box
	feedPublisher *feedpub.Publisher
}

func New(
	contentRepo repositories.ContentRepository,
	contentCache *contentcache.Cache,
	hotFeed *hotfeed.Feed,
	publishBox *publishbox.Box,
	feedPublisher *feedpub.Publisher,
) *Projector {
	return &Projector{
		contentRepo:   contentRepo,
		contentCache:  contentCache,
		hotFeed:       hotFeed,
		publishBox:    publishBox,
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
