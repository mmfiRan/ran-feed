package userservicelogic

import (
	"context"

	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type GetMeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewGetMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeLogic {
	return &GetMeLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetMeLogic) GetMe(in *user.GetMeReq) (*user.GetMeRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	var (
		u           *user.UserInfo
		countsReply *counterservice.GetUserProfileCountsRes
	)
	err := mr.Finish(
		func() error {
			userDO, err := l.userRepo.GetByID(in.UserId)
			if err != nil {
				return err
			}
			if userDO == nil {
				return errorx.NewMsg("用户不存在")
			}

			u = &user.UserInfo{
				UserId:   userDO.ID,
				Mobile:   userDO.Mobile,
				Nickname: userDO.Nickname,
				Avatar:   userDO.Avatar,
				Bio:      userDO.Bio,
				Gender:   user.Gender(userDO.Gender),
				Status:   user.UserStatus(userDO.Status),
			}
			return nil
		},
		func() error {
			resp, err := l.svcCtx.CountRpc.GetUserProfileCounts(l.ctx, &counterservice.GetUserProfileCountsReq{
				UserId: in.UserId,
			})
			if err != nil {
				countsReply = &counterservice.GetUserProfileCountsRes{}
				return nil
			}
			countsReply = resp
			return nil
		},
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询我的信息失败"))
	}
	if countsReply == nil {
		countsReply = &counterservice.GetUserProfileCountsRes{}
	}

	return &user.GetMeRes{
		UserInfo:              u,
		FolloweeCount:         countsReply.GetFollowingCount(),
		FollowerCount:         countsReply.GetFollowedCount(),
		LikeReceivedCount:     countsReply.GetLikeCount(),
		FavoriteReceivedCount: countsReply.GetFavoriteCount(),
	}, nil
}
