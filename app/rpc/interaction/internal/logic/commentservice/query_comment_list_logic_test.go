package commentservicelogic

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/common/consts"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/app/rpc/user/user"
)

// fakeCommentRepo 测试桩 只实现列表读用到的方法 其余 panic 提示误用
type fakeCommentRepo struct {
	rootRows     []*model.RanFeedComment
	replyRows    []*model.RanFeedComment
	rootCounts   map[int64]int64
	parentCounts map[int64]int64

	gotLimit  int // 记录列表查询传入的 limit 验证多取 1 条
	gotCursor int64
}

func (f *fakeCommentRepo) WithTx(_ *query.Query) repositories.CommentRepository { return f }
func (f *fakeCommentRepo) Create(_ *do.CommentDO) (int64, error)                { panic("not used") }
func (f *fakeCommentRepo) GetByID(_ int64) (*do.CommentDO, error)               { panic("not used") }
func (f *fakeCommentRepo) MarkDeleted(_ int64, _ int64) error                   { panic("not used") }
func (f *fakeCommentRepo) DeleteByID(_ int64) error                             { panic("not used") }
func (f *fakeCommentRepo) HasReferences(_ int64) (bool, error)                  { panic("not used") }
func (f *fakeCommentRepo) ListByIDs(_ []int64) ([]*model.RanFeedComment, error) {
	panic("not used")
}

func (f *fakeCommentRepo) ListRootByContentID(_ int64, cursor int64, limit int) ([]*model.RanFeedComment, error) {
	f.gotLimit = limit
	f.gotCursor = cursor
	return f.rootRows, nil
}

func (f *fakeCommentRepo) ListReplyByRootID(_ int64, cursor int64, limit int) ([]*model.RanFeedComment, error) {
	f.gotLimit = limit
	f.gotCursor = cursor
	return f.replyRows, nil
}

func (f *fakeCommentRepo) BatchCountByParentIDs(_ []int64) (map[int64]int64, error) {
	return f.parentCounts, nil
}

func (f *fakeCommentRepo) BatchCountByRootIDs(_ []int64) (map[int64]int64, error) {
	return f.rootCounts, nil
}

// fakeUserRpc 只覆盖 BatchGetUser 其余方法走嵌入接口 未实现 调用即 panic
type fakeUserRpc struct {
	userservice.UserService
	users map[int64]*user.UserInfo
}

func (f *fakeUserRpc) BatchGetUser(_ context.Context, in *userservice.BatchGetUserReq, _ ...grpc.CallOption) (*userservice.BatchGetUserRes, error) {
	out := &userservice.BatchGetUserRes{}
	for _, id := range in.UserIds {
		if u, ok := f.users[id]; ok {
			out.Users = append(out.Users, u)
		}
	}
	return out, nil
}

func newCommentListLogic(repo repositories.CommentRepository, user userservice.UserService) *QueryCommentListLogic {
	ctx := context.Background()
	return &QueryCommentListLogic{
		ctx:         ctx,
		svcCtx:      &svc.ServiceContext{UserRpc: user},
		Logger:      logx.WithContext(ctx),
		commentRepo: repo,
	}
}

// cmtRow 构造一行评论 默认正常态
func cmtRow(id, contentID, userID, parentID, rootID int64, isDeleted, status int32) *model.RanFeedComment {
	return &model.RanFeedComment{
		ID:            id,
		ContentID:     contentID,
		UserID:        userID,
		ReplyToUserID: 0,
		ParentID:      parentID,
		RootID:        rootID,
		Comment:       "正文",
		Status:        status,
		IsDeleted:     isDeleted,
		CreatedAt:     time.Unix(1700000000, 0),
	}
}

func TestQueryCommentList_InvalidReq(t *testing.T) {
	logic := newCommentListLogic(&fakeCommentRepo{}, &fakeUserRpc{})

	_, err := logic.QueryCommentList(nil)
	assert.Error(t, err)

	_, err = logic.QueryCommentList(&interaction.QueryCommentListReq{ContentId: 0})
	assert.Error(t, err)
}

func TestQueryCommentList_PaginationHasMore(t *testing.T) {
	// pageSize=2 但 DB 返回 3 条 命中 hasMore 截断
	repo := &fakeCommentRepo{
		rootRows: []*model.RanFeedComment{
			cmtRow(30, 1, 100, 0, 0, 0, 10),
			cmtRow(20, 1, 101, 0, 0, 0, 10),
			cmtRow(10, 1, 102, 0, 0, 0, 10),
		},
	}
	logic := newCommentListLogic(repo, &fakeUserRpc{})

	out, err := logic.QueryCommentList(&interaction.QueryCommentListReq{ContentId: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.gotLimit, "应多取 1 条判 hasMore")
	require.Len(t, out.Comments, 2)
	assert.True(t, out.HasMore)
	assert.Equal(t, int64(20), out.NextCursor, "游标取当前页末条 id")
}

func TestQueryCommentList_LastPage(t *testing.T) {
	repo := &fakeCommentRepo{
		rootRows: []*model.RanFeedComment{
			cmtRow(30, 1, 100, 0, 0, 0, 10),
			cmtRow(20, 1, 101, 0, 0, 0, 10),
		},
	}
	logic := newCommentListLogic(repo, &fakeUserRpc{})

	out, err := logic.QueryCommentList(&interaction.QueryCommentListReq{ContentId: 1, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, out.Comments, 2)
	assert.False(t, out.HasMore, "恰好一页不应误报 hasMore")
	assert.Equal(t, int64(0), out.NextCursor)
}

func TestQueryCommentList_Empty(t *testing.T) {
	logic := newCommentListLogic(&fakeCommentRepo{rootRows: nil}, &fakeUserRpc{})

	out, err := logic.QueryCommentList(&interaction.QueryCommentListReq{ContentId: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Empty(t, out.Comments)
	assert.False(t, out.HasMore)
	assert.Equal(t, int64(0), out.NextCursor)
}

func TestQueryCommentList_Tombstone(t *testing.T) {
	// is_deleted=1 与 status=20 两种墓碑都应渲染占位并抹作者
	repo := &fakeCommentRepo{
		rootRows: []*model.RanFeedComment{
			cmtRow(30, 1, 100, 0, 0, 1, 10),
			cmtRow(20, 1, 101, 0, 0, 0, consts.CommentStatusDeleted),
		},
	}
	logic := newCommentListLogic(repo, &fakeUserRpc{})

	out, err := logic.QueryCommentList(&interaction.QueryCommentListReq{ContentId: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, out.Comments, 2)
	for _, c := range out.Comments {
		assert.Equal(t, commentDeletedText, c.Comment)
		assert.Equal(t, consts.CommentStatusDeleted, c.Status)
		assert.Equal(t, int64(0), c.UserId)
	}
}

func TestQueryCommentList_ReplyCountAndUsers(t *testing.T) {
	repo := &fakeCommentRepo{
		rootRows: []*model.RanFeedComment{
			cmtRow(30, 1, 100, 0, 0, 0, 10),
			cmtRow(20, 1, 0, 0, 0, 1, 10), // 墓碑 不补用户
		},
		rootCounts: map[int64]int64{30: 5},
	}
	user := &fakeUserRpc{users: map[int64]*user.UserInfo{
		100: {UserId: 100, Nickname: "张三", Avatar: "a.png"},
	}}
	logic := newCommentListLogic(repo, user)

	out, err := logic.QueryCommentList(&interaction.QueryCommentListReq{ContentId: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, out.Comments, 2)

	assert.Equal(t, int64(5), out.Comments[0].ReplyCount)
	assert.Equal(t, "张三", out.Comments[0].UserName)
	assert.Equal(t, "a.png", out.Comments[0].UserAvatar)

	assert.Equal(t, int64(0), out.Comments[1].ReplyCount, "无回复数映射为 0")
	assert.Equal(t, "", out.Comments[1].UserName, "墓碑不补用户")
}
