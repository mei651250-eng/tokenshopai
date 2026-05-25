package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QuotaType 配额类型
type QuotaType string

const (
	QuotaDailyTokens   QuotaType = "daily_tokens"
	QuotaMonthlyTokens QuotaType = "monthly_tokens"
	QuotaDailyRequests QuotaType = "daily_requests"
	QuotaMonthlyAmount QuotaType = "monthly_amount" // 月消费上限（分）
)

// QuotaStatus 配额状态
type QuotaStatus struct {
	TenantID    string `json:"tenant_id"`
	QuotaType   QuotaType `json:"quota_type"`
	Limit       int64  `json:"limit"`
	Used        int64  `json:"used"`
	Remaining   int64  `json:"remaining"`
	Percentage  float64 `json:"percentage"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	IsExceeded  bool   `json:"is_exceeded"`
}

// QuotaConfig 配额配置
type QuotaConfig struct {
	TenantID    string    `json:"tenant_id"`
	QuotaType   QuotaType `json:"quota_type"`
	Limit       int64     `json:"limit"`
	PeriodDays  int       `json:"period_days"` // 周期天数
	AlertAt     float64   `json:"alert_at"`    // 告警阈值百分比，如 0.8 = 80%
	BlockAt     float64   `json:"block_at"`    // 阻断阈值百分比，如 1.0 = 100%
	CreatedAt   int64     `json:"created_at"`
	UpdatedAt   int64     `json:"updated_at"`
}

// QuotaAlert 配额告警
type QuotaAlert struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	QuotaType  QuotaType `json:"quota_type"`
	AlertLevel string    `json:"alert_level"` // warning, critical, blocked
	Usage      int64     `json:"usage"`
	Limit      int64     `json:"limit"`
	Message    string    `json:"message"`
	CreatedAt  int64     `json:"created_at"`
}

// QuotaService 配额服务
type QuotaService struct {
	logger *zap.Logger
	rdb    *redis.Client
	alerts []AlertHandler
}

// AlertHandler 告警处理器
type AlertHandler interface {
	HandleQuotaAlert(ctx context.Context, alert *QuotaAlert) error
	Name() string
}

// NewQuotaService 创建配额服务
func NewQuotaService(logger *zap.Logger, rdb *redis.Client) *QuotaService {
	return &QuotaService{
		logger: logger,
		rdb:    rdb,
		alerts: []AlertHandler{},
	}
}

// RegisterAlertHandler 注册告警处理器
func (s *QuotaService) RegisterAlertHandler(handler AlertHandler) {
	s.alerts = append(s.alerts, handler)
}

// SetQuota 设置租户配额
func (s *QuotaService) SetQuota(ctx context.Context, config *QuotaConfig) error {
	config.UpdatedAt = time.Now().Unix()
	if config.CreatedAt == 0 {
		config.CreatedAt = config.UpdatedAt
	}

	key := fmt.Sprintf("quota:config:%s:%s", config.TenantID, config.QuotaType)
	return s.rdb.HSet(ctx, key, map[string]interface{}{
		"tenant_id":    config.TenantID,
		"quota_type":   string(config.QuotaType),
		"limit":        config.Limit,
		"period_days":  config.PeriodDays,
		"alert_at":     config.AlertAt,
		"block_at":     config.BlockAt,
		"created_at":   config.CreatedAt,
		"updated_at":   config.UpdatedAt,
	}).Err()
}

// GetQuotaConfig 获取配额配置
func (s *QuotaService) GetQuotaConfig(ctx context.Context, tenantID string, quotaType QuotaType) (*QuotaConfig, error) {
	key := fmt.Sprintf("quota:config:%s:%s", tenantID, quotaType)
	data, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil // 无配额限制
	}

	var limit int64
	var periodDays int
	var alertAt, blockAt float64
	fmt.Sscanf(data["limit"], "%d", &limit)
	fmt.Sscanf(data["period_days"], "%d", &periodDays)
	fmt.Sscanf(data["alert_at"], "%f", &alertAt)
	fmt.Sscanf(data["block_at"], "%f", &blockAt)

	return &QuotaConfig{
		TenantID:   tenantID,
		QuotaType:  quotaType,
		Limit:      limit,
		PeriodDays: periodDays,
		AlertAt:    alertAt,
		BlockAt:    blockAt,
	}, nil
}

// CheckAndConsume 检查并消费配额（原子操作）
// 返回是否允许、当前状态
func (s *QuotaService) CheckAndConsume(ctx context.Context, tenantID string, quotaType QuotaType, amount int64) (bool, *QuotaStatus, error) {
	// 获取配额配置
	config, err := s.GetQuotaConfig(ctx, tenantID, quotaType)
	if err != nil {
		return false, nil, err
	}
	if config == nil || config.Limit <= 0 {
		// 无配额限制，允许
		return true, nil, nil
	}

	// 计算当前周期
	now := time.Now()
	periodStart := now.Truncate(time.Duration(config.PeriodDays) * 24 * time.Hour)
	usageKey := fmt.Sprintf("quota:usage:%s:%s:%d", tenantID, quotaType, periodStart.Unix())

	// 原子递增
	newUsage, err := s.rdb.IncrBy(ctx, usageKey, amount).Result()
	if err != nil {
		return false, nil, fmt.Errorf("increment quota usage: %w", err)
	}

	// 首次使用设置过期时间
	if newUsage == amount {
		expiry := time.Duration(config.PeriodDays) * 24 * time.Hour
		s.rdb.Expire(ctx, usageKey, expiry)
	}

	percentage := float64(newUsage) / float64(config.Limit)
	isExceeded := newUsage > config.Limit

	status := &QuotaStatus{
		TenantID:    tenantID,
		QuotaType:   quotaType,
		Limit:       config.Limit,
		Used:        newUsage,
		Remaining:   max(0, config.Limit-newUsage),
		Percentage:  percentage,
		PeriodStart: periodStart.Unix(),
		PeriodEnd:   periodStart.Add(time.Duration(config.PeriodDays) * 24 * time.Hour).Unix(),
		IsExceeded:  isExceeded,
	}

	// 超出阻断阈值，回滚并拒绝
	if isExceeded && config.BlockAt <= 1.0 {
		s.rdb.DecrBy(ctx, usageKey, amount)
		status.Remaining = 0

		s.fireAlert(ctx, &QuotaAlert{
			ID:         uuid.New().String(),
			TenantID:   tenantID,
			QuotaType:  quotaType,
			AlertLevel: "blocked",
			Usage:      newUsage,
			Limit:      config.Limit,
			Message:    fmt.Sprintf("配额已用尽: %s 使用 %d/%d", quotaType, newUsage, config.Limit),
			CreatedAt:  now.Unix(),
		})
		return false, status, nil
	}

	// 超出告警阈值，发送告警但仍允许
	if percentage >= config.AlertAt {
		s.fireAlert(ctx, &QuotaAlert{
			ID:         uuid.New().String(),
			TenantID:   tenantID,
			QuotaType:  quotaType,
			AlertLevel: "warning",
			Usage:      newUsage,
			Limit:      config.Limit,
			Message:    fmt.Sprintf("配额使用率 %.1f%%: %s 使用 %d/%d", percentage*100, quotaType, newUsage, config.Limit),
			CreatedAt:  now.Unix(),
		})
	}

	return true, status, nil
}

// GetQuotaStatus 获取配额状态
func (s *QuotaService) GetQuotaStatus(ctx context.Context, tenantID string, quotaType QuotaType) (*QuotaStatus, error) {
	config, err := s.GetQuotaConfig(ctx, tenantID, quotaType)
	if err != nil {
		return nil, err
	}
	if config == nil || config.Limit <= 0 {
		return &QuotaStatus{TenantID: tenantID, QuotaType: quotaType}, nil
	}

	now := time.Now()
	periodStart := now.Truncate(time.Duration(config.PeriodDays) * 24 * time.Hour)
	usageKey := fmt.Sprintf("quota:usage:%s:%s:%d", tenantID, quotaType, periodStart.Unix())

	used, _ := s.rdb.Get(ctx, usageKey).Int64()
	percentage := float64(used) / float64(config.Limit)

	return &QuotaStatus{
		TenantID:    tenantID,
		QuotaType:   quotaType,
		Limit:       config.Limit,
		Used:        used,
		Remaining:   max(0, config.Limit-used),
		Percentage:  percentage,
		PeriodStart: periodStart.Unix(),
		PeriodEnd:   periodStart.Add(time.Duration(config.PeriodDays) * 24 * time.Hour).Unix(),
		IsExceeded:  used > config.Limit,
	}, nil
}

// GetAllQuotaStatus 获取租户所有配额状态
func (s *QuotaService) GetAllQuotaStatus(ctx context.Context, tenantID string) ([]*QuotaStatus, error) {
	quotaTypes := []QuotaType{QuotaDailyTokens, QuotaMonthlyTokens, QuotaDailyRequests, QuotaMonthlyAmount}
	var statuses []*QuotaStatus
	for _, qt := range quotaTypes {
		status, err := s.GetQuotaStatus(ctx, tenantID, qt)
		if err != nil {
			return nil, err
		}
		if status != nil {
			statuses = append(statuses, status)
		}
	}
	return statuses, nil
}

// ResetQuota 重置配额（管理员操作）
func (s *QuotaService) ResetQuota(ctx context.Context, tenantID string, quotaType QuotaType) error {
	config, err := s.GetQuotaConfig(ctx, tenantID, quotaType)
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("quota config not found for tenant %s type %s", tenantID, quotaType)
	}

	now := time.Now()
	periodStart := now.Truncate(time.Duration(config.PeriodDays) * 24 * time.Hour)
	usageKey := fmt.Sprintf("quota:usage:%s:%s:%d", tenantID, quotaType, periodStart.Unix())

	return s.rdb.Del(ctx, usageKey).Err()
}

// fireAlert 触发告警
func (s *QuotaService) fireAlert(ctx context.Context, alert *QuotaAlert) {
	s.logger.Warn("quota alert",
		zap.String("tenant_id", alert.TenantID),
		zap.String("type", string(alert.QuotaType)),
		zap.String("level", alert.AlertLevel),
		zap.String("message", alert.Message),
	)

	for _, handler := range s.alerts {
		if err := handler.HandleQuotaAlert(ctx, alert); err != nil {
			s.logger.Error("alert handler failed",
				zap.String("handler", handler.Name()),
				zap.Error(err),
			)
		}
	}
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
