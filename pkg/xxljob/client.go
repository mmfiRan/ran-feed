package xxljob

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AdminClient struct {
	addresses   []string
	http        *http.Client
	accessToken string
	validator   ResultValidator
}

type ResultValidator func(statusCode int, body []byte) error

func NewAdminClient(addresses []string, timeout time.Duration, accessToken string) *AdminClient {
	return &AdminClient{
		addresses:   addresses,
		http:        &http.Client{Timeout: timeout},
		accessToken: accessToken,
		validator:   StrictReturnTValidator,
	}
}

func (c *AdminClient) SetResultValidator(v ResultValidator) {
	if v != nil {
		c.validator = v
	}
}

func (c *AdminClient) Register(ctx context.Context, param RegistryParam) error {
	return c.postOne(ctx, "/api/registry", param)
}

func (c *AdminClient) Unregister(ctx context.Context, param RegistryParam) error {
	return c.postOne(ctx, "/api/registryRemove", param)
}

func (c *AdminClient) Callback(ctx context.Context, params []HandleCallbackParam) error {
	return c.postOne(ctx, "/api/callback", params)
}

func (c *AdminClient) postOne(ctx context.Context, path string, body interface{}) error {
	if len(c.addresses) == 0 {
		return errors.New("xxljob: admin address required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var lastErr error
	for _, addr := range c.addresses {
		addr = strings.TrimRight(addr, "/")
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+path, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if c.accessToken != "" {
			req.Header.Set(HeaderAccessToken, c.accessToken)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if c.validator != nil {
			if err := c.validator(resp.StatusCode, b); err != nil {
				lastErr = err
				continue
			}
		}
		return nil
	}
	return lastErr
}

func StrictReturnTValidator(statusCode int, body []byte) error {
	if statusCode/100 != 2 {
		return errors.New(http.StatusText(statusCode))
	}
	var rt returnTRaw
	if err := json.Unmarshal(body, &rt); err != nil {
		return err
	}
	if rt.Code != SuccessCode {
		return fmt.Errorf("xxljob admin error: code=%d msg=%s", rt.Code, rt.Msg)
	}
	return nil
}
