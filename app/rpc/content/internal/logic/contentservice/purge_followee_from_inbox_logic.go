package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type PurgeFolloweeFromInboxLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewPurgeFolloweeFromInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PurgeFolloweeFromInboxLogic {
	return &PurgeFolloweeFromInboxLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

// PurgeFolloweeFromInbox 取关后从 follower 收件箱清理 followee 窗口内已扩散内容 对称于 BackfillFollowInbox
func (l *PurgeFolloweeFromInboxLogic) PurgeFolloweeFromInbox(in *content.PurgeFolloweeFromInboxReq) (*content.PurgeFolloweeFromInboxRes, error) {
	if in == nil {
		return &content.PurgeFolloweeFromInboxRes{
			RemovedCount: 0,
		}, nil
	}
	if in.FollowerId <= 0 || in.FolloweeId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// 关注关系变更 失效 viewer 拉模式集 取关后读路径才会停止拉该作者
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, rediskey.BuildFollowPullKey(in.FollowerId)); err != nil {
		l.Errorf("失效拉模式关注集失败 viewerID=%d err=%v", in.FollowerId, err)
	}

	// 拉模式内容从未推入 inbox 无需 ZREM 查询失败仍继续清理 ZREM 对拉模式无害对小号必要
	if isBig, err := l.svcCtx.FeedPublisher.IsBigVAuthor(l.ctx, in.FolloweeId); err != nil {
		l.Errorf("查询大 V 集合失败 followeeID=%d err=%v", in.FolloweeId, err)
	} else if isBig {
		return &content.PurgeFolloweeFromInboxRes{
			RemovedCount: 0,
		}, nil
	}

	days := contentconsts.WindowDays
	contents, err := loadFolloweeWindowContent(l.ctx, l.svcCtx, l.contentRepo, in.FolloweeId, followwindow.CutoffMillis(days), int(contentconsts.TimelineKeepN))
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return &content.PurgeFolloweeFromInboxRes{
			RemovedCount: 0,
		}, nil
	}

	members := make([]any, 0, len(contents))
	for _, c := range contents {
		members = append(members, strconv.FormatInt(c.id, 10))
	}

	inboxKey := rediskey.BuildFollowInboxKey(in.FollowerId)
	removed, err := l.svcCtx.Redis.ZremCtx(l.ctx, inboxKey, members...)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("清理关注收件箱失败"))
	}
	return &content.PurgeFolloweeFromInboxRes{
		RemovedCount: int32(removed),
	}, nil
}
