// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"

	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentUploadsCredentialsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContentUploadsCredentialsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentUploadsCredentialsLogic {
	return &ContentUploadsCredentialsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContentUploadsCredentialsLogic) ContentUploadsCredentials(req *types.ContentUploadsCredentialsReq) (resp *types.ContentUploadsCredentialsRes, err error) {
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户ID失败"))
	}

	rpcResp, err := l.svcCtx.ContentRpc.Uploads(l.ctx, &content.ContentUploadsCredentialsReq{
		UserId:   userID,
		Scene:    content.UploadScene(req.Scene),
		FileExt:  content.FileExt(req.FileExt),
		FileSize: req.FileSize,
		FileName: req.FileName,
	})
	if err != nil {
		return nil, err
	}

	return &types.ContentUploadsCredentialsRes{
		Url:        rpcResp.Url,
		Method:     rpcResp.Method,
		ObjectKey:  rpcResp.ObjectKey,
		FormFields: rpcResp.FormFields,
		ExpiredAt:  rpcResp.ExpiredAt.AsTime().Unix(),
	}, nil
}
