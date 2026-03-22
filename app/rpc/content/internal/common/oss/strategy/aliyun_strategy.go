package strategy

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
	"path/filepath"
	"ran-feed/app/rpc/content/internal/common/oss"
	"strconv"
	"time"

	"github.com/aliyun/credentials-go/credentials"
)

const (
	product          = "oss"
	signatureVersion = "OSS4-HMAC-SHA256"
	aliyunV4Request  = "aliyun_v4_request"
	aliyunV4Prefix   = "aliyun_v4"
)

// AliyunConfig 阿里云 OSS 配置
type AliyunConfig struct {
	Region          string
	BucketName      string
	AccessKeyId     string
	AccessKeySecret string
	RoleArn         string
	RoleSessionName string
	DurationSeconds int64
	UploadDir       string
}

// AliyunStrategy 阿里云 OSS 策略
type AliyunStrategy struct {
	config *AliyunConfig
}

// NewAliyunStrategy 创建阿里云 OSS 策略
func NewAliyunStrategy(config *AliyunConfig) *AliyunStrategy {
	return &AliyunStrategy{
		config: config,
	}
}

// GenerateUploadCredential 生成上传凭证
func (s *AliyunStrategy) GenerateUploadCredential(ctx context.Context, policy *oss.UploadPolicy) (*oss.UploadCredential, error) {
	// 1. 获取 STS 临时凭证
	cred, err := s.getSTSCredential()
	if err != nil {
		return nil, fmt.Errorf("获取 STS 凭证失败: %w", err)
	}

	// 2. 生成对象键（文件存储路径）
	objectKey := s.generateObjectKey(policy)

	// 3. 构建 Policy
	utcTime := time.Now().UTC()
	date := utcTime.Format("20060102")
	expiration := utcTime.Add(time.Duration(s.config.DurationSeconds) * time.Second)

	policyDoc := s.buildPolicyDocument(objectKey, date, expiration, utcTime, cred)

	// 4. 对 Policy 进行 Base64 编码
	policyJSON, err := json.Marshal(policyDoc)
	if err != nil {
		return nil, fmt.Errorf("序列化 policy 失败: %w", err)
	}
	encodedPolicy := base64.StdEncoding.EncodeToString(policyJSON)

	// 5. 生成签名
	signature := s.generateSignature(encodedPolicy, date, *cred.AccessKeySecret)

	// 6. 构建返回数据
	host := fmt.Sprintf("https://%s.oss-%s.aliyuncs.com", s.config.BucketName, s.config.Region)
	credential := fmt.Sprintf("%s/%s/%s/%s/%s",
		*cred.AccessKeyId, date, s.config.Region, product, aliyunV4Request)

	return &oss.UploadCredential{
		ObjectKey: objectKey,
		FormData: &oss.FormData{
			Host:             host,
			Policy:           encodedPolicy,
			Signature:        signature,
			SecurityToken:    *cred.SecurityToken,
			SignatureVersion: signatureVersion,
			Credential:       credential,
			Date:             utcTime.Format("20060102T150405Z"),
			Key:              objectKey,
		},
		ExpiredAt: expiration.Unix(),
	}, nil
}

// getSTSCredential 获取 STS 临时凭证
func (s *AliyunStrategy) getSTSCredential() (*credentials.CredentialModel, error) {
	config := new(credentials.Config).
		SetType("ram_role_arn").
		SetAccessKeyId(s.config.AccessKeyId).
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

// generateObjectKey 生成对象键（文件存储路径）
// 格式：{uploadDir}/{scene}/{userId}/{date}/{timestamp}_{filename}
func (s *AliyunStrategy) generateObjectKey(policy *oss.UploadPolicy) string {
	now := time.Now()
	datePath := now.Format("20060102")
	timestamp := now.UnixMilli()

	// 构建路径：uploads/scene/userId/20240108/1704672000000_filename.ext
	return filepath.Join(
		s.config.UploadDir,
		policy.Scene,
		strconv.FormatInt(policy.UserId, 10),
		datePath,
		fmt.Sprintf("%d_%s", timestamp, policy.FileName),
	)
}

// buildPolicyDocument 构建 Policy 文档
func (s *AliyunStrategy) buildPolicyDocument(objectKey, date string, expiration, utcTime time.Time, cred *credentials.CredentialModel) map[string]any {
	return map[string]any{
		"expiration": expiration.Format("2006-01-02T15:04:05.000Z"),
		"conditions": []any{
			map[string]string{"bucket": s.config.BucketName},
			map[string]string{"x-oss-signature-version": signatureVersion},
			map[string]string{
				"x-oss-credential": fmt.Sprintf("%s/%s/%s/%s/%s",
					*cred.AccessKeyId, date, s.config.Region, product, aliyunV4Request),
			},
			map[string]string{"x-oss-date": utcTime.Format("20060102T150405Z")},
			map[string]string{"x-oss-security-token": *cred.SecurityToken},
		},
	}
}

// generateSignature 生成 OSS4 签名
func (s *AliyunStrategy) generateSignature(stringToSign, date, accessKeySecret string) string {
	hmacHash := func() hash.Hash { return sha256.New() }

	// 构建 signing key
	signingKey := aliyunV4Prefix + accessKeySecret

	// 第一层：date
	h1 := hmac.New(hmacHash, []byte(signingKey))
	io.WriteString(h1, date)
	h1Key := h1.Sum(nil)

	// 第二层：region
	h2 := hmac.New(hmacHash, h1Key)
	io.WriteString(h2, s.config.Region)
	h2Key := h2.Sum(nil)

	// 第三层：product
	h3 := hmac.New(hmacHash, h2Key)
	io.WriteString(h3, product)
	h3Key := h3.Sum(nil)

	// 第四层：request type
	h4 := hmac.New(hmacHash, h3Key)
	io.WriteString(h4, aliyunV4Request)
	h4Key := h4.Sum(nil)

	// 最终签名
	h := hmac.New(hmacHash, h4Key)
	io.WriteString(h, stringToSign)

	return hex.EncodeToString(h.Sum(nil))
}
