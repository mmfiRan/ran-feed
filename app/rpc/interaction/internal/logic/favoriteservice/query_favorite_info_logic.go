package favoriteservicelogic

import (
	"context"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewQueryFavoriteInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteInfoLogic {
	return &QueryFavoriteInfoLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryFavoriteInfoLogic) QueryFavoriteInfo(in *interaction.QueryFavoriteInfoReq) (*interaction.QueryFavoriteInfoRes, error) {
	favoriteCount, err := l.queryFavoriteCountFromCountRPC(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏计数失败"))
	}

	// 未登录用户默认未收藏 只返回计数
	if in.UserId <= 0 {
		return l.buildResp(in, favoriteCount, false), nil
	}

	isFavorited, err := l.favoriteRepo.IsFavorited(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询是否收藏失败"))
	}

	return l.buildResp(in, favoriteCount, isFavorited), nil
}

func (l *QueryFavoriteInfoLogic) buildResp(in *interaction.QueryFavoriteInfoReq, favoriteCount int64, isFavorited bool) *interaction.QueryFavoriteInfoRes {
	return &interaction.QueryFavoriteInfoRes{
		ContentId:     in.ContentId,
		Scene:         in.Scene,
		FavoriteCount: favoriteCount,
		IsFavorited:   isFavorited,
	}
}

func (l *QueryFavoriteInfoLogic) queryFavoriteCountFromCountRPC(contentID int64) (int64, error) {
	resp, err := l.svcCtx.CountRpc.GetCount(l.ctx, &count.GetCountReq{
		BizType:    count.BizType_FAVORITE,
		TargetType: count.TargetType_CONTENT,
		TargetId:   contentID,
	})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, nil
	}
	return resp.Value, nil
}
