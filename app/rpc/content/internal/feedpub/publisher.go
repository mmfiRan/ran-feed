// Package feedpub 内容进入 feed 的发布副作用 由 ServiceContext 注入
// 只依赖具体组件(redis/followRpc/config/producer) 不依赖 svcCtx 避免 svc 反向引用逻辑层
package feedpub

import (
	"context"
	"strconv"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/mq/event"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/pkg/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// FanOutPublisher 扇出分批消息生产者 抽象便于替换与测试
type FanOutPublisher interface {
	PublishBatch(ctx context.Context, batch *event.FanOutBatch) error
}

type Publisher struct {
	redis     *redis.Redis
	followRpc followservice.FollowService
	producer  FanOutPublisher
}

func NewPublisher(rds *redis.Redis, followRpc followservice.FollowService, producer FanOutPublisher) *Publisher {
	return &Publisher{
		redis:     rds,
		followRpc: followRpc,
		producer:  producer,
	}
}

// Publish 内容进入 feed 时的副作用汇总 由 feed 消费者消费发布事件触发
// 写作者 publish zset + 登记热榜脏集合 + 投递分批扇出消息
// 消费者已在后台 同步执行并返回错误 失败由 kafka 重投重放(内部均幂等)
func (p *Publisher) Publish(ctx context.Context, contentID, authorID, publishedAtMillis int64, visibility contentEnum.VisibilityEnum) error {
	// 发件箱只装公开内容 与扇出 热榜种子 大 V merge 口径一致 私密内容不进任何 feed
	if visibility != contentEnum.VisibilityPublic {
		return nil
	}
	feedKey := rediskey.BuildUserPublishFeedKey(authorID)
	if err := p.writeUserPublishZSet(ctx, feedKey, contentID, publishedAtMillis); err != nil {
		return err
	}
	if err := p.writePublishHotSeed(ctx, contentID); err != nil {
		return err
	}
	return p.fanOutToFollowers(ctx, authorID, contentID, publishedAtMillis, visibility)
}

// IsBigVAuthor 命中全局大 V 集合即大 V 读失败保守按非大 V 处理 backfill/purge/fanout 共用
func (p *Publisher) IsBigVAuthor(ctx context.Context, authorID int64) (bool, error) {
	return p.redis.SismemberCtx(ctx, rediskey.RedisFeedBigVGlobalKey, strconv.FormatInt(authorID, 10))
}

// writeUserPublishZSet 写单条内容到作者 publish zset score=published_at
// publish zset 承载作者全量发布历史 不按时间裁剪 cutoff=0 仅 keepN 与 TTL 控量
func (p *Publisher) writeUserPublishZSet(ctx context.Context, feedKey string, contentID, publishedAtMillis int64) error {
	days := contentconsts.WindowDays
	args := followwindow.WriteArgs(contentconsts.TimelineKeepN, 0, followwindow.TTLSeconds(days), publishedAtMillis, contentID)
	_, err := p.redis.EvalCtx(ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

// writePublishHotSeed 发布即登记脏 把新内容放进热榜脏集合 让下一轮增量算分
func (p *Publisher) writePublishHotSeed(ctx context.Context, contentID int64) error {
	if contentID <= 0 {
		return nil
	}
	shard := int(contentID % consts.HotDirtyShards)
	dirtyKey := rediskey.BuildHotFeedDirtyKey(shard)
	_, err := p.redis.SaddCtx(ctx, dirtyKey, strconv.FormatInt(contentID, 10))
	return err
}

// fanOutToFollowers 按粉丝游标分页 每页投一条扇出消息 由独立消费者逐批写收件箱
// 大 V 跳过由读路径 merge 命中大 V 查询失败返回错误重试避免误判非大 V 造成扩散风暴
// feed 消费者只做分页与投递 O(1) 级动作 O(粉丝数) 的写收件箱下沉到扇出消费者
func (p *Publisher) fanOutToFollowers(ctx context.Context, authorID, contentID, publishedAtMillis int64, visibility contentEnum.VisibilityEnum) error {
	if visibility != contentEnum.VisibilityPublic {
		return nil
	}
	if authorID <= 0 || contentID <= 0 {
		return nil
	}

	isBig, err := p.IsBigVAuthor(ctx, authorID)
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
