package likeservicelogic

import (
	"context"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryLikeInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	batchQueryIsLikedLogic *BatchQueryIsLikedLogic
}

func NewBatchQueryLikeInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryLikeInfoLogic {
	return &BatchQueryLikeInfoLogic{
		ctx:                    ctx,
		svcCtx:                 svcCtx,
		Logger:                 logx.WithContext(ctx),
		batchQueryIsLikedLogic: NewBatchQueryIsLikedLogic(ctx, svcCtx),
	}
}

func (l *BatchQueryLikeInfoLogic) BatchQueryLikeInfo(in *interaction.BatchQueryLikeInfoReq) (*interaction.BatchQueryLikeInfoRes, error) {
	if in == nil || len(in.LikeInfos) == 0 {
		return &interaction.BatchQueryLikeInfoRes{
			LikeInfos: []*interaction.QueryLikeInfoRes{},
		}, nil
	}

	uniqIDs := make([]int64, 0, len(in.LikeInfos))
	seen := make(map[int64]struct{}, len(in.LikeInfos))
	for _, info := range in.LikeInfos {
		if info == nil || info.ContentId <= 0 {
			continue
		}
		if _, ok := seen[info.ContentId]; ok {
			continue
		}
		seen[info.ContentId] = struct{}{}
		uniqIDs = append(uniqIDs, info.ContentId)
	}

	likeCountMap, err := l.queryLikeCountMap(uniqIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
	}
	likedMap := map[int64]bool{}
	if in.UserId > 0 {
		likedMap, err = l.queryIsLikedMap(in.UserId, uniqIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询点赞信息失败"))
		}
	}

	out := &interaction.BatchQueryLikeInfoRes{
		LikeInfos: make([]*interaction.QueryLikeInfoRes, 0, len(in.LikeInfos)),
	}
	for _, info := range in.LikeInfos {
		if info == nil {
			continue
		}
		out.LikeInfos = append(out.LikeInfos, &interaction.QueryLikeInfoRes{
			ContentId: info.ContentId,
			Scene:     info.Scene,
			LikeCount: likeCountMap[info.ContentId],
			IsLiked:   likedMap[info.ContentId],
		})
	}
	return out, nil
}

func (l *BatchQueryLikeInfoLogic) queryLikeCountMap(contentIDs []int64) (map[int64]int64, error) {
	res := make(map[int64]int64, len(contentIDs))
	if len(contentIDs) == 0 {
		return res, nil
	}

	keys := make([]*count.CountKey, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		keys = append(keys, &count.CountKey{
			BizType:    count.BizType_LIKE,
			TargetType: count.TargetType_CONTENT,
			TargetId:   contentID,
		})
	}
	resp, err := l.svcCtx.CountRpc.BatchGetCount(l.ctx, &count.BatchGetCountReq{
		Keys: keys,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range resp.GetItems() {
		if item == nil || item.GetKey() == nil {
			continue
		}
		res[item.GetKey().GetTargetId()] = item.GetValue()
	}
	return res, nil
}

func (l *BatchQueryLikeInfoLogic) queryIsLikedMap(userID int64, contentIDs []int64) (map[int64]bool, error) {
	res := make(map[int64]bool, len(contentIDs))
	if userID <= 0 || len(contentIDs) == 0 {
		return res, nil
	}

	likeInfos := make([]*interaction.LikeInfo, 0, len(contentIDs))
	for _, contentID := range contentIDs {
		likeInfos = append(likeInfos, &interaction.LikeInfo{
			ContentId: contentID,
		})
	}
	req := &interaction.BatchQueryIsLikedReq{
		UserId:    &userID,
		LikeInfos: likeInfos,
	}
	resp, err := l.batchQueryIsLikedLogic.BatchQueryIsLiked(req)
	if err != nil {
		return nil, err
	}
	for _, item := range resp.GetIsLikedInfos() {
		if item == nil {
			continue
		}
		res[item.ContentId] = item.IsLiked
	}
	return res, nil
}
