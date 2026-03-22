// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type QueryUserProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryUserProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserProfileLogic {
	return &QueryUserProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserProfileLogic) QueryUserProfile(req *types.QueryUserProfileReq) (resp *types.QueryUserProfileRes, err error) {

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	userID := req.UserId

	var (
		profileResp *user.GetUserProfileRes
		followResp  *interaction.GetFollowSummaryRes
		countResp   *content.GetUserContentCountRes
		userCounts  *counterservice.GetUserProfileCountsRes
	)

	err = mr.Finish(
		func() error {
			r, e := l.svcCtx.UserRpc.GetUserProfile(l.ctx, &user.GetUserProfileReq{
				UserId: userID,
			})
			if e != nil {
				return e
			}
			profileResp = r
			return nil
		},
		func() error {
			r, e := l.svcCtx.CountRpc.GetUserProfileCounts(l.ctx, &counterservice.GetUserProfileCountsReq{
				UserId: viewerID,
			})
			if e != nil {
				return e
			}
			userCounts = r
			return nil
		},
		func() error {
			r, e := l.svcCtx.ContentRpc.GetUserContentCount(l.ctx, &content.GetUserContentCountReq{
				UserId: userID,
			})
			if e != nil {
				return e
			}
			countResp = r
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if profileResp == nil || profileResp.UserProfile == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	resp = &types.QueryUserProfileRes{
		UserProfileInfo: types.UserProfileInfo{
			UserId:   profileResp.UserProfile.UserId,
			Nickname: profileResp.UserProfile.Nickname,
			Avatar:   profileResp.UserProfile.Avatar,
			Bio:      profileResp.UserProfile.Bio,
			Gender:   int32(profileResp.UserProfile.Gender),
		},
		UserProfileCounts:  types.UserProfileCounts{},
		ViewerProfileState: types.ViewerProfileState{},
	}

	if followResp != nil {
		resp.ViewerProfileState.IsFollowing = followResp.IsFollowing
	}
	if userCounts != nil {
		resp.UserProfileCounts.FolloweeCount = userCounts.FollowingCount
		resp.UserProfileCounts.FollowerCount = userCounts.FollowedCount
		resp.UserProfileCounts.LikeReceivedCount = userCounts.LikeCount
		resp.UserProfileCounts.FavoriteReceivedCount = userCounts.FavoriteCount
	}
	if countResp != nil {
		resp.UserProfileCounts.ContentCount = countResp.ContentCount
	}

	return resp, nil
}
