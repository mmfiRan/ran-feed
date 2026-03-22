package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserContentCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewGetUserContentCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserContentCountLogic {
	return &GetUserContentCountLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetUserContentCountLogic) GetUserContentCount(in *content.GetUserContentCountReq) (*content.GetUserContentCountRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	status := int32(content.ContentStatus_PUBLISHED)
	visibility := int32(content.Visibility_PUBLIC)

	cnt, err := l.contentRepo.CountByAuthor(status, visibility, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询作品数失败"))
	}

	return &content.GetUserContentCountRes{
		ContentCount: cnt,
	}, nil
}
