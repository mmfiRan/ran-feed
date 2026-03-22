package commentservicelogic

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/interaction/internal/common/utils/lua"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	commentStatusNormal int32 = 10
	commentVersion      int32 = 1
)

type CommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *CommentLogic) Comment(in *interaction.CommentReq) (*interaction.CommentRes, error) {
	parentID := in.ParentId
	rootID := in.RootId
	replyToUserID := in.ReplyToUserId

	if parentID == 0 {
		if rootID != 0 || replyToUserID != 0 {
			return nil, errorx.NewMsg("一级评论不允许设置root_id/reply_to_user_id")
		}
		rootID = 0
	} else {
		parentComment, err := l.commentRepo.GetByID(parentID)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询父评论失败"))
		}
		if parentComment == nil {
			return nil, errorx.NewMsg("父评论不存在")
		}
		if parentComment.Status != commentStatusNormal || parentComment.IsDeleted == 1 {
			return nil, errorx.NewMsg("父评论不可回复")
		}
		if parentComment.ContentID != in.ContentId {
			return nil, errorx.NewMsg("父评论与内容不匹配")
		}

		derivedRootID := parentComment.RootID
		if parentComment.ParentID == 0 {
			derivedRootID = parentComment.ID
		}
		if rootID == 0 {
			rootID = derivedRootID
		} else if rootID != derivedRootID {
			return nil, errorx.NewMsg("root_id参数错误")
		}
		if replyToUserID <= 0 {
			replyToUserID = parentComment.UserID
		}
	}

	commentDO := &do.CommentDO{
		ContentID:     in.ContentId,
		ContentUserID: in.ContentUserId,
		UserID:        in.UserId,
		ReplyToUserID: replyToUserID,
		ParentID:      parentID,
		RootID:        rootID,
		Comment:       in.Comment,
		Status:        commentStatusNormal,
		Version:       commentVersion,
		CreatedBy:     in.UserId,
		UpdatedBy:     in.UserId,
	}

	commentID, err := l.commentRepo.Create(commentDO)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建评论失败"))
	}

	if parentID == 0 {
		idxKey := rediskey.BuildCommentIdxContentKey(strconv.FormatInt(in.ContentId, 10))
		objKey := rediskey.BuildCommentObjKey(strconv.FormatInt(commentID, 10))
		createdAt := time.Now().Unix()
		userName := ""
		userAvatar := ""
		replyCount := int64(0)
		resp, uerr := l.svcCtx.UserRpc.GetUser(l.ctx, &userservice.GetUserReq{
			UserId: in.UserId,
		})
		if uerr != nil {
			l.Errorf("查询用户信息失败: %v, user_id=%d", uerr, in.UserId)
		}
		userName = resp.UserInfo.Nickname
		userAvatar = resp.UserInfo.Avatar
		_, err = l.svcCtx.Redis.EvalCtx(
			l.ctx,
			luautils.UpdateCommentCacheScript,
			[]string{objKey, idxKey},
			strconv.FormatInt(int64(rediskey.RedisCommentObjExpireSeconds), 10),
			strconv.FormatInt(int64(rediskey.RedisCommentIdxExpireSeconds), 10),
			strconv.FormatInt(int64(rediskey.RedisCommentIdxKeepLatestN), 10),
			strconv.FormatInt(commentID, 10),
			strconv.FormatInt(commentDO.ContentID, 10),
			strconv.FormatInt(commentDO.UserID, 10),
			strconv.FormatInt(commentDO.ReplyToUserID, 10),
			strconv.FormatInt(commentDO.ParentID, 10),
			strconv.FormatInt(commentDO.RootID, 10),
			commentDO.Comment,
			strconv.FormatInt(createdAt, 10),
			strconv.FormatInt(int64(commentDO.Status), 10),
			userName,
			userAvatar,
			strconv.FormatInt(replyCount, 10),
		)
		if err != nil {
			l.Errorf("更新评论缓存失败: %v, content_id=%d, comment_id=%d", err, in.ContentId, commentID)
		}
	} else {
		// 旁路缓存策略：回复创建后删除父评论对象缓存 + 根评论索引缓存，触发下次读回源重建
		delKeys := []string{
			rediskey.BuildCommentObjKey(strconv.FormatInt(parentID, 10)),
		}
		if rootID > 0 && rootID != parentID {
			delKeys = append(delKeys, rediskey.BuildCommentObjKey(strconv.FormatInt(rootID, 10)))
		}
		if rootID > 0 {
			delKeys = append(delKeys, rediskey.BuildCommentIdxRootKey(strconv.FormatInt(rootID, 10)))
		}
		if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, delKeys...); delErr != nil {
			l.Errorf("删除回复相关缓存失败: %v, parent_id=%d, root_id=%d", delErr, parentID, rootID)
		}
	}

	return &interaction.CommentRes{
		CommentId: commentID,
	}, nil
}
