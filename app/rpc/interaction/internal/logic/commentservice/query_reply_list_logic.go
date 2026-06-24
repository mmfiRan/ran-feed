package commentservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryReplyListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	commentRepo repositories.CommentRepository
}

func NewQueryReplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryReplyListLogic {
	return &QueryReplyListLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		commentRepo: repositories.NewCommentRepository(ctx, svcCtx.MysqlDb),
	}
}

// QueryReplyList 楼层回复列表只走 DB 翻页 旁挂直接回复数与用户信息 墓碑保留占位
func (l *QueryReplyListLogic) QueryReplyList(in *interaction.QueryReplyListReq) (*interaction.QueryReplyListRes, error) {
	if in == nil || in.RootId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := clampCommentPageSize(int(in.PageSize))

	// 多取 1 条判 hasMore
	rows, err := l.commentRepo.ListReplyByRootID(in.RootId, in.Cursor, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询回复列表失败"))
	}
	rows, hasMore := pageCommentRows(rows, pageSize)

	replies := make([]*interaction.CommentItem, 0, len(rows))
	parentIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		replies = append(replies, buildCommentItemFromRow(row))
		parentIDs = append(parentIDs, row.ID)
	}

	if len(parentIDs) > 0 {
		replyCountMap, err := l.commentRepo.BatchCountByParentIDs(parentIDs)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询回复数失败"))
		}
		for _, c := range replies {
			c.ReplyCount = replyCountMap[c.CommentId]
		}
	}
	fillCommentUsers(l.ctx, l.svcCtx, l.Logger, replies)

	return &interaction.QueryReplyListRes{
		RootId:     in.RootId,
		Replies:    replies,
		NextCursor: nextCommentCursor(replies, hasMore),
		HasMore:    hasMore,
	}, nil
}
