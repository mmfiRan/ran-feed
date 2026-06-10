package likeservicelogic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/common/consts"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"
)

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

	// 用户维度缓存：避免 content 维度大 key，用 _mincid 区分冷热数据
	// 返回 trusted 表示缓存是否完整可信：可信则按 changed 决定是否发事件（重复点赞可省投递）
	changed, trusted, err := l.processLike(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("点赞处理失败"))
	}

	if !trusted || changed {
		l.asyncPublish(func(ctx context.Context) {
			l.publishLikeEvent(ctx, in.UserId, in.ContentId, in.ContentUserId, scene)
		})
	}

	return &interaction.LikeRes{}, nil
}

func (l *LikeLogic) processLike(userID, contentID int64) (changed, trusted bool, err error) {
	userLikeKey := rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))

	result, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.LikeUserHashScript,
		[]string{userLikeKey},
		strconv.FormatInt(contentID, 10),
		strconv.FormatInt(rediskey.RedisLikeUserHashCapacity, 10),
		strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10),
	)
	if err != nil {
		return false, false, err
	}
	return parseChangedTrusted(result)
}

// publishLikeEvent 发布点赞事件
func (l *LikeLogic) publishLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendLikeEvent(ctx, userID, contentID, contentUserID, scene)
}

// asyncPublish 在独立后台 ctx 中异步投递事件
func (l *LikeLogic) asyncPublish(publish func(ctx context.Context)) {
	bgCtx := context.WithoutCancel(l.ctx)
	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(bgCtx, consts.LikeEventPublishTimeout)
		defer cancel()
		publish(ctx)
	})
}

// parseChangedTrusted 解析点赞脚本返回的 {changed, trusted}
func parseChangedTrusted(result interface{}) (changed, trusted bool, err error) {
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return false, false, errorx.NewMsg("解析点赞脚本返回值失败")
	}
	changedVal, _ := arr[0].(int64)
	trustedVal, _ := arr[1].(int64)
	return changedVal == 1, trustedVal == 1, nil
}
