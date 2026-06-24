package commentservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
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

	// 列表读已改走 DB 创建不再维护缓存 obj 由首次 by-id 读惰性回填

	return &interaction.CommentRes{
		CommentId: commentID,
	}, nil
}
