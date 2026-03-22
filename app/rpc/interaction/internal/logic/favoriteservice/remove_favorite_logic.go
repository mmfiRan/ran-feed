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

	scene := in.Scene.String()
	contentIDStr := strconv.FormatInt(in.ContentId, 10)
	userIDStr := strconv.FormatInt(in.UserId, 10)

	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)

	// 取消收藏时删除关系缓存（计数由 count rpc 提供，不在 interaction 侧维护）
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, relKey); delErr != nil {
		l.Errorf("删除收藏缓存失败: %v", delErr)
	}

	// 取消收藏时移除列表缓存（不存在也不会报错）
	favKey := rediskey.BuildUserFavoriteFeedKey(userIDStr)
	if _, uerr := l.svcCtx.Redis.ZremCtx(l.ctx, favKey, contentIDStr); uerr != nil {
		l.Errorf("更新收藏列表缓存失败: %v, user_id=%d, content_id=%d", uerr, in.UserId, in.ContentId)
	}

	return &interaction.RemoveFavoriteRes{}, nil
}
