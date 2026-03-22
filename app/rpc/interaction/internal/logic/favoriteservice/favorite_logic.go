package favoriteservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
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

	inserted, err := l.favoriteRepo.Upsert(&do.FavoriteDO{
		UserID:        in.UserId,
		ContentID:     in.ContentId,
		ContentUserID: in.ContentUserId,
		CreatedBy:     in.UserId,
		UpdatedBy:     in.UserId,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("收藏失败"))
	}

	scene := in.Scene.String()
	contentIDStr := strconv.FormatInt(in.ContentId, 10)
	userIDStr := strconv.FormatInt(in.UserId, 10)

	relKey := rediskey.BuildFavoriteRelKey(scene, userIDStr, contentIDStr)

	// 收藏时删除关系缓存
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, relKey); delErr != nil {
		l.Errorf("删除收藏缓存失败: %v", delErr)
	}

	// 若用户收藏列表缓存存在则追加；缓存不存在则不写，等读回源重建
	if inserted {
		row, qerr := l.favoriteRepo.GetByUserAndContent(in.UserId, in.ContentId)
		if qerr != nil {
			l.Errorf("查询收藏记录失败: %v, user_id=%d, content_id=%d", qerr, in.UserId, in.ContentId)
		} else if row != nil && row.ID > 0 {
			favKey := rediskey.BuildUserFavoriteFeedKey(userIDStr)
			score := strconv.FormatInt(row.ID, 10)
			member := contentIDStr
			if _, uerr := l.svcCtx.Redis.EvalCtx(
				l.ctx,
				luautils.AddUserFavoriteIfExistsScript,
				[]string{favKey},
				score,
				member,
				strconv.FormatInt(5000, 10),
			); uerr != nil {
				l.Errorf("更新收藏列表缓存失败: %v, user_id=%d, content_id=%d", uerr, in.UserId, in.ContentId)
			}
		}
	}

	return &interaction.FavoriteRes{}, nil
}
