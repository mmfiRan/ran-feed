package commentservicelogic

import (
	"context"
	"math/rand"
	"strconv"
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

type QueryReplyListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewQueryReplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryReplyListLogic {
	return &QueryReplyListLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryReplyListLogic) QueryReplyList(in *interaction.QueryReplyListReq) (*interaction.QueryReplyListRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.RootId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	in.PageSize = uint32(pageSize)

	replies, nextCursor, hasMore, missIDs, cacheResult := l.queryFromRedis(in)
	if cacheResult == CacheHit {
		if len(missIDs) > 0 {
			filled, fillErr := l.refillMissingComments(missIDs)
			if fillErr != nil {
				return l.rebuildCacheWithLock(in)
			}
			replies = mergeRefilledComments(replies, filled)
		}
		fillCommentTombstones(replies, missIDs, 0)
		fillCommentUsersAndCache(l.ctx, l.svcCtx, l.Logger, replies)
		return &interaction.QueryReplyListRes{
			RootId:     in.RootId,
			Replies:    replies,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		}, nil
	}

	return l.rebuildCacheWithLock(in)
}

func (l *QueryReplyListLogic) queryFromRedis(in *interaction.QueryReplyListReq) (replies []*interaction.CommentItem, nextCursor int64, hasMore bool, missIDs []int64, result CacheResult) {
	idxKey := rediskey.BuildCommentIdxRootKey(strconv.FormatInt(in.RootId, 10))

	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryCommentListScript,
		[]string{idxKey},
		strconv.FormatInt(in.Cursor, 10),
		strconv.FormatInt(int64(in.PageSize), 10),
		strconv.FormatInt(int64(rediskey.RedisCommentIdxExpireSeconds), 10),
		strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10),
	)
	if err != nil {
		return nil, 0, false, nil, CacheError
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, 0, false, nil, CacheError
	}
	exists, _ := arr[0].(int64)
	if exists == 0 {
		return nil, 0, false, nil, CacheMiss
	}

	nextCursor, _ = arr[1].(int64)
	hasMoreVal, _ := arr[2].(int64)
	hasMore = hasMoreVal == 1

	const chunkSize = 12
	if (len(arr)-3)%chunkSize != 0 {
		return nil, 0, false, nil, CacheError
	}

	items := make([]*interaction.CommentItem, 0, (len(arr)-3)/chunkSize)
	miss := make([]int64, 0)
	for i := 3; i+chunkSize-1 < len(arr); i += chunkSize {
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
		}

		if rootID == 0 {
			rootID = in.RootId
		}
		items = append(items, &interaction.CommentItem{
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
		})
	}

	return items, nextCursor, hasMore, miss, CacheHit
}

func (l *QueryReplyListLogic) rebuildCacheWithLock(in *interaction.QueryReplyListReq) (*interaction.QueryReplyListRes, error) {
	lockKey := rediskey.GetRedisPrefixKey("lock:rebuild:reply_list", strconv.FormatInt(in.RootId, 10))
	lock := redislock.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(rebuildLockExpire)

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
			replies, nextCursor, hasMore, missIDs, cacheResult := l.queryFromRedis(in)
			if cacheResult == CacheHit {
				if len(missIDs) > 0 {
					filled, fillErr := l.refillMissingComments(missIDs)
					if fillErr != nil {
						return nil, fillErr
					}
					replies = mergeRefilledComments(replies, filled)
				}
				fillCommentTombstones(replies, missIDs, 0)
				fillCommentUsersAndCache(l.ctx, l.svcCtx, l.Logger, replies)
				return &interaction.QueryReplyListRes{RootId: in.RootId, Replies: replies, NextCursor: nextCursor, HasMore: hasMore}, nil
			}
		}
		resp, derr := l.queryFromDB(in)
		if derr != nil {
			return nil, derr
		}
		return resp, nil
	}

	// 双检
	replies, nextCursor, hasMore, missIDs, cacheResult := l.queryFromRedis(in)
	if cacheResult == CacheHit {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放分布式锁失败: %v", releaseErr)
		}
		if len(missIDs) > 0 {
			filled, fillErr := l.refillMissingComments(missIDs)
			if fillErr != nil {
				return nil, fillErr
			}
			replies = mergeRefilledComments(replies, filled)
		}
		fillCommentTombstones(replies, missIDs, 0)
		fillCommentUsersAndCache(l.ctx, l.svcCtx, l.Logger, replies)
		return &interaction.QueryReplyListRes{RootId: in.RootId, Replies: replies, NextCursor: nextCursor, HasMore: hasMore}, nil
	}

	resp, derr := l.queryFromDB(in)
	if derr != nil {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放分布式锁失败: %v", releaseErr)
		}
		return nil, derr
	}
	defer func() {
		if releaseOk, releaseErr := lock.ReleaseCtx(l.ctx); !releaseOk || releaseErr != nil {
			l.Errorf("释放分布式锁失败: %v", releaseErr)
		}
	}()
	l.rebuildCacheBestEffort(in.RootId, resp.Replies)

	return resp, nil
}

func (l *QueryReplyListLogic) refillMissingComments(missIDs []int64) ([]*interaction.CommentItem, error) {
	if len(missIDs) == 0 {
		return nil, nil
	}
	refillLogic := NewRefillCommentCacheLogic(l.ctx, l.svcCtx)
	resp, err := refillLogic.RefillCommentCache(&interaction.RefillCommentCacheReq{CommentIds: missIDs})
	if err != nil {
		return nil, err
	}
	return resp.Comments, nil
}

func (l *QueryReplyListLogic) queryFromDB(in *interaction.QueryReplyListReq) (*interaction.QueryReplyListRes, error) {
	rows, err := l.commentRepo.ListReplyByRootID(in.RootId, in.Cursor, int(in.PageSize))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询回复列表失败"))
	}

	replies := make([]*interaction.CommentItem, 0, len(rows))
	parentIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		parentIDs = append(parentIDs, row.ID)
		isDeleted := row.IsDeleted == 1 || row.Status == commentStatusDeleted
		commentText := row.Comment
		status := row.Status
		userID := row.UserID
		if isDeleted {
			commentText = "该评论已删除"
			status = commentStatusDeleted
			userID = 0
		}
		replies = append(replies, &interaction.CommentItem{
			CommentId:     row.ID,
			ContentId:     row.ContentID,
			UserId:        userID,
			ReplyToUserId: row.ReplyToUserID,
			ParentId:      row.ParentID,
			RootId:        row.RootID,
			Comment:       commentText,
			CreatedAt:     row.CreatedAt.Unix(),
			Status:        status,
		})
	}
	if len(parentIDs) > 0 {
		replyCountMap, err := l.commentRepo.BatchCountByParentIDs(parentIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询回复数失败"))
		}
		for _, c := range replies {
			if c == nil {
				continue
			}
			c.ReplyCount = replyCountMap[c.CommentId]
		}
	}
	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, replies)

	nextCursor := int64(0)
	hasMore := false
	if len(rows) >= int(in.PageSize) {
		last := rows[len(rows)-1]
		if last != nil {
			nextCursor = last.ID
			hasMore = nextCursor > 0
		}
	}

	return &interaction.QueryReplyListRes{RootId: in.RootId, Replies: replies, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (l *QueryReplyListLogic) rebuildCacheBestEffort(rootID int64, replies []*interaction.CommentItem) {
	if rootID <= 0 || len(replies) == 0 {
		return
	}
	idxKey := rediskey.BuildCommentIdxRootKey(strconv.FormatInt(rootID, 10))
	for _, c := range replies {
		if c == nil || c.CommentId <= 0 {
			continue
		}
		objKey := rediskey.BuildCommentObjKey(strconv.FormatInt(c.CommentId, 10))
		if c.CreatedAt <= 0 {
			c.CreatedAt = time.Now().Unix()
		}
		_, err := l.svcCtx.Redis.EvalCtx(
			l.ctx,
			luautils.UpdateCommentCacheScript,
			[]string{objKey, idxKey},
			strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10),
			strconv.FormatInt(int64(rediskey.RedisCommentIdxExpireSeconds), 10),
			strconv.FormatInt(int64(rediskey.RedisCommentIdxKeepLatestN), 10),
			strconv.FormatInt(c.CommentId, 10),
			strconv.FormatInt(c.ContentId, 10),
			strconv.FormatInt(c.UserId, 10),
			strconv.FormatInt(c.ReplyToUserId, 10),
			strconv.FormatInt(c.ParentId, 10),
			strconv.FormatInt(c.RootId, 10),
			c.Comment,
			strconv.FormatInt(c.CreatedAt, 10),
			strconv.FormatInt(int64(c.Status), 10),
			c.UserName,
			c.UserAvatar,
			strconv.FormatInt(c.ReplyCount, 10),
		)
		if err != nil {
			l.Errorf("重建回复缓存失败: %v, root_id=%d, comment_id=%d", err, rootID, c.CommentId)
		}
	}
}
