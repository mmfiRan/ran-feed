package feedprojector

import (
	"context"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/pkg/event/contentevent"
)

// syncPublished 触发 feed 流的发布 先失效二级缓存 再处理一级缓存
// 事件到达时内容可能已不在可见状态 以库内当前状态为准 不满足则退化成清理
func (p *Projector) syncPublished(ctx context.Context, evt *contentevent.ContentEvent) error {
	row, err := p.contentRepo.GetDetailByID(ctx, evt.ContentID)
	if err != nil {
		return err
	}
	if row == nil ||
		row.Status != contentEnum.ContentStatusPublished.Int32() ||
		row.Visibility != contentEnum.VisibilityPublic.Int32() {
		return p.cleanup(ctx, evt.ContentID, evt.AuthorID)
	}
	// 先失效 L2 再写 L1
	if err = p.contentCache.Invalidate(ctx, evt.ContentID); err != nil {
		return err
	}
	var publishedAtMillis int64
	if row.PublishedAt != nil {
		publishedAtMillis = row.PublishedAt.UnixMilli()
	}
	return p.feedPublisher.Publish(ctx, evt.ContentID, row.UserID, publishedAtMillis, contentEnum.VisibilityEnum(row.Visibility))
}
