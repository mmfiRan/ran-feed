package contentservicelogic

import (
	"context"
	"ran-feed/app/rpc/content/internal/common/oss"
	"ran-feed/pkg/errorx"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadsLogic {
	return &UploadsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UploadsLogic) Uploads(in *content.ContentUploadsCredentialsReq) (*content.ContentUploadsCredentialsRes, error) {
	policy := &oss.UploadPolicy{
		Scene:    in.Scene.String(),
		FileExt:  in.FileExt.String(),
		FileSize: in.FileSize,
		FileName: in.FileName,
		UserId:   in.UserId,
	}
	credential, err := l.svcCtx.OssContext.GenerateUploadCredential(l.ctx, policy)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成上传凭证失败"))
	}
	return &content.ContentUploadsCredentialsRes{
		ObjectKey: credential.ObjectKey,
		FormData: &content.OssFormData{
			Host:             credential.FormData.Host,
			Policy:           credential.FormData.Policy,
			Signature:        credential.FormData.Signature,
			SecurityToken:    credential.FormData.SecurityToken,
			SignatureVersion: credential.FormData.SignatureVersion,
			Credential:       credential.FormData.Credential,
			Date:             credential.FormData.Date,
			Key:              credential.FormData.Key,
		},
		ExpiredAt: credential.ExpiredAt,
	}, nil
}
