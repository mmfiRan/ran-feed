package favoriteservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (l *RemoveFavoriteLogic) RemoveFavorite(in *interaction.RemoveFavoriteReq) (*emptypb.Empty, error) {
	_, err := l.favoriteRepo.DeleteByUserAndContent(in.UserId, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消收藏失败"))
	}

	// 失效收藏流缓存 跨域数据走对应 RPC 不直删 content 侧 key 失败只记日志由 TTL 兜底
	if _, cerr := l.svcCtx.ContentRpc.ClearUserFavoriteCache(l.ctx, &content.ClearUserFavoriteCacheReq{UserId: in.UserId}); cerr != nil {
		l.Errorf("失效收藏流缓存失败: %v, user_id=%d", cerr, in.UserId)
	}

	return &emptypb.Empty{}, nil
}
