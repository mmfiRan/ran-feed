package oss

import (
	"context"
	"fmt"
)

// Context 策略上下文
type Context struct {
	strategy Strategy
}

func NewContext(strategy Strategy) *Context {
	return &Context{
		strategy: strategy,
	}
}

func (c *Context) SetStrategy(strategy Strategy) {
	c.strategy = strategy
}

// GenerateUploadCredential 生成上传凭证
func (c *Context) GenerateUploadCredential(ctx context.Context, policy *UploadPolicy) (*UploadCredential, error) {
	if c.strategy == nil {
		return nil, fmt.Errorf("OSS策略未设置")
	}
	return c.strategy.GenerateUploadCredential(ctx, policy)
}
