// Package feedpub 负责内容进入 feed 的发布副作用汇总 由 ServiceContext 注入
// 它只做编排 各份状态的写入都委托给对应的 owner 组件
package feedpub

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/internal/common/component/bigv"
	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/mq/event"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/pkg/consts"
	sharedkey "ran-feed/pkg/rediskey"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// FanOutPublisher 扇出分批消息生产者
type FanOutPublisher interface {
	PublishBatch(ctx context.Context, batch *event.FanOutBatch) error
}

type Publisher struct {
	redis      *redis.Redis
	followRpc  followservice.FollowService
	bigv       *bigv.Set
	publishBox *publishbox.Box
	producer   FanOutPublisher
}

func NewPublisher(
	rds *redis.Redis,
	followRpc followservice.FollowService,
	bigvSet *bigv.Set,
	publishBox *publishbox.Box,
	producer FanOutPublisher,
) *Publisher {
	return &Publisher{
		redis:      rds,
		followRpc:  followRpc,
		bigv:       bigvSet,
		publishBox: publishBox,
		producer:   producer,
	}
}

// Publish 内容进入 feed 时的副作用汇总 由 feed 消费者消费发布事件触发
// 写作者发件箱 加 登记热榜脏集合 加 投递分批扇出消息
// 消费者已在后台 同步执行并返回错误 失败由 kafka 重投重放(内部均幂等)
func (p *Publisher) Publish(ctx context.Context, contentID, authorID, publishedAtMillis int64, visibility contentEnum.VisibilityEnum) error {
	// 发件箱只装公开内容 与扇出 热榜种子 大 V merge 口径一致 私密内容不进任何 feed
	if visibility != contentEnum.VisibilityPublic {
		return nil
	}
	if err := p.publishBox.Write(ctx, authorID, contentID, publishedAtMillis); err != nil {
		return err
	}
	if err := p.writePublishHotSeed(ctx, contentID); err != nil {
		return err
	}
	return p.fanOutToFollowers(ctx, authorID, contentID, publishedAtMillis)
}

// writePublishHotSeed 发布即登记脏 把新内容放进热榜脏集合 让下一轮增量算分
func (p *Publisher) writePublishHotSeed(ctx context.Context, contentID int64) error {
	if contentID <= 0 {
		return nil
	}
	shard := int(contentID % consts.HotDirtyShards)
	_, err := p.redis.SaddCtx(ctx, sharedkey.HotFeedDirty(shard), strconv.FormatInt(contentID, 10))
	return err
}

// fanOutToFollowers 按粉丝游标分页 每页投一条扇出消息 由独立消费者逐批写收件箱
// 大 V 跳过由读路径 merge 大 V 判定失败宁可报错重试 误判成非大 V 会造成扩散风暴
func (p *Publisher) fanOutToFollowers(ctx context.Context, authorID, contentID, publishedAtMillis int64) error {
	if authorID <= 0 || contentID <= 0 {
		return nil
	}

	isBig, err := p.bigv.IsBigV(ctx, authorID)
	if err != nil {
		return err
	}
	if isBig {
		return nil
	}

	batchSize := contentconsts.FollowFanOutBatchSize
	cursor := int64(0)
	total := 0
	for {
		resp, err := p.followRpc.ListFollowers(ctx, &followservice.ListFollowersReq{
			UserId:   authorID,
			Cursor:   cursor,
			PageSize: uint32(batchSize),
		})
		if err != nil {
			return err
		}
		if resp == nil || len(resp.FollowerUserIds) == 0 {
			break
		}

		followerIDs := make([]int64, 0, len(resp.FollowerUserIds))
		for _, followerID := range resp.FollowerUserIds {
			if followerID > 0 {
				followerIDs = append(followerIDs, followerID)
			}
		}
		if len(followerIDs) > 0 {
			if err := p.publishBatch(ctx, contentID, authorID, publishedAtMillis, followerIDs); err != nil {
				return err
			}
			total += len(followerIDs)
		}

		if !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		cursor = resp.NextCursor
	}

	logx.WithContext(ctx).Infof("fan-out 分批投递完成 authorID=%d contentID=%d followers=%d", authorID, contentID, total)
	return nil
}

func (p *Publisher) publishBatch(ctx context.Context, contentID, authorID, publishedAtMillis int64, followerIDs []int64) error {
	if p.producer == nil {
		logx.WithContext(ctx).Errorf("扇出生产者未配置 跳过投递 contentID=%d followers=%d", contentID, len(followerIDs))
		return nil
	}
	return p.producer.PublishBatch(ctx, &event.FanOutBatch{
		ContentID:   contentID,
		AuthorID:    authorID,
		PublishedAt: publishedAtMillis,
		FollowerIDs: followerIDs,
	})
}
