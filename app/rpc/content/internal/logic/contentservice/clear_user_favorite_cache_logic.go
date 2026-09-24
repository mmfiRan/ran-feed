package contentservicelogic

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"
	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearUserFavoriteCacheLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewClearUserFavoriteCacheLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearUserFavoriteCacheLogic {
	return &ClearUserFavoriteCacheLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ClearUserFavoriteCache 失效某用户收藏流缓存 供 interaction 域收藏变更后跨域调用
// 收藏属 interaction 域 跨域不直删 content 侧 key 由本域提供入口 失效失败只记日志由 TTL 兜底
func (l *ClearUserFavoriteCacheLogic) ClearUserFavoriteCache(in *content.ClearUserFavoriteCacheReq) (*emptypb.Empty, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.NewMsg("用户id不能<=0")
	}
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, rediskey.BuildUserFavoriteFeedKey(in.UserId)); err != nil {
		l.Errorf("失效收藏流缓存失败 userID=%d err=%v", in.UserId, err)
	}
	return &emptypb.Empty{}, nil
}
