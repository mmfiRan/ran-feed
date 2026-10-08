package feedservicelogic

import (
	"context"
	"testing"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

// mockContentRepo 仅覆盖兜底查询路径用到的方法 其余嵌接口占位
type mockContentRepo struct {
	repositories.ContentRepository
	score    float64
	scoreErr error
	rows     []*model.RanFeedContent

	gotCursorScore float64
	gotCursorID    int64
	gotLimit       int
}

func (m *mockContentRepo) GetHotScoreByID(_ context.Context, contentID int64) (float64, error) {
	return m.score, m.scoreErr
}

func (m *mockContentRepo) ListRecommendByHotScoreCursor(_ context.Context, status, visibility int32, cursorScore float64, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	m.gotCursorScore, m.gotCursorID, m.gotLimit = cursorScore, cursorID, limit
	return m.rows, nil
}

func newTestRecommendLogic(repo repositories.ContentRepository) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:         context.Background(),
		Logger:      logx.WithContext(context.Background()),
		contentRepo: repo,
	}
}

// TestPageFromDB_OverFetchAndCursor 兜底查询多取一条判 hasMore 游标取本页末条
func TestPageFromDB_OverFetchAndCursor(t *testing.T) {
	repo := &mockContentRepo{rows: []*model.RanFeedContent{{ID: 900}, {ID: 300}, {ID: 100}}}
	l := newTestRecommendLogic(repo)

	res, err := l.pageFromDB("", 2)
	require.NoError(t, err)

	assert.Equal(t, 3, repo.gotLimit, "应按 pageSize+1 多取一条判 hasMore")
	assert.Equal(t, []int64{900, 300}, res.ContentIDs)
	assert.True(t, res.HasMore)
	assert.Equal(t, "300", res.NextCursor, "游标取本页末条 字符串契约")
}

// TestPageFromDB_LastPage 正好取满不溢出时 hasMore 为假
func TestPageFromDB_LastPage(t *testing.T) {
	repo := &mockContentRepo{rows: []*model.RanFeedContent{{ID: 900}, {ID: 300}}}
	l := newTestRecommendLogic(repo)

	res, err := l.pageFromDB("", 2)
	require.NoError(t, err)

	assert.False(t, res.HasMore)
	assert.Equal(t, "", res.NextCursor)
}

// TestPageFromDB_CursorResolvedByScore 带游标时用该内容分值定位翻页位置
func TestPageFromDB_CursorResolvedByScore(t *testing.T) {
	repo := &mockContentRepo{score: 5.5, rows: []*model.RanFeedContent{{ID: 100}}}
	l := newTestRecommendLogic(repo)

	_, err := l.pageFromDB("300", 2)
	require.NoError(t, err)

	assert.Equal(t, int64(300), repo.gotCursorID)
	assert.Equal(t, 5.5, repo.gotCursorScore)
}

// TestPageFromDB_CursorScoreMissFallsBackToFirstPage 游标内容已删取不到分值 退化为首页不返空
func TestPageFromDB_CursorScoreMissFallsBackToFirstPage(t *testing.T) {
	repo := &mockContentRepo{scoreErr: assert.AnError, rows: []*model.RanFeedContent{{ID: 900}}}
	l := newTestRecommendLogic(repo)

	_, err := l.pageFromDB("300", 2)
	require.NoError(t, err)

	assert.Equal(t, int64(0), repo.gotCursorID, "取不到分值应退化为首页")
	assert.Equal(t, float64(0), repo.gotCursorScore)
}
