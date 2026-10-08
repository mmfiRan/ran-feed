package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/component/favoritebox"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFavoriteFeedLogic struct {
	ctx context.Context
	logx.Logger
	favoriteBox *favoritebox.Box
	resolver    *contentresolver.Resolver
}

func NewUserFavoriteFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteFeedLogic {
	return &UserFavoriteFeedLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		favoriteBox: svcCtx.FavoriteBox,
		resolver:    svcCtx.ContentResolver,
	}
}

func (l *UserFavoriteFeedLogic) UserFavoriteFeed(in *content.UserFavoriteFeedReq) (*content.UserFavoriteFeedRes, error) {
	if in == nil {
		return emptyUserFavoriteFeedRes(), nil
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("用户id不能<=0")
	}
	pageSize := utils.ClampPageSize(in.PageSize)

	page, err := l.favoriteBox.Query(l.ctx, in.UserId, in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(page.ContentIDs) == 0 {
		return emptyUserFavoriteFeedRes(), nil
	}

	// 列表按被访问者(owner)取 点赞态按实际访问者(viewer)算 二者不能混用
	entries, err := l.resolver.Resolve(l.ctx, page.ContentIDs, favoriteViewerID(in), true)
	if err != nil {
		return nil, err
	}
	items := buildContentItems(entries)
	if len(items) == 0 {
		// 过滤后为空也返回原始游标 避免整页死内容导致翻页中断
		return &content.UserFavoriteFeedRes{Items: []*content.ContentItem{}, NextCursor: page.NextCursor, HasMore: page.HasMore}, nil
	}

	return &content.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}

func emptyUserFavoriteFeedRes() *content.UserFavoriteFeedRes {
	return &content.UserFavoriteFeedRes{
		Items:      []*content.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}

// favoriteViewerID 取实际访问者 未登录或未传为 0 匿名
// 收藏列表按 owner(user_id) 取 点赞态必须按 viewer(viewer_id) 算
func favoriteViewerID(in *content.UserFavoriteFeedReq) int64 {
	if in == nil || in.ViewerId == nil {
		return 0
	}
	return *in.ViewerId
}
