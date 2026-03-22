package strategy

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	commonoss "ran-feed/app/rpc/user/internal/common/oss"
)

// AliyunConfig 阿里云 OSS 配置
type AliyunConfig struct {
	Region          string
	BucketName      string
	AccessKeyId     string
	AccessKeySecret string
	Endpoint        string
	UploadDir       string
	PublicHost      string
}

// AliyunStrategy 阿里云 OSS 策略
type AliyunStrategy struct {
	config *AliyunConfig
}

func NewAliyunStrategy(config *AliyunConfig) *AliyunStrategy {
	return &AliyunStrategy{config: config}
}

func (s *AliyunStrategy) UploadObject(ctx context.Context, req *commonoss.UploadRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("无效上传请求")
	}
	if req.ObjectKey == "" {
		return "", fmt.Errorf("ObjectKey 不能为空")
	}
	bucketName := normalizeConfigValue(s.config.BucketName)
	if bucketName == "" {
		return "", fmt.Errorf("OSS BucketName 不能为空")
	}
	region := normalizeConfigValue(s.config.Region)

	endpoint := normalizeConfigValue(s.config.Endpoint)
	if endpoint == "" {
		if region == "" {
			return "", fmt.Errorf("OSS Region 不能为空")
		}
		endpoint = fmt.Sprintf("https://oss-%s.aliyuncs.com", region)
	}
	endpoint = normalizeEndpoint(endpoint, bucketName)

	client, err := oss.New(endpoint, s.config.AccessKeyId, s.config.AccessKeySecret)
	if err != nil {
		return "", err
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return "", err
	}

	objectKey := req.ObjectKey
	if dir := normalizeConfigValue(s.config.UploadDir); dir != "" {
		objectKey = path.Join(dir, objectKey)
	}

	options := make([]oss.Option, 0, 2)
	if ct := strings.TrimSpace(req.ContentType); ct != "" {
		options = append(options, oss.ContentType(ct))
	}
	if cc := strings.TrimSpace(req.CacheControl); cc != "" {
		options = append(options, oss.CacheControl(cc))
	}

	if err := bucket.PutObject(objectKey, bytes.NewReader(req.Content), options...); err != nil {
		return "", err
	}

	publicHost := strings.TrimRight(normalizeConfigValue(s.config.PublicHost), "/")
	if publicHost == "" {
		if region == "" {
			return "", fmt.Errorf("OSS Region 不能为空")
		}
		publicHost = fmt.Sprintf("https://%s.oss-%s.aliyuncs.com", bucketName, region)
	}

	return publicHost + "/" + objectKey, nil
}

func normalizeConfigValue(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
		return ""
	}
	return v
}

func normalizeEndpoint(endpoint, bucketName string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return endpoint
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	if strings.HasPrefix(u.Host, bucketName+".") {
		u.Host = strings.TrimPrefix(u.Host, bucketName+".")
	}
	return u.String()
}
