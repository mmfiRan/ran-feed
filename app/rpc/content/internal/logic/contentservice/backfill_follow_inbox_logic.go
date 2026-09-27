package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/feedpub"
	"ran-feed/app/rpc/content/internal/common/component/followwindow"
	"ran-feed/app/rpc/content/internal/common/component/redislock"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	backfillFollowInboxWindowCap = 200
)

type BackfillFollowInboxLogic struct {
	ctx context.Context
	logx.Logger
	redis         *redis.Redis
	feedPublisher *feedpub.Publisher
	contentRepo   repositories.ContentRepository
}

func NewBackfillFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackfillFollowInboxLogic {
	return &BackfillFollowInboxLogic{
		ctx:           ctx,
		Logger:        logx.WithContext(ctx),
		redis:         svcCtx.Redis,
		feedPublisher: svcCtx.FeedPublisher,
		contentRepo:   svcCtx.ContentRepository,
	}
}

func (l *BackfillFollowInboxLogic) BackfillFollowInbox(in *content.BackfillFollowInboxReq) (*content.BackfillFollowInboxRes, error) {
	if in == nil {
		return &content.BackfillFollowInboxRes{
			AddedCount: 0,
		}, nil
	}
	if in.FollowerId <= 0 || in.FolloweeId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// 关注关系变更 失效 viewer 拉模式集 下次读按新关注集合重建两半
	if _, err := l.redis.DelCtx(l.ctx, rediskey.BuildFollowPullKey(in.FollowerId)); err != nil {
		l.Errorf("失效拉模式关注集失败 viewerID=%d err=%v", in.FollowerId, err)
	}

	// 拉模式作者不回填 其内容由读路径查其发件箱覆盖 与写扩散对称 查询失败保守仍回填
	if isBig, berr := l.feedPublisher.IsBigVAuthor(l.ctx, in.FolloweeId); berr != nil {
		l.Errorf("查询大 V 集合失败 followeeID=%d err=%v", in.FolloweeId, berr)
	} else if isBig {
		return &content.BackfillFollowInboxRes{
			AddedCount: 0,
		}, nil
	}

	limit := int(in.Limit)
	if limit <= 0 || limit > backfillFollowInboxWindowCap {
		limit = backfillFollowInboxWindowCap
	}
	contents, err := loadFolloweeWindowContent(l.ctx, l.redis, l.contentRepo, in.FolloweeId, followwindow.CutoffMillis(), limit)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return &content.BackfillFollowInboxRes{AddedCount: 0}, nil
	}

	inboxKey := rediskey.BuildFollowInboxKey(in.FollowerId)
	addedCount, err := l.updateInbox(inboxKey, contents)
	if err != nil {
		return nil, err
	}

	return &content.BackfillFollowInboxRes{
		AddedCount: int32(addedCount),
	}, nil
}

func (l *BackfillFollowInboxLogic) updateInbox(inboxKey string, contents []followeeContent) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	args := make([]any, 0, 3+len(contents)*2)
	args = append(args,
		strconv.FormatInt(contentconsts.TimelineKeepN, 10),
		strconv.FormatInt(followwindow.CutoffMillis(), 10),
		strconv.Itoa(followwindow.TTLSeconds()),
	)
	for _, c := range contents {
		// score=published_at member=content_id
		args = append(args, strconv.FormatInt(c.publishedAt, 10), strconv.FormatInt(c.id, 10))
	}
	res, err := l.redis.EvalCtx(l.ctx, redislock.BackfillFollowInboxZSetScript, []string{inboxKey}, args...)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("回填关注收件箱失败"))
	}
	added, _ := res.(int64)
	return int(added), nil
}
