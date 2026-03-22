package oss

import "context"

// UploadRequest 上传请求
type UploadRequest struct {
	ObjectKey    string
	Content      []byte
	ContentType  string
	CacheControl string
}

// Strategy 云存储策略接口
type Strategy interface {
	UploadObject(ctx context.Context, req *UploadRequest) (string, error)
}
