package refund

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RefundStatus 退款状态
type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "pending"
	RefundStatusApproved  RefundStatus = "approved"
	RefundStatusRejected  RefundStatus = "rejected"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusCompleted RefundStatus = "completed"
	RefundStatusFailed    RefundStatus = "failed"
)

// RefundOrder 退款订单
type RefundOrder struct {
	ID              string       `json:"id"`
	UserID          string       `json:"user_id"`
	TenantID        string       `json:"tenant_id"`
	OrderNo         string       `json:"order_no"`          // 退款单号
	PaymentOrderNo  string       `json:"payment_order_no"`  // 原支付单号
	Amount          int64        `json:"amount"`            // 退款金额（分）
	Currency        string       `json:"currency"`
	Reason          string       `json:"reason"`            // 退款原因
	Status          RefundStatus `json:"status"`
	ChannelRefundNo string       `json:"channel_refund_no,omitempty"` // 渠道退款单号
	RejectReason    string       `json:"reject_reason,omitempty"`
	ReviewedBy      string       `json:"reviewed_by,omitempty"`
	ReviewedAt      *int64       `json:"reviewed_at,omitempty"`
	CompletedAt     *int64       `json:"completed_at,omitempty"`
	CreatedAt       int64        `json:"created_at"`
}

// RefundService 退款服务
type RefundService struct {
	logger *zap.Logger
	rdb    *redis.Client
}

// NewRefundService 创建退款服务
func NewRefundService(logger *zap.Logger, rdb *redis.Client) *RefundService {
	return &RefundService{logger: logger, rdb: rdb}
}

// CreateRefundOrder 创建退款申请
func (s *RefundService) CreateRefundOrder(ctx context.Context, order *RefundOrder) error {
	// 验证原支付订单
	paymentKey := fmt.Sprintf("payment:order:%s", order.PaymentOrderNo)
	paymentData, err := s.rdb.HGetAll(ctx, paymentKey).Result()
	if err != nil {
		return fmt.Errorf("get payment order: %w", err)
	}
	if len(paymentData) == 0 {
		return fmt.Errorf("payment order not found: %s", order.PaymentOrderNo)
	}

	// 检查支付订单状态
	if paymentData["status"] != "completed" {
		return fmt.Errorf("payment order status is not completed, cannot refund")
	}

	// 检查退款金额不超过实付金额
	actualAmount := parseInt64(paymentData["actual_amount"])
	if order.Amount > actualAmount {
		return fmt.Errorf("refund amount %d exceeds payment amount %d", order.Amount, actualAmount)
	}

	// 检查是否已有进行中的退款
	existingKey := fmt.Sprintf("refund:payment:%s", order.PaymentOrderNo)
	existing, _ := s.rdb.HGetAll(ctx, existingKey).Result()
	if len(existing) > 0 && existing["status"] != "rejected" && existing["status"] != "completed" && existing["status"] != "failed" {
		return fmt.Errorf("refund already exists for payment order %s", order.PaymentOrderNo)
	}

	// 检查余额是否充足（退款需要从用户余额中扣除）
	balanceKey := fmt.Sprintf("balance:%s:%s", order.TenantID, order.UserID)
	balance, err := s.rdb.Get(ctx, balanceKey).Int64()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("get balance: %w", err)
	}
	if balance < order.Amount {
		return fmt.Errorf("insufficient balance for refund: balance %d < refund %d", balance, order.Amount)
	}

	order.ID = uuid.New().String()
	order.OrderNo = fmt.Sprintf("RF%s%s", time.Now().Format("20060102150405"), generateShortID())
	order.Status = RefundStatusPending
	order.CreatedAt = time.Now().Unix()

	// 存储退款订单
	orderKey := fmt.Sprintf("refund:order:%s", order.OrderNo)
	if err := s.rdb.HSet(ctx, orderKey, map[string]interface{}{
		"id":               order.ID,
		"user_id":          order.UserID,
		"tenant_id":        order.TenantID,
		"order_no":         order.OrderNo,
		"payment_order_no": order.PaymentOrderNo,
		"amount":           order.Amount,
		"currency":         order.Currency,
		"reason":           order.Reason,
		"status":           string(order.Status),
		"created_at":       order.CreatedAt,
	}).Err(); err != nil {
		return fmt.Errorf("store refund order: %w", err)
	}

	// 用户退款订单索引
	userRefundsKey := fmt.Sprintf("refund:user:%s", order.UserID)
	s.rdb.ZAdd(ctx, userRefundsKey, &redis.Z{
		Score:  float64(order.CreatedAt),
		Member: order.OrderNo,
	})

	// 关联原支付订单
	s.rdb.HSet(ctx, existingKey, map[string]interface{}{
		"refund_order_no": order.OrderNo,
		"status":          string(order.Status),
	})

	s.logger.Info("refund order created",
		zap.String("order_no", order.OrderNo),
		zap.String("payment_order_no", order.PaymentOrderNo),
		zap.Int64("amount", order.Amount),
	)
	return nil
}

// ReviewRefundOrder 审核退款申请
func (s *RefundService) ReviewRefundOrder(ctx context.Context, orderNo, reviewerID string, approved bool, reason string) error {
	orderKey := fmt.Sprintf("refund:order:%s", orderNo)
	data, err := s.rdb.HGetAll(ctx, orderKey).Result()
	if err != nil {
		return fmt.Errorf("get refund order: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("refund order not found: %s", orderNo)
	}

	now := time.Now().Unix()
	updates := map[string]interface{}{
		"reviewed_by": reviewerID,
		"reviewed_at": now,
	}

	if approved {
		updates["status"] = string(RefundStatusApproved)

		// 从用户余额中扣除退款金额
		balanceKey := fmt.Sprintf("balance:%s:%s", data["tenant_id"], data["user_id"])
		amount := parseInt64(data["amount"])
		newBalance, err := s.rdb.DecrBy(ctx, balanceKey, amount).Result()
		if err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}
		if newBalance < 0 {
			// 回滚
			s.rdb.IncrBy(ctx, balanceKey, amount)
			return fmt.Errorf("insufficient balance after deduction")
		}

		// 标记为处理中，等待渠道退款
		updates["status"] = string(RefundStatusProcessing)

		s.logger.Info("refund approved and processing",
			zap.String("order_no", orderNo),
			zap.Int64("amount", amount),
		)
	} else {
		updates["status"] = string(RefundStatusRejected)
		updates["reject_reason"] = reason

		s.logger.Info("refund rejected",
			zap.String("order_no", orderNo),
			zap.String("reason", reason),
		)
	}

	s.rdb.HSet(ctx, orderKey, updates)

	// 更新关联的原支付订单
	existingKey := fmt.Sprintf("refund:payment:%s", data["payment_order_no"])
	s.rdb.HSet(ctx, existingKey, "status", updates["status"])

	return nil
}

// CompleteRefund 完成退款（渠道回调后调用）
func (s *RefundService) CompleteRefund(ctx context.Context, orderNo, channelRefundNo string, success bool) error {
	orderKey := fmt.Sprintf("refund:order:%s", orderNo)
	now := time.Now().Unix()

	updates := map[string]interface{}{
		"channel_refund_no": channelRefundNo,
	}

	if success {
		updates["status"] = string(RefundStatusCompleted)
		updates["completed_at"] = now
	} else {
		updates["status"] = string(RefundStatusFailed)

		// 退款失败，退还余额
		data, _ := s.rdb.HGetAll(ctx, orderKey).Result()
		amount := parseInt64(data["amount"])
		if amount > 0 {
			balanceKey := fmt.Sprintf("balance:%s:%s", data["tenant_id"], data["user_id"])
			s.rdb.IncrBy(ctx, balanceKey, amount)
		}
	}

	s.rdb.HSet(ctx, orderKey, updates)
	return nil
}

// GetRefundOrder 获取退款订单
func (s *RefundService) GetRefundOrder(ctx context.Context, orderNo string) (*RefundOrder, error) {
	orderKey := fmt.Sprintf("refund:order:%s", orderNo)
	data, err := s.rdb.HGetAll(ctx, orderKey).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("refund order not found: %s", orderNo)
	}

	return &RefundOrder{
		ID:              data["id"],
		UserID:          data["user_id"],
		TenantID:        data["tenant_id"],
		OrderNo:         data["order_no"],
		PaymentOrderNo:  data["payment_order_no"],
		Amount:          parseInt64(data["amount"]),
		Currency:        data["currency"],
		Reason:          data["reason"],
		Status:          RefundStatus(data["status"]),
		ChannelRefundNo: data["channel_refund_no"],
		RejectReason:    data["reject_reason"],
		ReviewedBy:      data["reviewed_by"],
		CreatedAt:       parseInt64(data["created_at"]),
	}, nil
}

// ListRefundOrders 获取用户退款订单列表
func (s *RefundService) ListRefundOrders(ctx context.Context, userID string, offset, limit int64) ([]*RefundOrder, error) {
	userRefundsKey := fmt.Sprintf("refund:user:%s", userID)
	orderNos, err := s.rdb.ZRevRange(ctx, userRefundsKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	var orders []*RefundOrder
	for _, orderNo := range orderNos {
		order, err := s.GetRefundOrder(ctx, orderNo)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}
	return orders, nil
}

// ListPendingRefunds 获取待审核退款列表
func (s *RefundService) ListPendingRefunds(ctx context.Context, tenantID string, offset, limit int64) ([]*RefundOrder, error) {
	// 使用集合存储待审核的退款订单号
	pendingKey := fmt.Sprintf("refund:pending:%s", tenantID)
	orderNos, err := s.rdb.ZRevRange(ctx, pendingKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	var orders []*RefundOrder
	for _, orderNo := range orderNos {
		order, err := s.GetRefundOrder(ctx, orderNo)
		if err != nil {
			continue
		}
		if order.Status == RefundStatusPending {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

// ==================== 辅助函数 ====================

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func generateShortID() string {
	id := uuid.New().String()
	return id[:8]
}
