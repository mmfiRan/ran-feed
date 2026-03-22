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
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type UserFavoriteFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo      repositories.ContentRepository
	publishFeedLogic *UserPublishFeedLogic
}

func NewUserFavoriteFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteFeedLogic {
	return &UserFavoriteFeedLogic{
		ctx:              ctx,
		svcCtx:           svcCtx,
		Logger:           logx.WithContext(ctx),
		contentRepo:      repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		publishFeedLogic: NewUserPublishFeedLogic(ctx, svcCtx),
	}
}

func (l *UserFavoriteFeedLogic) UserFavoriteFeed(in *content.UserFavoriteFeedReq) (*content.UserFavoriteFeedRes, error) {
	if in == nil {
		return emptyUserFavoriteFeedRes(), nil
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	feedKey := buildUserFavoriteFeedKey(in.UserId)
	ids, nextCursor, hasMore, err := l.loadPageIDs(feedKey, in.UserId, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	contents, err := l.loadContents(ids)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	return l.assembleResponse(l.publishFeedLogic, contents, nextCursor, hasMore, in.UserId)

}

const (
	userFavoriteFeedKeepN = 5000

	userFavoriteFeedRebuildLockTTLSeconds = 30
	userFavoriteFeedRebuildRetryTimes     = 3
	userFavoriteFeedRebuildRetryInterval  = 80 * time.Millisecond
)

func emptyUserFavoriteFeedRes() *content.UserFavoriteFeedRes {
	return &content.UserFavoriteFeedRes{
		Items:      []*content.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

func buildUserFavoriteFeedKey(userID int64) string {
	return rediskey.BuildUserFavoriteFeedKey(userID)
}

func buildUserFavoriteFeedRebuildLockKey(userID int64) string {
	return rediskey.GetRedisPrefixKey("feed:user:favorite:lock", strconv.FormatInt(userID, 10))
}

func (l *UserFavoriteFeedLogic) loadPageIDs(feedKey string, userID int64, cursor string, pageSize int) ([]int64, string, bool, error) {
	ids, nextCursor, hasMore, cacheExists, err := l.queryUserFavoriteIDs(feedKey, cursor, pageSize)
	if err != nil {
		return nil, "", false, err
	}
	if cacheExists {
		return ids, nextCursor, hasMore, nil
	}

	lockKey := buildUserFavoriteFeedRebuildLockKey(userID)
	rebuildLock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	rebuildLock.SetExpire(userFavoriteFeedRebuildLockTTLSeconds)
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

		allRows, qerr := l.listAllFavorites(userID)
		if qerr != nil {
			return nil, "", false, errorx.Wrap(l.ctx, qerr, errorx.NewMsg("查询收藏列表失败"))
		}
		if len(allRows) == 0 {
			return nil, "", false, nil
		}
		if uerr := l.updateUserFavoriteCache(feedKey, allRows); uerr != nil {
			l.Errorf("回填用户收藏列表缓存失败:%v", uerr)
		}

		pageRows := l.pageFavoriteRows(allRows, cursor, pageSize)
		if len(pageRows) > pageSize {
			hasMore = true
			nextCursor = strconv.FormatInt(pageRows[pageSize-1].FavoriteId, 10)
			pageRows = pageRows[:pageSize]
		} else {
			hasMore = false
			nextCursor = ""
		}

		res := make([]int64, 0, len(pageRows))
		for _, row := range pageRows {
			if row == nil || row.ContentId <= 0 {
				continue
			}
			res = append(res, row.ContentId)
		}
		return res, nextCursor, hasMore, nil
	}

	for i := 0; i < userFavoriteFeedRebuildRetryTimes; i++ {
		time.Sleep(userFavoriteFeedRebuildRetryInterval)
		ids, nextCursor, hasMore, cacheExists, err = l.queryUserFavoriteIDs(feedKey, cursor, pageSize)
		if err != nil {
			return nil, "", false, err
		}
		if cacheExists {
			return ids, nextCursor, hasMore, nil
		}
	}
	return nil, "", false, errorx.NewMsg("查询失败请稍后重试")
}

func (l *UserFavoriteFeedLogic) queryUserFavoriteIDs(feedKey, cursor string, pageSize int) ([]int64, string, bool, bool, error) {
	res, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryUserFavoriteZSetScript,
		[]string{feedKey},
		cursor,
		strconv.FormatInt(int64(pageSize), 10),
	)
	if err != nil {
		return nil, "", false, false, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return nil, "", false, false, errorx.NewMsg("查询收藏列表失败")
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

func (l *UserFavoriteFeedLogic) updateUserFavoriteCache(feedKey string, rows []*favoriteservice.FavoriteItem) error {
	if len(rows) == 0 {
		return nil
	}
	args := make([]interface{}, 0, 1+len(rows)*2)
	args = append(args, strconv.FormatInt(int64(userFavoriteFeedKeepN), 10))
	for _, r := range rows {
		if r == nil || r.ContentId <= 0 {
			continue
		}
		score := strconv.FormatInt(r.FavoriteId, 10)
		member := strconv.FormatInt(r.ContentId, 10)
		args = append(args, score, member)
	}
	_, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.UpdateUserPublishZSetScript,
		[]string{feedKey},
		args...,
	)
	return err
}

func (l *UserFavoriteFeedLogic) pageFavoriteRows(allRows []*favoriteservice.FavoriteItem, cursor string, pageSize int) []*favoriteservice.FavoriteItem {
	if len(allRows) == 0 {
		return allRows
	}

	cursorScore := int64(0)
	if cursor != "" {
		v, err := strconv.ParseInt(cursor, 10, 64)
		if err == nil && v > 0 {
			cursorScore = v
		}
	}

	res := make([]*favoriteservice.FavoriteItem, 0, pageSize+1)
	for _, r := range allRows {
		if r == nil {
			continue
		}
		score := r.FavoriteId
		if cursorScore > 0 && score >= cursorScore {
			continue
		}
		res = append(res, r)
		if len(res) >= pageSize+1 {
			break
		}
	}
	return res
}

func (l *UserFavoriteFeedLogic) listAllFavorites(userID int64) ([]*favoriteservice.FavoriteItem, error) {
	cursor := int64(0)
	pageSize := uint32(500)
	out := make([]*favoriteservice.FavoriteItem, 0)
	for {
		resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteList(l.ctx, &favoriteservice.QueryFavoriteListReq{
			UserId:   userID,
			Cursor:   cursor,
			PageSize: pageSize,
		})
		logx.Infof("QueryFavoriteList: cursor=%d, pageSize=%d, resp=%v, err=%v", cursor, pageSize, resp, err)
		if err != nil {
			return nil, err
		}
		if resp != nil && len(resp.Items) > 0 {
			out = append(out, resp.Items...)
		}
		if resp == nil || !resp.HasMore || resp.NextCursor <= 0 {
			break
		}
		cursor = resp.NextCursor
		if len(out) >= userFavoriteFeedKeepN {
			break
		}
	}
	if len(out) > userFavoriteFeedKeepN {
		out = out[:userFavoriteFeedKeepN]
	}
	return out, nil
}

func (l *UserFavoriteFeedLogic) loadContents(ids []int64) ([]*model.RanFeedContent, error) {
	contentMap, err := l.contentRepo.BatchGetPublishedByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏内容失败"))
	}

	contents := make([]*model.RanFeedContent, 0, len(ids))
	for _, id := range ids {
		if row, ok := contentMap[id]; ok && row != nil {
			contents = append(contents, row)
		}
	}
	return contents, nil
}

func (l *UserFavoriteFeedLogic) assembleResponse(helper *UserPublishFeedLogic, contents []*model.RanFeedContent, nextCursor string, hasMore bool, userID int64) (*content.UserFavoriteFeedRes, error) {
	userMap, likedMap, likeCountMap, err := helper.buildUserAndLikeMaps(contents, userID)
	if err != nil {
		return nil, err
	}
	articleMap, videoMap, err := helper.buildBriefMaps(contents)
	if err != nil {
		return nil, err
	}
	items := helper.buildItems(contents, articleMap, videoMap, userMap, likedMap, likeCountMap)
	return &content.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
