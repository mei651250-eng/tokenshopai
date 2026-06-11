package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuditLog 审计日志
type AuditLog struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	TraceID      string    `json:"trace_id" gorm:"index"`
	TenantID     string    `json:"tenant_id" gorm:"index"`
	UserID       string    `json:"user_id" gorm:"index"`
	APIKeyID     string    `json:"api_key_id"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id"`
	ModelName    string    `json:"model_name,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	StatusCode   int       `json:"status_code"`
	LatencyMs    int       `json:"latency_ms"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	TotalTokens  int       `json:"total_tokens,omitempty"`
	Amount       int64     `json:"amount,omitempty"`
	Currency     string    `json:"currency,omitempty"`
	ClientIP     string    `json:"client_ip"`
	UserAgent    string    `json:"user_agent"`
	Error        string    `json:"error,omitempty"`
	RequestHash  string    `json:"request_hash,omitempty"`
	ResponseHash string    `json:"response_hash,omitempty"`
	Desensitized bool      `json:"desensitized"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// AuditQuery 审计查询
type AuditQuery struct {
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id,omitempty"`
	Action     string     `json:"action,omitempty"`
	Resource   string     `json:"resource,omitempty"`
	ModelName  string     `json:"model_name,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	StatusCode int        `json:"status_code,omitempty"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
}

// AuditService 审计服务
type AuditService struct {
	logger *zap.Logger
	db     *gorm.DB
}

// AuditAction 常量
const (
	ActionAPICall      = "api_call"
	ActionLogin        = "login"
	ActionLogout       = "logout"
	ActionKeyCreate    = "key_create"
	ActionKeyRevoke    = "key_revoke"
	ActionConfigChange = "config_change"
	ActionBillingTopup = "billing_topup"
	ActionExport       = "export"
	ActionSecurity     = "security_event"
)

// NewAuditService 创建审计服务
func NewAuditService(logger *zap.Logger, db *gorm.DB) *AuditService {
	return &AuditService{logger: logger, db: db}
}

// AutoMigrate 自动迁移
func (s *AuditService) AutoMigrate() error {
	return s.db.AutoMigrate(&AuditLog{})
}

// Log 记录审计日志
func (s *AuditService) Log(ctx context.Context, log *AuditLog) error {
	if log.ID == "" {
		log.ID = generateAuditID()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		s.logger.Error("failed to write audit log", zap.Error(err))
		return err
	}
	return nil
}

// LogAPICall 记录API调用审计日志
func (s *AuditService) LogAPICall(ctx context.Context, tenantID, userID, apiKeyID, modelName, provider string, statusCode, latencyMs, inputTokens, outputTokens int, amount int64, currency, clientIP, userAgent string, err error) {
	log := &AuditLog{
		TenantID:     tenantID,
		UserID:       userID,
		APIKeyID:     apiKeyID,
		Action:       ActionAPICall,
		Resource:     "model",
		ModelName:    modelName,
		Provider:     provider,
		StatusCode:   statusCode,
		LatencyMs:    latencyMs,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Amount:       amount,
		Currency:     currency,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
	}
	if err != nil {
		log.Error = err.Error()
	}
	s.Log(ctx, log)
}

// Query 查询审计日志
func (s *AuditService) Query(ctx context.Context, q *AuditQuery) ([]*AuditLog, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}

	query := s.db.WithContext(ctx).Model(&AuditLog{})

	if q.TenantID != "" {
		query = query.Where("tenant_id = ?", q.TenantID)
	}
	if q.UserID != "" {
		query = query.Where("user_id = ?", q.UserID)
	}
	if q.Action != "" {
		query = query.Where("action = ?", q.Action)
	}
	if q.Resource != "" {
		query = query.Where("resource = ?", q.Resource)
	}
	if q.ModelName != "" {
		query = query.Where("model_name = ?", q.ModelName)
	}
	if q.StartTime != nil {
		query = query.Where("created_at >= ?", q.StartTime)
	}
	if q.EndTime != nil {
		query = query.Where("created_at <= ?", q.EndTime)
	}
	if q.StatusCode > 0 {
		query = query.Where("status_code = ?", q.StatusCode)
	}

	var total int64
	query.Count(&total)

	var logs []*AuditLog
	offset := (q.Page - 1) * q.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(q.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// HashContent 对请求/响应内容做脱敏哈希（不存原文）
func HashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// generateAuditID 生成审计日志ID
func generateAuditID() string {
	b := make([]byte, 16)
	time.Now().UnixMilli()
	return hex.EncodeToString(b)
}
