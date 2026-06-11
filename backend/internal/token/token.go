package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TokenStatus 令牌状态
type TokenStatus string

const (
	TokenStatusActive  TokenStatus = "active"
	TokenStatusRevoked TokenStatus = "revoked"
	TokenStatusExpired TokenStatus = "expired"
)

// AccessToken 令牌（sk-xxx格式，用于API二次分发）
type AccessToken struct {
	ID           string       `json:"id" gorm:"primaryKey"`
	Name         string       `json:"name"`
	Key          string       `json:"key,omitempty"`           // sk-xxx 格式，创建后仅显示一次
	KeyHash      string       `json:"key_hash" gorm:"index"`   // 存储hash用于验证
	UserID       string       `json:"user_id" gorm:"index"`
	TenantID     string       `json:"tenant_id" gorm:"index"`
	Status       TokenStatus  `json:"status"`
	QuotaTotal   int64        `json:"quota_total"`             // 总额度（分），-1为无限
	QuotaUsed    int64        `json:"quota_used"`              // 已用额度（分）
	QuotaRemain  int64        `json:"quota_remain" gorm:"-"`   // 剩余额度
	Models       []string     `json:"models" gorm:"serializer:json"` // 允许的模型列表，空为全部
	AllowedIPs   []string     `json:"allowed_ips" gorm:"serializer:json"` // 允许的IP范围
	RateLimitRPM int          `json:"rate_limit_rpm"`          // 每分钟请求限制
	RateLimitTPM int          `json:"rate_limit_tpm"`          // 每分钟Token限制
	Group        string       `json:"group"`                   // 令牌分组（对应不同倍率）
	ExpiresAt    int64        `json:"expires_at"`              // 过期时间，0为永不过期
	LastUsedAt   int64        `json:"last_used_at"`
	TotalRequests int64       `json:"total_requests"`
	CreatedAt    int64        `json:"created_at"`
	UpdatedAt    int64        `json:"updated_at"`
}

// CreateTokenRequest 创建令牌请求
type CreateTokenRequest struct {
	Name         string   `json:"name" binding:"required"`
	QuotaTotal   int64    `json:"quota_total"`
	Models       []string `json:"models"`
	AllowedIPs   []string `json:"allowed_ips"`
	RateLimitRPM int      `json:"rate_limit_rpm"`
	RateLimitTPM int      `json:"rate_limit_tpm"`
	Group        string   `json:"group"`
	ExpiresAt    int64    `json:"expires_at"`
}

// TokenService 令牌服务
type TokenService struct {
	logger *zap.Logger
	db     *gorm.DB
	mu     sync.RWMutex
}

// NewTokenService 创建令牌服务
func NewTokenService(logger *zap.Logger, db *gorm.DB) *TokenService {
	return &TokenService{logger: logger, db: db}
}

// AutoMigrate 自动迁移
func (s *TokenService) AutoMigrate() error {
	return s.db.AutoMigrate(&AccessToken{})
}

// generateTokenKey 生成 sk-xxx 格式的令牌
func generateTokenKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk-" + hex.EncodeToString(b)
}

// hashToken 对令牌进行哈希
func hashToken(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// CreateToken 创建令牌
func (s *TokenService) CreateToken(ctx context.Context, userID, tenantID string, req *CreateTokenRequest) (*AccessToken, error) {
	key := generateTokenKey()
	now := time.Now().Unix()

	token := &AccessToken{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Key:          key,
		KeyHash:      hashToken(key),
		UserID:       userID,
		TenantID:     tenantID,
		Status:       TokenStatusActive,
		QuotaTotal:   req.QuotaTotal,
		QuotaUsed:    0,
		Models:       req.Models,
		AllowedIPs:   req.AllowedIPs,
		RateLimitRPM: req.RateLimitRPM,
		RateLimitTPM: req.RateLimitTPM,
		Group:        req.Group,
		ExpiresAt:    req.ExpiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.db.WithContext(ctx).Create(token).Error; err != nil {
		return nil, err
	}

	s.logger.Info("access token created",
		zap.String("token_id", token.ID),
		zap.String("user_id", userID),
		zap.String("name", req.Name),
	)

	return token, nil
}

// ValidateToken 验证令牌
func (s *TokenService) ValidateToken(ctx context.Context, key string) (*AccessToken, error) {
	hash := hashToken(key)

	var token AccessToken
	if err := s.db.WithContext(ctx).Where("key_hash = ? AND status = ?", hash, TokenStatusActive).First(&token).Error; err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	// 检查过期
	if token.ExpiresAt > 0 && token.ExpiresAt < time.Now().Unix() {
		s.db.WithContext(ctx).Model(&token).Update("status", TokenStatusExpired)
		return nil, fmt.Errorf("token expired")
	}

	// 检查额度
	if token.QuotaTotal > 0 && token.QuotaUsed >= token.QuotaTotal {
		return nil, fmt.Errorf("token quota exhausted")
	}

	// 更新最后使用时间
	s.db.WithContext(ctx).Model(&token).Updates(map[string]interface{}{
		"last_used_at":   time.Now().Unix(),
		"total_requests": gorm.Expr("total_requests + 1"),
	})

	token.QuotaRemain = token.QuotaTotal - token.QuotaUsed
	if token.QuotaTotal < 0 {
		token.QuotaRemain = -1 // 无限
	}

	return &token, nil
}

// CheckModelPermission 检查模型权限
func (s *TokenService) CheckModelPermission(token *AccessToken, modelName string) bool {
	if len(token.Models) == 0 {
		return true // 空列表表示允许所有
	}
	for _, m := range token.Models {
		if m == modelName || strings.HasPrefix(modelName, strings.TrimSuffix(m, "*")) {
			return true
		}
	}
	return false
}

// CheckIPPermission 检查IP权限
func (s *TokenService) CheckIPPermission(token *AccessToken, clientIP string) bool {
	if len(token.AllowedIPs) == 0 {
		return true // 空列表表示允许所有
	}
	for _, ip := range token.AllowedIPs {
		if ip == clientIP || strings.HasSuffix(clientIP, strings.TrimPrefix(ip, "*")) {
			return true
		}
		// CIDR 简化匹配
		if strings.Contains(ip, "/") && strings.HasPrefix(clientIP, strings.Split(ip, "/")[0][:strings.LastIndex(strings.Split(ip, "/")[0], ".")]) {
			return true
		}
	}
	return false
}

// DeductQuota 扣减额度
func (s *TokenService) DeductQuota(ctx context.Context, tokenID string, amount int64) error {
	result := s.db.WithContext(ctx).Model(&AccessToken{}).Where("id = ? AND (quota_total < 0 OR quota_used + ? <= quota_total)", tokenID, amount).
		Updates(map[string]interface{}{
			"quota_used":  gorm.Expr("quota_used + ?", amount),
			"updated_at":  time.Now().Unix(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient quota")
	}
	return nil
}

// RevokeToken 吊销令牌
func (s *TokenService) RevokeToken(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Model(&AccessToken{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     TokenStatusRevoked,
		"updated_at": time.Now().Unix(),
	}).Error
}

// DeleteToken 删除令牌
func (s *TokenService) DeleteToken(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&AccessToken{}).Error
}

// UpdateToken 更新令牌
func (s *TokenService) UpdateToken(ctx context.Context, id string, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&AccessToken{}).Where("id = ?", id).Updates(updates).Error
}

// ListTokens 列出令牌
func (s *TokenService) ListTokens(ctx context.Context, userID, tenantID string) ([]*AccessToken, error) {
	var tokens []*AccessToken
	query := s.db.WithContext(ctx)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	query = query.Order("created_at DESC")

	if err := query.Find(&tokens).Error; err != nil {
		return nil, err
	}
	for _, t := range tokens {
		t.Key = "" // 不返回key
		t.QuotaRemain = t.QuotaTotal - t.QuotaUsed
		if t.QuotaTotal < 0 {
			t.QuotaRemain = -1
		}
	}
	return tokens, nil
}

// GetToken 获取令牌
func (s *TokenService) GetToken(ctx context.Context, id string) (*AccessToken, error) {
	var token AccessToken
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&token).Error; err != nil {
		return nil, err
	}
	token.Key = ""
	token.QuotaRemain = token.QuotaTotal - token.QuotaUsed
	if token.QuotaTotal < 0 {
		token.QuotaRemain = -1
	}
	return &token, nil
}
