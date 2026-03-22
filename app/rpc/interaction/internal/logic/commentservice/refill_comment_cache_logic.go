package commentservicelogic

import (
	"context"
	"hash/fnv"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	redislock "github.com/zeromicro/go-zero/core/stores/redis"
)

type RefillCommentCacheLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewRefillCommentCacheLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefillCommentCacheLogic {
	return &RefillCommentCacheLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RefillCommentCacheLogic) RefillCommentCache(in *interaction.RefillCommentCacheReq) (*interaction.RefillCommentCacheRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if len(in.CommentIds) == 0 {
		return &interaction.RefillCommentCacheRes{
			Comments: []*interaction.CommentItem{},
		}, nil
	}

	ordered := make([]int64, 0, len(in.CommentIds))
	seen := make(map[int64]struct{}, len(in.CommentIds))
	for _, id := range in.CommentIds {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	if len(ordered) == 0 {
		return &interaction.RefillCommentCacheRes{
			Comments: []*interaction.CommentItem{},
		}, nil
	}

	cacheHit, miss, cacheErr := l.batchGetFromRedis(ordered)
	if cacheErr != nil {
		cacheHit = map[int64]*interaction.CommentItem{}
		miss = ordered
	}
	if len(miss) == 0 {
		return &interaction.RefillCommentCacheRes{
			Comments: pickOrderedComments(ordered, cacheHit, nil),
		}, nil
	}

	lockKey := l.buildRefillLockKey(miss)
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(30)

	lockAcquired, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取分布式锁失败"))
	}

	if !lockAcquired {
		const (
			maxRetry    = 5
			baseSleepMs = 30
			jitterMs    = 50
		)
		for i := 0; i < maxRetry; i++ {
			select {
			case <-l.ctx.Done():
				return nil, l.ctx.Err()
			default:
			}
			sleep := time.Duration(baseSleepMs+rand.Intn(jitterMs)) * time.Millisecond
			time.Sleep(sleep)

			reHit, reMiss, reErr := l.batchGetFromRedis(miss)
			if reErr == nil && len(reMiss) == 0 {
				for id, it := range reHit {
					cacheHit[id] = it
				}
				return &interaction.RefillCommentCacheRes{
					Comments: pickOrderedComments(ordered, cacheHit, nil),
				}, nil
			}
		}

		dbMap, _, derr := l.queryFromDB(miss)
		if derr != nil {
			return nil, derr
		}
		for id, it := range dbMap {
			cacheHit[id] = it
		}
		return &interaction.RefillCommentCacheRes{
			Comments: pickOrderedComments(ordered, cacheHit, nil),
		}, nil
	}

	defer func() {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放分布式锁失败: %v", releaseErr)
		}
	}()

	reHit, reMiss, reErr := l.batchGetFromRedis(miss)
	if reErr == nil && len(reMiss) == 0 {
		for id, it := range reHit {
			cacheHit[id] = it
		}
		return &interaction.RefillCommentCacheRes{
			Comments: pickOrderedComments(ordered, cacheHit, nil),
		}, nil
	}

	dbMap, _, derr := l.queryFromDB(miss)
	if derr != nil {
		return nil, derr
	}
	for id, it := range dbMap {
		cacheHit[id] = it
	}
	return &interaction.RefillCommentCacheRes{
		Comments: pickOrderedComments(ordered, cacheHit, nil),
	}, nil
}

func (l *RefillCommentCacheLogic) batchGetFromRedis(ids []int64) (map[int64]*interaction.CommentItem, []int64, error) {
	keys := make([]string, 0, len(ids))
	argv := make([]string, 0, len(ids)+1)
	argv = append(argv, strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10))
	for _, id := range ids {
		keys = append(keys, rediskey.BuildCommentObjKey(strconv.FormatInt(id, 10)))
		argv = append(argv, strconv.FormatInt(id, 10))
	}
	args := make([]any, 0, len(argv))
	for _, v := range argv {
		args = append(args, v)
	}

	result, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.BatchGetCommentObjsScript, keys, args...)
	if err != nil {
		l.Errorf("RefillCommentCache redis lua失败: %v", err)
		return nil, nil, err
	}

	arr, ok := result.([]interface{})
	if !ok {
		return nil, nil, errorx.NewMsg("缓存数据格式错误")
	}

	const chunkSize = 12
	if len(arr)%chunkSize != 0 {
		return nil, nil, errorx.NewMsg("缓存数据长度错误")
	}

	cacheHit := make(map[int64]*interaction.CommentItem, len(ids))
	miss := make([]int64, 0)
	for i := 0; i+chunkSize-1 < len(arr); i += chunkSize {
		id := parseInt64(arr[i])
		if id <= 0 {
			continue
		}
		contentID := parseInt64(arr[i+1])
		userID := parseInt64(arr[i+2])
		replyToUserID := parseInt64(arr[i+3])
		parentID := parseInt64(arr[i+4])
		rootID := parseInt64(arr[i+5])
		comment, _ := arr[i+6].(string)
		createdAt := parseInt64(arr[i+7])
		status := int32(parseInt64(arr[i+8]))
		userName, _ := arr[i+9].(string)
		userAvatar, _ := arr[i+10].(string)
		replyCount := parseInt64(arr[i+11])

		if contentID == 0 && userID == 0 && comment == "" {
			miss = append(miss, id)
			continue
		}
		cacheHit[id] = &interaction.CommentItem{
			CommentId:     id,
			ContentId:     contentID,
			UserId:        userID,
			ReplyToUserId: replyToUserID,
			ParentId:      parentID,
			RootId:        rootID,
			Comment:       comment,
			CreatedAt:     createdAt,
			Status:        status,
			UserName:      userName,
			UserAvatar:    userAvatar,
			ReplyCount:    replyCount,
		}
	}
	return cacheHit, miss, nil
}

func (l *RefillCommentCacheLogic) queryFromDB(ids []int64) (map[int64]*interaction.CommentItem, []int64, error) {
	rows, err := l.commentRepo.ListByIDs(ids)
	if err != nil {
		return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论失败"))
	}
	dbMap := make(map[int64]*interaction.CommentItem, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		isDeleted := r.IsDeleted == 1 || r.Status == commentStatusDeleted
		commentText := r.Comment
		status := r.Status
		userID := r.UserID
		if isDeleted {
			commentText = "该评论已删除"
			status = commentStatusDeleted
			userID = 0
		}
		dbMap[r.ID] = &interaction.CommentItem{
			CommentId:     r.ID,
			ContentId:     r.ContentID,
			UserId:        userID,
			ReplyToUserId: r.ReplyToUserID,
			ParentId:      r.ParentID,
			RootId:        r.RootID,
			Comment:       commentText,
			CreatedAt:     r.CreatedAt.Unix(),
			Status:        status,
		}
	}
	if len(ids) > 0 {
		rootIDs := make([]int64, 0)
		parentIDs := make([]int64, 0)
		for _, it := range dbMap {
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				rootIDs = append(rootIDs, it.CommentId)
			} else {
				parentIDs = append(parentIDs, it.CommentId)
			}
		}

		rootCountMap := make(map[int64]int64)
		parentCountMap := make(map[int64]int64)
		if len(rootIDs) > 0 {
			var err error
			rootCountMap, err = l.commentRepo.BatchCountByRootIDs(rootIDs)
			if err != nil {
				return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
			}
		}
		if len(parentIDs) > 0 {
			var err error
			parentCountMap, err = l.commentRepo.BatchCountByParentIDs(parentIDs)
			if err != nil {
				return nil, nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
			}
		}

		for _, it := range dbMap {
			if it == nil {
				continue
			}
			if it.ParentId == 0 {
				it.ReplyCount = rootCountMap[it.CommentId]
			} else {
				it.ReplyCount = parentCountMap[it.CommentId]
			}
		}
	}

	comments := make([]*interaction.CommentItem, 0, len(dbMap))
	for _, it := range dbMap {
		comments = append(comments, it)
	}
	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, comments)

	itemsToFill := make([]*interaction.CommentItem, 0, len(dbMap))
	for _, it := range dbMap {
		itemsToFill = append(itemsToFill, it)
	}
	l.fillObjCacheBestEffort(itemsToFill)

	miss := make([]int64, 0)
	for _, id := range ids {
		if _, ok := dbMap[id]; !ok {
			miss = append(miss, id)
		}
	}
	return dbMap, miss, nil
}

func (l *RefillCommentCacheLogic) fillObjCacheBestEffort(items []*interaction.CommentItem) {
	for _, c := range items {
		if c == nil || c.CommentId <= 0 {
			continue
		}
		objKey := rediskey.BuildCommentObjKey(strconv.FormatInt(c.CommentId, 10))
		createdAt := c.CreatedAt
		if createdAt <= 0 {
			createdAt = time.Now().Unix()
		}
		_, err := l.svcCtx.Redis.EvalCtx(
			l.ctx,
			luautils.UpdateCommentObjScript,
			[]string{objKey},
			strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10),
			strconv.FormatInt(c.CommentId, 10),
			strconv.FormatInt(c.ContentId, 10),
			strconv.FormatInt(c.UserId, 10),
			strconv.FormatInt(c.ReplyToUserId, 10),
			strconv.FormatInt(c.ParentId, 10),
			strconv.FormatInt(c.RootId, 10),
			c.Comment,
			strconv.FormatInt(createdAt, 10),
			strconv.FormatInt(int64(c.Status), 10),
			c.UserName,
			c.UserAvatar,
			strconv.FormatInt(c.ReplyCount, 10),
		)
		if err != nil {
			l.Errorf("回填评论对象缓存失败: %v, comment_id=%d", err, c.CommentId)
		}
	}
}

func (l *RefillCommentCacheLogic) buildRefillLockKey(ids []int64) string {
	idStrs := make([]string, 0, len(ids))
	for _, id := range ids {
		idStrs = append(idStrs, strconv.FormatInt(id, 10))
	}
	raw := strings.Join(idStrs, ",")
	h := fnv.New64a()
	_, _ = h.Write([]byte(raw))
	return rediskey.GetRedisPrefixKey("lock:refill:comment_cache", strconv.FormatUint(h.Sum64(), 10))
}

func pickOrderedComments(ids []int64, cache map[int64]*interaction.CommentItem, db map[int64]*interaction.CommentItem) []*interaction.CommentItem {
	out := make([]*interaction.CommentItem, 0, len(ids))
	for _, id := range ids {
		if it, ok := cache[id]; ok {
			out = append(out, it)
			continue
		}
		if db != nil {
			if it, ok := db[id]; ok {
				out = append(out, it)
			}
		}
	}
	return out
}
