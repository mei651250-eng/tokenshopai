package tenant

import (
	"time"
)

// Tenant 租户
type Tenant struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug" gorm:"uniqueIndex"`
	Status      TenantStatus `json:"status"`
	Plan        PlanType  `json:"plan"`
	Region      string    `json:"region"`        // 所属区域 cn, us, eu, ap
	Language    string    `json:"language"`      // 默认语言
	Currency    string    `json:"currency"`      // 默认货币
	Timezone    string    `json:"timezone"`       // 时区
	MaxUsers    int       `json:"max_users"`
	MaxAPIKeys  int       `json:"max_api_keys"`
	MaxModels   int       `json:"max_models"`
	MaxQPS      int       `json:"max_qps"`
	Isolation   IsolationType `json:"isolation"`  // 隔离级别
	Config      TenantConfig `json:"config" gorm:"serializer:json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type TenantStatus string

const (
	TenantActive   TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
	TenantExpired  TenantStatus = "expired"
	TenantTrial    TenantStatus = "trial"
)

type PlanType string

const (
	PlanFree      PlanType = "free"
	PlanStarter   PlanType = "starter"
	PlanPro       PlanType = "pro"
	PlanEnterprise PlanType = "enterprise"
	PlanCustom    PlanType = "custom"
)

type IsolationType string

const (
	IsolationLogical IsolationType = "logical"  // 逻辑隔离（共享数据库，tenant_id区分）
	IsolationSchema  IsolationType = "schema"   // Schema隔离（同库不同Schema）
	IsolationPhysical IsolationType = "physical" // 物理隔离（独立数据库）
)

// TenantConfig 租户配置
type TenantConfig struct {
	CustomBranding    bool     `json:"custom_branding"`
	LogoURL          string   `json:"logo_url,omitempty"`
	PrimaryColor     string   `json:"primary_color,omitempty"`
	AllowedProviders []string `json:"allowed_providers,omitempty"` // 限制可用的模型供应商
	RateLimitRPS    int      `json:"rate_limit_rps,omitempty"`
	IPWhitelist     []string `json:"ip_whitelist,omitempty"`
	IPBlacklist     []string `json:"ip_blacklist,omitempty"`
	DataRetention   int      `json:"data_retention_days,omitempty"` // 数据保留天数
	EnableAudit     bool     `json:"enable_audit"`
	EnableDesensitize bool   `json:"enable_desensitize"`
}

// TenantQuota 租户配额
type TenantQuota struct {
	TenantID       string `json:"tenant_id" gorm:"primaryKey"`
	ModelName      string `json:"model_name"`
	DailyTokens   int64  `json:"daily_tokens"`
	MonthlyTokens  int64  `json:"monthly_tokens"`
	UsedDailyTokens int64 `json:"used_daily_tokens"`
	UsedMonthlyTokens int64 `json:"used_monthly_tokens"`
	PeriodStart   time.Time `json:"period_start"`
}
