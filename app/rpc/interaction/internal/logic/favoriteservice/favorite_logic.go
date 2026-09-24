package favoriteservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (l *FavoriteLogic) Favorite(in *interaction.FavoriteReq) (*emptypb.Empty, error) {

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

	// 失效收藏流缓存 跨域数据走对应 RPC 不直删 content 侧 key 失败只记日志由 TTL 兜底
	if _, cerr := l.svcCtx.ContentRpc.ClearUserFavoriteCache(l.ctx, &content.ClearUserFavoriteCacheReq{UserId: in.UserId}); cerr != nil {
		l.Errorf("失效收藏流缓存失败: %v, user_id=%d", cerr, in.UserId)
	}

	return &emptypb.Empty{}, nil
}
