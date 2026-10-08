package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/favoritebox"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClearUserFavoriteCacheLogic struct {
	ctx context.Context
	logx.Logger
	favoriteBox *favoritebox.Box
}

func NewClearUserFavoriteCacheLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearUserFavoriteCacheLogic {
	return &ClearUserFavoriteCacheLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		favoriteBox: svcCtx.FavoriteBox,
	}
}

// ClearUserFavoriteCache 失效某用户收藏流缓存 供 interaction 域收藏变更后跨域调用
// 收藏属 interaction 域 跨域不直删 content 侧 key 由本域提供入口 失效失败只记日志由 TTL 兜底
func (l *ClearUserFavoriteCacheLogic) ClearUserFavoriteCache(in *content.ClearUserFavoriteCacheReq) (*emptypb.Empty, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.NewMsg("用户id不能<=0")
	}
	if err := l.favoriteBox.Invalidate(l.ctx, in.UserId); err != nil {
		l.Errorf("失效收藏流缓存失败 userID=%d err=%v", in.UserId, err)
	}
	return &emptypb.Empty{}, nil
}
