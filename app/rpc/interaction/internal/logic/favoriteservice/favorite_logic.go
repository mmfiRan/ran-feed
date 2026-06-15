package favoriteservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type FavoriteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FavoriteLogic {
	return &FavoriteLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FavoriteLogic) Favorite(in *interaction.FavoriteReq) (*interaction.FavoriteRes, error) {

	_, err := l.favoriteRepo.Upsert(&do.FavoriteDO{
		UserID:        in.UserId,
		ContentID:     in.ContentId,
		ContentUserID: in.ContentUserId,
		CreatedBy:     in.UserId,
		UpdatedBy:     in.UserId,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("收藏失败"))
	}

	favKey := rediskey.BuildUserFavoriteFeedKey(strconv.FormatInt(in.UserId, 10))
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, favKey); delErr != nil {
		l.Errorf("删除收藏列表缓存失败: %v, user_id=%d", delErr, in.UserId)
	}

	return &interaction.FavoriteRes{}, nil
}
