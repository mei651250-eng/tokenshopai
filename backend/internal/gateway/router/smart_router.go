package router

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/tokenhub/backend/internal/gateway"
	"go.uber.org/zap"
)

// RouteStrategy 路由策略
type RouteStrategy string

const (
	StrategyCheapest    RouteStrategy = "cheapest"     // 最低成本
	StrategyFastest     RouteStrategy = "fastest"       // 最低延迟
	StrategyRoundRobin  RouteStrategy = "round_robin"   // 轮询
	StrategyWeighted    RouteStrategy = "weighted"      // 加权随机
	StrategyFailover    RouteStrategy = "failover"       // 故障转移
	StrategySmart      RouteStrategy = "smart"          // 智能路由（综合评分）
)

// SmartRouteResult 智能路由结果
type SmartRouteResult struct {
	ModelConfig *gateway.ModelConfig `json:"model_config"`
	Strategy    RouteStrategy        `json:"strategy"`
	Score       float64              `json:"score"`
	LatencyMs   int                  `json:"latency_ms"`
	CostPer1K   float64              `json:"cost_per_1k"`
}

// SmartRouter 智能路由器
type SmartRouter struct {
	logger   *zap.Logger
	rdb      *redis.Client
	strategy RouteStrategy
	mu       sync.RWMutex
	counters map[string]int64 // 轮询计数器
}

// RouterConfig 路由器配置
type RouterConfig struct {
	DefaultStrategy RouteStrategy `json:"default_strategy"`
	CostWeight      float64      `json:"cost_weight"`       // 成本权重（0-1）
	LatencyWeight   float64      `json:"latency_weight"`     // 延迟权重（0-1）
	ReliabilityWeight float64    `json:"reliability_weight"` // 可靠性权重（0-1）
	MaxLatencyMs    int          `json:"max_latency_ms"`     // 最大可接受延迟
	MinSuccessRate  float64      `json:"min_success_rate"`   // 最低成功率阈值
}

// DefaultRouterConfig 默认路由配置
var DefaultRouterConfig = RouterConfig{
	DefaultStrategy:  StrategySmart,
	CostWeight:       0.4,
	LatencyWeight:    0.3,
	ReliabilityWeight: 0.3,
	MaxLatencyMs:     30000,
	MinSuccessRate:   0.95,
}

// NewSmartRouter 创建智能路由器
func NewSmartRouter(logger *zap.Logger, rdb *redis.Client, strategy RouteStrategy) *SmartRouter {
	return &SmartRouter{
		logger:   logger,
		rdb:      rdb,
		strategy: strategy,
		counters: make(map[string]int64),
	}
}

// Route 路由请求到最佳上游
func (sr *SmartRouter) Route(ctx context.Context, modelName string, configs []*gateway.ModelConfig) (*SmartRouteResult, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no available model configs for %s", modelName)
	}

	// 过滤不可用的
	var available []*gateway.ModelConfig
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		available = append(available, cfg)
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no enabled model configs for %s", modelName)
	}

	// 根据策略选择
	switch sr.strategy {
	case StrategyCheapest:
		return sr.routeCheapest(available), nil
	case StrategyFastest:
		return sr.routeFastest(available), nil
	case StrategyRoundRobin:
		return sr.routeRoundRobin(modelName, available), nil
	case StrategyWeighted:
		return sr.routeWeighted(available), nil
	case StrategyFailover:
		return sr.routeFailover(available), nil
	case StrategySmart:
		return sr.routeSmart(available), nil
	default:
		return sr.routeSmart(available), nil
	}
}

// routeCheapest 最低成本路由
func (sr *SmartRouter) routeCheapest(configs []*gateway.ModelConfig) *SmartRouteResult {
	sort.Slice(configs, func(i, j int) bool {
		costI := configs[i].InputPrice + configs[i].OutputPrice
		costJ := configs[j].InputPrice + configs[j].OutputPrice
		return costI < costJ
	})

	cfg := configs[0]
	return &SmartRouteResult{
		ModelConfig: cfg,
		Strategy:    StrategyCheapest,
		CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
		LatencyMs:   cfg.LatencyMs,
	}
}

// routeFastest 最低延迟路由
func (sr *SmartRouter) routeFastest(configs []*gateway.ModelConfig) *SmartRouteResult {
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].LatencyMs < configs[j].LatencyMs
	})

	cfg := configs[0]
	return &SmartRouteResult{
		ModelConfig: cfg,
		Strategy:    StrategyFastest,
		LatencyMs:   cfg.LatencyMs,
		CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
	}
}

// routeRoundRobin 轮询路由
func (sr *SmartRouter) routeRoundRobin(modelName string, configs []*gateway.ModelConfig) *SmartRouteResult {
	sr.mu.Lock()
	sr.counters[modelName]++
	idx := sr.counters[modelName] % int64(len(configs))
	sr.mu.Unlock()

	cfg := configs[idx]
	return &SmartRouteResult{
		ModelConfig: cfg,
		Strategy:    StrategyRoundRobin,
		CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
		LatencyMs:   cfg.LatencyMs,
	}
}

// routeWeighted 加权随机路由
func (sr *SmartRouter) routeWeighted(configs []*gateway.ModelConfig) *SmartRouteResult {
	totalWeight := 0
	for _, cfg := range configs {
		totalWeight += cfg.Weight
	}

	if totalWeight == 0 {
		// 权重都是0，等概率选择
		cfg := configs[0]
		return &SmartRouteResult{
			ModelConfig: cfg,
			Strategy:    StrategyWeighted,
			CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
			LatencyMs:   cfg.LatencyMs,
		}
	}

	// 加权随机
	target := time.Now().UnixNano() % int64(totalWeight)
	cumWeight := 0
	for _, cfg := range configs {
		cumWeight += cfg.Weight
		if int64(cumWeight) > target {
			return &SmartRouteResult{
				ModelConfig: cfg,
				Strategy:    StrategyWeighted,
				CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
				LatencyMs:   cfg.LatencyMs,
			}
		}
	}

	cfg := configs[len(configs)-1]
	return &SmartRouteResult{
		ModelConfig: cfg,
		Strategy:    StrategyWeighted,
		CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
		LatencyMs:   cfg.LatencyMs,
	}
}

// routeFailover 故障转移路由
func (sr *SmartRouter) routeFailover(configs []*gateway.ModelConfig) *SmartRouteResult {
	// 按优先级排序
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Priority < configs[j].Priority
	})

	// 选择第一个可用的
	for _, cfg := range configs {
		if cfg.SuccessRate >= DefaultRouterConfig.MinSuccessRate {
			return &SmartRouteResult{
				ModelConfig: cfg,
				Strategy:    StrategyFailover,
				CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
				LatencyMs:   cfg.LatencyMs,
			}
		}
	}

	// 所有都不太健康，选优先级最高的
	cfg := configs[0]
	return &SmartRouteResult{
		ModelConfig: cfg,
		Strategy:    StrategyFailover,
		CostPer1K:   cfg.InputPrice + cfg.OutputPrice,
		LatencyMs:   cfg.LatencyMs,
	}
}

// routeSmart 智能路由（综合评分）
func (sr *SmartRouter) routeSmart(configs []*gateway.ModelConfig) *SmartRouteResult {
	cfg := DefaultRouterConfig

	type scored struct {
		config *gateway.ModelConfig
		score  float64
	}

	var candidates []scored
	maxCost := 0.0
	maxLatency := 0.0

	// 归一化参数
	for _, c := range configs {
		cost := c.InputPrice + c.OutputPrice
		if cost > maxCost {
			maxCost = cost
		}
		if float64(c.LatencyMs) > maxLatency {
			maxLatency = float64(c.LatencyMs)
		}
	}

	for _, c := range configs {
		// 成本评分：成本越低分越高
		costScore := 1.0
		if maxCost > 0 {
			costScore = 1.0 - (c.InputPrice+c.OutputPrice)/maxCost
		}

		// 延迟评分：延迟越低分越高
		latencyScore := 1.0
		if maxLatency > 0 {
			latencyScore = 1.0 - float64(c.LatencyMs)/maxLatency
		}

		// 可靠性评分
		reliabilityScore := c.SuccessRate

		// 综合评分
		totalScore := cfg.CostWeight*costScore +
			cfg.LatencyWeight*latencyScore +
			cfg.ReliabilityWeight*reliabilityScore

		candidates = append(candidates, scored{config: c, score: totalScore})
	}

	// 按评分排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	best := candidates[0]
	return &SmartRouteResult{
		ModelConfig: best.config,
		Strategy:    StrategySmart,
		Score:       math.Round(best.score*100) / 100,
		CostPer1K:   best.config.InputPrice + best.config.OutputPrice,
		LatencyMs:   best.config.LatencyMs,
	}
}

// RecordMetrics 记录上游性能指标（用于动态调整路由）
func (sr *SmartRouter) RecordMetrics(ctx context.Context, modelID string, latencyMs int, success bool) {
	key := fmt.Sprintf("router:metrics:%s", modelID)
	now := time.Now().Unix()

	// 使用滑动窗口更新平均延迟和成功率
	sr.rdb.HSet(ctx, key, map[string]interface{}{
		"last_latency": latencyMs,
		"last_success": success,
		"updated_at":   now,
	})

	// 更新延迟均值（简化版，生产应使用 HDR Histogram）
	avgLatency, _ := sr.rdb.HGet(ctx, key, "avg_latency_ms").Int()
	if avgLatency == 0 {
		sr.rdb.HSet(ctx, key, "avg_latency_ms", latencyMs)
	} else {
		newAvg := (avgLatency*9 + latencyMs) / 10 // 指数移动平均
		sr.rdb.HSet(ctx, key, "avg_latency_ms", newAvg)
	}
}
