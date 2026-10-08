package feedprojector

import (
	"context"
)

// cleanup 删除或下架的清理 摘热榜 加 摘作者发件箱 加 失效内容详情二级缓存
// follower 收件箱不主动清 读路径按已发布加公开回源过滤 死内容自然不可见
func (p *Projector) cleanup(ctx context.Context, contentID, authorID int64) error {
	if err := p.hotFeed.Remove(ctx, contentID); err != nil {
		return err
	}
	if err := p.publishBox.Remove(ctx, authorID, contentID); err != nil {
		return err
	}
	return p.contentCache.Invalidate(ctx, contentID)
}
