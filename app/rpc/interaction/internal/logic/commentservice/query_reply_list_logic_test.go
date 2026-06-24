package commentservicelogic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"
)

func newReplyListLogic(repo repositories.CommentRepository, user userservice.UserService) *QueryReplyListLogic {
	ctx := context.Background()
	return &QueryReplyListLogic{
		ctx:         ctx,
		svcCtx:      &svc.ServiceContext{UserRpc: user},
		Logger:      logx.WithContext(ctx),
		commentRepo: repo,
	}
}

func TestQueryReplyList_InvalidReq(t *testing.T) {
	logic := newReplyListLogic(&fakeCommentRepo{}, &fakeUserRpc{})

	_, err := logic.QueryReplyList(nil)
	assert.Error(t, err)

	_, err = logic.QueryReplyList(&interaction.QueryReplyListReq{RootId: 0})
	assert.Error(t, err)
}

func TestQueryReplyList_PaginationAndCount(t *testing.T) {
	// pageSize=2 DB 返回 3 条 回复数走 parent 维度
	repo := &fakeCommentRepo{
		replyRows: []*model.RanFeedComment{
			cmtRow(33, 1, 100, 9, 9, 0, 10),
			cmtRow(22, 1, 101, 9, 9, 0, 10),
			cmtRow(11, 1, 102, 9, 9, 0, 10),
		},
		parentCounts: map[int64]int64{33: 2},
	}
	logic := newReplyListLogic(repo, &fakeUserRpc{})

	out, err := logic.QueryReplyList(&interaction.QueryReplyListReq{RootId: 9, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.gotLimit, "应多取 1 条判 hasMore")
	require.Len(t, out.Replies, 2)
	assert.True(t, out.HasMore)
	assert.Equal(t, int64(22), out.NextCursor)
	assert.Equal(t, int64(9), out.RootId)
	assert.Equal(t, int64(2), out.Replies[0].ReplyCount)
}

func TestQueryReplyList_Empty(t *testing.T) {
	logic := newReplyListLogic(&fakeCommentRepo{replyRows: nil}, &fakeUserRpc{})

	out, err := logic.QueryReplyList(&interaction.QueryReplyListReq{RootId: 9, PageSize: 20})
	require.NoError(t, err)
	assert.Empty(t, out.Replies)
	assert.False(t, out.HasMore)
	assert.Equal(t, int64(0), out.NextCursor)
}
