package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event/contentevent"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeleteContentLogic struct {
	ctx context.Context
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
	outboxRepo  repositories.ContentOutboxRepository
}

func NewDeleteContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteContentLogic {
	return &DeleteContentLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		contentRepo: svcCtx.ContentRepository,
		articleRepo: svcCtx.ArticleRepository,
		videoRepo:   svcCtx.VideoRepository,
		outboxRepo:  svcCtx.ContentOutboxRepository,
	}
}

func (l *DeleteContentLogic) DeleteContent(in *content.DeleteContentReq) (*emptypb.Empty, error) {
	row, err := l.contentRepo.GetByIDBrief(l.ctx, in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除内容失败"))
	}
	// 查不到 已软删 与非法 id 都返回 nil 不判会解引用崩
	if row == nil {
		return nil, errorx.NewMsg("内容不存在或无权限")
	}
	if row.UserID != in.UserId {
		return nil, errorx.NewMsg("不是发布内容用户无法删除该内容")
	}

	if err = query.Q.Transaction(func(tx *query.Query) error {
		contentRepo := l.contentRepo.WithTx(tx)
		articleRepo := l.articleRepo.WithTx(tx)
		videoRepo := l.videoRepo.WithTx(tx)

		if contentEnum.ContentTypeEnum(row.ContentType) == contentEnum.ContentTypeArticle {
			if derr := articleRepo.DeleteByContentID(l.ctx, in.ContentId); derr != nil {
				return derr
			}
		}
		if contentEnum.ContentTypeEnum(row.ContentType) == contentEnum.ContentTypeVideo {
			if derr := videoRepo.DeleteByContentID(l.ctx, in.ContentId); derr != nil {
				return derr
			}
		}
		if derr := contentRepo.DeleteByID(l.ctx, in.ContentId); derr != nil {
			return derr
		}

		// 写发件箱,清理由消费者消费删除事件完成
		return l.outboxRepo.WithTx(tx).CreateEvent(l.ctx, contentevent.NewContentDeletedEvent(in.ContentId, in.UserId))
	}); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除失败"))
	}

	return &emptypb.Empty{}, nil
}
