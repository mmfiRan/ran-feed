package favoriteservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveFavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewRemoveFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFavoriteLogic {
	return &RemoveFavoriteLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *RemoveFavoriteLogic) RemoveFavorite(in *interaction.RemoveFavoriteReq) (*interaction.RemoveFavoriteRes, error) {
	_, err := l.favoriteRepo.DeleteByUserAndContent(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消收藏失败"))
	}

	// 旁路缓存 取消后删除头部缓存 下次读重建
	favKey := rediskey.BuildUserFavoriteFeedKey(strconv.FormatInt(in.UserId, 10))
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, favKey); delErr != nil {
		l.Errorf("删除收藏列表缓存失败: %v, user_id=%d", delErr, in.UserId)
	}

	return &interaction.RemoveFavoriteRes{}, nil
}
