package feedservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
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
			nextCursor = strconv.FormatInt(pageRows[pageSize-1].ID, 10)
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
	// Lua 返回: [keyExists, hasMore, nextCursor, id1, id2, ...]
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserPublishZSetScript,
		[]string{feedKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布列表失败"))
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询发布列表失败")
	}

	existsVal, _ := luaReplyInt64(arr[0])
	cacheExists := existsVal == 1
	if !cacheExists {
		return nil, "", false, false, nil
	}

	hasMoreVal, _ := luaReplyInt64(arr[1])
	hasMore := hasMoreVal == 1
	nextCursor := ""
	if hasMore {
		if s, ok := luaReplyString(arr[2]); ok {
			nextCursor = s
		}
	}

	ids := make([]int64, 0, len(arr)-3)
	for i := 3; i < len(arr); i++ {
		s, _ := luaReplyString(arr[i])
		if s == "" {
			continue
		}
		id, parseErr := strconv.ParseInt(s, 10, 64)
		if parseErr != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nextCursor, hasMore, true, nil
}

func (l *UserPublishFeedLogic) queryUserPublishAllFromDB(authorID int64) ([]*model.RanFeedContent, error) {
	rows, err := l.contentRepo.ListPublishedByAuthor(authorID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询发布内容失败"))
	}
	return rows, nil
}

func (l *UserPublishFeedLogic) updateUserPublishCache(feedKey string, rows []*model.RanFeedContent) error {
	if len(rows) == 0 {
		return nil
	}
	args := make([]interface{}, 0, 1+len(rows)*2)
	args = append(args, strconv.FormatInt(int64(userPublishFeedKeepN), 10))
	// UpdateUserPublishZSetScript 参数约定: [keep_latest_n, member, score, ...]
	// 这里使用 content_id 作为 member 和 score，天然可做时间倒序游标分页。
	for _, r := range rows {
		if r == nil {
			continue
		}
		idStr := strconv.FormatInt(r.ID, 10)
		args = append(args, idStr, idStr)
	}
	_, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.UpdateUserPublishZSetScript,
		[]string{feedKey},
		args...,
	)
	return err
}

func (l *UserPublishFeedLogic) pageUserPublishRows(allRows []*model.RanFeedContent, cursor string, pageSize int) []*model.RanFeedContent {
	if len(allRows) == 0 {
		return allRows
	}

	cursorID := int64(0)
	if cursor != "" {
		v, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil && v > 0 {
			cursorID = v
		}
	}

	res := make([]*model.RanFeedContent, 0, pageSize+1)
	// 多取 1 条用于判断 hasMore，避免额外 count 查询。
	for _, r := range allRows {
		if r == nil {
			continue
		}
		if cursorID > 0 && r.ID >= cursorID {
			continue
		}
		res = append(res, r)
		if len(res) >= pageSize+1 {
			break
		}
	}
	return res
}

