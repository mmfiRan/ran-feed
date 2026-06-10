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

type UnlikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnlikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlikeLogic {
	return &UnlikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UnlikeLogic) Unlike(in *interaction.UnlikeReq) (*interaction.UnlikeRes, error) {
	scene := in.Scene.String()

	changed, trusted, err := l.processUnlike(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消点赞失败"))
	}

	if !trusted || changed {
		l.asyncPublish(func(ctx context.Context) {
			l.publishCancelLikeEvent(ctx, in.UserId, in.ContentId, in.ContentUserId, scene)
		})
	}

	return &interaction.UnlikeRes{}, nil
}

func (l *UnlikeLogic) processUnlike(userID, contentID int64) (changed, trusted bool, err error) {
	if userID <= 0 || contentID <= 0 {
		return false, false, nil
	}

	userLikeKey := rediskey.BuildLikeUserKey(strconv.FormatInt(userID, 10))

	result, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.CancelLikeUserHashScript,
		[]string{userLikeKey},
		strconv.FormatInt(contentID, 10),
		strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10),
	)
	if err != nil {
		return false, false, err
	}
	return parseChangedTrusted(result)
}

func (l *UnlikeLogic) publishCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendCancelLikeEvent(ctx, userID, contentID, contentUserID, scene)
}

// asyncPublish 在独立后台 ctx 中异步投递事件 避免被请求 ctx 取消
func (l *UnlikeLogic) asyncPublish(publish func(ctx context.Context)) {
	bgCtx := context.WithoutCancel(l.ctx)
	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(bgCtx, consts.LikeEventPublishTimeout)
		defer cancel()
		publish(ctx)
	})
}
