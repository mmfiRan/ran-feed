package userservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetUserForIndexLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetUserForIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetUserForIndexLogic {
	return &BatchGetUserForIndexLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatchGetUserForIndex 增量回源 只返回可索引用户的原始投影 不可索引的 id 直接缺席由 search 判删
func (l *BatchGetUserForIndexLogic) BatchGetUserForIndex(in *user.BatchGetUserForIndexReq) (*user.BatchGetUserForIndexRes, error) {
	if in == nil || len(in.UserIds) == 0 {
		return &user.BatchGetUserForIndexRes{Items: []*user.UserIndexItem{}}, nil
	}

	userRepo := repositories.NewUserRepository(l.ctx, l.svcCtx.MysqlDb)
	rows, err := userRepo.BatchGetActiveForIndex(in.UserIds)
	if err != nil {
		return nil, err
	}

	items := make([]*user.UserIndexItem, 0, len(rows))
	for _, id := range in.UserIds {
		if row := rows[id]; row != nil {
			items = append(items, buildUserIndexItem(row))
		}
	}
	return &user.BatchGetUserForIndexRes{Items: items}, nil
}
