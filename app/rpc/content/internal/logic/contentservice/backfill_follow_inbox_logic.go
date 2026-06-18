package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	backfillFollowInboxWindowCap = 200
	backfillFollowInboxKeepN     = 5000
)

type BackfillFollowInboxLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewBackfillFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackfillFollowInboxLogic {
	return &BackfillFollowInboxLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
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

	// 关注关系变更 失效 viewer 大 V 列表缓存 读路径按新关注集合重算
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, rediskey.BuildFollowBigVKey(in.FollowerId)); err != nil {
		l.Errorf("失效大 V 列表缓存失败 viewerID=%d err=%v", in.FollowerId, err)
	}

	// 大 V 跳过回填 其内容由读路径 merge 大 V publish zset 覆盖 与写扩散对称 查询失败保守仍回填
	if isBig, berr := isBigVAuthor(l.ctx, l.svcCtx, in.FolloweeId); berr != nil {
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
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	contents, err := loadFolloweeWindowContent(l.ctx, l.svcCtx, l.contentRepo, in.FolloweeId, followwindow.CutoffMillis(days), limit)
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
	days := l.svcCtx.Config.FollowFanOut.DeadlineWindowDays
	args := make([]any, 0, 3+len(contents)*2)
	args = append(args,
		strconv.FormatInt(backfillFollowInboxKeepN, 10),
		strconv.FormatInt(followwindow.CutoffMillis(days), 10),
		strconv.Itoa(followwindow.TTLSeconds(days)),
	)
	for _, c := range contents {
		// score=published_at member=content_id
		args = append(args, strconv.FormatInt(c.publishedAt, 10), strconv.FormatInt(c.id, 10))
	}
	res, err := l.svcCtx.Redis.EvalCtx(l.ctx, luautils.BackfillFollowInboxZSetScript, []string{inboxKey}, args...)
	if err != nil {
		return 0, errorx.Wrap(l.ctx, err, errorx.NewMsg("回填关注收件箱失败"))
	}
	added, _ := res.(int64)
	return int(added), nil
}
