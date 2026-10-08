// Package followfeed 负责关注流两半 收件箱 feed:follow:inbox 与拉模式集 feed:follow:pull 的全部读写
// 两半共同维持一条不变式 推侧并集拉侧覆盖 viewer 的全部关注 任一半缺失都会漏内容
package followfeed

import (
	"context"

	"ran-feed/app/rpc/content/internal/common/component/bigv"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/pkg/cache"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// followeesScanLimit 重建时扫描关注上限 推拉两半共用单一上限
	// 两条上限不一致会造出既没推也不拉的黑洞
	followeesScanLimit = 5000
	// followeesPageSize 翻关注列表的单页大小
	followeesPageSize = 500
)

// ScoredID 收件箱 zset 的成员 ID 是 content_id Score 是发布时间毫秒
type ScoredID struct {
	ID    int64
	Score int64
}

// Sources 关注流一页的两侧来源 推侧是本页候选 拉侧是待查发件箱的作者
type Sources struct {
	PushItems   []ScoredID
	PushHasMore bool
	PullAuthors []int64
}

// Feed 关注流两半的读写入口
type Feed struct {
	redis       *redis.Redis
	locker      *cache.DistLocker
	followRpc   followservice.FollowService
	bigv        *bigv.Set
	contentRepo repositories.ContentRepository
}

func New(
	redisClient *redis.Redis,
	locker *cache.DistLocker,
	followRpc followservice.FollowService,
	bigvSet *bigv.Set,
	contentRepo repositories.ContentRepository,
) *Feed {
	return &Feed{
		redis:       redisClient,
		locker:      locker,
		followRpc:   followRpc,
		bigv:        bigvSet,
		contentRepo: contentRepo,
	}
}

// Sources 取本页推侧候选与拉侧作者集
// 两半独立判就绪独立重建 拉侧只要翻关注列表 推侧要扫库 代价差一个量级 不该互相拖
// 任一步失败或等锁超时都降级直接回源 不返回空
func (f *Feed) Sources(ctx context.Context, viewerID, cursorScore int64, pageSize int) Sources {
	pullAuthors, pullReady := f.ensurePull(ctx, viewerID)
	if !pullReady {
		return f.fromSource(ctx, viewerID)
	}

	inboxReady, err := f.ensureInbox(ctx, viewerID, pullAuthors)
	if err != nil {
		f.logger(ctx).Errorf("准备关注收件箱失败 viewerID=%d err=%v", viewerID, err)
	}
	if !inboxReady {
		return f.fromSource(ctx, viewerID)
	}

	items, hasMore, qerr := f.queryInbox(ctx, viewerID, cursorScore, pageSize)
	if qerr != nil {
		f.logger(ctx).Errorf("查询关注收件箱失败 viewerID=%d err=%v", viewerID, qerr)
		return f.fromSource(ctx, viewerID)
	}
	return Sources{PushItems: items, PushHasMore: hasMore, PullAuthors: pullAuthors}
}

// listFolloweesCapped 按 limit 截断的关注列表分页拉取
func (f *Feed) listFolloweesCapped(ctx context.Context, viewerID int64, limit int) ([]int64, error) {
	followees := make([]int64, 0)
	cursor := int64(0)
	for len(followees) < limit {
		pageSize := uint32(followeesPageSize)
		if remain := limit - len(followees); remain < int(pageSize) {
			pageSize = uint32(remain)
		}
		resp, err := f.followRpc.ListFollowees(ctx, &followservice.ListFolloweesReq{
			UserId:   viewerID,
			Cursor:   cursor,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.FollowUserIds) == 0 {
			break
		}
		followees = append(followees, resp.FollowUserIds...)
		if !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		cursor = resp.NextCursor
	}
	if len(followees) > limit {
		followees = followees[:limit]
	}
	return followees, nil
}

// splitFollowees 把关注列表切成拉侧大 V 与推侧小号 判定失败时全部当小号 宁可多推不可漏
func (f *Feed) splitFollowees(ctx context.Context, followees []int64) (bigVs, small []int64, err error) {
	if len(followees) == 0 {
		return nil, nil, nil
	}
	bigVs, err = f.bigv.Filter(ctx, followees)
	if err != nil {
		return nil, nil, err
	}
	return bigVs, excludeIDs(followees, bigVs), nil
}

// excludeIDs 返回 all 中不在 excluded 里的元素 保序
func excludeIDs(all, excluded []int64) []int64 {
	if len(all) == 0 {
		return nil
	}
	if len(excluded) == 0 {
		return all
	}
	excludedSet := make(map[int64]struct{}, len(excluded))
	for _, id := range excluded {
		excludedSet[id] = struct{}{}
	}
	out := make([]int64, 0, len(all))
	for _, id := range all {
		if _, ok := excludedSet[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}
