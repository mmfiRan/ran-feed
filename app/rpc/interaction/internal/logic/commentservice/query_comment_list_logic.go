package commentservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/common/consts"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const commentDeletedText = "该评论已删除"

type QueryCommentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewQueryCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryCommentListLogic {
	return &QueryCommentListLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

// QueryCommentList 一级评论列表只走 DB 翻页 旁挂回复数与用户信息 墓碑保留占位
func (l *QueryCommentListLogic) QueryCommentList(in *interaction.QueryCommentListReq) (*interaction.QueryCommentListRes, error) {
	if in == nil || in.ContentId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := clampCommentPageSize(int(in.PageSize))

	// 多取 1 条判 hasMore
	rows, err := l.commentRepo.ListRootByContentID(in.ContentId, in.Cursor, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论列表失败"))
	}
	rows, hasMore := pageCommentRows(rows, pageSize)

	comments := make([]*interaction.CommentItem, 0, len(rows))
	rootIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		comments = append(comments, buildCommentItemFromRow(row))
		rootIDs = append(rootIDs, row.ID)
	}

	if len(rootIDs) > 0 {
		replyCountMap, err := l.commentRepo.BatchCountByRootIDs(rootIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论回复数失败"))
		}
		for _, c := range comments {
			c.ReplyCount = replyCountMap[c.CommentId]
		}
	}
	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, comments)

	return &interaction.QueryCommentListRes{
		Comments:   comments,
		NextCursor: nextCommentCursor(comments, hasMore),
		HasMore:    hasMore,
	}, nil
}

// clampCommentPageSize 收敛分页大小 默认 20 上限 100
func clampCommentPageSize(pageSize int) int {
	if pageSize <= 0 {
		return 20
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

// pageCommentRows 按多取 1 条的约定截断当前页并给出 hasMore
func pageCommentRows(rows []*model.RanFeedComment, pageSize int) ([]*model.RanFeedComment, bool) {
	if len(rows) > pageSize {
		return rows[:pageSize], true
	}
	return rows, false
}

// nextCommentCursor 下一页游标取当前页末条 id 仅在还有更多时给出
func nextCommentCursor(comments []*interaction.CommentItem, hasMore bool) int64 {
	if !hasMore || len(comments) == 0 {
		return 0
	}
	return comments[len(comments)-1].CommentId
}

// buildCommentItemFromRow 行转 DTO 墓碑 is_deleted 或 status 删除态 换占位文案并抹作者
func buildCommentItemFromRow(row *model.RanFeedComment) *interaction.CommentItem {
	commentText := row.Comment
	status := row.Status
	userID := row.UserID
	if row.IsDeleted == 1 || row.Status == consts.CommentStatusDeleted {
		commentText = commentDeletedText
		status = consts.CommentStatusDeleted
		userID = 0
	}
	return &interaction.CommentItem{
		CommentId:     row.ID,
		ContentId:     row.ContentID,
		UserId:        userID,
		ReplyToUserId: row.ReplyToUserID,
		ParentId:      row.ParentID,
		RootId:        row.RootID,
		Comment:       commentText,
		CreatedAt:     row.CreatedAt.Unix(),
		Status:        status,
	}
}
