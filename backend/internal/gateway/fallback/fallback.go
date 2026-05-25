package fallback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tokenhub/backend/internal/gateway"
	"github.com/tokenhub/backend/internal/gateway/lb"
	"go.uber.org/zap"
)

// FallbackManager 降级管理器
type FallbackManager struct {
	logger     *zap.Logger
	lb         lb.LoadBalancer
	maxRetries int
	retryDelay time.Duration

	mu      sync.RWMutex
	fails   map[string]*failCounter // model_id -> fail counter
	circuit map[string]*circuitBreaker
}

type failCounter struct {
	count       int
	lastFailAt  time.Time
	cooldown    time.Duration // 冷却时间
}

type circuitBreaker struct {
	state       circuitState
	failures    int
	lastFailAt  time.Time
	threshold   int           // 连续失败阈值
	timeout     time.Duration // 熔断恢复超时
}

type circuitState int

const (
	circuitClosed   circuitState = iota // 正常
	circuitOpen                          // 熔断
	circuitHalfOpen                      // 半开
)

// NewFallbackManager 创建降级管理器
func NewFallbackManager(logger *zap.Logger, strategy lb.Strategy, maxRetries int) *FallbackManager {
	return &FallbackManager{
		logger:     logger,
		lb:         lb.NewLoadBalancer(strategy),
		maxRetries: maxRetries,
		retryDelay: 100 * time.Millisecond,
		fails:      make(map[string]*failCounter),
		circuit:    make(map[string]*circuitBreaker),
	}
}

// RouteResult 路由结果
type RouteResult struct {
	Model      *gateway.ModelConfig
	Attempt    int
	Fallback   bool
	FromModel  string // 从哪个模型降级来的
	Error      error
}

// RouteWithFallback 带降级的路由调度
func (fm *FallbackManager) RouteWithFallback(
	ctx context.Context,
	models []*gateway.ModelConfig,
	excludeIDs []string,
) (*RouteResult, error) {
	available := fm.filterAvailable(models, excludeIDs)
	if len(available) == 0 {
		return nil, fmt.Errorf("all models unavailable, excluded: %v", excludeIDs)
	}

	selected, err := fm.lb.Select(available)
	if err != nil {
		return nil, err
	}

	result := &RouteResult{
		Model:     selected,
		Attempt:   1,
		Fallback:  len(excludeIDs) > 0,
		FromModel: "",
	}

	if len(excludeIDs) > 0 {
		result.FromModel = excludeIDs[len(excludeIDs)-1]
	}

	return result, nil
}

// RecordSuccess 记录成功
func (fm *FallbackManager) RecordSuccess(modelID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// 重置失败计数
	delete(fm.fails, modelID)

	// 半开状态下恢复
	if cb, ok := fm.circuit[modelID]; ok {
		cb.failures = 0
		cb.state = circuitClosed
	}
}

// RecordFailure 记录失败
func (fm *FallbackManager) RecordFailure(modelID string, err error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.logger.Warn("model request failed",
		zap.String("model_id", modelID),
		zap.Error(err),
	)

	// 更新失败计数
	counter, ok := fm.fails[modelID]
	if !ok {
		counter = &failCounter{
			cooldown: 30 * time.Second,
		}
		fm.fails[modelID] = counter
	}
	counter.count++
	counter.lastFailAt = time.Now()

	// 更新熔断器
	cb, ok := fm.circuit[modelID]
	if !ok {
		cb = &circuitBreaker{
			threshold: fm.maxRetries,
			timeout:   60 * time.Second,
		}
		fm.circuit[modelID] = cb
	}
	cb.failures++
	cb.lastFailAt = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = circuitOpen
		fm.logger.Error("circuit breaker opened",
			zap.String("model_id", modelID),
			zap.Int("failures", cb.failures),
		)
	}
}

// filterAvailable 过滤不可用的模型
func (fm *FallbackManager) filterAvailable(models []*gateway.ModelConfig, excludeIDs []string) []*gateway.ModelConfig {
	excludeSet := make(map[string]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var available []*gateway.ModelConfig
	for _, m := range models {
		if !m.Enabled {
			continue
		}
		if excludeSet[m.ID] {
			continue
		}

		// 检查熔断器状态
		if cb, ok := fm.circuit[m.ID]; ok {
			switch cb.state {
			case circuitOpen:
				// 检查是否可以进入半开状态
				if time.Since(cb.lastFailAt) > cb.timeout {
					cb.state = circuitHalfOpen
				} else {
					continue // 熔断中，跳过
				}
			case circuitHalfOpen:
				// 半开状态允许少量请求通过
			}
		}

		// 检查冷却期
		if counter, ok := fm.fails[m.ID]; ok {
			if time.Since(counter.lastFailAt) < counter.cooldown {
				continue
			}
		}

		available = append(available, m)
	}

	return available
}

// GetCircuitStates 获取熔断器状态（用于监控）
func (fm *FallbackManager) GetCircuitStates() map[string]string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	states := make(map[string]string)
	for id, cb := range fm.circuit {
		switch cb.state {
		case circuitClosed:
			states[id] = "closed"
		case circuitOpen:
			states[id] = "open"
		case circuitHalfOpen:
			states[id] = "half_open"
		}
	}
	return states
}
