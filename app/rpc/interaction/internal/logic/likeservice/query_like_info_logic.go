package likeservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		likeRepo: repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryLikeInfoLogic) QueryLikeInfo(in *interaction.QueryLikeInfoReq) (*interaction.QueryLikeInfoRes, error) {
	scene := in.Scene.String()
	contentID := in.ContentId

	// 计数来自 CountService
	likeCount, err := l.queryFromCountService(contentID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞数失败"))
	}

	isLiked, err := l.queryIsLiked(scene, in.UserId, contentID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询是否点赞失败"))
	}

	return l.buildResp(in, likeCount, isLiked), nil
}

func (l *QueryLikeInfoLogic) buildResp(in *interaction.QueryLikeInfoReq, likeCount int64, isLiked bool) *interaction.QueryLikeInfoRes {
	return &interaction.QueryLikeInfoRes{
		ContentId: in.ContentId,
		Scene:     in.Scene,
		LikeCount: likeCount,
		IsLiked:   isLiked,
	}
}

func (l *QueryLikeInfoLogic) queryIsLiked(scene string, userID, contentID int64) (bool, error) {
	if userID <= 0 || contentID <= 0 {
		return false, nil
	}

	userLikeKey := redis.BuildLikeUserKey(strconv.FormatInt(userID, 10))

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.QueryIsLikedUserHashScript,
		[]string{userLikeKey},
		strconv.FormatInt(contentID, 10),
		strconv.FormatInt(redis.RedisLikeExpireSeconds, 10),
	)
	if err != nil {
		return l.likeRepo.IsLiked(userID, contentID)
	}
	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 3 {
		return l.likeRepo.IsLiked(userID, contentID)
	}

	exists, _ := arr[0].(int64)
	liked, _ := arr[1].(int64)
	minCidStr, _ := arr[2].(string)

	if exists == 0 {
		return l.likeRepo.IsLiked(userID, contentID)
	}

	// 有 mincid 时才做冷热判断；没有 mincid 视为热数据
	if minCidStr != "" {
		if minCid, parseErr := strconv.ParseInt(minCidStr, 10, 64); parseErr == nil {
			if contentID < minCid {
				return l.likeRepo.IsLiked(userID, contentID)
			}
		}
	}

	return liked == 1, nil
}

func (l *QueryLikeInfoLogic) queryFromCountService(contentID int64) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}
	res, err := l.svcCtx.CountRpc.GetCount(l.ctx, &count.GetCountReq{
		BizType:    count.BizType_LIKE,
		TargetType: count.TargetType_CONTENT,
		TargetId:   contentID,
	})
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	return res.Value, nil
}
