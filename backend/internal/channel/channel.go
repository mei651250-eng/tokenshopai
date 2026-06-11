package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tokenhub/backend/internal/gateway"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ChannelStatus 渠道状态
type ChannelStatus string

const (
	ChannelStatusActive   ChannelStatus = "active"
	ChannelStatusDisabled ChannelStatus = "disabled"
	ChannelStatusError    ChannelStatus = "error"
	ChannelStatusTesting  ChannelStatus = "testing"
)

// Channel 渠道 - 一个模型可有多个渠道（不同API Key/端点）
type Channel struct {
	ID            string              `json:"id" gorm:"primaryKey"`
	Name          string              `json:"name"`
	Provider      gateway.ModelProvider `json:"provider"`
	ModelID       string              `json:"model_id"`
	ModelName     string              `json:"model_name"`
	Endpoint      string              `json:"endpoint"`
	APIKey        string              `json:"api_key,omitempty"`
	APIKeys       []string            `json:"api_keys" gorm:"-"`
	KeyIndex      int                 `json:"key_index" gorm:"-"`
	Group         string              `json:"group"`
	Weight        int                 `json:"weight"`
	Priority      int                 `json:"priority"`
	MaxTokens     int                 `json:"max_tokens"`
	InputPrice    float64             `json:"input_price"`
	OutputPrice   float64             `json:"output_price"`
	Currency      string              `json:"currency"`
	Multiplier    float64             `json:"multiplier"`
	Streamable    bool                `json:"streamable"`
	Enabled       bool                `json:"enabled"`
	Status        ChannelStatus       `json:"status"`
	LatencyMs     int                 `json:"latency_ms"`
	SuccessRate   float64             `json:"success_rate"`
	TotalRequests int64               `json:"total_requests"`
	FailedRequests int64              `json:"failed_requests"`
	LastTestAt    int64               `json:"last_test_at"`
	LastError     string              `json:"last_error"`
	TenantID      string              `json:"tenant_id" gorm:"index"`
	Tags          []string            `json:"tags" gorm:"serializer:json"`
	CreatedAt     int64               `json:"created_at"`
	UpdatedAt     int64               `json:"updated_at"`
}

// ChannelTestResult 渠道测试结果
type ChannelTestResult struct {
	ChannelID string   `json:"channel_id"`
	Success   bool     `json:"success"`
	LatencyMs int      `json:"latency_ms"`
	Error     string   `json:"error,omitempty"`
	ModelList []string `json:"model_list,omitempty"`
}

// ChannelService 渠道服务
type ChannelService struct {
	logger   *zap.Logger
	db       *gorm.DB
	mu       sync.RWMutex
	affinity map[string]string // user_id -> channel_id (渠道亲和性)
}

// NewChannelService 创建渠道服务
func NewChannelService(logger *zap.Logger, db *gorm.DB) *ChannelService {
	return &ChannelService{
		logger:   logger,
		db:       db,
		affinity: make(map[string]string),
	}
}

// AutoMigrate 自动迁移
func (s *ChannelService) AutoMigrate() error {
	return s.db.AutoMigrate(&Channel{})
}

// CreateChannel 创建渠道
func (s *ChannelService) CreateChannel(ctx context.Context, ch *Channel) error {
	if ch.ID == "" {
		ch.ID = uuid.New().String()
	}
	now := time.Now().Unix()
	ch.CreatedAt = now
	ch.UpdatedAt = now
	if ch.Status == "" {
		ch.Status = ChannelStatusActive
	}
	if ch.Multiplier == 0 {
		ch.Multiplier = 1.0
	}
	if ch.Currency == "" {
		ch.Currency = "CNY"
	}
	s.parseAPIKeys(ch)
	if err := s.db.WithContext(ctx).Create(ch).Error; err != nil {
		return err
	}
	// 自动同步到 ModelConfig（渠道创建时确保模型列表可见）
	s.syncToModelConfig(ctx, ch)
	return nil
}

// UpdateChannel 更新渠道
func (s *ChannelService) UpdateChannel(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().Unix()
	result := s.db.WithContext(ctx).Model(&Channel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("channel not found: %s", id)
	}
	// 更新后同步 ModelConfig
	var ch Channel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&ch).Error; err == nil {
		s.syncToModelConfig(ctx, &ch)
	}
	return nil
}

// DeleteChannel 删除渠道
func (s *ChannelService) DeleteChannel(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&Channel{}).Error
}

// GetChannel 获取渠道
func (s *ChannelService) GetChannel(ctx context.Context, id string) (*Channel, error) {
	var ch Channel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&ch).Error; err != nil {
		return nil, err
	}
	s.parseAPIKeys(&ch)
	return &ch, nil
}

// ListChannels 列出渠道
func (s *ChannelService) ListChannels(ctx context.Context, provider, group, modelName string, enabled *bool) ([]*Channel, error) {
	query := s.db.WithContext(ctx).Model(&Channel{})
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if group != "" {
		query = query.Where("`group` = ?", group)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	query = query.Order("priority ASC, weight DESC")

	var channels []*Channel
	if err := query.Find(&channels).Error; err != nil {
		return nil, err
	}
	for _, ch := range channels {
		s.parseAPIKeys(ch)
	}
	return channels, nil
}

// GetChannelsForModel 获取指定模型的所有可用渠道
func (s *ChannelService) GetChannelsForModel(ctx context.Context, modelName, tenantID string) ([]*Channel, error) {
	query := s.db.WithContext(ctx).Where("model_name = ? AND enabled = ? AND status != ?", modelName, true, ChannelStatusDisabled)
	if tenantID != "" {
		query = query.Where("tenant_id = ? OR tenant_id = ''", tenantID)
	}
	query = query.Order("priority ASC, weight DESC")

	var channels []*Channel
	if err := query.Find(&channels).Error; err != nil {
		return nil, err
	}
	for _, ch := range channels {
		s.parseAPIKeys(ch)
	}
	return channels, nil
}

// RotateAPIKey 轮换API Key
func (s *ChannelService) RotateAPIKey(ch *Channel) string {
	if len(ch.APIKeys) <= 1 {
		if len(ch.APIKeys) == 1 {
			return ch.APIKeys[0]
		}
		return ch.APIKey
	}
	ch.KeyIndex = (ch.KeyIndex + 1) % len(ch.APIKeys)
	return ch.APIKeys[ch.KeyIndex]
}

// GetCurrentAPIKey 获取当前使用的API Key
func (s *ChannelService) GetCurrentAPIKey(ch *Channel) string {
	if len(ch.APIKeys) == 0 {
		return ch.APIKey
	}
	if ch.KeyIndex >= len(ch.APIKeys) {
		ch.KeyIndex = 0
	}
	return ch.APIKeys[ch.KeyIndex]
}

// RecordSuccess 记录渠道请求成功
func (s *ChannelService) RecordSuccess(ctx context.Context, channelID string, latencyMs int) {
	s.db.WithContext(ctx).Model(&Channel{}).Where("id = ?", channelID).Updates(map[string]interface{}{
		"total_requests": gorm.Expr("total_requests + 1"),
		"latency_ms":     latencyMs,
		"status":         ChannelStatusActive,
		"last_test_at":   time.Now().Unix(),
	})
}

// RecordFailure 记录渠道请求失败
func (s *ChannelService) RecordFailure(ctx context.Context, channelID string, errMsg string) {
	s.db.WithContext(ctx).Model(&Channel{}).Where("id = ?", channelID).Updates(map[string]interface{}{
		"total_requests":  gorm.Expr("total_requests + 1"),
		"failed_requests": gorm.Expr("failed_requests + 1"),
		"last_error":      errMsg,
		"last_test_at":    time.Now().Unix(),
	})

	// 自动禁用失败率过高的渠道
	var ch Channel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&ch).Error; err == nil {
		if ch.TotalRequests > 10 {
			failRate := float64(ch.FailedRequests) / float64(ch.TotalRequests)
			if failRate > 0.5 {
				s.db.WithContext(ctx).Model(&Channel{}).Where("id = ?", channelID).Updates(map[string]interface{}{
					"status":  ChannelStatusError,
					"enabled": false,
				})
				s.logger.Warn("channel auto-disabled due to high failure rate",
					zap.String("channel_id", channelID),
					zap.Float64("fail_rate", failRate),
				)
			}
		}
	}
}

// TestChannel 测试渠道可用性
func (s *ChannelService) TestChannel(ctx context.Context, channelID string) *ChannelTestResult {
	var ch Channel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&ch).Error; err != nil {
		return &ChannelTestResult{ChannelID: channelID, Success: false, Error: "channel not found"}
	}
	s.parseAPIKeys(&ch)

	apiKey := s.GetCurrentAPIKey(&ch)
	if apiKey == "" {
		return &ChannelTestResult{ChannelID: channelID, Success: false, Error: "no API key configured"}
	}

	start := time.Now()
	result := &ChannelTestResult{ChannelID: channelID}

	req, err := http.NewRequest("GET", ch.Endpoint+"/models", nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		result.Error = err.Error()
		s.RecordFailure(ctx, channelID, err.Error())
		return result
	}
	defer resp.Body.Close()

	result.LatencyMs = int(time.Since(start).Milliseconds())

	if resp.StatusCode == http.StatusOK {
		result.Success = true
		s.RecordSuccess(ctx, channelID, result.LatencyMs)

		var modelsResp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err == nil {
			for _, m := range modelsResp.Data {
				result.ModelList = append(result.ModelList, m.ID)
			}
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
		s.RecordFailure(ctx, channelID, result.Error)
	}

	return result
}

// BatchTestChannels 批量测试渠道
func (s *ChannelService) BatchTestChannels(ctx context.Context) []*ChannelTestResult {
	var channels []*Channel
	s.db.WithContext(ctx).Where("enabled = ?", true).Find(&channels)

	var results []*ChannelTestResult
	for _, ch := range channels {
		results = append(results, s.TestChannel(ctx, ch.ID))
	}
	return results
}

// GetAffinity 获取渠道亲和性
func (s *ChannelService) GetAffinity(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.affinity[userID]
}

// SetAffinity 设置渠道亲和性
func (s *ChannelService) SetAffinity(userID, channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.affinity[userID] = channelID
}

// BatchCreateChannels 批量创建渠道
func (s *ChannelService) BatchCreateChannels(ctx context.Context, channels []*Channel) (created int, skipped int, failed int) {
	for _, ch := range channels {
		var count int64
		s.db.WithContext(ctx).Model(&Channel{}).Where("model_name = ? AND provider = ? AND model_id = ?", ch.ModelName, ch.Provider, ch.ModelID).Count(&count)
		if count > 0 {
			skipped++
			continue
		}
		if err := s.CreateChannel(ctx, ch); err != nil {
			failed++
			s.logger.Error("batch create channel failed", zap.String("model_id", ch.ModelID), zap.Error(err))
		} else {
			created++
		}
	}
	return
}

// GetChannelStats 获取渠道统计
func (s *ChannelService) GetChannelStats(ctx context.Context) map[string]interface{} {
	var total, active, errorCount, disabled int64
	s.db.WithContext(ctx).Model(&Channel{}).Count(&total)
	s.db.WithContext(ctx).Model(&Channel{}).Where("status = ?", ChannelStatusActive).Count(&active)
	s.db.WithContext(ctx).Model(&Channel{}).Where("status = ?", ChannelStatusError).Count(&errorCount)
	s.db.WithContext(ctx).Model(&Channel{}).Where("status = ?", ChannelStatusDisabled).Count(&disabled)

	return map[string]interface{}{
		"total":    total,
		"active":   active,
		"error":    errorCount,
		"disabled": disabled,
	}
}

// parseAPIKeys 解析多个API Key
func (s *ChannelService) parseAPIKeys(ch *Channel) {
	if ch.APIKey == "" {
		ch.APIKeys = []string{}
		return
	}
	keys := strings.Split(ch.APIKey, ",")
	ch.APIKeys = make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			ch.APIKeys = append(ch.APIKeys, k)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// syncToModelConfig 渠道创建/更新时自动同步到 ModelConfig 表
// 确保通过渠道创建的模型在模型列表页面可见
func (s *ChannelService) syncToModelConfig(ctx context.Context, ch *Channel) {
	var mc gateway.ModelConfig
	err := s.db.WithContext(ctx).Where("model_id = ? AND provider = ?", ch.ModelID, ch.Provider).First(&mc).Error
	if err != nil {
		// 不存在，创建新的 ModelConfig
		mc = gateway.ModelConfig{
			ID:          uuid.New().String(),
			Name:        ch.ModelName,
			Provider:    ch.Provider,
			ModelID:     ch.ModelID,
			Endpoint:    ch.Endpoint,
			MaxTokens:   ch.MaxTokens,
			InputPrice:  ch.InputPrice,
			OutputPrice: ch.OutputPrice,
			Currency:    ch.Currency,
			Weight:      ch.Weight,
			Priority:    ch.Priority,
			Enabled:     ch.Enabled,
			Streamable:  ch.Streamable,
			TenantID:    ch.TenantID,
			LatencyMs:   ch.LatencyMs,
			SuccessRate: ch.SuccessRate,
			CreatedAt:   ch.CreatedAt,
		}
		if err := s.db.WithContext(ctx).Create(&mc).Error; err != nil {
			s.logger.Error("failed to sync channel to model_config", zap.Error(err))
		} else {
			s.logger.Info("synced channel to model_config", zap.String("model_id", ch.ModelID))
		}
	} else {
		// 已存在，更新
		updates := map[string]interface{}{
			"endpoint":     ch.Endpoint,
			"max_tokens":   ch.MaxTokens,
			"input_price":  ch.InputPrice,
			"output_price": ch.OutputPrice,
			"currency":     ch.Currency,
			"weight":       ch.Weight,
			"priority":     ch.Priority,
			"enabled":      ch.Enabled,
			"streamable":   ch.Streamable,
			"latency_ms":   ch.LatencyMs,
			"success_rate": ch.SuccessRate,
		}
		if err := s.db.WithContext(ctx).Model(&mc).Updates(updates).Error; err != nil {
			s.logger.Error("failed to update model_config from channel", zap.Error(err))
		}
	}
}
