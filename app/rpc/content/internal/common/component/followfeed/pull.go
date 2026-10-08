package followfeed

import (
	"context"
	"strconv"
	"time"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// InvalidatePull 失效 viewer 的拉模式集 关注关系变更后调用 下次读按新关注集合重算
func (f *Feed) InvalidatePull(ctx context.Context, viewerID int64) error {
	if viewerID <= 0 {
		return nil
	}
	_, err := f.redis.DelCtx(ctx, rediskey.BuildFollowPullKey(viewerID))
	return err
}

// ensurePull 拉模式集在即直读 否则重算一次 只翻关注列表不扫库 代价远低于收件箱重建 故不抢锁
// 拉侧 TTL 比收件箱短 让大 V 判定能周期性刷新 棘轮语义下集合只增不减 重算期间的重叠由合并去重兜住
func (f *Feed) ensurePull(ctx context.Context, viewerID int64) ([]int64, bool) {
	pullKey := rediskey.BuildFollowPullKey(viewerID)
	if f.keyExists(ctx, pullKey) {
		return f.readPull(ctx, pullKey), true
	}

	followees, err := f.listFolloweesCapped(ctx, viewerID, followeesScanLimit)
	if err != nil {
		f.logger(ctx).Errorf("查询关注列表失败 viewerID=%d err=%v", viewerID, err)
		return nil, false
	}
	bigVs, _, err := f.splitFollowees(ctx, followees)
	if err != nil {
		f.logger(ctx).Errorf("判定大 V 关注失败 viewerID=%d err=%v", viewerID, err)
		return nil, false
	}
	if werr := f.writePull(ctx, pullKey, bigVs); werr != nil {
		f.logger(ctx).Errorf("写拉模式关注集失败 viewerID=%d err=%v", viewerID, werr)
		return nil, false
	}
	return bigVs, true
}

// readPull 读拉模式集 已剔除哨兵
func (f *Feed) readPull(ctx context.Context, pullKey string) []int64 {
	members, err := f.redis.SmembersCtx(ctx, pullKey)
	if err != nil {
		f.logger(ctx).Errorf("读拉模式关注集失败 pullKey=%s err=%v", pullKey, err)
		return nil
	}
	return parseIDMembers(members)
}

// writePull 写拉模式集 空集写哨兵防穿透
func (f *Feed) writePull(ctx context.Context, pullKey string, authorIDs []int64) error {
	members := make([]any, 0, len(authorIDs)+1)
	if len(authorIDs) == 0 {
		members = append(members, rediskey.FollowPullEmptySentinel)
	} else {
		for _, id := range authorIDs {
			members = append(members, strconv.FormatInt(id, 10))
		}
	}
	ttl := time.Duration(contentconsts.PullSetTTLSeconds) * time.Second
	return f.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, pullKey)
		pipe.SAdd(ctx, pullKey, members...)
		pipe.Expire(ctx, pullKey, ttl)
		return nil
	})
}

// fromSource 缓存不可用时的安全降级 直接回源出窗口内全部推侧候选与拉侧作者集 不写缓存 不返回空
// 游标过滤与截断交调用方的合并步骤 由它算 hasMore
func (f *Feed) fromSource(ctx context.Context, viewerID int64) Sources {
	followees, err := f.listFolloweesCapped(ctx, viewerID, followeesScanLimit)
	if err != nil {
		f.logger(ctx).Errorf("降级查询关注列表失败 viewerID=%d err=%v", viewerID, err)
		return Sources{}
	}
	bigVs, small, err := f.splitFollowees(ctx, followees)
	if err != nil {
		f.logger(ctx).Errorf("降级判定大 V 关注失败 viewerID=%d err=%v", viewerID, err)
		return Sources{}
	}
	if len(small) == 0 {
		return Sources{PullAuthors: bigVs}
	}

	rows, err := f.contentRepo.ListFollowByAuthorsCursor(ctx,
		contentEnum.ContentStatusPublished.Int32(),
		contentEnum.VisibilityPublic.Int32(),
		small, 0, int(contentconsts.TimelineKeepN),
	)
	if err != nil {
		f.logger(ctx).Errorf("降级查询关注内容失败 viewerID=%d err=%v", viewerID, err)
		return Sources{PullAuthors: bigVs}
	}
	return Sources{PushItems: rowsToScored(rows), PullAuthors: bigVs}
}

// parseIDMembers 解析集合成员为 id 跳过空串与哨兵
func parseIDMembers(members []string) []int64 {
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m == "" || m == rediskey.FollowPullEmptySentinel {
			continue
		}
		id, err := strconv.ParseInt(m, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}
