package counterservicelogic

import (
	"context"
	"errors"
	"testing"

	"ran-feed/app/rpc/count/count"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

type fakeBatchGetter struct {
	calls   int
	lastReq *count.BatchGetCountReq
	resp    *count.BatchGetCountRes
	err     error
}

func (f *fakeBatchGetter) BatchGetCount(in *count.BatchGetCountReq) (*count.BatchGetCountRes, error) {
	f.calls++
	f.lastReq = in
	return f.resp, f.err
}

func newTestContentCountsLogic(getter countBatchGetter) *BatchGetContentCountsLogic {
	ctx := context.Background()
	return &BatchGetContentCountsLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		batchGetter: getter,
	}
}

func TestNormalizeContentIDs(t *testing.T) {
	tests := []struct {
		name string
		in   []int64
		want []int64
	}{
		{"空", nil, []int64{}},
		{"去重保序", []int64{3, 1, 3, 2, 1}, []int64{3, 1, 2}},
		{"过滤非正", []int64{0, -1, 5, 0, 7}, []int64{5, 7}},
		{"全非法", []int64{0, -2}, []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeContentIDs(tt.in))
		})
	}
}

func TestBuildContentCountKeys(t *testing.T) {
	keys := buildContentCountKeys([]int64{10, 20})
	require.Len(t, keys, 6)

	wantBiz := []count.BizType{
		count.BizType_BIZ_TYPE_LIKE, count.BizType_BIZ_TYPE_FAVORITE, count.BizType_BIZ_TYPE_COMMENT,
		count.BizType_BIZ_TYPE_LIKE, count.BizType_BIZ_TYPE_FAVORITE, count.BizType_BIZ_TYPE_COMMENT,
	}
	wantTarget := []int64{10, 10, 10, 20, 20, 20}
	for i, k := range keys {
		assert.Equal(t, wantBiz[i], k.BizType, "第 %d 个键 biz 不符", i)
		assert.Equal(t, count.TargetType_TARGET_TYPE_CONTENT, k.TargetType)
		assert.Equal(t, wantTarget[i], k.TargetId)
	}
}

func TestFoldContentCounts(t *testing.T) {
	items := []*count.CountValueItem{
		{Key: &count.CountKey{BizType: count.BizType_BIZ_TYPE_LIKE, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: 100}, Value: 5},
		{Key: &count.CountKey{BizType: count.BizType_BIZ_TYPE_FAVORITE, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: 100}, Value: 3},
		{Key: &count.CountKey{BizType: count.BizType_BIZ_TYPE_COMMENT, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: 100}, Value: 2},
		{Key: &count.CountKey{BizType: count.BizType_BIZ_TYPE_LIKE, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: 200}, Value: 9},
		nil,
		{Key: nil, Value: 1},
	}
	got := foldContentCounts(items)

	require.Contains(t, got, int64(100))
	assert.Equal(t, int64(5), got[100].LikeCount)
	assert.Equal(t, int64(3), got[100].FavoriteCount)
	assert.Equal(t, int64(2), got[100].CommentCount)

	require.Contains(t, got, int64(200))
	assert.Equal(t, int64(9), got[200].LikeCount)
	assert.Equal(t, int64(0), got[200].FavoriteCount)
	assert.Len(t, got, 2)
}

func TestBatchGetContentCounts_EmptyInput(t *testing.T) {
	getter := &fakeBatchGetter{}
	l := newTestContentCountsLogic(getter)

	resp, err := l.BatchGetContentCounts(&count.BatchGetContentCountsReq{})
	require.NoError(t, err)
	assert.Empty(t, resp.GetItems())
	assert.Equal(t, 0, getter.calls, "空输入不应调用下游")
}

func TestBatchGetContentCounts_OrderedAndZeroFilled(t *testing.T) {
	getter := &fakeBatchGetter{
		resp: &count.BatchGetCountRes{
			Items: []*count.CountValueItem{
				{Key: &count.CountKey{BizType: count.BizType_BIZ_TYPE_LIKE, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: 2}, Value: 7},
				{Key: &count.CountKey{BizType: count.BizType_BIZ_TYPE_COMMENT, TargetType: count.TargetType_TARGET_TYPE_CONTENT, TargetId: 2}, Value: 4},
			},
		},
	}
	l := newTestContentCountsLogic(getter)

	resp, err := l.BatchGetContentCounts(&count.BatchGetContentCountsReq{ContentIds: []int64{1, 2, 2, 3}})
	require.NoError(t, err)
	require.Len(t, resp.GetItems(), 3, "重复入参只出一次")

	assert.Equal(t, int64(1), resp.GetItems()[0].GetContentId())
	assert.Equal(t, int64(0), resp.GetItems()[0].GetLikeCount(), "无记录补零")

	assert.Equal(t, int64(2), resp.GetItems()[1].GetContentId())
	assert.Equal(t, int64(7), resp.GetItems()[1].GetLikeCount())
	assert.Equal(t, int64(4), resp.GetItems()[1].GetCommentCount())

	assert.Equal(t, int64(3), resp.GetItems()[2].GetContentId())

	require.NotNil(t, getter.lastReq)
	assert.Len(t, getter.lastReq.GetKeys(), 9, "三个去重后 id 各三个键")
}

func TestBatchGetContentCounts_PropagatesError(t *testing.T) {
	getter := &fakeBatchGetter{err: errors.New("boom")}
	l := newTestContentCountsLogic(getter)

	resp, err := l.BatchGetContentCounts(&count.BatchGetContentCountsReq{ContentIds: []int64{1}})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 1, getter.calls)
}
