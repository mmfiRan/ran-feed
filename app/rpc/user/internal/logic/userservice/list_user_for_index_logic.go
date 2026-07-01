package userservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUserForIndexLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListUserForIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUserForIndexLogic {
	return &ListUserForIndexLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListUserForIndex 全量重建 按 id 游标扫可索引用户 组装原始投影 空返回表示扫完
func (l *ListUserForIndexLogic) ListUserForIndex(in *user.ListUserForIndexReq) (*user.ListUserForIndexRes, error) {
	if in == nil {
		return &user.ListUserForIndexRes{Items: []*user.UserIndexItem{}}, nil
	}

	limit := int(in.Limit)
	if limit <= 0 || limit > maxIndexPageSize {
		limit = maxIndexPageSize
	}

	userRepo := repositories.NewUserRepository(l.ctx, l.svcCtx.MysqlDb)
	rows, err := userRepo.ScanActiveForIndex(in.Cursor, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*user.UserIndexItem, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items = append(items, buildUserIndexItem(row))
	}
	return &user.ListUserForIndexRes{Items: items}, nil
}
