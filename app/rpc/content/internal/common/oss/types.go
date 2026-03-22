package oss

import "context"

// UploadPolicy 上传策略请求
type UploadPolicy struct {
	Scene    string // 上传场景
	FileExt  string // 文件扩展名
	FileSize int64  // 文件大小
	FileName string // 文件名
	UserId   int64  // 用户ID
}

// UploadCredential 上传凭证响应
type UploadCredential struct {
	ObjectKey string    // 对象键（文件存储路径）
	FormData  *FormData // 表单数据
	ExpiredAt int64     // 过期时间（Unix时间戳）
}

// FormData OSS 表单数据
type FormData struct {
	Host             string // OSS 上传地址
	Policy           string // Base64 编码的策略
	Signature        string // 签名
	SecurityToken    string // STS 临时凭证 token
	SignatureVersion string // 签名版本（如：OSS4-HMAC-SHA256）
	Credential       string // 凭证字符串
	Date             string // UTC 时间
	Key              string // 对象键
}

// Strategy 云存储策略接口
type Strategy interface {
	// GenerateUploadCredential 生成上传凭证
	GenerateUploadCredential(ctx context.Context, policy *UploadPolicy) (*UploadCredential, error)
}
