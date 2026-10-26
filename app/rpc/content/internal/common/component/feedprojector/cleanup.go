package feedprojector

import (
	"context"
)

// cleanup 删除或下架热榜加作者发件箱加失效内容详情二级缓存
func (p *Projector) cleanup(ctx context.Context, contentID, authorID int64) error {
	if err := p.contentCache.Invalidate(ctx, contentID); err != nil {
		return err
	}
	if err := p.hotFeed.Remove(ctx, contentID); err != nil {
		return err
	}
	if err := p.publishBox.Remove(ctx, authorID, contentID); err != nil {
		return err
	}
	return nil
}
