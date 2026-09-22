// Package oss 云存储直传凭证
package oss

import "context"

type Request struct {
	UserID      int64  // 上传用户
	Scene       string // 业务场景路径前缀
	ContentType string // 文件类型
	MaxBytes    int64  // 文件大小上限
	FileName    string // 文件名
}

// Credential 直传票据 前端据此自行上传
type Credential struct {
	URL       string            // 上传 URL
	Method    string            // HTTP 方法
	ObjectKey string            // 对象键
	Fields    map[string]string // 表单字段
	Headers   map[string]string // 预签名签名头
	ExpiredAt int64             // 过期时间
}

// Strategy 云存储直传凭证策略 每家云一个实现
type Strategy interface {
	Generate(ctx context.Context, req *Request) (*Credential, error)
}
