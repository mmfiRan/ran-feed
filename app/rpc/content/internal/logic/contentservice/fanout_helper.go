package contentservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	defaultBigVThreshold     int64 = 5000
	defaultFanOutBatchSize   int   = 500
	defaultFanOutInboxKeepN  int64 = 5000
	fanOutBackgroundTimeout        = 30 * time.Second
)

// fanOutToFollowersAsync 发布后异步推送到 follower 收件箱
// 小账号（粉丝数 < 阈值）走推；大 V 跳过，由读路径 merge
// 与 publish 主流程解耦，错误只记日志
func fanOutToFollowersAsync(svcCtx *svc.ServiceContext, authorID, contentID int64, visibility content.Visibility) {
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

		// 1. 查粉丝数，判断是否大 V
		threshold := svcCtx.Config.FollowFanOut.BigVFollowerThreshold
		if threshold <= 0 {
			threshold = defaultBigVThreshold
		}
		followerCount, err := getFollowerCount(ctx, svcCtx, authorID)
		if err != nil {
			logger.Errorf("fan-out 查粉丝数失败 authorID=%d: %v", authorID, err)
			return
		}
		if followerCount >= threshold {
			// 大 V：跳过推送，由读路径 merge
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

		pushToFollowers(ctx, svcCtx, authorID, contentID, batchSize, keepN, logger)
	})
}

func getFollowerCount(ctx context.Context, svcCtx *svc.ServiceContext, authorID int64) (int64, error) {
	resp, err := svcCtx.CountRpc.GetCount(ctx, &count.GetCountReq{
		BizType:    count.BizType_FOLLOWED,
		TargetType: count.TargetType_USER,
		TargetId:   authorID,
	})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, nil
	}
	return resp.Value, nil
}

func pushToFollowers(ctx context.Context, svcCtx *svc.ServiceContext, authorID, contentID int64, batchSize int, keepN int64, logger logx.Logger) {
	contentIDStr := strconv.FormatInt(contentID, 10)
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
				strconv.FormatInt(keepN, 10),
				contentIDStr, contentIDStr,
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