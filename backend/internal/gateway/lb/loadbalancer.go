package lb

import (
	"math/rand"
	"sync"
	"time"

	"github.com/tokenhub/backend/internal/gateway"
)

// Strategy 负载均衡策略
type Strategy string

const (
	StrategyWeightedRandom  Strategy = "weighted_random"
	StrategyRoundRobin      Strategy = "round_robin"
	StrategyLeastLatency    Strategy = "least_latency"
	StrategyLeastCost       Strategy = "least_cost"
	StrategyBestSuccessRate Strategy = "best_success_rate"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error)
	Strategy() Strategy
}

// --- 加权随机 ---

type WeightedRandomLB struct {
	rng *rand.Rand
}

func NewWeightedRandomLB() *WeightedRandomLB {
	return &WeightedRandomLB{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (lb *WeightedRandomLB) Strategy() Strategy { return StrategyWeightedRandom }

func (lb *WeightedRandomLB) Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error) {
	if len(models) == 0 {
		return nil, ErrNoAvailableModels
	}

	totalWeight := 0
	for _, m := range models {
		totalWeight += m.Weight
	}

	r := lb.rng.Intn(totalWeight)
	cumulative := 0
	for _, m := range models {
		cumulative += m.Weight
		if r < cumulative {
			return m, nil
		}
	}

	return models[0], nil
}

// --- 轮询 ---

type RoundRobinLB struct {
	mu    sync.Mutex
	index int
}

func NewRoundRobinLB() *RoundRobinLB {
	return &RoundRobinLB{}
}

func (lb *RoundRobinLB) Strategy() Strategy { return StrategyRoundRobin }

func (lb *RoundRobinLB) Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error) {
	if len(models) == 0 {
		return nil, ErrNoAvailableModels
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	model := models[lb.index%len(models)]
	lb.index++
	return model, nil
}

// --- 最低延迟 ---

type LeastLatencyLB struct{}

func NewLeastLatencyLB() *LeastLatencyLB { return &LeastLatencyLB{} }

func (lb *LeastLatencyLB) Strategy() Strategy { return StrategyLeastLatency }

func (lb *LeastLatencyLB) Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error) {
	if len(models) == 0 {
		return nil, ErrNoAvailableModels
	}

	var best *gateway.ModelConfig
	minLatency := int(^uint(0) >> 1) // MaxInt
	for _, m := range models {
		if m.LatencyMs < minLatency {
			minLatency = m.LatencyMs
			best = m
		}
	}
	return best, nil
}

// --- 最低成本 ---

type LeastCostLB struct{}

func NewLeastCostLB() *LeastCostLB { return &LeastCostLB{} }

func (lb *LeastCostLB) Strategy() Strategy { return StrategyLeastCost }

func (lb *LeastCostLB) Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error) {
	if len(models) == 0 {
		return nil, ErrNoAvailableModels
	}

	var best *gateway.ModelConfig
	minCost := float64(^uint(0) >> 1)
	for _, m := range models {
		cost := m.InputPrice + m.OutputPrice
		if cost < minCost {
			minCost = cost
			best = m
		}
	}
	return best, nil
}

// --- 最高成功率 ---

type BestSuccessRateLB struct{}

func NewBestSuccessRateLB() *BestSuccessRateLB { return &BestSuccessRateLB{} }

func (lb *BestSuccessRateLB) Strategy() Strategy { return StrategyBestSuccessRate }

func (lb *BestSuccessRateLB) Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error) {
	if len(models) == 0 {
		return nil, ErrNoAvailableModels
	}

	var best *gateway.ModelConfig
	maxRate := float64(-1)
	for _, m := range models {
		if m.SuccessRate > maxRate {
			maxRate = m.SuccessRate
			best = m
		}
	}
	return best, nil
}

// --- 综合评分 ---

type CompositeScoreLB struct {
	LatencyWeight float64 // 延迟权重
	CostWeight    float64 // 成本权重
	RateWeight    float64 // 成功率权重
}

func NewCompositeScoreLB() *CompositeScoreLB {
	return &CompositeScoreLB{
		LatencyWeight: 0.4,
		CostWeight:    0.3,
		RateWeight:    0.3,
	}
}

func (lb *CompositeScoreLB) Strategy() Strategy { return "composite_score" }

func (lb *CompositeScoreLB) Select(models []*gateway.ModelConfig) (*gateway.ModelConfig, error) {
	if len(models) == 0 {
		return nil, ErrNoAvailableModels
	}

	var best *gateway.ModelConfig
	bestScore := float64(-1)

	maxLatency := 0.0
	maxCost := 0.0
	for _, m := range models {
		if float64(m.LatencyMs) > maxLatency {
			maxLatency = float64(m.LatencyMs)
		}
		cost := m.InputPrice + m.OutputPrice
		if cost > maxCost {
			maxCost = cost
		}
	}

	for _, m := range models {
		latencyScore := 1.0
		costScore := 1.0
		if maxLatency > 0 {
			latencyScore = 1.0 - float64(m.LatencyMs)/maxLatency
		}
		if maxCost > 0 {
			costScore = 1.0 - (m.InputPrice+m.OutputPrice)/maxCost
		}

		score := lb.LatencyWeight*latencyScore +
			lb.CostWeight*costScore +
			lb.RateWeight*m.SuccessRate

		if score > bestScore {
			bestScore = score
			best = m
		}
	}

	return best, nil
}

// NewLoadBalancer 根据策略创建负载均衡器
func NewLoadBalancer(strategy Strategy) LoadBalancer {
	switch strategy {
	case StrategyWeightedRandom:
		return NewWeightedRandomLB()
	case StrategyRoundRobin:
		return NewRoundRobinLB()
	case StrategyLeastLatency:
		return NewLeastLatencyLB()
	case StrategyLeastCost:
		return NewLeastCostLB()
	case StrategyBestSuccessRate:
		return NewBestSuccessRateLB()
	default:
		return NewWeightedRandomLB()
	}
}
