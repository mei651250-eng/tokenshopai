package audit

import (
	"time"
)

// AuditLog 审计日志
type AuditLog struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	TraceID     string    `json:"trace_id" gorm:"index"`
	TenantID    string    `json:"tenant_id" gorm:"index"`
	UserID      string    `json:"user_id" gorm:"index"`
	APIKeyID    string    `json:"api_key_id"`
	Action      string    `json:"action"`          // api_call, login, config_change, etc.
	Resource    string    `json:"resource"`        // model, apikey, tenant, etc.
	ResourceID  string    `json:"resource_id"`
	ModelName   string    `json:"model_name,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   int       `json:"latency_ms"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	TotalTokens  int       `json:"total_tokens,omitempty"`
	Amount      int64     `json:"amount,omitempty"` // 费用（分）
	Currency    string    `json:"currency,omitempty"`
	ClientIP    string    `json:"client_ip"`
	UserAgent   string    `json:"user_agent"`
	Error       string    `json:"error,omitempty"`
	RequestHash string    `json:"request_hash,omitempty"` // 请求内容哈希（不存原文）
	ResponseHash string   `json:"response_hash,omitempty"`
	Desensitized bool     `json:"desensitized"`     // 是否经过脱敏
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}

// AuditQuery 审计查询
type AuditQuery struct {
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id,omitempty"`
	Action     string    `json:"action,omitempty"`
	Resource   string    `json:"resource,omitempty"`
	ModelName  string    `json:"model_name,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// AuditService 审计服务
type AuditService struct {
	// 写入 ClickHouse 用于海量日志分析
	// 写入 PostgreSQL 用于事务性审计
}

// AuditAction 常量
const (
	ActionAPICall     = "api_call"
	ActionLogin       = "login"
	ActionLogout      = "logout"
	ActionKeyCreate   = "key_create"
	ActionKeyRevoke   = "key_revoke"
	ActionConfigChange = "config_change"
	ActionBillingTopup = "billing_topup"
	ActionExport      = "export"
	ActionSecurity    = "security_event"
)
