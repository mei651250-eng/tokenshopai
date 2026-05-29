package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tokenhub/backend/internal/gateway"
	"github.com/tokenhub/backend/internal/gateway/fallback"
	"github.com/tokenhub/backend/internal/gateway/lb"
	"github.com/tokenhub/backend/internal/gateway/protocol"
	"go.uber.org/zap"
)

// ModelRouter 模型路由器
type ModelRouter struct {
	logger    *zap.Logger
	registry  *protocol.ConverterRegistry
	fallback  *fallback.FallbackManager
	strategy  lb.Strategy

	mu     sync.RWMutex
	models map[string][]*gateway.ModelConfig // model_name -> configs (多个供应商实例)
}

// NewModelRouter 创建模型路由器
func NewModelRouter(
	logger *zap.Logger,
	strategy lb.Strategy,
	maxRetries int,
) *ModelRouter {
	return &ModelRouter{
		logger:   logger,
		registry: protocol.NewConverterRegistry(),
		fallback: fallback.NewFallbackManager(logger, strategy, maxRetries),
		strategy: strategy,
		models:   make(map[string][]*gateway.ModelConfig),
	}
}

// RouteResult 路由结果
type RouteResult struct {
	Converter    protocol.ProtocolConverter
	ModelConfig  *gateway.ModelConfig
	Attempt      int
	FallbackFrom string // 降级来源模型
}

// Route 执行路由选择
func (r *ModelRouter) Route(ctx context.Context, req *gateway.ChatRequest) (*RouteResult, error) {
	modelName := req.Model
	if modelName == "" {
		return nil, fmt.Errorf("model name is required")
	}

	// 1. 获取该模型名称的所有可用实例
	instances := r.getModelInstances(modelName, req.TenantID)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances for model: %s", modelName)
	}

	// 2. 通过降级管理器选择最优实例
	var excludeIDs []string
	maxAttempts := 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fbResult, err := r.fallback.RouteWithFallback(ctx, instances, excludeIDs)
		if err != nil {
			return nil, fmt.Errorf("route failed after %d attempts: %w", attempt, err)
		}

		selected := fbResult.Model

		// 3. 获取协议转换器
		converter, err := r.registry.Get(selected.Provider)
		if err != nil {
			r.logger.Warn("no converter for provider, excluding model",
				zap.String("provider", string(selected.Provider)),
				zap.String("model_id", selected.ID),
			)
			excludeIDs = append(excludeIDs, selected.ID)
			continue
		}

		return &RouteResult{
			Converter:    converter,
			ModelConfig:  selected,
			Attempt:      attempt,
			FallbackFrom: fbResult.FromModel,
		}, nil
	}

	return nil, fmt.Errorf("all attempts exhausted for model: %s", modelName)
}

// RecordSuccess 记录请求成功
func (r *ModelRouter) RecordSuccess(modelID string) {
	r.fallback.RecordSuccess(modelID)
}

// RecordFailure 记录请求失败
func (r *ModelRouter) RecordFailure(modelID string, err error) {
	r.fallback.RecordFailure(modelID, err)
}

// RegisterModel 注册模型实例
func (r *ModelRouter) RegisterModel(config *gateway.ModelConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := config.Name
	if config.TenantID != "" {
		key = config.TenantID + ":" + config.Name
	}
	r.models[key] = append(r.models[key], config)

	r.logger.Info("model registered",
		zap.String("model_id", config.ID),
		zap.String("name", config.Name),
		zap.String("provider", string(config.Provider)),
	)
}

// UnregisterModel 注销模型实例
func (r *ModelRouter) UnregisterModel(modelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, instances := range r.models {
		for i, m := range instances {
			if m.ID == modelID {
				r.models[key] = append(instances[:i], instances[i+1:]...)
				break
			}
		}
	}
}

// UpdateModelStats 更新模型统计（延迟、成功率）
func (r *ModelRouter) UpdateModelStats(modelID string, latencyMs int, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, instances := range r.models {
		for _, m := range instances {
			if m.ID == modelID {
				// 简单的指数移动平均
				alpha := 0.3
				m.LatencyMs = int(float64(m.LatencyMs)*(1-alpha) + float64(latencyMs)*alpha)
				if success {
					m.SuccessRate = m.SuccessRate*(1-alpha) + 1.0*alpha
				} else {
					m.SuccessRate = m.SuccessRate*(1-alpha) + 0.0*alpha
				}
				m.UpdatedAt = time.Now().Unix()
				break
			}
		}
	}
}

// getModelInstances 获取模型实例列表
func (r *ModelRouter) getModelInstances(modelName, tenantID string) []*gateway.ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 优先获取租户私有模型
	if tenantID != "" {
		key := tenantID + ":" + modelName
		if instances, ok := r.models[key]; ok {
			return instances
		}
	}

	// 回退到公共模型
	if instances, ok := r.models[modelName]; ok {
		return instances
	}

	return nil
}

// GetCircuitStates 获取熔断器状态
func (r *ModelRouter) GetCircuitStates() map[string]string {
	return r.fallback.GetCircuitStates()
}

// GetAllModels 获取所有已注册模型
func (r *ModelRouter) GetAllModels() []*gateway.ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []*gateway.ModelConfig
	for _, instances := range r.models {
		all = append(all, instances...)
	}
	return all
}
