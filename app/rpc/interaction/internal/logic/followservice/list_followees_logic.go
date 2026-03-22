package followservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFolloweesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewListFolloweesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFolloweesLogic {
	return &ListFolloweesLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *ListFolloweesLogic) ListFollowees(in *interaction.ListFolloweesReq) (*interaction.ListFolloweesRes, error) {
	if in == nil {
		return &interaction.ListFolloweesRes{FollowUserIds: []int64{}, NextCursor: 0, HasMore: false}, nil
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	cursor := in.Cursor
	ids, err := l.followRepo.ListFolloweesByCursor(in.UserId, cursor, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
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

	return &interaction.ListFolloweesRes{
		FollowUserIds: ids,
		NextCursor:    nextCursor,
		HasMore:       hasMore,
	}, nil
}
