package admincontentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetContentStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewAdminSetContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetContentStatusLogic {
	return &AdminSetContentStatusLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

// AdminSetContentStatus 下架/恢复
// 状态翻转后失效二级缓存
func (l *AdminSetContentStatusLogic) AdminSetContentStatus(in *content.AdminSetContentStatusReq) (*content.AdminSetContentStatusRes, error) {
	target := in.GetStatus()
	from, err := l.flipSourceStatus(target)
	if err != nil {
		return nil, err
	}

	affected, err := l.contentRepo.AdminUpdateStatus(in.ContentId, int32(from), int32(target), in.GetOperatorId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新内容状态失败"))
	}
	if affected == 0 {
		return nil, errorx.NewMsg("内容不存在或状态已变更，请刷新后重试")
	}

	// 失效二级缓存
	if err = contentcache.Invalidate(l.ctx, l.svcCtx.Redis, in.ContentId); err != nil {
		l.Errorf("失效内容详情二级缓存失败 contentID=%d err=%v", in.ContentId, err)
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
