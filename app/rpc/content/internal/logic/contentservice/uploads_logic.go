package contentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/convert"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/oss"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UploadsLogic struct {
	ctx context.Context
	logx.Logger
	ossStrategy oss.Strategy
}

func NewUploadsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadsLogic {
	return &UploadsLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		ossStrategy: svcCtx.OssStrategy,
	}
}

func (l *UploadsLogic) Uploads(in *content.ContentUploadsCredentialsReq) (*content.ContentUploadsCredentialsRes, error) {
	scene, ok := convert.UploadSceneFromPB(in.Scene)
	if !ok {
		return nil, errorx.NewMsg("上传场景取值非法")
	}
	fileExt, ok := convert.FileExtFromPB(in.FileExt)
	if !ok {
		return nil, errorx.NewMsg("文件扩展名取值非法")
	}

	req := &oss.Request{
		UserID:      in.UserId,
		Scene:       scene.Path(),
		ContentType: fileExt.MIME(),
		MaxBytes:    in.FileSize,
		FileName:    convert.SanitizeFileName(in.FileName, fileExt.Suffix()),
	}

	credential, err := l.ossStrategy.Generate(l.ctx, req)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成上传凭证失败"))
	}

	return &content.ContentUploadsCredentialsRes{
		Url:        credential.URL,
		Method:     credential.Method,
		ObjectKey:  credential.ObjectKey,
		FormFields: credential.Fields,
		ExpiredAt:  timestamppb.New(time.Unix(credential.ExpiredAt, 0)),
	}, nil
}
