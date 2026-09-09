package followservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFollowersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewListFollowersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFollowersLogic {
	return &ListFollowersLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *ListFollowersLogic) ListFollowers(in *interaction.ListFollowersReq) (*interaction.ListFollowersRes, error) {
	if in == nil {
		return &interaction.ListFollowersRes{FollowerUserIds: []int64{}, NextCursor: 0, HasMore: false}, nil
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}

	ids, err := l.followRepo.ListFollowersByCursor(in.UserId, in.Cursor, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}

	hasMore := false
	if len(ids) > pageSize {
		hasMore = true
		ids = ids[:pageSize]
	}

	nextCursor := int64(0)
	if hasMore && len(ids) > 0 {
		nextCursor = ids[len(ids)-1]
	}

	return &interaction.ListFollowersRes{
		FollowerUserIds: ids,
		NextCursor:      nextCursor,
		HasMore:         hasMore,
	}, nil
}
