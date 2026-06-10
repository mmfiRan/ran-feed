package likeservicelogic

import (
	"context"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLikeInfoLogic struct {
	ctx                    context.Context
	svcCtx                 *svc.ServiceContext
	batchQueryIsLikedLogic *BatchQueryIsLikedLogic
	logx.Logger
	likeRepo repositories.LikeRepository
}

func NewQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLikeInfoLogic {
	return &QueryLikeInfoLogic{
		ctx:                    ctx,
		svcCtx:                 svcCtx,
		Logger:                 logx.WithContext(ctx),
		likeRepo:               repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
		batchQueryIsLikedLogic: NewBatchQueryIsLikedLogic(ctx, svcCtx),
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

// queryIsLiked 复用批量路径 保证单条与批量行为完全一致
func (l *QueryLikeInfoLogic) queryIsLiked(scene string, userID, contentID int64) (bool, error) {
	if userID <= 0 || contentID <= 0 {
		return false, nil
	}
	uid := userID
	res, err := l.batchQueryIsLikedLogic.BatchQueryIsLiked(&interaction.BatchQueryIsLikedReq{
		UserId: &uid,
		LikeInfos: []*interaction.LikeInfo{
			{ContentId: contentID},
		},
	})
	if err != nil {
		return false, err
	}
	if res == nil || len(res.IsLikedInfos) == 0 {
		return false, nil
	}
	return res.IsLikedInfos[0].IsLiked, nil
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
