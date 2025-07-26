package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-ops/internal/util"
)

// HTTPClient HTTP 客户端接口
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AIHTTPClient AI 专用 HTTP 客户端
type AIHTTPClient struct {
	client  HTTPClient
	timeout time.Duration
	baseURL string
	headers map[string]string
}

// NewAIHTTPClient 创建新的 AI HTTP 客户端
func NewAIHTTPClient(baseURL string, timeout time.Duration) *AIHTTPClient {
	return &AIHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
		baseURL: baseURL,
		headers: make(map[string]string),
	}
}

// SetHeader 设置请求头
func (c *AIHTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetHeaders 批量设置请求头
func (c *AIHTTPClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

// Post 发送 POST 请求
func (c *AIHTTPClient) Post(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	url := c.baseURL
	if endpoint != "" {
		url = strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	}

	var body io.Reader
	var jsonData []byte
	var err error

	if payload != nil {
		jsonData, err = json.Marshal(payload)
		if err != nil {
			return nil, NewAIError(ErrCodeInvalidParameters, "failed to marshal request payload", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, NewAIError(ErrCodeInvalidParameters, "failed to create HTTP request", err)
	}

	// 设置默认请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ai-ops/1.0")

	// 设置自定义请求头
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// 记录详细的请求信息
	headersMap := make(map[string]string)
	for name, values := range req.Header {
		headersMap[name] = strings.Join(values, ", ")
	}
	util.Debugw("发送 HTTP POST 请求", map[string]interface{}{
		"url":     url,
		"headers": headersMap,
		"body":    string(jsonData),
	})

	resp, err := c.client.Do(req)
	if err != nil {
		util.Errorw("HTTP 请求失败", map[string]interface{}{"error": err, "url": url})
		return nil, NewAIError(ErrCodeNetworkFailed, "HTTP request failed", err)
	}

	// util.Debugw("收到 HTTP 响应", map[string]interface{}{
	// 	"url":    url,
	// 	"status": resp.Status,
	// })

	return resp, nil
}

// Get 发送 GET 请求
func (c *AIHTTPClient) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewAIError(ErrCodeInvalidParameters, "failed to create HTTP request", err)
	}

	// 设置默认请求头
	req.Header.Set("User-Agent", "ai-ops/1.0")

	// 设置自定义请求头
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, NewAIError(ErrCodeNetworkFailed, "HTTP request failed", err)
	}

	return resp, nil
}

// PostJSON 发送 JSON POST 请求并解析响应
func (c *AIHTTPClient) PostJSON(ctx context.Context, endpoint string, payload interface{}, result interface{}) error {
	resp, err := c.Post(ctx, endpoint, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := c.handleHTTPError(resp); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return NewAIError(ErrCodeInvalidResponse, "failed to decode response", err)
		}
	}

	return nil
}

// GetJSON 发送 GET 请求并解析 JSON 响应
func (c *AIHTTPClient) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := c.handleHTTPError(resp); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return NewAIError(ErrCodeInvalidResponse, "failed to decode response", err)
		}
	}

	return nil
}

// handleHTTPError 处理 HTTP 错误状态码
func (c *AIHTTPClient) handleHTTPError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// 读取错误响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewAIError(ErrCodeNetworkFailed, fmt.Sprintf("HTTP %d: failed to read error response", resp.StatusCode), err)
	}

	// 记录错误响应
	util.Warnw("HTTP 错误响应", map[string]interface{}{
		"status": resp.Status,
		"body":   string(body),
		"url":    resp.Request.URL.String(),
	})

	switch resp.StatusCode {
	case 400:
		return NewAIErrorWithDetails(ErrCodeInvalidParameters, "Bad Request", string(body), nil)
	case 401:
		return NewAIErrorWithDetails(ErrCodeAPIKeyMissing, "Unauthorized", string(body), nil)
	case 403:
		return NewAIErrorWithDetails(ErrCodeAPIKeyMissing, "Forbidden", string(body), nil)
	case 429:
		return NewAIErrorWithDetails(ErrCodeRateLimited, "Rate Limited", string(body), nil)
	case 500, 502, 503, 504:
		return NewAIErrorWithDetails(ErrCodeNetworkFailed, "Server Error", string(body), nil)
	default:
		return NewAIErrorWithDetails(ErrCodeNetworkFailed, fmt.Sprintf("HTTP %d", resp.StatusCode), string(body), nil)
	}
}

// RetryableHTTPClient 支持重试的 HTTP 客户端
type RetryableHTTPClient struct {
	*AIHTTPClient
	maxRetries int
	retryDelay time.Duration
}

// NewRetryableHTTPClient 创建支持重试的 HTTP 客户端
func NewRetryableHTTPClient(baseURL string, timeout time.Duration, maxRetries int, retryDelay time.Duration) *RetryableHTTPClient {
	return &RetryableHTTPClient{
		AIHTTPClient: NewAIHTTPClient(baseURL, timeout),
		maxRetries:   maxRetries,
		retryDelay:   retryDelay,
	}
}

// PostJSONWithRetry 带重试的 JSON POST 请求
func (c *RetryableHTTPClient) PostJSONWithRetry(ctx context.Context, endpoint string, payload interface{}, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := c.retryDelay * time.Duration(1<<(attempt-1))
			util.Debugw("请求失败，正在重试...", map[string]interface{}{
				"attempt":  attempt,
				"backoff":  backoff.String(),
				"last_err": lastErr.Error(),
			})
			select {
			case <-ctx.Done():
				return NewAIError(ErrCodeContextCanceled, "request context canceled", ctx.Err())
			case <-time.After(backoff):
			}
		}

		err := c.PostJSON(ctx, endpoint, payload, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试
		if !c.shouldRetry(err) {
			break
		}
	}

	return lastErr
}

// shouldRetry 判断是否应该重试
func (c *RetryableHTTPClient) shouldRetry(err error) bool {
	if aiErr, ok := err.(*AIError); ok {
		switch aiErr.Code {
		case ErrCodeNetworkFailed, ErrCodeTimeout, ErrCodeRateLimited:
			return true
		default:
			return false
		}
	}
	return false
}
