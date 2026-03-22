package userservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewBatchGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetUserLogic {
	return &BatchGetUserLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchGetUserLogic) BatchGetUser(in *user.BatchGetUserReq) (*user.BatchGetUserRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if len(in.UserIds) == 0 {
		return &user.BatchGetUserRes{
			Users: []*user.UserInfo{},
		}, nil
	}

	seen := make(map[int64]struct{}, len(in.UserIds))
	ids := make([]int64, 0, len(in.UserIds))
	for _, id := range in.UserIds {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return &user.BatchGetUserRes{Users: []*user.UserInfo{}}, nil
	}

	userMap, err := l.userRepo.BatchGetByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("批量查询用户失败"))
	}

	users := make([]*user.UserInfo, 0, len(ids))
	for _, id := range ids {
		u := userMap[id]
		if u == nil {
			continue
		}
		users = append(users, &user.UserInfo{
			UserId:   u.ID,
			Username: u.Username,
			Mobile:   u.Mobile,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Bio:      u.Bio,
			Gender:   user.Gender(u.Gender),
			Status:   user.UserStatus(u.Status),
		})
	}

	return &user.BatchGetUserRes{
		Users: users,
	}, nil
}
