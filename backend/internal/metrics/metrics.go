package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// PrometheusMetrics Prometheus指标采集
type PrometheusMetrics struct {
	logger *zap.Logger
	rdb    *redis.Client
	mu     sync.RWMutex

	// 计数器
	totalRequests   int64
	totalErrors     int64
	totalTokens     int64
	totalAmount     int64

	// 延迟数据（使用滑动窗口，保留最近10000条）
	latencies []int
	maxLatencies int
}

// NewPrometheusMetrics 创建Prometheus指标
func NewPrometheusMetrics(logger *zap.Logger, rdb *redis.Client) *PrometheusMetrics {
	return &PrometheusMetrics{
		logger:       logger,
		rdb:          rdb,
		latencies:    make([]int, 0, 10000),
		maxLatencies: 10000,
	}
}

// RecordRequest 记录请求
func (m *PrometheusMetrics) RecordRequest(ctx context.Context, model, provider, tenantID string, latencyMs int, success bool, tokens int, amount int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	if !success {
		m.totalErrors++
	}
	m.totalTokens += int64(tokens)
	m.totalAmount += amount
	m.latencies = append(m.latencies, latencyMs)
	// 滑动窗口：超过上限则截断
	if len(m.latencies) > m.maxLatencies {
		m.latencies = m.latencies[len(m.latencies)-m.maxLatencies:]
	}

	// 写入Redis时间序列
	minuteKey := time.Now().Format("200601021504")

	// 全局指标
	m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:global:requests:%s", minuteKey), 1)
	m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:global:tokens:%s", minuteKey), int64(tokens))
	m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:global:amount:%s", minuteKey), amount)

	if !success {
		m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:global:errors:%s", minuteKey), 1)
	}

	// 按模型指标
	m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:model:%s:requests:%s", model, minuteKey), 1)
	m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:model:%s:tokens:%s", model, minuteKey), int64(tokens))
	m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:model:%s:latency:%s", model, minuteKey), int64(latencyMs))

	// 按租户指标
	if tenantID != "" {
		m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:tenant:%s:requests:%s", tenantID, minuteKey), 1)
		m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:tenant:%s:tokens:%s", tenantID, minuteKey), int64(tokens))
		m.rdb.IncrBy(ctx, fmt.Sprintf("metrics:tenant:%s:amount:%s", tenantID, minuteKey), amount)
	}
}

// GetMetrics 获取当前指标
func (m *PrometheusMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := 0
	if len(m.latencies) > 0 {
		sum := 0
		for _, l := range m.latencies {
			sum += l
		}
		avgLatency = sum / len(m.latencies)
	}

	p50, p90, p99 := m.percentiles()

	errorRate := 0.0
	if m.totalRequests > 0 {
		errorRate = float64(m.totalErrors) / float64(m.totalRequests) * 100
	}

	return map[string]interface{}{
		"total_requests":  m.totalRequests,
		"total_errors":   m.totalErrors,
		"total_tokens":   m.totalTokens,
		"total_amount":   m.totalAmount,
		"error_rate":     errorRate,
		"avg_latency_ms": avgLatency,
		"p50_latency_ms": p50,
		"p90_latency_ms": p90,
		"p99_latency_ms": p99,
	}
}

// Reset 重置指标
func (m *PrometheusMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalRequests = 0
	m.totalErrors = 0
	m.totalTokens = 0
	m.totalAmount = 0
	m.latencies = m.latencies[:0]
}

// ServeHTTP 暴露 Prometheus 指标端点（简化版，文本格式）
func (m *PrometheusMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metrics := m.GetMetrics()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP tokenhub_requests_total Total API requests\n")
	fmt.Fprintf(w, "# TYPE tokenhub_requests_total counter\n")
	fmt.Fprintf(w, "tokenhub_requests_total %v\n\n", metrics["total_requests"])

	fmt.Fprintf(w, "# HELP tokenhub_errors_total Total API errors\n")
	fmt.Fprintf(w, "# TYPE tokenhub_errors_total counter\n")
	fmt.Fprintf(w, "tokenhub_errors_total %v\n\n", metrics["total_errors"])

	fmt.Fprintf(w, "# HELP tokenhub_tokens_total Total tokens consumed\n")
	fmt.Fprintf(w, "# TYPE tokenhub_tokens_total counter\n")
	fmt.Fprintf(w, "tokenhub_tokens_total %v\n\n", metrics["total_tokens"])

	fmt.Fprintf(w, "# HELP tokenhub_amount_total Total billing amount in cents\n")
	fmt.Fprintf(w, "# TYPE tokenhub_amount_total counter\n")
	fmt.Fprintf(w, "tokenhub_amount_total %v\n\n", metrics["total_amount"])

	fmt.Fprintf(w, "# HELP tokenhub_error_rate Error rate percentage\n")
	fmt.Fprintf(w, "# TYPE tokenhub_error_rate gauge\n")
	fmt.Fprintf(w, "tokenhub_error_rate %.2f\n\n", metrics["error_rate"])

	fmt.Fprintf(w, "# HELP tokenhub_latency_avg Average latency in ms\n")
	fmt.Fprintf(w, "# TYPE tokenhub_latency_avg gauge\n")
	fmt.Fprintf(w, "tokenhub_latency_avg %v\n\n", metrics["avg_latency_ms"])

	fmt.Fprintf(w, "# HELP tokenhub_latency_p50 P50 latency in ms\n")
	fmt.Fprintf(w, "# TYPE tokenhub_latency_p50 gauge\n")
	fmt.Fprintf(w, "tokenhub_latency_p50 %v\n\n", metrics["p50_latency_ms"])

	fmt.Fprintf(w, "# HELP tokenhub_latency_p90 P90 latency in ms\n")
	fmt.Fprintf(w, "# TYPE tokenhub_latency_p90 gauge\n")
	fmt.Fprintf(w, "tokenhub_latency_p90 %v\n\n", metrics["p90_latency_ms"])

	fmt.Fprintf(w, "# HELP tokenhub_latency_p99 P99 latency in ms\n")
	fmt.Fprintf(w, "# TYPE tokenhub_latency_p99 gauge\n")
	fmt.Fprintf(w, "tokenhub_latency_p99 %v\n", metrics["p99_latency_ms"])
}

// percentiles 计算延迟百分位
func (m *PrometheusMetrics) percentiles() (int, int, int) {
	n := len(m.latencies)
	if n == 0 {
		return 0, 0, 0
	}

	sorted := make([]int, n)
	copy(sorted, m.latencies)
	sort.Ints(sorted)

	p50 := sorted[n*50/100]
	p90 := sorted[n*90/100]
	p99Idx := n * 99 / 100
	if p99Idx >= n {
		p99Idx = n - 1
	}
	p99 := sorted[p99Idx]

	return p50, p90, p99
}

// AlertManager 告警管理
type AlertManager struct {
	logger  *zap.Logger
	rdb     *redis.Client
	rules   []AlertRule
	channels []AlertChannel
}

// AlertRule 告警规则
type AlertRule struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Metric     string  `json:"metric"`     // error_rate, latency_p99, balance, success_rate
	Condition  string  `json:"condition"`  // gt, lt, eq
	Threshold  float64 `json:"threshold"`
	Severity   string  `json:"severity"`   // info, warning, critical
	Cooldown   int     `json:"cooldown"`   // 冷却时间（秒）
	Enabled    bool    `json:"enabled"`
}

// AlertChannel 告警通道
type AlertChannel interface {
	Send(ctx context.Context, rule *AlertRule, value float64, message string) error
	Name() string
}

// NewAlertManager 创建告警管理
func NewAlertManager(logger *zap.Logger, rdb *redis.Client) *AlertManager {
	return &AlertManager{
		logger:  logger,
		rdb:     rdb,
		rules:   make([]AlertRule, 0),
		channels: make([]AlertChannel, 0),
	}
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule AlertRule) {
	am.rules = append(am.rules, rule)
}

// AddChannel 添加告警通道
func (am *AlertManager) AddChannel(ch AlertChannel) {
	am.channels = append(am.channels, ch)
}

// Check 检查告警
func (am *AlertManager) Check(ctx context.Context, metrics map[string]interface{}) {
	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		value, ok := metrics[rule.Metric]
		if !ok {
			continue
		}

		var val float64
		switch v := value.(type) {
		case float64:
			val = v
		case int64:
			val = float64(v)
		case int:
			val = float64(v)
		default:
			continue
		}

		triggered := false
		switch rule.Condition {
		case "gt":
			triggered = val > rule.Threshold
		case "lt":
			triggered = val < rule.Threshold
		case "eq":
			triggered = val == rule.Threshold
		}

		if !triggered {
			continue
		}

		// 检查冷却
		cooldownKey := fmt.Sprintf("alert:cooldown:%s", rule.ID)
		exists, _ := am.rdb.Exists(ctx, cooldownKey).Result()
		if exists > 0 {
			continue
		}

		message := fmt.Sprintf("告警 [%s]: %s 当前值 %.2f %s 阈值 %.2f",
			rule.Severity, rule.Name, val, rule.Condition, rule.Threshold)

		am.logger.Warn("alert triggered",
			zap.String("rule", rule.Name),
			zap.Float64("value", val),
			zap.Float64("threshold", rule.Threshold),
		)

		// 发送告警
		for _, ch := range am.channels {
			if err := ch.Send(ctx, &rule, val, message); err != nil {
				am.logger.Error("alert send failed",
					zap.String("channel", ch.Name()),
					zap.Error(err),
				)
			}
		}

		// 设置冷却
		am.rdb.Set(ctx, cooldownKey, 1, time.Duration(rule.Cooldown)*time.Second)
	}
}
