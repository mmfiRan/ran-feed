// Package feedpub 内容进入 feed 的发布副作用 由 ServiceContext 注入
// 只依赖具体组件(redis/followRpc/config) 不依赖 svcCtx 避免 svc 反向引用逻辑层
package feedpub

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	publishFeedKeepN        int64 = 5000
	defaultFanOutBatchSize  int   = 500
	defaultFanOutInboxKeepN int64 = 5000
	fanOutBackgroundTimeout       = 30 * time.Second
)

type Publisher struct {
	redis     *redis.Redis
	followRpc followservice.FollowService
	conf      config.FollowFanOutConfig
}

func NewPublisher(rds *redis.Redis, followRpc followservice.FollowService, conf config.FollowFanOutConfig) *Publisher {
	return &Publisher{redis: rds, followRpc: followRpc, conf: conf}
}

// Publish 内容进入 feed 时的副作用汇总 先审后发下由审核通过触发
// 写作者 publish zset + 公开内容登记热榜脏集合 + 小账号扩散到 follower inbox 错误只记日志不阻断
// 须在数据落库事务提交后调用(内含 Redis/RPC 副作用)
func (p *Publisher) Publish(ctx context.Context, contentID, authorID, publishedAtMillis int64, visibility content.Visibility) {
	logger := logx.WithContext(ctx)
	feedKey := rediskey.BuildUserPublishFeedKey(authorID)
	if err := p.writeUserPublishZSet(ctx, feedKey, contentID, publishedAtMillis); err != nil {
		logger.Errorf("更新用户发布列表缓存失败 contentId=%d: %v", contentID, err)
	}
	if visibility == content.Visibility_VISIBILITY_PUBLIC {
		if err := p.writePublishHotSeed(ctx, contentID); err != nil {
			logger.Errorf("写热榜增量失败 contentId=%d: %v", contentID, err)
		}
	}
	// 推拉结合 小账号 fan-out 到 follower inbox 大 V 跳过
	p.fanOutToFollowersAsync(authorID, contentID, publishedAtMillis, visibility)
}

// IsBigVAuthor 命中全局大 V 集合即大 V 读失败保守按非大 V 处理 backfill/purge/fanout 共用
func (p *Publisher) IsBigVAuthor(ctx context.Context, authorID int64) (bool, error) {
	return p.redis.SismemberCtx(ctx, rediskey.RedisFeedBigVGlobalKey, strconv.FormatInt(authorID, 10))
}

// writeUserPublishZSet 写单条内容到作者 publish zset score=published_at
// publish zset 承载作者全量发布历史 不按时间裁剪 cutoff=0 仅 keepN 与 TTL 控量
func (p *Publisher) writeUserPublishZSet(ctx context.Context, feedKey string, contentID, publishedAtMillis int64) error {
	days := p.conf.DeadlineWindowDays
	args := followwindow.WriteArgs(publishFeedKeepN, 0, followwindow.TTLSeconds(days), publishedAtMillis, contentID)
	_, err := p.redis.EvalCtx(ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

// writePublishHotSeed 发布即登记脏 把新内容放进热榜脏集合 让下一轮快更算分
// 0 互动新内容靠加法时间项进 TopN 冷启动自带解药 见 HOT_FEED_DESIGN 第2节 无需额外 seed 分值
func (p *Publisher) writePublishHotSeed(ctx context.Context, contentID int64) error {
	if contentID <= 0 {
		return nil
	}
	shard := int(contentID % int64(rediskey.RedisFeedHotIncDefaultShards))
	dirtyKey := rediskey.BuildHotFeedDirtyKey(shard)
	_, err := p.redis.SaddCtx(ctx, dirtyKey, strconv.FormatInt(contentID, 10))
	return err
}

// fanOutToFollowersAsync 发布后异步推送到 follower 收件箱
// 小账号（粉丝数 < 阈值）走推；大 V 跳过，由读路径 merge
// 与 publish 主流程解耦，错误只记日志
func (p *Publisher) fanOutToFollowersAsync(authorID, contentID, publishedAtMillis int64, visibility content.Visibility) {
	// 仅 PUBLIC 内容才推（PRIVATE 不进 feed）
	if visibility != content.Visibility_VISIBILITY_PUBLIC {
		return
	}
	if authorID <= 0 || contentID <= 0 {
		return
	}

	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), fanOutBackgroundTimeout)
		defer cancel()
		logger := logx.WithContext(ctx)

		// 1. 命中全局大 V 集合则跳过推送 由读路径 merge 查询失败保守跳过避免误发扩散风暴
		isBig, err := p.IsBigVAuthor(ctx, authorID)
		if err != nil {
			logger.Errorf("fan-out 查大 V 集合失败 authorID=%d: %v", authorID, err)
			return
		}
		if isBig {
			return
		}

		// 2. 小账号：分批拉取 follower 列表并写入各自 inbox
		batchSize := p.conf.BatchSize
		if batchSize <= 0 {
			batchSize = defaultFanOutBatchSize
		}
		keepN := p.conf.InboxKeepN
		if keepN <= 0 {
			keepN = defaultFanOutInboxKeepN
		}

		p.pushToFollowers(ctx, authorID, contentID, publishedAtMillis, batchSize, keepN, logger)
	})
}

func (p *Publisher) pushToFollowers(ctx context.Context, authorID, contentID, publishedAtMillis int64, batchSize int, keepN int64, logger logx.Logger) {
	days := p.conf.DeadlineWindowDays
	inboxArgs := followwindow.WriteArgs(keepN, followwindow.CutoffMillis(days), followwindow.TTLSeconds(days), publishedAtMillis, contentID)
	cursor := int64(0)
	totalPushed := 0

	for {
		resp, err := p.followRpc.ListFollowers(ctx, &followservice.ListFollowersReq{
			UserId:   authorID,
			Cursor:   cursor,
			PageSize: uint32(batchSize),
		})
		if err != nil {
			logger.Errorf("fan-out 拉粉丝列表失败 authorID=%d cursor=%d: %v", authorID, cursor, err)
			return
		}
		if resp == nil || len(resp.FollowerUserIds) == 0 {
			break
		}

		for _, followerID := range resp.FollowerUserIds {
			if followerID <= 0 {
				continue
			}
			inboxKey := rediskey.BuildFollowInboxKey(followerID)
			if _, err := p.redis.EvalCtx(
				ctx,
				luautils.UpdateFollowInboxZSetScript,
				[]string{inboxKey},
				inboxArgs...,
			); err != nil {
				// 单个 follower 失败不影响其他人，只记日志
				logger.Errorf("fan-out 写 inbox 失败 followerID=%d contentID=%d: %v", followerID, contentID, err)
				continue
			}
			totalPushed++
		}

		if !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		cursor = resp.NextCursor
	}

	logger.Infof("fan-out 完成 authorID=%d contentID=%d pushed=%d", authorID, contentID, totalPushed)
}
