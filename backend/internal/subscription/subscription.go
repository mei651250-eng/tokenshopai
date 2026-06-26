package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PlanType 订阅类型
type PlanType string

const (
	PlanTrial    PlanType = "trial"     // 试用（3天）
	PlanMonthly  PlanType = "monthly"   // 月订阅
	PlanQuarterly PlanType = "quarterly" // 季订阅
	PlanAnnual   PlanType = "annual"    // 年订阅
	PlanPAYG     PlanType = "payg"      // 按量付费
)

// PlanStatus 计划状态
type PlanStatus string

const (
	PlanStatusActive   PlanStatus = "active"
	PlanStatusInactive PlanStatus = "inactive"
)

// SubscriptionPlan 订阅计划定义（GORM 模型）
type SubscriptionPlan struct {
	ID          string     `json:"id" gorm:"primaryKey;size:36"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	Type        PlanType   `json:"type" gorm:"size:20;not null;index"`
	Description string     `json:"description" gorm:"size:512"`
	SortOrder   int        `json:"sort_order" gorm:"default:0"`

	// 周期与价格
	DurationDays int     `json:"duration_days" gorm:"default:0"` // 有效天数，PAYG 为 0
	Price        float64 `json:"price"`                           // 价格（元），PAYG 为 0
	Currency     string  `json:"currency" gorm:"default:CNY"`

	// 配额限制
	TokenLimit    int64   `json:"token_limit"`     // 总 Token 限额，0=无限
	RequestLimit  int64   `json:"request_limit"`   // 总请求次数限额，0=无限
	DailyReqLimit int64   `json:"daily_req_limit"` // 每日请求限额，0=无限
	MaxModels     int     `json:"max_models"`      // 可访问模型数，0=无限
	ModelList     string  `json:"model_list" gorm:"type:text"` // 允许的模型列表（JSON 数组）

	// 按量付费价格（仅 PAYG 类型有效）
	InputTokenPrice  float64 `json:"input_token_price"`  // 输入 Token 单价（元/千Token）
	OutputTokenPrice float64 `json:"output_token_price"` // 输出 Token 单价（元/千Token）

	// 功能特性
	Features string `json:"features" gorm:"type:text"` // JSON: ["api_access","priority_support",...]

	Status    PlanStatus `json:"status" gorm:"size:20;default:active;index"`
	IsDefault bool       `json:"is_default" gorm:"default:false"` // 是否为默认套餐

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// SubStatus 用户订阅状态
type SubStatus string

const (
	SubStatusActive    SubStatus = "active"    // 使用中
	SubStatusExpired   SubStatus = "expired"   // 已过期
	SubStatusCancelled SubStatus = "cancelled" // 已取消
	SubStatusSuspended SubStatus = "suspended" // 已暂停
)

// UserSubscription 用户订阅记录（GORM 模型）
type UserSubscription struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	UserID    string    `json:"user_id" gorm:"size:36;index"`
	TenantID  string    `json:"tenant_id" gorm:"size:36;index"`
	PlanID    string    `json:"plan_id" gorm:"size:36"`
	PlanName  string    `json:"plan_name" gorm:"size:128"`
	PlanType  PlanType  `json:"plan_type" gorm:"size:20"`

	Status    SubStatus `json:"status" gorm:"size:20;index"`
	AutoRenew bool      `json:"auto_renew" gorm:"default:false"`

	// 用量追踪
	TokenUsed   int64 `json:"token_used" gorm:"default:0"`   // 已用 Token
	RequestUsed int64 `json:"request_used" gorm:"default:0"` // 已用请求数

	// 时间
	StartAt  time.Time  `json:"start_at"`
	EndAt    time.Time  `json:"end_at"`
	TrialEnd *time.Time `json:"trial_end,omitempty"` // 试用结束时间

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SubscriptionService 订阅服务
type SubscriptionService struct {
	logger *zap.Logger
	db     *gorm.DB
	rdb    *redis.Client
}

// NewSubscriptionService 创建订阅服务
func NewSubscriptionService(logger *zap.Logger, db *gorm.DB, rdb *redis.Client) *SubscriptionService {
	return &SubscriptionService{logger: logger, db: db, rdb: rdb}
}

// ==================== 计划管理（Admin） ====================

// CreatePlan 创建订阅计划
func (s *SubscriptionService) CreatePlan(ctx context.Context, plan *SubscriptionPlan) error {
	plan.ID = uuid.New().String()
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = plan.CreatedAt
	if plan.Currency == "" {
		plan.Currency = "CNY"
	}
	return s.db.WithContext(ctx).Create(plan).Error
}

// UpdatePlan 更新订阅计划
func (s *SubscriptionService) UpdatePlan(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return s.db.WithContext(ctx).Model(&SubscriptionPlan{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePlan 删除订阅计划（软删除）
func (s *SubscriptionService) DeletePlan(ctx context.Context, id string) error {
	// 检查是否有用户正在使用
	var count int64
	s.db.WithContext(ctx).Model(&UserSubscription{}).Where("plan_id = ? AND status = ?", id, SubStatusActive).Count(&count)
	if count > 0 {
		return fmt.Errorf("cannot delete: %d active subscribers", count)
	}
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&SubscriptionPlan{}).Error
}

// GetPlan 获取单个计划
func (s *SubscriptionService) GetPlan(ctx context.Context, id string) (*SubscriptionPlan, error) {
	var plan SubscriptionPlan
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&plan).Error
	return &plan, err
}

// ListPlans 列出所有启用的计划
func (s *SubscriptionService) ListPlans(ctx context.Context, planType string) ([]SubscriptionPlan, error) {
	var plans []SubscriptionPlan
	query := s.db.WithContext(ctx).Where("status = ?", PlanStatusActive)
	if planType != "" && planType != "all" {
		query = query.Where("type = ?", planType)
	}
	err := query.Order("sort_order ASC, type, price ASC").Find(&plans).Error
	return plans, err
}

// ListAllPlans 列出所有计划（含禁用）
func (s *SubscriptionService) ListAllPlans(ctx context.Context) ([]SubscriptionPlan, error) {
	var plans []SubscriptionPlan
	err := s.db.WithContext(ctx).Order("sort_order ASC, type, price ASC").Find(&plans).Error
	return plans, err
}

// SeedDefaultPlans 初始化默认计划
func (s *SubscriptionService) SeedDefaultPlans(ctx context.Context) error {
	var count int64
	s.db.WithContext(ctx).Model(&SubscriptionPlan{}).Count(&count)
	if count > 0 {
		return nil // 已有计划，不重复初始化
	}

	plans := []SubscriptionPlan{
		{
			Name:         "3天免费试用",
			Type:         PlanTrial,
			Description:  "新用户3天免费试用，赠送1元额度(1000Token)",
			SortOrder:    1,
			DurationDays: 3,
			Price:        0,
			TokenLimit:   1000,
			RequestLimit: 200,
			DailyReqLimit: 100,
			MaxModels:    0,
			Status:       PlanStatusActive,
			IsDefault:    true,
		},
		{
			Name:         "月度订阅",
			Type:         PlanMonthly,
			Description:  "按月订阅，享受稳定服务",
			SortOrder:    2,
			DurationDays: 30,
			Price:        29.9,
			TokenLimit:   5000000,
			RequestLimit: 0,
			DailyReqLimit: 500,
			MaxModels:    0,
			Status:       PlanStatusActive,
		},
		{
			Name:         "季度订阅",
			Type:         PlanQuarterly,
			Description:  "季度订阅享8折优惠",
			SortOrder:    3,
			DurationDays: 90,
			Price:        71.76,
			TokenLimit:   20000000,
			RequestLimit: 0,
			DailyReqLimit: 1000,
			MaxModels:    0,
			Status:       PlanStatusActive,
		},
		{
			Name:         "年度订阅",
			Type:         PlanAnnual,
			Description:  "年度订阅享6折，最超值",
			SortOrder:    4,
			DurationDays: 365,
			Price:        215.28,
			TokenLimit:   100000000,
			RequestLimit: 0,
			DailyReqLimit: 2000,
			MaxModels:    0,
			Status:       PlanStatusActive,
		},
		{
			Name:             "按量付费",
			Type:             PlanPAYG,
			Description:      "随用随付，灵活控制成本",
			SortOrder:        5,
			DurationDays:     0,
			Price:            0,
			TokenLimit:       0,
			RequestLimit:     0,
			DailyReqLimit:    0,
			MaxModels:        0,
			InputTokenPrice:  0.01, // 输入 ¥0.01/千Token
			OutputTokenPrice: 0.03, // 输出 ¥0.03/千Token
			Status:           PlanStatusActive,
		},
	}

	for i := range plans {
		plans[i].ID = uuid.New().String()
		now := time.Now()
		plans[i].CreatedAt = now
		plans[i].UpdatedAt = now
		if plans[i].Currency == "" {
			plans[i].Currency = "CNY"
		}
	}

	return s.db.WithContext(ctx).Create(&plans).Error
}

// GetDefaultPlan 获取默认计划（一般为试用）
func (s *SubscriptionService) GetDefaultPlan(ctx context.Context, planType PlanType) (*SubscriptionPlan, error) {
	var plan SubscriptionPlan
	err := s.db.WithContext(ctx).
		Where("type = ? AND status = ? AND is_default = ?", planType, PlanStatusActive, true).
		First(&plan).Error
	return &plan, err
}

// ==================== 用户订阅管理 ====================

// Subscribe 用户订阅计划
func (s *SubscriptionService) Subscribe(ctx context.Context, userID, tenantID, planID string) (*UserSubscription, error) {
	plan, err := s.GetPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
	if plan.Status != PlanStatusActive {
		return nil, fmt.Errorf("plan is not available")
	}

	now := time.Now()

	// 检查试用是否已用过
	if plan.Type == PlanTrial {
		var trialCount int64
		s.db.WithContext(ctx).Model(&UserSubscription{}).
			Where("user_id = ? AND plan_type = ?", userID, PlanTrial).Count(&trialCount)
		if trialCount > 0 {
			return nil, fmt.Errorf("trial already used")
		}
	}

	// 取消当前活跃订阅
	s.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("user_id = ? AND status = ?", userID, SubStatusActive).
		Updates(map[string]interface{}{
			"status":     SubStatusCancelled,
			"updated_at": now,
		})

	sub := &UserSubscription{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		PlanID:    planID,
		PlanName:  plan.Name,
		PlanType:  plan.Type,
		Status:    SubStatusActive,
		AutoRenew: plan.Type != PlanTrial,
		StartAt:   now,
		EndAt:     now.AddDate(0, 0, plan.DurationDays),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if plan.DurationDays == 0 {
		// PAYG 无过期时间
		sub.EndAt = time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	}

	return sub, s.db.WithContext(ctx).Create(sub).Error
}

// GetUserSubscription 获取用户当前活跃订阅
func (s *SubscriptionService) GetUserSubscription(ctx context.Context, userID string) (*UserSubscription, error) {
	var sub UserSubscription
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, SubStatusActive).
		Order("created_at DESC").
		First(&sub).Error
	return &sub, err
}

// GetUserSubscriptionPlan 获取用户当前订阅及对应计划
func (s *SubscriptionService) GetUserSubscriptionPlan(ctx context.Context, userID string) (*UserSubscription, *SubscriptionPlan, error) {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("no active subscription")
	}
	plan, err := s.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return sub, nil, err
	}
	return sub, plan, nil
}

// CheckSubscriptionActive 检查订阅是否有效
func (s *SubscriptionService) CheckSubscriptionActive(ctx context.Context, userID string) (bool, *UserSubscription) {
	sub, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, nil
	}

	// 检查是否过期
	if time.Now().After(sub.EndAt) {
		s.db.WithContext(ctx).Model(&UserSubscription{}).
			Where("id = ?", sub.ID).
			Updates(map[string]interface{}{
				"status":     SubStatusExpired,
				"updated_at": time.Now(),
			})
		return false, sub
	}

	return true, sub
}

// CheckTokenQuota 检查 Token 配额是否足够（返回剩余量）
func (s *SubscriptionService) CheckTokenQuota(ctx context.Context, sub *UserSubscription, plan *SubscriptionPlan, needed int64) (bool, int64) {
	if plan.TokenLimit == 0 {
		return true, -1 // 无限配额
	}
	remaining := plan.TokenLimit - sub.TokenUsed - needed
	return remaining >= 0, remaining
}

// IncrementUsage 增加用量
func (s *SubscriptionService) IncrementUsage(ctx context.Context, subID string, tokens, requests int64) error {
	return s.db.WithContext(ctx).Model(&UserSubscription{}).Where("id = ?", subID).Updates(map[string]interface{}{
		"token_used":   gorm.Expr("token_used + ?", tokens),
		"request_used": gorm.Expr("request_used + ?", requests),
		"updated_at":   time.Now(),
	}).Error
}

// RenewSubscription 续费
func (s *SubscriptionService) RenewSubscription(ctx context.Context, subID string) error {
	var sub UserSubscription
	if err := s.db.WithContext(ctx).Where("id = ?", subID).First(&sub).Error; err != nil {
		return err
	}

	plan, err := s.GetPlan(ctx, sub.PlanID)
	if err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":     SubStatusActive,
		"start_at":   now,
		"end_at":     now.AddDate(0, 0, plan.DurationDays),
		"updated_at": now,
	}

	// 重置用量追踪（可选：是否累积）
	// 如果累积则注释下面两行
	// updates["token_used"] = 0
	// updates["request_used"] = 0

	return s.db.WithContext(ctx).Model(&UserSubscription{}).Where("id = ?", subID).Updates(updates).Error
}

// CancelSubscription 取消订阅
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("user_id = ? AND status = ?", userID, SubStatusActive).
		Updates(map[string]interface{}{
			"status":     SubStatusCancelled,
			"updated_at": time.Now(),
		}).Error
}

// ListUserSubscriptions 列出用户订阅历史
func (s *SubscriptionService) ListUserSubscriptions(ctx context.Context, userID string, limit int) ([]UserSubscription, error) {
	var subs []UserSubscription
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&subs).Error
	return subs, err
}

// ==================== 定时任务 ====================

// ExpireSubscriptions 定时过期检查（配合 cron）
func (s *SubscriptionService) ExpireSubscriptions(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("status = ? AND end_at < ?", SubStatusActive, time.Now()).
		Updates(map[string]interface{}{
			"status":     SubStatusExpired,
			"updated_at": time.Now(),
		})
	return result.RowsAffected, result.Error
}

// ==================== 统计分析 ====================

// GetSubscriptionStats 获取订阅统计
func (s *SubscriptionService) GetSubscriptionStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var activeCount int64
	s.db.WithContext(ctx).Model(&UserSubscription{}).Where("status = ?", SubStatusActive).Count(&activeCount)
	stats["active_subscribers"] = activeCount

	var trialCount int64
	s.db.WithContext(ctx).Model(&UserSubscription{}).Where("plan_type = ? AND status = ?", PlanTrial, SubStatusActive).Count(&trialCount)
	stats["trial_users"] = trialCount

	var paidCount int64
	s.db.WithContext(ctx).Model(&UserSubscription{}).Where("status = ? AND plan_type != ?", SubStatusActive, PlanTrial).Count(&paidCount)
	stats["paid_subscribers"] = paidCount

	return stats, nil
}

// ==================== JSON 辅助 ====================

// GetFeatures 解析功能特性 JSON
func (p *SubscriptionPlan) GetFeatures() []string {
	if p.Features == "" {
		return nil
	}
	var features []string
	json.Unmarshal([]byte(p.Features), &features)
	return features
}

// SetFeatures 设置功能特性
func (p *SubscriptionPlan) SetFeatures(features []string) {
	data, _ := json.Marshal(features)
	p.Features = string(data)
}

// GetModelList 获取模型列表
func (p *SubscriptionPlan) GetModelList() []string {
	if p.ModelList == "" {
		return nil
	}
	var models []string
	json.Unmarshal([]byte(p.ModelList), &models)
	return models
}
