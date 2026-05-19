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

	// 解耦 content-rpc：content-rpc 故障或内容已删除时，仍允许用户清理本地点赞态。
	// contentUserID=0 由 Kafka 下游识别为"作者未知"，跳过用户级获赞数变更，
	// content 级点赞数仍可由 contentID 维度独立更新。
	contentUserID := int64(0)
	contentDetail, cerr := l.svcCtx.ContentRpc.GetContentDetail(l.ctx, &content.GetContentDetailReq{
		ContentId: in.ContentId,
	})
	if cerr != nil {
		l.Errorf("取消点赞时查询内容详情失败，降级 contentUserID=0 继续: content_id=%d, err=%v", in.ContentId, cerr)
	} else if contentDetail != nil && contentDetail.Detail != nil {
		contentUserID = contentDetail.Detail.AuthorId
	}

	changed, err := l.processUnlike(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消点赞失败"))
	}

	// 请求 ctx 在 handler 返回后会被 cancel，Kafka 异步发送必须用独立 bg ctx + timeout。
	if changed {
		threading.GoSafe(func() {
			ctx, cancel := context.WithTimeout(context.Background(), likeEventPublishTimeout)
			defer cancel()
			l.publishCancelLikeEvent(ctx, in.UserId, in.ContentId, contentUserID, scene)
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

func (l *UnlikeLogic) publishCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
	l.svcCtx.LikeProducer.SendCancelLikeEvent(ctx, userID, contentID, contentUserID, scene)
}
