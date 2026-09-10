package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeleteContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *DeleteContentLogic) DeleteContent(in *content.DeleteContentReq) (*emptypb.Empty, error) {

	row, err := l.contentRepo.GetByIDBrief(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在")
	}
	if row.UserID != in.UserId {
		return nil, errorx.NewMsg("无权限")
	}

	if err := query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepo.WithTx(tx)
		articleRepo := l.articleRepo.WithTx(tx)
		videoRepo := l.videoRepo.WithTx(tx)

		if row.ContentType == int32(content.ContentType_CONTENT_TYPE_ARTICLE) {
			if derr := articleRepo.DeleteByContentID(in.ContentId); derr != nil {
				return derr
			}
		}
		if row.ContentType == int32(content.ContentType_CONTENT_TYPE_VIDEO) {
			if derr := videoRepo.DeleteByContentID(in.ContentId); derr != nil {
				return derr
			}
		}
		if derr := contentRepo.DeleteByID(in.ContentId); derr != nil {
			return derr
		}
		return nil
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除失败"))
	}

	contentIDStr := strconv.FormatInt(in.ContentId, 10)
	pubKey := rediskey.GetRedisPrefixKey(rediskey.RedisFeedUserPublishPrefix, strconv.FormatInt(in.UserId, 10))
	if err := l.svcCtx.Redis.PipelinedCtx(l.ctx, func(pipe redis.Pipeliner) error {
		pipe.ZRem(l.ctx, pubKey, contentIDStr)
		pipe.ZRem(l.ctx, rediskey.RedisFeedHotGlobalKey, contentIDStr)
		return nil
	}); err != nil {
		l.Errorf("删除内容缓存失败: %v", err)
	}

	// 失效内容详情二级缓存 防止删后仍命中旧详情
	if err := contentcache.Invalidate(l.ctx, l.svcCtx.Redis, in.ContentId); err != nil {
		l.Errorf("失效内容详情二级缓存失败 contentID=%d err=%v", in.ContentId, err)
	}

	return &emptypb.Empty{}, nil
}
