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

func (m *mockContentRepo) GetHotScoreByID(contentID int64) (float64, error) {
	return m.score, m.scoreErr
}

func (m *mockContentRepo) ListRecommendByHotScoreCursor(status, visibility int32, cursorScore float64, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
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

// TestQueryFromDB_OverFetchAndCursor 兜底查询多取一条判 hasMore 游标取本页末条
func TestQueryFromDB_OverFetchAndCursor(t *testing.T) {
	repo := &mockContentRepo{rows: []*model.RanFeedContent{{ID: 900}, {ID: 300}, {ID: 100}}}
	l := newTestRecommendLogic(repo)

	res, err := l.queryFromDB("", 2)
	require.NoError(t, err)

	assert.Equal(t, 3, repo.gotLimit, "应按 pageSize+1 多取一条判 hasMore")
	assert.Equal(t, []int64{900, 300}, res.ids)
	assert.True(t, res.hasMore)
	assert.Equal(t, "300", res.nextCursor, "游标取本页末条 字符串契约")
}

// TestQueryFromDB_LastPage 正好取满不溢出时 hasMore 为假
func TestQueryFromDB_LastPage(t *testing.T) {
	repo := &mockContentRepo{rows: []*model.RanFeedContent{{ID: 900}, {ID: 300}}}
	l := newTestRecommendLogic(repo)

	res, err := l.queryFromDB("", 2)
	require.NoError(t, err)

	assert.False(t, res.hasMore)
	assert.Equal(t, "", res.nextCursor)
}

// TestQueryFromDB_CursorResolvedByScore 带游标时用该内容分值定位翻页位置
func TestQueryFromDB_CursorResolvedByScore(t *testing.T) {
	repo := &mockContentRepo{score: 5.5, rows: []*model.RanFeedContent{{ID: 100}}}
	l := newTestRecommendLogic(repo)

	_, err := l.queryFromDB("300", 2)
	require.NoError(t, err)

	assert.Equal(t, int64(300), repo.gotCursorID)
	assert.Equal(t, 5.5, repo.gotCursorScore)
}

// TestQueryFromDB_CursorScoreMissFallsBackToFirstPage 游标内容已删取不到分值 退化为首页不返空
func TestQueryFromDB_CursorScoreMissFallsBackToFirstPage(t *testing.T) {
	repo := &mockContentRepo{scoreErr: assert.AnError, rows: []*model.RanFeedContent{{ID: 900}}}
	l := newTestRecommendLogic(repo)

	_, err := l.queryFromDB("300", 2)
	require.NoError(t, err)

	assert.Equal(t, int64(0), repo.gotCursorID, "取不到分值应退化为首页")
	assert.Equal(t, float64(0), repo.gotCursorScore)
}
