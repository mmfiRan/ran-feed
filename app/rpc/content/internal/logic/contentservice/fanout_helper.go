package contentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	defaultFanOutBatchSize  int   = 500
	defaultFanOutInboxKeepN int64 = 5000
	fanOutBackgroundTimeout       = 30 * time.Second
)

// RunPublishFeedEffects 内容进入 feed 时的副作用汇总 先审后发下由审核通过触发(原发布时触发)
// 写作者 publish zset + 公开内容登记热榜脏集合 + 小账号扩散到 follower inbox 错误只记日志不阻断
// 供发布链路与 admin 审核链路复用 须在数据落库事务提交后调用(内含 Redis/RPC 副作用)
func RunPublishFeedEffects(ctx context.Context, svcCtx *svc.ServiceContext, contentID, userID, publishedAtMillis int64, visibility content.Visibility) {
	logger := logx.WithContext(ctx)
	feedKey := rediskey.BuildUserPublishFeedKey(userID)
	if err := writeUserPublishZSet(ctx, svcCtx, feedKey, contentID, publishedAtMillis); err != nil {
		logger.Errorf("更新用户发布列表缓存失败 contentId=%d: %v", contentID, err)
	}
	if shouldSeedHotIncrement(visibility) {
		if err := writePublishHotSeed(ctx, svcCtx, contentID); err != nil {
			logger.Errorf("写热榜增量失败 contentId=%d: %v", contentID, err)
		}
	}
	// 推拉结合 小账号 fan-out 到 follower inbox 大 V 跳过
	fanOutToFollowersAsync(svcCtx, userID, contentID, publishedAtMillis, visibility)
}

// writeUserPublishZSet 写单条内容到作者 publish zset score=published_at
// publish zset 承载作者全量发布历史 不按时间裁剪 cutoff=0 仅 keepN 与 TTL 控量
func writeUserPublishZSet(ctx context.Context, svcCtx *svc.ServiceContext, feedKey string, contentID, publishedAtMillis int64) error {
	days := svcCtx.Config.FollowFanOut.DeadlineWindowDays
	args := followwindow.WriteArgs(userPublishFeedKeepN, 0, followwindow.TTLSeconds(days), publishedAtMillis, contentID)
	_, err := svcCtx.Redis.EvalCtx(ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

// fanOutToFollowersAsync 发布后异步推送到 follower 收件箱
// 小账号（粉丝数 < 阈值）走推；大 V 跳过，由读路径 merge
// 与 publish 主流程解耦，错误只记日志
func fanOutToFollowersAsync(svcCtx *svc.ServiceContext, authorID, contentID, publishedAtMillis int64, visibility content.Visibility) {
	// 仅 PUBLIC 内容才推（PRIVATE 不进 feed）
	if visibility != content.Visibility_PUBLIC {
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
		isBig, err := isBigVAuthor(ctx, svcCtx, authorID)
		if err != nil {
			logger.Errorf("fan-out 查大 V 集合失败 authorID=%d: %v", authorID, err)
			return
		}
		if isBig {
			return
		}

		// 2. 小账号：分批拉取 follower 列表并写入各自 inbox
		batchSize := svcCtx.Config.FollowFanOut.BatchSize
		if batchSize <= 0 {
			batchSize = defaultFanOutBatchSize
		}
		keepN := svcCtx.Config.FollowFanOut.InboxKeepN
		if keepN <= 0 {
			keepN = defaultFanOutInboxKeepN
		}

		pushToFollowers(ctx, svcCtx, authorID, contentID, publishedAtMillis, batchSize, keepN, logger)
	})
}

func pushToFollowers(ctx context.Context, svcCtx *svc.ServiceContext, authorID, contentID, publishedAtMillis int64, batchSize int, keepN int64, logger logx.Logger) {
	days := svcCtx.Config.FollowFanOut.DeadlineWindowDays
	inboxArgs := followwindow.WriteArgs(keepN, followwindow.CutoffMillis(days), followwindow.TTLSeconds(days), publishedAtMillis, contentID)
	cursor := int64(0)
	totalPushed := 0

	for {
		resp, err := svcCtx.FollowRpc.ListFollowers(ctx, &followservice.ListFollowersReq{
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
			if _, err := svcCtx.Redis.EvalCtx(
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
