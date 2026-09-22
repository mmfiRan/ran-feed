package admincontentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/event"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetContentStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	outboxRepo  repositories.ContentOutboxRepository
}

func NewAdminSetContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetContentStatusLogic {
	return &AdminSetContentStatusLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		outboxRepo:  repositories.NewContentOutboxRepository(ctx, svcCtx.MysqlDb),
	}
}

// AdminSetContentStatus 下架/恢复 状态翻转后由 feed 消费者据事件清理或重进 feed
func (l *AdminSetContentStatusLogic) AdminSetContentStatus(in *content.AdminSetContentStatusReq) (*content.AdminSetContentStatusRes, error) {
	target := in.GetStatus()
	from, err := l.flipSourceStatus(target)
	if err != nil {
		return nil, err
	}

	row, err := l.contentRepo.AdminGetByID(in.ContentId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("内容不存在")
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		affected, uerr := l.contentRepo.WithTx(tx).AdminUpdateStatus(in.ContentId, int32(from), int32(target), in.GetOperatorId())
		if uerr != nil {
			return uerr
		}
		if affected == 0 {
			return errorx.NewMsg("内容不存在或状态已变更 请刷新后重试")
		}
		return l.outboxRepo.WithTx(tx).CreateEvent(buildStatusEvent(target, row))
	})
	if err != nil {
		// 业务错误直接透出保留具体提示 非预期错误才包装
		if bizErr, ok := err.(*errorx.BizError); ok {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新内容状态失败"))
	}

	return &content.AdminSetContentStatusRes{
		Status: utils.ContentStatusValue(int32(target)),
	}, nil
}

func (l *AdminSetContentStatusLogic) flipSourceStatus(target content.ContentStatus) (content.ContentStatus, error) {
	switch target {
	case content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN:
		return content.ContentStatus_CONTENT_STATUS_PUBLISHED, nil
	case content.ContentStatus_CONTENT_STATUS_PUBLISHED:
		return content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN, nil
	default:
		return content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, errorx.NewMsg("不支持的目标状态")
	}
}

// buildStatusEvent 下架发下架事件 恢复发恢复事件 重进 feed
func buildStatusEvent(target content.ContentStatus, row *model.RanFeedContent) *event.ContentEvent {
	if target == content.ContentStatus_CONTENT_STATUS_PUBLISHED {
		var publishedAtMillis int64
		if row.PublishedAt != nil {
			publishedAtMillis = row.PublishedAt.UnixMilli()
		}
		return &event.ContentEvent{
			EventType:   contentenums.EventTypeRestored,
			ContentID:   row.ID,
			AuthorID:    row.UserID,
			ContentType: row.ContentType,
			Visibility:  row.Visibility,
			PublishedAt: publishedAtMillis,
		}
	}
	return &event.ContentEvent{
		EventType: contentenums.EventTypeTakenDown,
		ContentID: row.ID,
		AuthorID:  row.UserID,
	}
}
