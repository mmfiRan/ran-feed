package likeservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryIsLikedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewBatchQueryIsLikedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryIsLikedLogic {
	return &BatchQueryIsLikedLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchQueryIsLikedLogic) BatchQueryIsLiked(in *interaction.BatchQueryIsLikedReq) (*interaction.BatchQueryIsLikedRes, error) {
	if in == nil || len(in.LikeInfos) == 0 {
		return &interaction.BatchQueryIsLikedRes{
			IsLikedInfos: []*interaction.IsLikedInfo{},
		}, nil
	}

	out := &interaction.BatchQueryIsLikedRes{
		IsLikedInfos: make([]*interaction.IsLikedInfo, 0, len(in.LikeInfos)),
	}
	for _, info := range in.LikeInfos {
		if info == nil {
			continue
		}
		out.IsLikedInfos = append(out.IsLikedInfos, &interaction.IsLikedInfo{
			ContentId: info.ContentId,
			Scene:     info.Scene,
			IsLiked:   false,
		})
	}
	if len(out.IsLikedInfos) == 0 || in.UserId == nil {
		return out, nil
	}

	contentIDs := make([]int64, 0, len(out.IsLikedInfos))
	for _, item := range out.IsLikedInfos {
		contentIDs = append(contentIDs, item.ContentId)
	}

	userLikeKey := rediskey.BuildLikeUserKey(strconv.FormatInt(*in.UserId, 10))
	cacheExists, minCID, cacheLikedMap, err := l.queryBatchIsLikedFromCache(userLikeKey, contentIDs)
	if err != nil {
		l.Errorf("批量查询点赞缓存失败，降级默认未点赞: user_id=%d, err=%v", *in.UserId, err)
		return out, nil
	}
	if !cacheExists {
		return out, nil
	}

	dbQueryIDs := make([]int64, 0)
	for _, item := range out.IsLikedInfos {
		cid := item.ContentId
		if cid <= 0 {
			continue
		}
		if minCID > 0 && cid < minCID {
			dbQueryIDs = append(dbQueryIDs, cid)
			continue
		}
		item.IsLiked = cacheLikedMap[cid]
	}

	if len(dbQueryIDs) > 0 {
		dbLikedMap, dbErr := l.likeRepo.BatchIsLiked(*in.UserId, dbQueryIDs)
		if dbErr != nil {
			l.Errorf("批量查询冷数据点赞DB失败，降级默认未点赞: user_id=%d, err=%v", *in.UserId, dbErr)
			return out, nil
		}
		for _, item := range out.IsLikedInfos {
			cid := item.ContentId
			if cid > 0 && minCID > 0 && cid < minCID {
				item.IsLiked = dbLikedMap[cid]
			}
		}
	}

	return out, nil
}

func (l *BatchQueryIsLikedLogic) queryBatchIsLikedFromCache(userLikeKey string, contentIDs []int64) (exists bool, minCID int64, likedMap map[int64]bool, err error) {
	likedMap = make(map[int64]bool, len(contentIDs))
	if len(contentIDs) == 0 {
		return false, 0, likedMap, nil
	}

	uniqIDs := make([]int64, 0, len(contentIDs))
	seen := make(map[int64]struct{}, len(contentIDs))
	for _, cid := range contentIDs {
		if cid <= 0 {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}
		uniqIDs = append(uniqIDs, cid)
	}
	if len(uniqIDs) == 0 {
		return true, 0, likedMap, nil
	}

	args := make([]any, 0, len(uniqIDs)+1)
	args = append(args, strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10))
	for _, cid := range uniqIDs {
		args = append(args, strconv.FormatInt(cid, 10))
	}

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryIsLikedUserHashBatchScript,
		[]string{userLikeKey},
		args...,
	)
	if err != nil {
		return false, 0, likedMap, err
	}

	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 2 {
		return false, 0, likedMap, nil
	}

	existsVal, _ := arr[0].(int64)
	if existsVal == 0 {
		return false, 0, likedMap, nil
	}

	if minCIDStr, ok := arr[1].(string); ok && minCIDStr != "" {
		if parsed, parseErr := strconv.ParseInt(minCIDStr, 10, 64); parseErr == nil {
			minCID = parsed
		}
	}

	for idx, cid := range uniqIDs {
		luaIdx := idx + 2
		if luaIdx >= len(arr) {
			break
		}
		likedVal, _ := arr[luaIdx].(int64)
		likedMap[cid] = likedVal == 1
	}

	return true, minCID, likedMap, nil
}
