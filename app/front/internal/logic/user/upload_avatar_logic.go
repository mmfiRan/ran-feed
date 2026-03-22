package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"ran-feed/app/front/internal/common/consts"
	commonoss "ran-feed/app/front/internal/common/oss"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	uploadAvatarField = "file"
	maxAvatarSize     = 20 * 1024 * 1024
)

var allowedAvatarMimes = map[string]string{
	consts.ImageMimeJpeg: consts.ImageExtJpg,
	consts.ImageMimePng:  consts.ImageExtPng,
	consts.ImageMimeWebp: consts.ImageExtWebp,
	consts.ImageMimeGif:  consts.ImageExtGif,
}

type UploadAvatarLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadAvatarLogic {
	return &UploadAvatarLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadAvatarLogic) UploadAvatar(w http.ResponseWriter, r *http.Request) (*types.UploadAvatarRes, error) {

	// 限制上传体积，避免大文件占用内存
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+1024)

	// 解析 multipart 表单
	if err := r.ParseMultipartForm(maxAvatarSize + 1024); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("解析上传表单失败"))
	}

	file, _, err := r.FormFile(uploadAvatarField)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("未找到上传文件"))
	}
	defer file.Close()

	// 读取文件并校验大小
	content, err := io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("读取上传文件失败"))
	}
	if len(content) == 0 {
		return nil, errorx.NewMsg("上传文件为空")
	}
	if int64(len(content)) > maxAvatarSize {
		return nil, errorx.NewMsg("文件大小超过限制")
	}

	// 检测 MIME 并校验白名单
	mime := http.DetectContentType(content)
	ext, ok := allowedAvatarMimes[mime]
	if !ok {
		return nil, errorx.NewMsg("不支持的图片格式")
	}

	// 生成对象 Key 并上传到 OSS
	objectKey := l.buildObjectKey(ext)
	urlStr, err := l.svcCtx.OssContext.UploadObject(l.ctx, &commonoss.UploadRequest{
		ObjectKey:   objectKey,
		Content:     content,
		ContentType: mime,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("图片上传失败"))
	}

	return &types.UploadAvatarRes{
		Url:       urlStr,
		ObjectKey: l.publicObjectKey(objectKey),
		Mime:      mime,
		Size:      int64(len(content)),
	}, nil
}

func (l *UploadAvatarLogic) buildObjectKey(ext string) string {
	dateStr := time.Now().Format("20060102")
	return fmt.Sprintf("avatar/%s/%d_%s%s", dateStr, time.Now().UnixMilli(), l.randToken(), ext)
}

func (l *UploadAvatarLogic) publicObjectKey(objectKey string) string {
	uploadDir := strings.TrimSpace(l.svcCtx.Config.Oss.UploadDir)
	if uploadDir == "" {
		return objectKey
	}
	return path.Join(uploadDir, objectKey)
}

func (l *UploadAvatarLogic) randToken() string {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "=")
}
