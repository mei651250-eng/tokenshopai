package reconciliation

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ReconciliationStatus 对账状态
type ReconciliationStatus string

const (
	ReconStatusMatched   ReconciliationStatus = "matched"
	ReconStatusUnmatched ReconciliationStatus = "unmatched"
	ReconStatusShortage  ReconciliationStatus = "shortage"  // 短款：收到钱少于应收
	ReconStatusExcess    ReconciliationStatus = "excess"    // 溢款：收到钱多于应收
)

// DailySummary 日汇总
type DailySummary struct {
	ID                string `json:"id"`
	Date              string `json:"date"`               // YYYY-MM-DD
	TenantID          string `json:"tenant_id,omitempty"`

	// 收入
	PaymentCount      int64  `json:"payment_count"`       // 充值订单数
	PaymentAmount     int64  `json:"payment_amount"`       // 充值金额（分）
	PaymentFee        int64  `json:"payment_fee"`          // 手续费
	ActualIncome      int64  `json:"actual_income"`        // 实际到账

	// 支出
	APICallCount      int64  `json:"api_call_count"`       // API调用次数
	TokenConsumed     int64  `json:"token_consumed"`       // 消耗Token数
	APICost           int64  `json:"api_cost"`             // API成本（分）

	// 提现
	WithdrawalCount   int64  `json:"withdrawal_count"`
	WithdrawalAmount  int64  `json:"withdrawal_amount"`

	// 退款
	RefundCount       int64  `json:"refund_count"`
	RefundAmount      int64  `json:"refund_amount"`

	// 净利润
	GrossProfit       int64  `json:"gross_profit"`         // 毛利 = 实际到账 - API成本
	NetProfit         int64  `json:"net_profit"`           // 净利 = 毛利 - 退款 - 提现手续费

	// 对账状态
	ReconStatus       ReconciliationStatus `json:"recon_status"`
	Discrepancy       int64  `json:"discrepancy"`          // 差异金额
	Notes             string `json:"notes,omitempty"`

	CreatedAt         int64  `json:"created_at"`
}

// ReconciliationService 对账服务
type ReconciliationService struct {
	logger *zap.Logger
	rdb    *redis.Client
}

// NewReconciliationService 创建对账服务
func NewReconciliationService(logger *zap.Logger, rdb *redis.Client) *ReconciliationService {
	return &ReconciliationService{logger: logger, rdb: rdb}
}

// GenerateDailySummary 生成日汇总
func (s *ReconciliationService) GenerateDailySummary(ctx context.Context, date, tenantID string) (*DailySummary, error) {
	summary := &DailySummary{
		ID:       uuid.New().String(),
		Date:     date,
		TenantID: tenantID,
	}

	// 解析日期
	dayStart, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}
	dayEnd := dayStart.Add(24 * time.Hour)
	startTs := dayStart.Unix()
	endTs := dayEnd.Unix()

	if tenantID != "" {
		// 按租户统计
		summary.PaymentCount, summary.PaymentAmount = s.countPayments(ctx, tenantID, startTs, endTs)
		summary.APICallCount, summary.TokenConsumed, summary.APICost = s.countAPIUsage(ctx, tenantID, startTs, endTs)
		summary.WithdrawalCount, summary.WithdrawalAmount = s.countWithdrawals(ctx, tenantID, startTs, endTs)
		summary.RefundCount, summary.RefundAmount = s.countRefunds(ctx, tenantID, startTs, endTs)
	} else {
		// 全局统计
		summary.PaymentCount, summary.PaymentAmount = s.countPaymentsAll(ctx, startTs, endTs)
		summary.APICallCount, summary.TokenConsumed, summary.APICost = s.countAPIUsageAll(ctx, startTs, endTs)
		summary.WithdrawalCount, summary.WithdrawalAmount = s.countWithdrawalsAll(ctx, startTs, endTs)
		summary.RefundCount, summary.RefundAmount = s.countRefundsAll(ctx, startTs, endTs)
	}

	// 计算手续费（平均0.6%）
	summary.PaymentFee = int64(float64(summary.PaymentAmount) * 0.006)
	summary.ActualIncome = summary.PaymentAmount - summary.PaymentFee

	// 计算利润
	summary.GrossProfit = summary.ActualIncome - summary.APICost
	summary.NetProfit = summary.GrossProfit - summary.RefundAmount

	// 对账状态
	if summary.Discrepancy == 0 {
		summary.ReconStatus = ReconStatusMatched
	} else if summary.Discrepancy > 0 {
		summary.ReconStatus = ReconStatusExcess
	} else {
		summary.ReconStatus = ReconStatusShortage
	}
	summary.CreatedAt = time.Now().Unix()

	// 存储汇总
	summaryKey := fmt.Sprintf("recon:summary:%s:%s", date, tenantID)
	s.storeSummary(ctx, summaryKey, summary)

	s.logger.Info("daily summary generated",
		zap.String("date", date),
		zap.String("tenant_id", tenantID),
		zap.Int64("payment_amount", summary.PaymentAmount),
		zap.Int64("api_cost", summary.APICost),
		zap.Int64("net_profit", summary.NetProfit),
	)

	return summary, nil
}

// GetDailySummary 获取日汇总
func (s *ReconciliationService) GetDailySummary(ctx context.Context, date, tenantID string) (*DailySummary, error) {
	summaryKey := fmt.Sprintf("recon:summary:%s:%s", date, tenantID)
	data, err := s.rdb.HGetAll(ctx, summaryKey).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		// 汇总不存在，自动生成
		return s.GenerateDailySummary(ctx, date, tenantID)
	}
	return s.parseSummary(data), nil
}

// GetDateRangeSummary 获取日期范围汇总
func (s *ReconciliationService) GetDateRangeSummary(ctx context.Context, startDate, endDate, tenantID string) ([]*DailySummary, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, err
	}

	var summaries []*DailySummary
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		dateStr := d.Format("2006-01-02")
		summary, err := s.GetDailySummary(ctx, dateStr, tenantID)
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// GetAggregatedSummary 获取聚合汇总
func (s *ReconciliationService) GetAggregatedSummary(ctx context.Context, startDate, endDate, tenantID string) (*DailySummary, error) {
	summaries, err := s.GetDateRangeSummary(ctx, startDate, endDate, tenantID)
	if err != nil {
		return nil, err
	}

	agg := &DailySummary{
		ID:       uuid.New().String(),
		Date:     startDate + " ~ " + endDate,
		TenantID: tenantID,
	}

	for _, s := range summaries {
		agg.PaymentCount += s.PaymentCount
		agg.PaymentAmount += s.PaymentAmount
		agg.PaymentFee += s.PaymentFee
		agg.ActualIncome += s.ActualIncome
		agg.APICallCount += s.APICallCount
		agg.TokenConsumed += s.TokenConsumed
		agg.APICost += s.APICost
		agg.WithdrawalCount += s.WithdrawalCount
		agg.WithdrawalAmount += s.WithdrawalAmount
		agg.RefundCount += s.RefundCount
		agg.RefundAmount += s.RefundAmount
		agg.GrossProfit += s.GrossProfit
		agg.NetProfit += s.NetProfit
	}

	return agg, nil
}

// ==================== 内部方法 ====================

func (s *ReconciliationService) countPayments(ctx context.Context, tenantID string, startTs, endTs int64) (int64, int64) {
	key := fmt.Sprintf("payment:user:%s", tenantID)
	orderNos, err := s.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", startTs),
		Max: fmt.Sprintf("%d", endTs),
	}).Result()
	if err != nil {
		return 0, 0
	}

	var count, total int64
	for _, orderNo := range orderNos {
		orderKey := fmt.Sprintf("payment:order:%s", orderNo)
		data, err := s.rdb.HGetAll(ctx, orderKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		if data["status"] == "completed" {
			count++
			total += parseInt64(data["amount"])
		}
	}
	return count, total
}

func (s *ReconciliationService) countPaymentsAll(ctx context.Context, startTs, endTs int64) (int64, int64) {
	// 从全局支付统计键读取
	key := "stats:payments:daily"
	dateKey := time.Unix(startTs, 0).Format("20060102")
	count, _ := s.rdb.HGet(ctx, key, dateKey+"_count").Int64()
	amount, _ := s.rdb.HGet(ctx, key, dateKey+"_amount").Int64()
	return count, amount
}

func (s *ReconciliationService) countAPIUsage(ctx context.Context, tenantID string, startTs, endTs int64) (int64, int64, int64) {
	key := fmt.Sprintf("stats:api:%s:daily", tenantID)
	dateKey := time.Unix(startTs, 0).Format("20060102")
	count, _ := s.rdb.HGet(ctx, key, dateKey+"_count").Int64()
	tokens, _ := s.rdb.HGet(ctx, key, dateKey+"_tokens").Int64()
	cost, _ := s.rdb.HGet(ctx, key, dateKey+"_cost").Int64()
	return count, tokens, cost
}

func (s *ReconciliationService) countAPIUsageAll(ctx context.Context, startTs, endTs int64) (int64, int64, int64) {
	key := "stats:api:global:daily"
	dateKey := time.Unix(startTs, 0).Format("20060102")
	count, _ := s.rdb.HGet(ctx, key, dateKey+"_count").Int64()
	tokens, _ := s.rdb.HGet(ctx, key, dateKey+"_tokens").Int64()
	cost, _ := s.rdb.HGet(ctx, key, dateKey+"_cost").Int64()
	return count, tokens, cost
}

func (s *ReconciliationService) countWithdrawals(ctx context.Context, tenantID string, startTs, endTs int64) (int64, int64) {
	key := fmt.Sprintf("withdrawal:user:%s", tenantID)
	orderNos, err := s.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", startTs),
		Max: fmt.Sprintf("%d", endTs),
	}).Result()
	if err != nil {
		return 0, 0
	}

	var count, total int64
	for _, orderNo := range orderNos {
		orderKey := fmt.Sprintf("withdrawal:order:%s", orderNo)
		data, err := s.rdb.HGetAll(ctx, orderKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		if data["status"] == "completed" {
			count++
			total += parseInt64(data["amount"])
		}
	}
	return count, total
}

func (s *ReconciliationService) countWithdrawalsAll(ctx context.Context, startTs, endTs int64) (int64, int64) {
	return 0, 0
}

func (s *ReconciliationService) countRefunds(ctx context.Context, tenantID string, startTs, endTs int64) (int64, int64) {
	key := fmt.Sprintf("refund:user:%s", tenantID)
	orderNos, err := s.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", startTs),
		Max: fmt.Sprintf("%d", endTs),
	}).Result()
	if err != nil {
		return 0, 0
	}

	var count, total int64
	for _, orderNo := range orderNos {
		orderKey := fmt.Sprintf("refund:order:%s", orderNo)
		data, err := s.rdb.HGetAll(ctx, orderKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		if data["status"] == "completed" {
			count++
			total += parseInt64(data["amount"])
		}
	}
	return count, total
}

func (s *ReconciliationService) countRefundsAll(ctx context.Context, startTs, endTs int64) (int64, int64) {
	return 0, 0
}

func (s *ReconciliationService) storeSummary(ctx context.Context, key string, summary *DailySummary) {
	s.rdb.HSet(ctx, key, map[string]interface{}{
		"id":               summary.ID,
		"date":             summary.Date,
		"tenant_id":        summary.TenantID,
		"payment_count":    summary.PaymentCount,
		"payment_amount":   summary.PaymentAmount,
		"payment_fee":      summary.PaymentFee,
		"actual_income":    summary.ActualIncome,
		"api_call_count":   summary.APICallCount,
		"token_consumed":   summary.TokenConsumed,
		"api_cost":         summary.APICost,
		"withdrawal_count": summary.WithdrawalCount,
		"withdrawal_amount": summary.WithdrawalAmount,
		"refund_count":     summary.RefundCount,
		"refund_amount":    summary.RefundAmount,
		"gross_profit":     summary.GrossProfit,
		"net_profit":       summary.NetProfit,
		"recon_status":     string(summary.ReconStatus),
		"discrepancy":      summary.Discrepancy,
		"created_at":       summary.CreatedAt,
	})
}

func (s *ReconciliationService) parseSummary(data map[string]string) *DailySummary {
	return &DailySummary{
		ID:               data["id"],
		Date:             data["date"],
		TenantID:         data["tenant_id"],
		PaymentCount:     parseInt64(data["payment_count"]),
		PaymentAmount:    parseInt64(data["payment_amount"]),
		PaymentFee:       parseInt64(data["payment_fee"]),
		ActualIncome:     parseInt64(data["actual_income"]),
		APICallCount:     parseInt64(data["api_call_count"]),
		TokenConsumed:    parseInt64(data["token_consumed"]),
		APICost:          parseInt64(data["api_cost"]),
		WithdrawalCount:  parseInt64(data["withdrawal_count"]),
		WithdrawalAmount: parseInt64(data["withdrawal_amount"]),
		RefundCount:      parseInt64(data["refund_count"]),
		RefundAmount:     parseInt64(data["refund_amount"]),
		GrossProfit:      parseInt64(data["gross_profit"]),
		NetProfit:        parseInt64(data["net_profit"]),
		ReconStatus:      ReconciliationStatus(data["recon_status"]),
		Discrepancy:      parseInt64(data["discrepancy"]),
		CreatedAt:        parseInt64(data["created_at"]),
	}
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
