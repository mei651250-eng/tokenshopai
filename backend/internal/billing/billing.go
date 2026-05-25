package billing

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/tokenhub/backend/internal/gateway"
	"go.uber.org/zap"
)

//go:embed lua_scripts/deduct_balance.lua
var deductBalanceScript string

//go:embed lua_scripts/refresh_token_quota.lua
var refreshTokenQuotaScript string

// BillingService 计费服务
type BillingService struct {
	logger *zap.Logger
	rdb    *redis.Client
}

// NewBillingService 创建计费服务
func NewBillingService(logger *zap.Logger, rdb *redis.Client) *BillingService {
	return &BillingService{
		logger: logger,
		rdb:    rdb,
	}
}

// BillingRecord 计费记录
type BillingRecord struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id"`
	UserID          string  `json:"user_id"`
	APIKeyID        string  `json:"api_key_id"`
	ModelID         string  `json:"model_id"`
	ModelName       string  `json:"model_name"`
	Provider        string  `json:"provider"`
	TraceID         string  `json:"trace_id"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	Amount          int64   `json:"amount"`            // 扣费金额（分）
	Currency       string  `json:"currency"`
	BalanceBefore  int64   `json:"balance_before"`
	BalanceAfter   int64   `json:"balance_after"`
	BillingType    string  `json:"billing_type"`       // pay_as_you_go, package
	PackageID      string  `json:"package_id,omitempty"`
	CreatedAt      int64   `json:"created_at"`
}

// DeductRequest 扣费请求
type DeductRequest struct {
	TenantID     string
	UserID       string
	Usage        *gateway.Usage
	ModelConfig  *gateway.ModelConfig
	TraceID      string
	Currency     string
}

// DeductBalance 扣减余额（原子操作，Redis+Lua）
func (s *BillingService) DeductBalance(ctx context.Context, req *DeductRequest) (*BillingRecord, error) {
	// 1. 计算费用
	amount := s.calculateAmount(req.Usage, req.ModelConfig, req.Currency)

	if amount <= 0 {
		s.logger.Warn("calculated amount is zero, skipping deduction",
			zap.String("tenant_id", req.TenantID),
			zap.String("model_id", req.ModelConfig.ID),
		)
		return nil, nil
	}

	// 2. 构建Redis键
	balanceKey := fmt.Sprintf("balance:%s:%s", req.TenantID, req.UserID)
	frozenKey := fmt.Sprintf("frozen:%s:%s", req.TenantID, req.UserID)
	txID := uuid.New().String()
	idempotentKey := fmt.Sprintf("idempotent:%s", txID)

	// 3. 执行Lua原子扣费
	result, err := s.rdb.Eval(ctx, deductBalanceScript,
		[]string{balanceKey, frozenKey},
		amount, txID, 0, idempotentKey,
	).Int64()

	if err != nil {
		return nil, fmt.Errorf("execute deduct script: %w", err)
	}

	switch result {
	case 1:
		// 扣费成功
		s.logger.Info("balance deducted",
			zap.String("tenant_id", req.TenantID),
			zap.String("user_id", req.UserID),
			zap.Int64("amount", amount),
			zap.String("model", req.ModelConfig.Name),
		)
	case -1:
		return nil, fmt.Errorf("insufficient balance for tenant %s user %s", req.TenantID, req.UserID)
	case -2:
		s.logger.Warn("duplicate deduction ignored", zap.String("tx_id", txID))
		return nil, nil
	case -3:
		return nil, fmt.Errorf("invalid deduction parameters")
	default:
		return nil, fmt.Errorf("unknown deduct result: %d", result)
	}

	// 4. 获取扣费后余额
	newBalance, _ := s.rdb.Get(ctx, balanceKey).Int64()

	record := &BillingRecord{
		ID:           txID,
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		TraceID:      req.TraceID,
		ModelID:      req.ModelConfig.ID,
		ModelName:     req.ModelConfig.Name,
		Provider:     string(req.ModelConfig.Provider),
		InputTokens:  req.Usage.PromptTokens,
		OutputTokens: req.Usage.CompletionTokens,
		TotalTokens:  req.Usage.TotalTokens,
		Amount:       amount,
		Currency:     req.Currency,
		BalanceAfter: newBalance,
		BillingType:  "pay_as_you_go",
		CreatedAt:    time.Now().Unix(),
	}

	return record, nil
}

// calculateAmount 计算费用（分）
func (s *BillingService) calculateAmount(usage *gateway.Usage, model *gateway.ModelConfig, currency string) int64 {
	inputCost := float64(usage.PromptTokens) / 1000.0 * model.InputPrice
	outputCost := float64(usage.CompletionTokens) / 1000.0 * model.OutputPrice
	totalCost := inputCost + outputCost

	// 汇率转换（简化处理，实际应从汇率服务获取）
	convertedAmount := s.convertCurrency(totalCost, model.Currency, currency)

	// 转换为分
	amountInCents := int64(convertedAmount * 100)
	if amountInCents < 0 {
		return 0
	}
	return amountInCents
}

// convertCurrency 货币转换（简化版）
func (s *BillingService) convertCurrency(amount float64, from, to string) float64 {
	if from == to {
		return amount
	}
	// 简化的汇率表，实际应从外部汇率API获取
	rates := map[string]map[string]float64{
		"USD": {"CNY": 7.25, "JPY": 155.0, "KRW": 1350.0, "EUR": 0.92},
		"CNY": {"USD": 0.138, "JPY": 21.4, "KRW": 186.0, "EUR": 0.127},
	}

	if rateMap, ok := rates[from]; ok {
		if rate, ok := rateMap[to]; ok {
			return amount * rate
		}
	}

	// 通过USD中转
	if rate1, ok := rates[from]["USD"]; ok {
		if rate2, ok := rates["USD"][to]; ok {
			return amount * rate1 * rate2
		}
	}

	return amount
}

// GetBalance 获取余额
func (s *BillingService) GetBalance(ctx context.Context, tenantID, userID string) (int64, error) {
	key := fmt.Sprintf("balance:%s:%s", tenantID, userID)
	balance, err := s.rdb.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return balance, err
}

// TopUp 充值
func (s *BillingService) TopUp(ctx context.Context, tenantID, userID string, amount int64) error {
	key := fmt.Sprintf("balance:%s:%s", tenantID, userID)
	return s.rdb.IncrBy(ctx, key, amount).Err()
}

// CheckQuota 检查配额
func (s *BillingService) CheckQuota(ctx context.Context, tenantID, apiKeyID, model string, tokens int) (bool, error) {
	rateKey := fmt.Sprintf("ratelimit:%s:%s", tenantID, apiKeyID)
	quotaKey := fmt.Sprintf("quota:%s:%s:daily", tenantID, model)

	result, err := s.rdb.Eval(ctx, refreshTokenQuotaScript,
		[]string{rateKey, quotaKey},
		100, 10, time.Now().UnixMicro(), tokens, 1000000, 86400,
	).Int64()

	if err != nil {
		return false, err
	}

	return result >= 0, nil
}
