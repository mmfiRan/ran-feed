package feedservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	userPublishFeedKeepN = 5000

	userPublishFeedRebuildLockTTLSeconds = 30
	userPublishFeedRebuildRetryTimes     = 3
	userPublishFeedRebuildRetryInterval  = 80 * time.Millisecond
)

type UserPublishFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	resolver    *contentDetailResolver
}

func NewUserPublishFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishFeedLogic {
	return &UserPublishFeedLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		resolver:    newContentDetailResolver(ctx, svcCtx),
	}
}

func (l *UserPublishFeedLogic) UserPublishFeed(in *content.UserPublishFeedReq) (*content.UserPublishFeedRes, error) {
	if in == nil {
		return emptyUserPublishFeedRes(), nil
	}
	if in.AuthorId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	feedKey := buildUserPublishFeedKey(in.AuthorId)

	ids, nextCursor, hasMore, err := l.loadPageIDs(feedKey, in.AuthorId, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if ids == nil || len(ids) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	viewerID := int64(0)
	if in.ViewerId != nil {
		viewerID = *in.ViewerId
	}
	items, err := l.resolver.assembleItems(ids, viewerID, false)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	return &content.UserPublishFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyUserPublishFeedRes() *content.UserPublishFeedRes {
	return &content.UserPublishFeedRes{
		Items:      []*content.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func (l *UserPublishFeedLogic) loadPageIDs(feedKey string, authorID int64, cursor string, pageSize int) ([]int64, string, bool, error) {
	// 优先走 Redis zset，拿到当前页 content_id 列表
	ids, nextCursor, hasMore, cacheExists, err := l.queryUserPublishIDs(feedKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if cacheExists {
		return ids, nextCursor, hasMore, nil
	}

	// 缓存不存在时回源 DB，并用分布式锁避免并发重建击穿。
	lockKey := buildUserPublishFeedRebuildLockKey(authorID)
	rebuildLock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	rebuildLock.SetExpire(userPublishFeedRebuildLockTTLSeconds)
	locked, lockErr := rebuildLock.AcquireCtx(l.ctx)
	if lockErr != nil {
		return nil, "", false, errorx.Wrap(l.ctx, lockErr, errorx.NewMsg("查询失败请稍后重试"))
	}
	if locked {
		defer func() {
			if releaseOk, releaseErr := rebuildLock.ReleaseCtx(context.Background()); !releaseOk || releaseErr != nil {
				l.Errorf("释放分布式锁失败: %v", releaseErr)
			}
		}()

		// 抢到锁的请求负责全量构建该作者发布列表缓存
		allRows, qerr := l.queryUserPublishAllFromDB(authorID)
		if qerr != nil {
			return nil, "", false, qerr
		}
		if len(allRows) == 0 {
			return nil, "", false, nil
		}
		if uerr := l.updateUserPublishCache(feedKey, allRows); uerr != nil {
			l.Errorf("回填用户发布列表缓存失败:%v", uerr)
		}

		// 回源场景直接在内存中按 cursor 做一次分页，避免再走一遍 Redis
		pageRows := l.pageUserPublishRows(allRows, cursor, pageSize)
		if len(pageRows) > pageSize {
			hasMore = true
			if last := pageRows[pageSize-1]; last.PublishedAt != nil {
				nextCursor = strconv.FormatInt(last.PublishedAt.UnixMilli(), 10)
			}
			pageRows = pageRows[:pageSize]
		} else {
			hasMore = false
			nextCursor = ""
		}

		res := make([]int64, 0, len(pageRows))
		for _, row := range pageRows {
			if row == nil || row.ID <= 0 {
				continue
			}
			res = append(res, row.ID)
		}
		return res, nextCursor, hasMore, nil
	}

	// 未抢到锁的请求短暂等待并重试读取缓存。
	for i := 0; i < userPublishFeedRebuildRetryTimes; i++ {
		time.Sleep(userPublishFeedRebuildRetryInterval)
		ids, nextCursor, hasMore, cacheExists, err = l.queryUserPublishIDs(feedKey, cursor, pageSize)
		if err != nil {
			return nil, "", false, err
		}
		if cacheExists {
			return ids, nextCursor, hasMore, nil
		}
	}
	return nil, "", false, errorx.NewMsg("查询失败请稍后重试")
}

func buildUserPublishFeedKey(authorID int64) string {
	return rediskey.GetRedisPrefixKey("feed:user:publish", strconv.FormatInt(authorID, 10))
}

func buildUserPublishFeedRebuildLockKey(authorID int64) string {
	return rediskey.GetRedisPrefixKey("feed:user:publish:lock", strconv.FormatInt(authorID, 10))
}

func (l *UserPublishFeedLogic) queryUserPublishIDs(feedKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	// publish zset 承载全量历史 不按时间裁剪 cutoff=0 读时续期整 key TTL
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserPublishZSetScript,
		[]string{feedKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
		"0",
		strconv.Itoa(followwindow.TTLSeconds(days)),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布列表失败"))
	}
	items, nextCursor, hasMore, exists, ok := parseZSetReply(res)
	if !ok {
		return nil, "", false, false, errorx.NewMsg("查询发布列表失败")
	}
	if !exists {
		return nil, "", false, false, nil
	}
	return scoredIDsToIDs(items), nextCursor, hasMore, true, nil
}

func (l *UserPublishFeedLogic) queryUserPublishAllFromDB(authorID int64) ([]*model.RanFeedContent, error) {
	rows, err := l.contentRepo.ListPublishedByAuthor(authorID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布内容失败"))
	}
	return rows, nil
}

func (l *UserPublishFeedLogic) updateUserPublishCache(feedKey string, rows []*model.RanFeedContent) error {
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	pairs := make([]int64, 0, len(rows)*2)
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		pairs = append(pairs, r.PublishedAt.UnixMilli(), r.ID)
	}
	if len(pairs) == 0 {
		return nil
	}
	// publish zset 承载全量历史 不按时间裁剪 cutoff=0
	args := followwindow.WriteArgs(int64(userPublishFeedKeepN), 0, followwindow.TTLSeconds(days), pairs...)
	_, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.UpdateUserPublishZSetScript, []string{feedKey}, args...)
	return err
}

func (l *UserPublishFeedLogic) pageUserPublishRows(allRows []*model.RanFeedContent, cursor string, pageSize int) []*model.RanFeedContent {
	if len(allRows) == 0 {
		return allRows
	}

	// 游标按 published_at 毫秒 与 zset score 对齐 allRows 已按 published_at 倒序
	cursorMillis := int64(0)
	if cursor != "" {
		v, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil && v > 0 {
			cursorMillis = v
		}
	}

	res := make([]*model.RanFeedContent, 0, pageSize+1)
	// 多取 1 条用于判断 hasMore，避免额外 count 查询。
	for _, r := range allRows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		if cursorMillis > 0 && r.PublishedAt.UnixMilli() >= cursorMillis {
			continue
		}
		res = append(res, r)
		if len(res) >= pageSize+1 {
			break
		}
	}
	return res
}

