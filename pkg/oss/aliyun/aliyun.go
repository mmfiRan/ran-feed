// Package aliyun OSS 表单直传凭证实现
package aliyun

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"path"
	"strconv"
	"time"

	"github.com/aliyun/credentials-go/credentials"

	"ran-feed/pkg/oss"
)

const (
	signatureVersion = "OSS4-HMAC-SHA256"
	product          = "oss"
	aliyunV4Request  = "aliyun_v4_request"
	aliyunV4Prefix   = "aliyun_v4"
	defaultDuration  = int64(900)      // 凭证过期默认 15 分钟
	defaultMaxBytes  = int64(10 << 30) // 最大 10GB
)

// Config 阿里云 OSS 配置
type Config struct {
	Region          string
	BucketName      string
	AccessKeyID     string
	AccessKeySecret string
	RoleArn         string
	RoleSessionName string
	DurationSeconds int64
	UploadDir       string
}

// Strategy 阿里云 OSS 凭证策略
type Strategy struct {
	config Config
}

// New 创建阿里云策略
func New(config Config) *Strategy {
	if config.DurationSeconds <= 0 {
		config.DurationSeconds = defaultDuration
	}
	return &Strategy{
		config: config,
	}
}

// Generate 生成表单直传凭证
func (s *Strategy) Generate(ctx context.Context, req *oss.Request) (*oss.Credential, error) {
	cred, err := s.stsCredential()
	if err != nil {
		return nil, err
	}

	objectKey := s.objectKey(req)
	now := time.Now().UTC()
	date := now.Format("20060102")
	expiration := now.Add(time.Duration(s.config.DurationSeconds) * time.Second)
	credential := s.credential(*cred.AccessKeyId, date)

	policyDoc := s.policyDocument(objectKey, date, now, expiration, cred, req)
	policyJSON, err := json.Marshal(policyDoc)
	if err != nil {
		return nil, fmt.Errorf("序列化 policy 失败: %w", err)
	}
	encodedPolicy := base64.StdEncoding.EncodeToString(policyJSON)

	return &oss.Credential{
		URL:       fmt.Sprintf("https://%s.oss-%s.aliyuncs.com", s.config.BucketName, s.config.Region),
		Method:    "POST",
		ObjectKey: objectKey,
		Fields: map[string]string{
			"policy":                  encodedPolicy,
			"signature":               s.signature(encodedPolicy, date, *cred.AccessKeySecret),
			"x-oss-security-token":    *cred.SecurityToken,
			"x-oss-signature-version": signatureVersion,
			"x-oss-credential":        credential,
			"x-oss-date":              now.Format("20060102T150405Z"),
			"key":                     objectKey,
		},
		ExpiredAt: expiration.Unix(),
	}, nil
}

// objectKey 生成对象键
func (s *Strategy) objectKey(req *oss.Request) string {
	now := time.Now()
	return path.Join(
		s.config.UploadDir,
		req.Scene,
		strconv.FormatInt(req.UserID, 10),
		now.Format("20060102"),
		fmt.Sprintf("%d_%s", now.UnixMilli(), req.FileName),
	)
}

// policyDocument 构造 Policy
func (s *Strategy) policyDocument(objectKey, date string, now, expiration time.Time, cred *credentials.CredentialModel, req *oss.Request) map[string]any {
	conditions := []any{
		map[string]string{"bucket": s.config.BucketName},
		map[string]string{"key": objectKey},
		contentLengthRange(req.MaxBytes),
	}
	if req.ContentType != "" {
		conditions = append(conditions, []any{"starts-with", "$Content-Type", req.ContentType})
	}
	conditions = append(conditions,
		map[string]string{"x-oss-signature-version": signatureVersion},
		map[string]string{"x-oss-credential": s.credential(*cred.AccessKeyId, date)},
		map[string]string{"x-oss-date": now.Format("20060102T150405Z")},
		map[string]string{"x-oss-security-token": *cred.SecurityToken},
	)

	return map[string]any{
		"expiration": expiration.Format("2006-01-02T15:04:05.000Z"),
		"conditions": conditions,
	}
}

// contentLengthRange 大小上限条件
func contentLengthRange(max int64) []any {
	if max <= 0 {
		max = defaultMaxBytes
	}
	return []any{
		"content-length-range", int64(0), max,
	}
}

// credential 拼接 x-oss-credential
func (s *Strategy) credential(accessKeyID, date string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", accessKeyID, date, s.config.Region, product, aliyunV4Request)
}

// signature 对 policy 做 OSS4 签名
func (s *Strategy) signature(stringToSign, date, accessKeySecret string) string {
	hmacHash := func() hash.Hash {
		return sha256.New()
	}

	h1 := hmac.New(hmacHash, []byte(aliyunV4Prefix+accessKeySecret))
	_, _ = io.WriteString(h1, date)
	h2 := hmac.New(hmacHash, h1.Sum(nil))
	_, _ = io.WriteString(h2, s.config.Region)
	h3 := hmac.New(hmacHash, h2.Sum(nil))
	_, _ = io.WriteString(h3, product)
	h4 := hmac.New(hmacHash, h3.Sum(nil))
	_, _ = io.WriteString(h4, aliyunV4Request)

	h := hmac.New(hmacHash, h4.Sum(nil))
	_, _ = io.WriteString(h, stringToSign)

	return hex.EncodeToString(h.Sum(nil))
}

// stsCredential 获取 STS 临时凭证
func (s *Strategy) stsCredential() (*credentials.CredentialModel, error) {
	config := new(credentials.Config).
		SetType("ram_role_arn").
		SetAccessKeyId(s.config.AccessKeyID).
		SetAccessKeySecret(s.config.AccessKeySecret).
		SetRoleArn(s.config.RoleArn).
		SetRoleSessionName(s.config.RoleSessionName).
		SetPolicy("").
		SetRoleSessionExpiration(int(s.config.DurationSeconds))

	provider, err := credentials.NewCredential(config)
	if err != nil {
		return nil, fmt.Errorf("创建凭证提供器失败: %w", err)
	}

	cred, err := provider.GetCredential()
	if err != nil {
		return nil, fmt.Errorf("获取凭证失败: %w", err)
	}

	return cred, nil
}
