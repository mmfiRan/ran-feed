package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserContentCountLogic struct {
	ctx context.Context
	logx.Logger
	contentRepo repositories.ContentRepository
}

func NewGetUserContentCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserContentCountLogic {
	return &GetUserContentCountLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		contentRepo: svcCtx.ContentRepository,
	}
}

func (l *GetUserContentCountLogic) GetUserContentCount(in *content.GetUserContentCountReq) (*content.GetUserContentCountRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	cnt, err := l.contentRepo.CountByAuthor(l.ctx,
		contentEnum.ContentStatusPublished, contentEnum.VisibilityPublic, in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询作品数失败"))
	}

	return &content.GetUserContentCountRes{
		ContentCount: cnt,
	}, nil
}
