package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Tenant 租户
type Tenant struct {
	ID         string        `json:"id" gorm:"primaryKey"`
	Name       string        `json:"name"`
	Slug       string        `json:"slug" gorm:"uniqueIndex"`
	Status     TenantStatus  `json:"status"`
	Plan       PlanType      `json:"plan"`
	Region     string        `json:"region"`
	Language   string        `json:"language"`
	Currency   string        `json:"currency"`
	Timezone   string        `json:"timezone"`
	MaxUsers   int           `json:"max_users"`
	MaxAPIKeys int           `json:"max_api_keys"`
	MaxModels  int           `json:"max_models"`
	MaxQPS     int           `json:"max_qps"`
	Isolation  IsolationType `json:"isolation"`
	Config     TenantConfig  `json:"config" gorm:"serializer:json"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	ExpiresAt  *time.Time    `json:"expires_at,omitempty"`
}

type TenantStatus string

const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
	TenantExpired   TenantStatus = "expired"
	TenantTrial     TenantStatus = "trial"
)

type PlanType string

const (
	PlanFree       PlanType = "free"
	PlanStarter    PlanType = "starter"
	PlanPro        PlanType = "pro"
	PlanEnterprise PlanType = "enterprise"
	PlanCustom     PlanType = "custom"
)

type IsolationType string

const (
	IsolationLogical  IsolationType = "logical"
	IsolationSchema   IsolationType = "schema"
	IsolationPhysical IsolationType = "physical"
)

// TenantConfig 租户配置
type TenantConfig struct {
	CustomBranding    bool     `json:"custom_branding"`
	LogoURL          string   `json:"logo_url,omitempty"`
	PrimaryColor     string   `json:"primary_color,omitempty"`
	AllowedProviders []string `json:"allowed_providers,omitempty"`
	RateLimitRPS    int      `json:"rate_limit_rps,omitempty"`
	IPWhitelist     []string `json:"ip_whitelist,omitempty"`
	IPBlacklist     []string `json:"ip_blacklist,omitempty"`
	DataRetention   int      `json:"data_retention_days,omitempty"`
	EnableAudit     bool     `json:"enable_audit"`
	EnableDesensitize bool   `json:"enable_desensitize"`
}

// TenantQuota 租户配额
type TenantQuota struct {
	TenantID           string    `json:"tenant_id" gorm:"primaryKey"`
	ModelName         string    `json:"model_name"`
	DailyTokens       int64     `json:"daily_tokens"`
	MonthlyTokens     int64     `json:"monthly_tokens"`
	UsedDailyTokens   int64     `json:"used_daily_tokens"`
	UsedMonthlyTokens  int64    `json:"used_monthly_tokens"`
	PeriodStart       time.Time `json:"period_start"`
}

// TenantService 租户服务
type TenantService struct {
	logger *zap.Logger
	db     *gorm.DB
}

// NewTenantService 创建租户服务
func NewTenantService(logger *zap.Logger, db *gorm.DB) *TenantService {
	return &TenantService{logger: logger, db: db}
}

// AutoMigrate 自动迁移
func (s *TenantService) AutoMigrate() error {
	return s.db.AutoMigrate(&Tenant{}, &TenantQuota{})
}

// Create 创建租户
func (s *TenantService) Create(ctx context.Context, t *Tenant) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.Status == "" {
		t.Status = TenantActive
	}
	if t.Plan == "" {
		t.Plan = PlanFree
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	return s.db.WithContext(ctx).Create(t).Error
}

// GetByID 根据ID获取租户
func (s *TenantService) GetByID(ctx context.Context, id string) (*Tenant, error) {
	var t Tenant
	if err := s.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetBySlug 根据Slug获取租户
func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var t Tenant
	if err := s.db.WithContext(ctx).First(&t, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// List 列出租户
func (s *TenantService) List(ctx context.Context, status TenantStatus, page, pageSize int) ([]*Tenant, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.WithContext(ctx).Model(&Tenant{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var tenants []*Tenant
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tenants).Error; err != nil {
		return nil, 0, err
	}
	return tenants, total, nil
}

// Update 更新租户
func (s *TenantService) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	result := s.db.WithContext(ctx).Model(&Tenant{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete 删除租户（软删除：设为suspended）
func (s *TenantService) Delete(ctx context.Context, id string) error {
	return s.Update(ctx, id, map[string]interface{}{
		"status": TenantSuspended,
	})
}

// SetQuota 设置租户模型配额
func (s *TenantService) SetQuota(ctx context.Context, quota *TenantQuota) error {
	return s.db.WithContext(ctx).Save(quota).Error
}

// GetQuota 获取租户配额
func (s *TenantService) GetQuota(ctx context.Context, tenantID, modelName string) (*TenantQuota, error) {
	var q TenantQuota
	if err := s.db.WithContext(ctx).First(&q, "tenant_id = ? AND model_name = ?", tenantID, modelName).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// CheckQuota 检查租户配额
func (s *TenantService) CheckQuota(ctx context.Context, tenantID, modelName string, tokens int) (bool, error) {
	quota, err := s.GetQuota(ctx, tenantID, modelName)
	if err != nil {
		// 没有配额记录则不限制
		return true, nil
	}
	if quota.DailyTokens > 0 && quota.UsedDailyTokens+int64(tokens) > quota.DailyTokens {
		return false, nil
	}
	if quota.MonthlyTokens > 0 && quota.UsedMonthlyTokens+int64(tokens) > quota.MonthlyTokens {
		return false, nil
	}
	return true, nil
}

// ResetDailyQuotas 重置每日配额
func (s *TenantService) ResetDailyQuotas(ctx context.Context) error {
	return s.db.WithContext(ctx).Model(&TenantQuota{}).Where("1=1").Update("used_daily_tokens", 0).Error
}

// GetPlanDefaults 获取套餐默认配置
func GetPlanDefaults(plan PlanType) Tenant {
	defaults := map[PlanType]Tenant{
		PlanFree: {
			MaxUsers: 1, MaxAPIKeys: 2, MaxModels: 5, MaxQPS: 10,
			Isolation: IsolationLogical,
			Config: TenantConfig{EnableAudit: false, EnableDesensitize: true},
		},
		PlanStarter: {
			MaxUsers: 5, MaxAPIKeys: 10, MaxModels: 20, MaxQPS: 50,
			Isolation: IsolationLogical,
			Config: TenantConfig{EnableAudit: true, EnableDesensitize: true},
		},
		PlanPro: {
			MaxUsers: 20, MaxAPIKeys: 50, MaxModels: 100, MaxQPS: 200,
			Isolation: IsolationLogical,
			Config: TenantConfig{EnableAudit: true, EnableDesensitize: true},
		},
		PlanEnterprise: {
			MaxUsers: -1, MaxAPIKeys: -1, MaxModels: -1, MaxQPS: -1,
			Isolation: IsolationSchema,
			Config: TenantConfig{EnableAudit: true, EnableDesensitize: true, CustomBranding: true},
		},
	}
	if d, ok := defaults[plan]; ok {
		d.Plan = plan
		return d
	}
	return defaults[PlanFree]
}
