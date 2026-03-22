package commentservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"

	"github.com/zeromicro/go-zero/core/logx"
)

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
		if c.Status == commentStatusDeleted {
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

	userMap := make(map[int64]*userservice.UserInfo, len(resp.Users))
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

func fillCommentUsersAndCache(ctx context.Context, svcCtx *svc.ServiceContext, logger logx.Logger, items []*interaction.CommentItem) {
	if svcCtx == nil || svcCtx.UserRpc == nil || svcCtx.Redis == nil || len(items) == 0 {
		return
	}

	needFill := make([]*interaction.CommentItem, 0, len(items))
	for _, c := range items {
		if c == nil || c.UserId <= 0 {
			continue
		}
		if c.UserName == "" || c.UserAvatar == "" {
			needFill = append(needFill, c)
		}
	}
	if len(needFill) == 0 {
		return
	}

	fillCommentUsers(ctx, svcCtx, logger, needFill)

	for _, c := range needFill {
		if c == nil || c.CommentId <= 0 {
			continue
		}
		if c.UserName == "" && c.UserAvatar == "" {
			continue
		}
		objKey := rediskey.BuildCommentObjKey(strconv.FormatInt(c.CommentId, 10))
		createdAt := c.CreatedAt
		if createdAt <= 0 {
			createdAt = time.Now().Unix()
		}
		_, err := svcCtx.Redis.EvalCtx(
			ctx,
			luautils.UpdateCommentObjScript,
			[]string{objKey},
			strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10),
			strconv.FormatInt(c.CommentId, 10),
			strconv.FormatInt(c.ContentId, 10),
			strconv.FormatInt(c.UserId, 10),
			strconv.FormatInt(c.ReplyToUserId, 10),
			strconv.FormatInt(c.ParentId, 10),
			strconv.FormatInt(c.RootId, 10),
			c.Comment,
			strconv.FormatInt(createdAt, 10),
			strconv.FormatInt(int64(c.Status), 10),
			c.UserName,
			c.UserAvatar,
			strconv.FormatInt(c.ReplyCount, 10),
		)
		if err != nil {
			logger.Errorf("回填评论用户信息缓存失败: %v, comment_id=%d", err, c.CommentId)
		}
	}
}
