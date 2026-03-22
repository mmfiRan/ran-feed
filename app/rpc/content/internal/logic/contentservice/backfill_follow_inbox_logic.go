package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	backfillFollowInboxDefaultLimit = 20
	backfillFollowInboxMaxLimit     = 50
	backfillFollowInboxKeepN        = 5000
)

type BackfillFollowInboxLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBackfillFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackfillFollowInboxLogic {
	return &BackfillFollowInboxLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
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

	limit := int(in.Limit)
	if limit <= 0 {
		limit = backfillFollowInboxDefaultLimit
	}
	if limit > backfillFollowInboxMaxLimit {
		limit = backfillFollowInboxMaxLimit
	}

	publishKey := rediskey.BuildUserPublishFeedKey(in.FolloweeId)
	inboxKey := rediskey.BuildFollowInboxKey(in.FollowerId)

	candidates, err := l.loadFolloweeLatest(publishKey, limit)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &content.BackfillFollowInboxRes{
			AddedCount: 0,
		}, nil
	}

	addedCount, err := l.updateInbox(inboxKey, candidates)
	if err != nil {
		return nil, err
	}

	return &content.BackfillFollowInboxRes{AddedCount: int32(addedCount)}, nil
}

type inboxCandidate struct {
	score  string
	member string
}

func (l *BackfillFollowInboxLogic) loadFolloweeLatest(publishKey string, limit int) ([]inboxCandidate, error) {
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, publishKey)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注者发布列表失败"))
	}

	if exists {
		pairs, err := l.svcCtx.Redis.ZrevrangeWithScoresByFloatCtx(l.ctx, publishKey, 0, int64(limit-1))
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注者发布列表失败"))
		}
		return buildCandidatesFromPairs(pairs), nil
	}

	return nil, nil
}

func buildCandidatesFromPairs(pairs []zredis.FloatPair) []inboxCandidate {
	if len(pairs) == 0 {
		return nil
	}
	res := make([]inboxCandidate, 0, len(pairs))
	for _, p := range pairs {
		if p.Key == "" {
			continue
		}
		score := p.Key
		if _, err := strconv.ParseInt(p.Key, 10, 64); err != nil {
			score = strconv.FormatInt(int64(p.Score), 10)
		}
		res = append(res, inboxCandidate{
			score:  score,
			member: p.Key,
		})
	}
	return res
}

func (l *BackfillFollowInboxLogic) updateInbox(inboxKey string, candidates []inboxCandidate) (int, error) {
	if len(candidates) == 0 {
		return 0, nil
	}
	args := make([]any, 0, 1+len(candidates)*2)
	args = append(args, strconv.FormatInt(backfillFollowInboxKeepN, 10))
	for _, c := range candidates {
		if c.member == "" || c.score == "" {
			continue
		}
		args = append(args, c.score, c.member)
	}
	res, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.BackfillFollowInboxZSetScript, []string{inboxKey}, args...)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("回填关注收件箱失败"))
	}
	added, _ := res.(int64)
	return int(added), nil
}
