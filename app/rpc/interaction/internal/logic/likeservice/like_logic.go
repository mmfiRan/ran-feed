package likeservicelogic

import (
	"context"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"
)

const likeEventPublishTimeout = 5 * time.Second

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LikeLogic) Like(in *interaction.LikeReq) (*interaction.LikeRes, error) {

	scene := in.Scene.String()

	var changed bool

	// 通过redis处理点赞(用户维度缓存，避免content维度大key)
	changed, err := l.processLike(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("点赞处理失败"))
	}

	// 状态变化时发送事件。请求 ctx 在 handler 返回后会被 cancel，
	// Kafka 异步发送必须用独立 bg ctx + timeout，避免事件中途丢失导致计数不一致。
	if changed {
		threading.GoSafe(func() {
			ctx, cancel := context.WithTimeout(context.Background(), likeEventPublishTimeout)
			defer cancel()
			l.publishLikeEvent(ctx, in.UserId, in.ContentId, in.ContentUserId, scene)
		})
	}

	return &interaction.LikeRes{}, nil
}

func (l *LikeLogic) processLike(userID, contentID int64) (changed bool, err error) {

	contentIdStr := strconv.FormatInt(contentID, 10)
	userIdStr := strconv.FormatInt(userID, 10)
	userLikeKey := rediskey.BuildLikeUserKey(userIdStr)

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.LikeUserHashScript,
		[]string{userLikeKey},
		contentIdStr,
		strconv.FormatInt(rediskey.RedisLikeUserHashCapacity, 10),
		strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10),
	)
	if err != nil {
		return false, err
	}
	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 2 {
		return false, errorx.NewMsg("解析点赞脚本返回值失败")
	}
	changedVal, _ := arr[0].(int64)
	return changedVal == 1, nil
}

// publishLikeEvent 发布点赞事件
func (l *LikeLogic) publishLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendLikeEvent(ctx, userID, contentID, contentUserID, scene)
}
