package contentservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	contentutils "ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/oss"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	scene := contentEnum.UploadSceneEnum(in.Scene)
	fileExt := contentEnum.FileExtEnum(in.FileExt)

	req := &oss.Request{
		UserID:      in.UserId,
		Scene:       scene.Path(),
		ContentType: fileExt.MIME(),
		MaxBytes:    in.FileSize,
		FileName:    contentutils.SanitizeFileName(in.FileName, fileExt.Suffix()),
	}

	credential, err := l.svcCtx.OssStrategy.Generate(l.ctx, req)
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
