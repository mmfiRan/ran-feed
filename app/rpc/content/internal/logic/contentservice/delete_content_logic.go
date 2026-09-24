package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event/contentevent"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeleteContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
	outboxRepo  repositories.ContentOutboxRepository
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
		outboxRepo:  repositories.NewContentOutboxRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *DeleteContentLogic) DeleteContent(in *content.DeleteContentReq) (*emptypb.Empty, error) {
	row, err := l.contentRepo.GetByIDBrief(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除内容失败"))
	}
	if row.UserID != in.UserId {
		return nil, errorx.NewMsg("不是发布内容用户无法删除该内容")
	}

	if err = query.Q.Transaction(func(tx *query.Query) error {
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

		// 写发件箱,清理由消费者消费删除事件完成
		return l.outboxRepo.WithTx(tx).CreateEvent(contentevent.NewContentDeletedEvent(in.ContentId, in.UserId))
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除失败"))
	}

	return &emptypb.Empty{}, nil
}
