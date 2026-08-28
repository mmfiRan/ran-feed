package commentservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/common/consts"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

// fillCommentUsers 按 user_id 去重批量补昵称头像 删除态跳过
func fillCommentUsers(ctx context.Context, svcCtx *svc.ServiceContext, logger logx.Logger, items []*interaction.CommentItem) {
	if svcCtx == nil || svcCtx.UserRpc == nil || len(items) == 0 {
		return
	}

	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, c := range items {
		if c == nil || c.UserId <= 0 {
			continue
		}
		if c.Status == consts.CommentStatusDeleted {
			continue
		}
		if _, ok := seen[c.UserId]; ok {
			continue
		}
		seen[c.UserId] = struct{}{}
		ids = append(ids, c.UserId)
	}
	if len(ids) == 0 {
		return
	}

	resp, err := svcCtx.UserRpc.BatchGetUser(ctx, &userservice.BatchGetUserReq{UserIds: ids})
	if err != nil {
		logger.Errorf("批量查询用户信息失败: %v", err)
		return
	}

	userMap := make(map[int64]*user.UserInfo, len(resp.Users))
	for _, u := range resp.Users {
		if u == nil || u.UserId <= 0 {
			continue
		}
		userMap[u.UserId] = u
	}

	for _, c := range items {
		if c == nil || c.UserId <= 0 {
			continue
		}
		if u, ok := userMap[c.UserId]; ok {
			c.UserName = u.Nickname
			c.UserAvatar = u.Avatar
		}
	}
}
