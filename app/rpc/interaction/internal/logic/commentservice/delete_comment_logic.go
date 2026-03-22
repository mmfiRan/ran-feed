package commentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/interaction/interaction"
	rediskey "ran-feed/app/rpc/interaction/internal/common/consts/redis"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewDeleteCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCommentLogic {
	return &DeleteCommentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *DeleteCommentLogic) DeleteComment(in *interaction.DeleteCommentReq) (*interaction.DeleteCommentRes, error) {
	if in == nil || in.CommentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("用户未登录")
	}

	comment, err := l.commentRepo.GetByID(in.CommentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论失败"))
	}
	if comment == nil {
		return nil, errorx.NewMsg("评论不存在")
	}
	if comment.UserID != in.UserId {
		return nil, errorx.NewMsg("无权限删除评论")
	}
	if comment.IsDeleted == 1 {
		return &interaction.DeleteCommentRes{}, nil
	}

	hasRef, err := l.commentRepo.HasReferences(comment.ID)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关联评论失败"))
	}

	if hasRef {
		// 有后代，墓碑删除
		if err = l.commentRepo.MarkDeleted(comment.ID, in.UserId); err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除评论失败"))
		}
		l.invalidateCommentCache(comment)
		return &interaction.DeleteCommentRes{}, nil
	}

	// 无后代，物理删除
	if err = l.commentRepo.DeleteByID(comment.ID); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除评论失败"))
	}
	l.invalidateCommentCache(comment)
	l.removeFromIndex(comment)

	// 清理父链：若父评论为墓碑且已无子评论，则物理删除
	l.cleanupDeletedAncestors(comment.ParentID)

	return &interaction.DeleteCommentRes{}, nil
}

func (l *DeleteCommentLogic) invalidateCommentCache(c *do.CommentDO) {
	if c == nil {
		return
	}
	delKeys := []string{rediskey.BuildCommentObjKey(strconv.FormatInt(c.ID, 10))}
	if c.ParentID > 0 {
		delKeys = append(delKeys, rediskey.BuildCommentObjKey(strconv.FormatInt(c.ParentID, 10)))
	}
	if c.RootID > 0 && c.RootID != c.ID {
		delKeys = append(delKeys, rediskey.BuildCommentObjKey(strconv.FormatInt(c.RootID, 10)))
	}
	if _, delErr := l.svcCtx.Redis.DelCtx(l.ctx, delKeys...); delErr != nil {
		l.Errorf("删除评论缓存失败: %v, comment_id=%d", delErr, c.ID)
	}
}

func (l *DeleteCommentLogic) removeFromIndex(c *do.CommentDO) {
	if c == nil {
		return
	}
	if c.ParentID == 0 {
		idxKey := rediskey.BuildCommentIdxContentKey(strconv.FormatInt(c.ContentID, 10))
		if _, err := l.svcCtx.Redis.ZremCtx(l.ctx, idxKey, c.ID); err != nil {
			l.Errorf("删除评论索引失败: %v, comment_id=%d", err, c.ID)
		}
		return
	}
	if c.RootID > 0 {
		idxKey := rediskey.BuildCommentIdxRootKey(strconv.FormatInt(c.RootID, 10))
		if _, err := l.svcCtx.Redis.ZremCtx(l.ctx, idxKey, c.ID); err != nil {
			l.Errorf("删除回复索引失败: %v, comment_id=%d", err, c.ID)
		}
	}
}

func (l *DeleteCommentLogic) cleanupDeletedAncestors(parentID int64) {
	current := parentID
	for current > 0 {
		parent, err := l.commentRepo.GetByID(current)
		if err != nil {
			l.Errorf("查询父评论失败: %v, comment_id=%d", err, current)
			return
		}
		if parent == nil || parent.IsDeleted == 0 {
			return
		}

		hasRef, err := l.commentRepo.HasReferences(parent.ID)
		if err != nil {
			l.Errorf("查询父评论关联失败: %v, comment_id=%d", err, parent.ID)
			return
		}
		if hasRef {
			return
		}

		if err := l.commentRepo.DeleteByID(parent.ID); err != nil {
			l.Errorf("物理删除父评论失败: %v, comment_id=%d", err, parent.ID)
			return
		}
		l.invalidateCommentCache(parent)
		l.removeFromIndex(parent)

		current = parent.ParentID
	}
}
