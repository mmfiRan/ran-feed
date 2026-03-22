package likeservicelogic

import (
	"context"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
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

	contentUserID := int64(0)
	contentDetail, err := l.svcCtx.ContentRpc.GetContentDetail(l.ctx, &content.GetContentDetailReq{
		ContentId: in.ContentId,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容详情失败"))
	}
	if contentDetail != nil && contentDetail.Detail != nil {
		contentUserID = contentDetail.Detail.AuthorId
	}
	if contentUserID <= 0 {
		return nil, errorx.NewMsg("内容作者不存在")
	}

	changed, err := l.processUnlike(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消点赞失败"))
	}

	if changed {
		threading.GoSafe(func() {
			l.publishCancelLikeEvent(in.UserId, in.ContentId, contentUserID, scene)
		})
	}

	return &interaction.UnlikeRes{}, nil
}

func (l *UnlikeLogic) processUnlike(userID, contentID int64) (changed bool, err error) {
	if userID <= 0 || contentID <= 0 {
		return false, nil
	}

	contentIdStr := strconv.FormatInt(contentID, 10)
	userIdStr := strconv.FormatInt(userID, 10)
	userLikeKey := rediskey.BuildLikeUserKey(userIdStr)

	resultVal, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.CancelLikeUserHashScript,
		[]string{userLikeKey},
		contentIdStr,
		strconv.FormatInt(rediskey.RedisLikeExpireSeconds, 10),
	)
	if err != nil {
		return false, err
	}
	arr, ok := resultVal.([]interface{})
	if !ok || len(arr) < 2 {
		return false, errorx.NewMsg("解析取消点赞脚本返回值失败")
	}
	changedVal, _ := arr[0].(int64)
	if changedVal == 0 {
		return false, nil
	}
	return true, nil
}

func (l *UnlikeLogic) publishCancelLikeEvent(userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendCancelLikeEvent(l.ctx, userID, contentID, contentUserID, scene)
}
