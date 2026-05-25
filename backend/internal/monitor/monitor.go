package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MetricsData 监控指标
type MetricsData struct {
	Timestamp     int64   `json:"timestamp"`
	QPS           float64 `json:"qps"`
	ActiveConns   int64   `json:"active_connections"`
	TotalRequests int64   `json:"total_requests"`
	SuccessRate   float64 `json:"success_rate"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P90LatencyMs  float64 `json:"p90_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	InputTokens   int64   `json:"input_tokens"`
	OutputTokens  int64   `json:"output_tokens"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalAmount   int64   `json:"total_amount"`  // 总费用（分）
	Models        []ModelMetrics `json:"models"`
	Tenants       []TenantMetrics `json:"tenants"`
}

// ModelMetrics 模型级指标
type ModelMetrics struct {
	ModelID      string  `json:"model_id"`
	ModelName    string  `json:"model_name"`
	Provider     string  `json:"provider"`
	Requests     int64   `json:"requests"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Tokens       int64   `json:"tokens"`
	Errors       int64   `json:"errors"`
	CircuitState string  `json:"circuit_state"`
}

// TenantMetrics 租户级指标
type TenantMetrics struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Requests   int64  `json:"requests"`
	Tokens     int64  `json:"tokens"`
	Amount     int64  `json:"amount"`
	QPS        float64 `json:"qps"`
}

// MonitorService 监控服务
type MonitorService struct {
	logger   *zap.Logger
	mu       sync.RWMutex
	metrics  *MetricsData
	clients  map[*websocket.Conn]bool
	broadcast chan *MetricsData
	stopCh   chan struct{}
}

// NewMonitorService 创建监控服务
func NewMonitorService(logger *zap.Logger) *MonitorService {
	m := &MonitorService{
		logger:   logger,
		metrics:  &MetricsData{Models: []ModelMetrics{}, Tenants: []TenantMetrics{}},
		clients:  make(map[*websocket.Conn]bool),
		broadcast: make(chan *MetricsData, 100),
		stopCh:   make(chan struct{}),
	}
	go m.broadcastLoop()
	return m
}

// RecordRequest 记录请求指标
func (m *MonitorService) RecordRequest(modelID, modelName, provider, tenantID string, latencyMs int, success bool, tokens int, amount int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalRequests++
	m.metrics.TotalTokens += int64(tokens)
	m.metrics.TotalAmount += amount
	m.metrics.Timestamp = time.Now().UnixMilli()

	// 更新延迟统计（简化版，生产环境应使用HDR Histogram）
	if success {
		m.metrics.SuccessRate = float64(m.metrics.SuccessRate*float64(m.metrics.TotalRequests-1)+1) / float64(m.metrics.TotalRequests)
	} else {
		m.metrics.SuccessRate = float64(m.metrics.SuccessRate*float64(m.metrics.TotalRequests-1)+0) / float64(m.metrics.TotalRequests)
	}

	// 更新模型指标
	found := false
	for i, model := range m.metrics.Models {
		if model.ModelID == modelID {
			m.metrics.Models[i].Requests++
			m.metrics.Models[i].Tokens += int64(tokens)
			m.metrics.Models[i].AvgLatencyMs = (model.AvgLatencyMs*float64(model.Requests-1) + float64(latencyMs)) / float64(model.Requests)
			if !success {
				m.metrics.Models[i].Errors++
			}
			found = true
			break
		}
	}
	if !found {
		m.metrics.Models = append(m.metrics.Models, ModelMetrics{
			ModelID:      modelID,
			ModelName:    modelName,
			Provider:     provider,
			Requests:     1,
			Tokens:       int64(tokens),
			AvgLatencyMs: float64(latencyMs),
		})
	}
}

// GetMetrics 获取当前指标
func (m *MonitorService) GetMetrics() *MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 深拷贝
	cp := *m.metrics
	cp.Models = make([]ModelMetrics, len(m.metrics.Models))
	copy(cp.Models, m.metrics.Models)
	cp.Tenants = make([]TenantMetrics, len(m.metrics.Tenants))
	copy(cp.Tenants, m.metrics.Tenants)
	return &cp
}

// RegisterClient 注册WebSocket客户端（大屏）
func (m *MonitorService) RegisterClient(conn *websocket.Conn) {
	m.mu.Lock()
	m.clients[conn] = true
	m.mu.Unlock()
}

// UnregisterClient 注销WebSocket客户端
func (m *MonitorService) UnregisterClient(conn *websocket.Conn) {
	m.mu.Lock()
	delete(m.clients, conn)
	m.mu.Unlock()
	conn.Close()
}

// broadcastLoop 广播循环
func (m *MonitorService) broadcastLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := m.GetMetrics()
			m.mu.RLock()
			for client := range m.clients {
				err := client.WriteJSON(metrics)
				if err != nil {
					m.mu.RUnlock()
					m.UnregisterClient(client)
					m.mu.RLock()
				}
			}
			m.mu.RUnlock()
		case <-m.stopCh:
			return
		}
	}
}

// Stop 停止监控服务
func (m *MonitorService) Stop() {
	close(m.stopCh)
}

// Alert 报警
type Alert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`    // error_rate, latency, balance, circuit
	Severity  string    `json:"severity"` // info, warning, critical
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	ModelID   string    `json:"model_id,omitempty"`
	TenantID  string    `json:"tenant_id,omitempty"`
	Metric    string    `json:"metric,omitempty"`
	Value     float64   `json:"value,omitempty"`
	Threshold float64   `json:"threshold,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AlertChannel 报警通道
type AlertChannel interface {
	Send(ctx context.Context, alert *Alert) error
	Name() string
}

// DingTalkChannel 钉钉报警通道
type DingTalkChannel struct {
	webhookURL string
}

func NewDingTalkChannel(webhookURL string) *DingTalkChannel {
	return &DingTalkChannel{webhookURL: webhookURL}
}

func (c *DingTalkChannel) Name() string { return "dingtalk" }

func (c *DingTalkChannel) Send(ctx context.Context, alert *Alert) error {
	// 实际实现：发送钉钉机器人消息
	return nil
}

// EmailChannel 邮件报警通道
type EmailChannel struct {
	smtpHost string
	smtpPort int
	from     string
	to       []string
}

func NewEmailChannel(host string, port int, from string, to []string) *EmailChannel {
	return &EmailChannel{smtpHost: host, smtpPort: port, from: from, to: to}
}

func (c *EmailChannel) Name() string { return "email" }

func (c *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	// 实际实现：发送邮件
	return nil
}
