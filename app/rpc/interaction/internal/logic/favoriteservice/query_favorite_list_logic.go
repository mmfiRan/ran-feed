package favoriteservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFavoriteListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	favoriteRepo repositories.FavoriteRepository
}

func NewQueryFavoriteListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFavoriteListLogic {
	return &QueryFavoriteListLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		favoriteRepo: repositories.NewFavoriteRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *QueryFavoriteListLogic) QueryFavoriteList(in *interaction.QueryFavoriteListReq) (*interaction.QueryFavoriteListRes, error) {
	if in == nil || in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	rows, err := l.favoriteRepo.ListByUserCursor(in.UserId, in.Cursor, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询收藏列表失败"))
	}

	hasMore := false
	if len(rows) > pageSize {
		hasMore = true
		rows = rows[:pageSize]
	}

	items := make([]*interaction.FavoriteItem, 0, len(rows))
	for _, r := range rows {
		if r == nil || r.ContentID <= 0 {
			continue
		}
		items = append(items, &interaction.FavoriteItem{
			FavoriteId:    r.ID,
			ContentId:     r.ContentID,
			ContentUserId: r.ContentUserID,
		})
	}

	nextCursor := int64(0)
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].FavoriteId
	}

	return &interaction.QueryFavoriteListRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
