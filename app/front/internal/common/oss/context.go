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
	return &Context{strategy: strategy}
}

func (c *Context) SetStrategy(strategy Strategy) {
	c.strategy = strategy
}

func (c *Context) UploadObject(ctx context.Context, req *UploadRequest) (string, error) {
	if c.strategy == nil {
		return "", fmt.Errorf("OSS策略未设置")
	}
	return c.strategy.UploadObject(ctx, req)
}
