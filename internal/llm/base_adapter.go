package llm

import (
	pkg "ai-ops/internal/pkg"
	"ai-ops/internal/pkg/errors"
	"ai-ops/internal/tools"
	"context"
	"sync"
	"time"
)

// BaseAdapter 基础适配器实现，提供通用功能
type BaseAdapter struct {
	// info 适配器信息
	info AdapterInfo

	// metrics 性能指标
	metrics AdapterMetrics

	// mu 互斥锁，保证线程安全
	mu sync.RWMutex

	// initialized 是否已初始化
	initialized bool

	// errorMapper 错误映射器
	errorMapper ErrorMapper
}

// NewBaseAdapter 创建新的基础适配器
func NewBaseAdapter(info AdapterInfo) *BaseAdapter {
	return &BaseAdapter{
		info:        info,
		metrics:     AdapterMetrics{},
		initialized: false,
	}
}

// GetAdapterInfo 获取适配器信息
func (b *BaseAdapter) GetAdapterInfo() AdapterInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.info
}

// HealthCheck 健康检查
func (b *BaseAdapter) HealthCheck(ctx context.Context) error {
	b.mu.RLock()
	initialized := b.initialized
	b.mu.RUnlock()

	if !initialized {
		return errors.NewError(errors.ErrCodeInvalidConfig, "adapter not initialized")
	}

	return nil
}

// ValidateConfig 验证配置（基础实现）
func (b *BaseAdapter) ValidateConfig(config interface{}) error {
	// 默认实现：不进行额外验证
	return nil
}

// Initialize 初始化适配器
// 说明：基础适配器不再在此处触发配置校验，避免“伪多态”导致子类校验被绕过。
// 配置校验应在：注册表验证器/具体客户端创建流程中完成。
func (b *BaseAdapter) Initialize(ctx context.Context, config interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.initialized = true

	pkg.Debugw("适配器初始化成功", map[string]interface{}{
		"adapter_name": b.info.Name,
		"adapter_type": b.info.Type,
	})

	return nil
}

// UpdateMetrics 更新性能指标
func (b *BaseAdapter) UpdateMetrics(responseTime int64, success bool, tokensUsed int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.metrics.RequestCount++
	b.metrics.LastRequestTime = time.Now().Unix()
	b.metrics.TokensUsed += tokensUsed

	if !success {
		b.metrics.ErrorCount++
	}

	// 计算平均响应时间（增量/加权）：avg_n = avg_{n-1} + (x_n - avg_{n-1}) / n
	n := b.metrics.RequestCount
	if n == 1 {
		b.metrics.AverageResponseTime = responseTime
	} else {
		diff := responseTime - b.metrics.AverageResponseTime
		b.metrics.AverageResponseTime = b.metrics.AverageResponseTime + diff/int64(n)
	}
}

// GetMetrics 获取性能指标
func (b *BaseAdapter) GetMetrics() AdapterMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.metrics
}

// SetErrorMapper 设置错误映射器
func (b *BaseAdapter) SetErrorMapper(mapper ErrorMapper) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.errorMapper = mapper
}

// MapError 映射错误
func (b *BaseAdapter) MapError(originalError error) error {
	b.mu.RLock()
	mapper := b.errorMapper
	b.mu.RUnlock()

	if mapper != nil {
		return mapper.MapError(originalError)
	}
	return originalError
}

// RecordError 记录错误
func (b *BaseAdapter) RecordError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.metrics.LastError = err.Error()
	}
}

// GetStatus 获取适配器状态
func (b *BaseAdapter) GetStatus() AdapterStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return AdapterStatus{
		Name:            b.info.Name,
		Healthy:         b.initialized,
		LastHealthCheck: time.Now().Unix(),
		Metrics:         b.metrics,
		LastError:       b.metrics.LastError,
	}
}

// ClientAdapterWrapper 客户端适配器包装器，将现有的AIClient包装为ModelAdapter
type ClientAdapterWrapper struct {
	*BaseAdapter
	client AIClient
}

// NewClientAdapterWrapper 创建新的客户端适配器包装器
func NewClientAdapterWrapper(client AIClient, info AdapterInfo) *ClientAdapterWrapper {
	w := &ClientAdapterWrapper{
		BaseAdapter: NewBaseAdapter(info),
		client:      client,
	}
	// 确保已初始化，避免健康检查失败；此处不做配置校验
	_ = w.Initialize(context.Background(), nil)
	return w
}

// SendMessage 发送消息并获取响应
func (w *ClientAdapterWrapper) SendMessage(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	startTime := time.Now()

	// 调用原始客户端
	response, err := w.client.SendMessage(ctx, messages, toolDefs)

	// 计算响应时间
	responseTime := time.Since(startTime).Milliseconds()

	// 更新指标
	var tokensUsed int64
	if response != nil {
		tokensUsed = int64(response.Usage.TotalTokens)
	}
	w.UpdateMetrics(responseTime, err == nil, tokensUsed)

	// 如果有错误，进行错误映射和记录
	if err != nil {
		w.RecordError(err)
		err = w.MapError(err)
	}

	return response, err
}

// GetModelInfo 获取模型信息
func (w *ClientAdapterWrapper) GetModelInfo() ModelInfo {
	return w.client.GetModelInfo()
}
