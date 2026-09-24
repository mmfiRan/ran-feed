package feedpub

import (
	"context"
	"sync"
	"testing"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	"ran-feed/app/rpc/content/internal/mq/event"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/pkg/consts"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
)

// mockFollowService 只实现被测路径需要的方法 返回预置粉丝
type mockFollowService struct {
	followers []int64
}

func (m *mockFollowService) ListFollowers(ctx context.Context, in *followservice.ListFollowersReq, opts ...grpc.CallOption) (*followservice.ListFollowersRes, error) {
	return &followservice.ListFollowersRes{FollowerUserIds: m.followers}, nil
}

func (m *mockFollowService) GetFollowSummary(ctx context.Context, in *followservice.GetFollowSummaryReq, opts ...grpc.CallOption) (*followservice.GetFollowSummaryRes, error) {
	return &followservice.GetFollowSummaryRes{}, nil
}

// 以下方法不在被测路径 返回零值占位

func (m *mockFollowService) FollowUser(ctx context.Context, in *followservice.FollowUserReq, opts ...grpc.CallOption) (*followservice.FollowUserRes, error) {
	return &followservice.FollowUserRes{}, nil
}

func (m *mockFollowService) UnfollowUser(ctx context.Context, in *followservice.UnfollowUserReq, opts ...grpc.CallOption) (*followservice.UnfollowUserRes, error) {
	return &followservice.UnfollowUserRes{}, nil
}

func (m *mockFollowService) ListFollowees(ctx context.Context, in *followservice.ListFolloweesReq, opts ...grpc.CallOption) (*followservice.ListFolloweesRes, error) {
	return &followservice.ListFolloweesRes{}, nil
}

// mockFanOutProducer 记录投递的扇出分批
type mockFanOutProducer struct {
	mu      sync.Mutex
	batches []*event.FanOutBatch
}

func (m *mockFanOutProducer) PublishBatch(ctx context.Context, batch *event.FanOutBatch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batches = append(m.batches, batch)
	return nil
}

func (m *mockFanOutProducer) sent() []*event.FanOutBatch {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*event.FanOutBatch(nil), m.batches...)
}

func newTestPublisher(t *testing.T, followRpc followservice.FollowService, producer FanOutPublisher) (*miniredis.Miniredis, *redis.Redis, *Publisher) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	return mr, r, NewPublisher(r, followRpc, producer)
}

// TestPublish_NonPublicSkipsFeedWrites 私密与未指定可见性都不应写任何 feed 结构 也不投扇出
func TestPublish_NonPublicSkipsFeedWrites(t *testing.T) {
	cases := []struct {
		name string
		vis  contentEnum.VisibilityEnum
	}{
		{"私密", contentEnum.VisibilityPrivate},
		{"未指定", contentEnum.VisibilityUnknown},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			producer := &mockFanOutProducer{}
			mr, _, p := newTestPublisher(t, &mockFollowService{}, producer)

			err := p.Publish(context.Background(), 1001, 2002, followwindow.NowMillis(), tt.vis)
			require.NoError(t, err)
			assert.Empty(t, mr.Keys(), "非公开内容不应写任何 feed 结构")
			assert.Empty(t, producer.sent(), "非公开内容不应投递扇出")
		})
	}
}

// TestPublish_PublicWritesBoxAndHotSeed 公开内容写作者发件箱与热榜脏集合 无粉丝则不投扇出
func TestPublish_PublicWritesBoxAndHotSeed(t *testing.T) {
	producer := &mockFanOutProducer{}
	mr, r, p := newTestPublisher(t, &mockFollowService{}, producer)
	ctx := context.Background()

	const contentID, authorID = 1001, 2002
	require.NoError(t, p.Publish(ctx, contentID, authorID, followwindow.NowMillis(), contentEnum.VisibilityPublic))

	members, err := r.ZrangeCtx(ctx, rediskey.BuildUserPublishFeedKey(authorID), 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"1001"}, members, "作者发件箱应含该公开内容")

	shard := int(contentID % consts.HotDirtyShards)
	inDirty, err := r.SismemberCtx(ctx, rediskey.BuildHotFeedDirtyKey(shard), "1001")
	require.NoError(t, err)
	assert.True(t, inDirty, "公开内容应登记进热榜脏集合")

	for _, k := range mr.Keys() {
		assert.NotContains(t, k, rediskey.RedisFeedFollowInboxPrefix, "无粉丝不应写任何 inbox")
	}
	assert.Empty(t, producer.sent(), "无粉丝不应投递扇出")
}

// TestPublish_PublicPublishesFanOutBatch 公开且有粉丝时按批投递扇出消息 不再同步写收件箱
func TestPublish_PublicPublishesFanOutBatch(t *testing.T) {
	producer := &mockFanOutProducer{}
	follow := &mockFollowService{followers: []int64{11, 22}}
	mr, _, p := newTestPublisher(t, follow, producer)

	require.NoError(t, p.Publish(context.Background(), 1001, 2002, followwindow.NowMillis(), contentEnum.VisibilityPublic))

	batches := producer.sent()
	require.Len(t, batches, 1)
	assert.Equal(t, int64(1001), batches[0].ContentID)
	assert.Equal(t, int64(2002), batches[0].AuthorID)
	assert.Equal(t, []int64{11, 22}, batches[0].FollowerIDs)

	for _, k := range mr.Keys() {
		assert.NotContains(t, k, rediskey.RedisFeedFollowInboxPrefix, "收件箱写入下沉到扇出消费者 不在此同步写")
	}
}
