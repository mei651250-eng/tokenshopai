package finance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ==================== 收款账号 ====================

// AccountType 账号类型
type AccountType string

const (
	AccountTypeAlipay     AccountType = "alipay"      // 支付宝
	AccountTypeWeChat     AccountType = "wechat"      // 微信
	AccountTypeBankCard   AccountType = "bank_card"   // 银行卡
	AccountTypePayPal     AccountType = "paypal"       // PayPal
	AccountTypePayoneer   AccountType = "payoneer"     // Payoneer
	AccountTypeWise       AccountType = "wise"         // Wise
	AccountTypeStripe     AccountType = "stripe"       // Stripe
	AccountTypeCrypto     AccountType = "crypto"       // 加密货币钱包
)

// SupportedAccountTypes 支持的收款账号类型
var SupportedAccountTypes = []AccountType{
	AccountTypeAlipay, AccountTypeWeChat, AccountTypeBankCard,
	AccountTypePayPal, AccountTypePayoneer, AccountTypeWise,
	AccountTypeStripe, AccountTypeCrypto,
}

// AccountTypeInfo 账号类型信息
var AccountTypeInfo = map[AccountType]map[string]string{
	AccountTypeAlipay:   {"name": "支付宝", "icon": "/icons/alipay.svg", "currency": "CNY"},
	AccountTypeWeChat:   {"name": "微信", "icon": "/icons/wechat_pay.svg", "currency": "CNY"},
	AccountTypeBankCard: {"name": "银行卡", "icon": "/icons/bank.svg", "currency": "CNY"},
	AccountTypePayPal:   {"name": "PayPal", "icon": "/icons/paypal.svg", "currency": "USD"},
	AccountTypePayoneer: {"name": "Payoneer", "icon": "/icons/payoneer.svg", "currency": "USD"},
	AccountTypeWise:     {"name": "Wise", "icon": "/icons/wise.svg", "currency": "USD"},
	AccountTypeStripe:   {"name": "Stripe", "icon": "/icons/stripe.svg", "currency": "USD"},
	AccountTypeCrypto:   {"name": "Crypto", "icon": "/icons/crypto.svg", "currency": "USDT"},
}

// ReceivingAccount 收款账号
type ReceivingAccount struct {
	ID           string      `json:"id"`
	UserID       string      `json:"user_id"`
	TenantID     string      `json:"tenant_id"`
	AccountType  AccountType `json:"account_type"`
	AccountName  string      `json:"account_name"`  // 账号持有人姓名
	AccountNo    string      `json:"account_no"`    // 账号/卡号（脱敏存储）
	BankName     string      `json:"bank_name,omitempty"`    // 银行名称（银行卡）
	BankBranch   string      `json:"bank_branch,omitempty"`  // 开户行（银行卡）
	QRCodeURL    string      `json:"qrcode_url,omitempty"`   // 收款码URL（支付宝/微信）
	WalletAddress string     `json:"wallet_address,omitempty"` // 钱包地址（加密货币）
	ChainType    string      `json:"chain_type,omitempty"`    // 链类型（加密货币）
	IsPrimary    bool        `json:"is_primary"`     // 是否默认
	Verified     bool        `json:"verified"`       // 是否已验证
	Enabled      bool        `json:"enabled"`        // 是否启用
	Label        string      `json:"label,omitempty"` // 备注
	CreatedAt    int64       `json:"created_at"`
	UpdatedAt    int64       `json:"updated_at"`
}

// ==================== 提现账户 ====================

// WithdrawalAccount 提现账户
type WithdrawalAccount struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id"`
	TenantID      string      `json:"tenant_id"`
	AccountType   AccountType `json:"account_type"`
	AccountName   string      `json:"account_name"`   // 持有人姓名
	AccountNo     string      `json:"account_no"`     // 账号/卡号
	BankName      string      `json:"bank_name,omitempty"`
	BankBranch    string      `json:"bank_branch,omitempty"`
	SwiftCode     string      `json:"swift_code,omitempty"`    // SWIFT代码（国际汇款）
	WalletAddress string      `json:"wallet_address,omitempty"` // 钱包地址
	ChainType     string      `json:"chain_type,omitempty"`
	IsPrimary     bool        `json:"is_primary"`
	Verified      bool        `json:"verified"`
	Label         string      `json:"label,omitempty"`
	CreatedAt     int64       `json:"created_at"`
	UpdatedAt     int64       `json:"updated_at"`
}

// ==================== 提现订单 ====================

// WithdrawalStatus 提现状态
type WithdrawalStatus string

const (
	WithdrawalStatusPending   WithdrawalStatus = "pending"    // 待审核
	WithdrawalStatusApproved  WithdrawalStatus = "approved"   // 审核通过
	WithdrawalStatusProcessing WithdrawalStatus = "processing" // 处理中
	WithdrawalStatusCompleted WithdrawalStatus = "completed"  // 已完成
	WithdrawalStatusRejected  WithdrawalStatus = "rejected"   // 已拒绝
	WithdrawalStatusFailed    WithdrawalStatus = "failed"     // 失败
	WithdrawalStatusCancelled WithdrawalStatus = "cancelled"  // 已取消
)

// WithdrawalOrder 提现订单
type WithdrawalOrder struct {
	ID              string            `json:"id"`
	UserID          string            `json:"user_id"`
	TenantID        string            `json:"tenant_id"`
	OrderNo         string            `json:"order_no"`
	AccountID       string            `json:"account_id"`        // 提现账户ID
	AccountType     AccountType       `json:"account_type"`
	AccountName     string            `json:"account_name"`
	AccountNo       string            `json:"account_no"`         // 脱敏
	Amount          int64             `json:"amount"`             // 提现金额（分）
	Currency        string            `json:"currency"`
	FeeAmount       int64             `json:"fee_amount"`         // 手续费（分）
	ActualAmount    int64             `json:"actual_amount"`      // 实际到账（分）
	Status          WithdrawalStatus  `json:"status"`
	Remark          string            `json:"remark,omitempty"`
	RejectReason    string            `json:"reject_reason,omitempty"`
	ReviewedBy      string            `json:"reviewed_by,omitempty"`
	ReviewedAt      *int64            `json:"reviewed_at,omitempty"`
	CompletedAt     *int64            `json:"completed_at,omitempty"`
	TxHash          string            `json:"tx_hash,omitempty"`  // 链上交易哈希
	CreatedAt       int64             `json:"created_at"`
}

// ==================== 财务服务 ====================

// FinanceService 财务服务
type FinanceService struct {
	logger *zap.Logger
	rdb    *redis.Client
}

// NewFinanceService 创建财务服务
func NewFinanceService(logger *zap.Logger, rdb *redis.Client) *FinanceService {
	return &FinanceService{logger: logger, rdb: rdb}
}

// --- 收款账号 ---

// CreateReceivingAccount 创建收款账号
func (s *FinanceService) CreateReceivingAccount(ctx context.Context, account *ReceivingAccount) error {
	account.ID = uuid.New().String()
	account.CreatedAt = time.Now().Unix()
	account.UpdatedAt = account.CreatedAt

	key := fmt.Sprintf("receiving:%s:%s", account.UserID, account.ID)
	if err := s.rdb.HSet(ctx, key, map[string]interface{}{
		"id":             account.ID,
		"user_id":        account.UserID,
		"tenant_id":      account.TenantID,
		"account_type":   string(account.AccountType),
		"account_name":   account.AccountName,
		"account_no":     maskAccountNo(account.AccountNo),
		"bank_name":      account.BankName,
		"bank_branch":    account.BankBranch,
		"qrcode_url":     account.QRCodeURL,
		"wallet_address": account.WalletAddress,
		"chain_type":     account.ChainType,
		"is_primary":     account.IsPrimary,
		"verified":       account.Verified,
		"enabled":        account.Enabled,
		"label":          account.Label,
		"created_at":     account.CreatedAt,
		"updated_at":     account.UpdatedAt,
	}).Err(); err != nil {
		return fmt.Errorf("store receiving account: %w", err)
	}

	// 用户收款账号列表索引
	listKey := fmt.Sprintf("receiving:list:%s", account.UserID)
	s.rdb.SAdd(ctx, listKey, account.ID)

	// 如果设为主账号，清除其他主账号标记
	if account.IsPrimary {
		s.clearOtherPrimary(ctx, account.UserID, "receiving", account.ID)
	}

	s.logger.Info("receiving account created",
		zap.String("user_id", account.UserID),
		zap.String("type", string(account.AccountType)),
	)
	return nil
}

// ListReceivingAccounts 获取收款账号列表
func (s *FinanceService) ListReceivingAccounts(ctx context.Context, userID string) ([]*ReceivingAccount, error) {
	listKey := fmt.Sprintf("receiving:list:%s", userID)
	ids, err := s.rdb.SMembers(ctx, listKey).Result()
	if err != nil {
		return nil, err
	}

	var accounts []*ReceivingAccount
	for _, id := range ids {
		key := fmt.Sprintf("receiving:%s:%s", userID, id)
		data, err := s.rdb.HGetAll(ctx, key).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		accounts = append(accounts, &ReceivingAccount{
			ID:            data["id"],
			UserID:        data["user_id"],
			TenantID:      data["tenant_id"],
			AccountType:   AccountType(data["account_type"]),
			AccountName:   data["account_name"],
			AccountNo:     data["account_no"],
			BankName:      data["bank_name"],
			BankBranch:    data["bank_branch"],
			QRCodeURL:     data["qrcode_url"],
			WalletAddress: data["wallet_address"],
			ChainType:     data["chain_type"],
			IsPrimary:     data["is_primary"] == "1",
			Verified:      data["verified"] == "1",
			Enabled:       data["enabled"] == "1",
			Label:         data["label"],
		})
	}
	return accounts, nil
}

// DeleteReceivingAccount 删除收款账号
func (s *FinanceService) DeleteReceivingAccount(ctx context.Context, userID, accountID string) error {
	key := fmt.Sprintf("receiving:%s:%s", userID, accountID)
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return err
	}
	listKey := fmt.Sprintf("receiving:list:%s", userID)
	s.rdb.SRem(ctx, listKey, accountID)
	return nil
}

// SetPrimaryReceivingAccount 设为默认收款账号
func (s *FinanceService) SetPrimaryReceivingAccount(ctx context.Context, userID, accountID string) error {
	key := fmt.Sprintf("receiving:%s:%s", userID, accountID)
	s.rdb.HSet(ctx, key, "is_primary", true, "updated_at", time.Now().Unix())
	s.clearOtherPrimary(ctx, userID, "receiving", accountID)
	return nil
}

// --- 提现账户 ---

// CreateWithdrawalAccount 创建提现账户
func (s *FinanceService) CreateWithdrawalAccount(ctx context.Context, account *WithdrawalAccount) error {
	account.ID = uuid.New().String()
	account.CreatedAt = time.Now().Unix()
	account.UpdatedAt = account.CreatedAt

	key := fmt.Sprintf("withdraw_account:%s:%s", account.UserID, account.ID)
	if err := s.rdb.HSet(ctx, key, map[string]interface{}{
		"id":             account.ID,
		"user_id":        account.UserID,
		"tenant_id":      account.TenantID,
		"account_type":   string(account.AccountType),
		"account_name":   account.AccountName,
		"account_no":     maskAccountNo(account.AccountNo),
		"bank_name":      account.BankName,
		"bank_branch":    account.BankBranch,
		"swift_code":     account.SwiftCode,
		"wallet_address": account.WalletAddress,
		"chain_type":     account.ChainType,
		"is_primary":     account.IsPrimary,
		"verified":       account.Verified,
		"label":          account.Label,
		"created_at":     account.CreatedAt,
		"updated_at":     account.UpdatedAt,
	}).Err(); err != nil {
		return fmt.Errorf("store withdrawal account: %w", err)
	}

	listKey := fmt.Sprintf("withdraw_account:list:%s", account.UserID)
	s.rdb.SAdd(ctx, listKey, account.ID)

	if account.IsPrimary {
		s.clearOtherPrimary(ctx, account.UserID, "withdraw_account", account.ID)
	}

	s.logger.Info("withdrawal account created",
		zap.String("user_id", account.UserID),
		zap.String("type", string(account.AccountType)),
	)
	return nil
}

// ListWithdrawalAccounts 获取提现账户列表
func (s *FinanceService) ListWithdrawalAccounts(ctx context.Context, userID string) ([]*WithdrawalAccount, error) {
	listKey := fmt.Sprintf("withdraw_account:list:%s", userID)
	ids, err := s.rdb.SMembers(ctx, listKey).Result()
	if err != nil {
		return nil, err
	}

	var accounts []*WithdrawalAccount
	for _, id := range ids {
		key := fmt.Sprintf("withdraw_account:%s:%s", userID, id)
		data, err := s.rdb.HGetAll(ctx, key).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		accounts = append(accounts, &WithdrawalAccount{
			ID:            data["id"],
			UserID:        data["user_id"],
			TenantID:      data["tenant_id"],
			AccountType:   AccountType(data["account_type"]),
			AccountName:   data["account_name"],
			AccountNo:     data["account_no"],
			BankName:      data["bank_name"],
			BankBranch:    data["bank_branch"],
			SwiftCode:     data["swift_code"],
			WalletAddress: data["wallet_address"],
			ChainType:     data["chain_type"],
			IsPrimary:     data["is_primary"] == "1",
			Verified:      data["verified"] == "1",
			Label:         data["label"],
		})
	}
	return accounts, nil
}

// DeleteWithdrawalAccount 删除提现账户
func (s *FinanceService) DeleteWithdrawalAccount(ctx context.Context, userID, accountID string) error {
	key := fmt.Sprintf("withdraw_account:%s:%s", userID, accountID)
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return err
	}
	listKey := fmt.Sprintf("withdraw_account:list:%s", userID)
	s.rdb.SRem(ctx, listKey, accountID)
	return nil
}

// --- 提现订单 ---

// CreateWithdrawalOrder 创建提现订单
func (s *FinanceService) CreateWithdrawalOrder(ctx context.Context, order *WithdrawalOrder) error {
	// 检查余额
	balanceKey := fmt.Sprintf("balance:%s:%s", order.TenantID, order.UserID)
	balance, err := s.rdb.Get(ctx, balanceKey).Int64()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("get balance: %w", err)
	}
	if balance < order.Amount {
		return fmt.Errorf("insufficient balance: %d < %d", balance, order.Amount)
	}

	// 冻结金额
	s.rdb.DecrBy(ctx, balanceKey, order.Amount)

	order.ID = uuid.New().String()
	order.OrderNo = fmt.Sprintf("WD%s%s", time.Now().Format("20060102150405"), generateFinanceShortID())
	order.Status = WithdrawalStatusPending
	order.CreatedAt = time.Now().Unix()

	// 计算手续费
	order.FeeAmount = s.calculateFee(order.Amount, order.AccountType)
	order.ActualAmount = order.Amount - order.FeeAmount

	// 存储订单
	orderKey := fmt.Sprintf("withdrawal:order:%s", order.OrderNo)
	if err := s.rdb.HSet(ctx, orderKey, map[string]interface{}{
		"id":            order.ID,
		"user_id":       order.UserID,
		"tenant_id":     order.TenantID,
		"order_no":      order.OrderNo,
		"account_id":    order.AccountID,
		"account_type":  string(order.AccountType),
		"account_name":  order.AccountName,
		"account_no":    order.AccountNo,
		"amount":        order.Amount,
		"currency":      order.Currency,
		"fee_amount":    order.FeeAmount,
		"actual_amount": order.ActualAmount,
		"status":        string(order.Status),
		"remark":        order.Remark,
		"created_at":    order.CreatedAt,
	}).Err(); err != nil {
		return fmt.Errorf("store withdrawal order: %w", err)
	}

	// 用户订单索引
	userOrdersKey := fmt.Sprintf("withdrawal:user:%s", order.UserID)
	s.rdb.ZAdd(ctx, userOrdersKey, &redis.Z{
		Score:  float64(order.CreatedAt),
		Member: order.OrderNo,
	})

	s.logger.Info("withdrawal order created",
		zap.String("order_no", order.OrderNo),
		zap.Int64("amount", order.Amount),
	)
	return nil
}

// ListWithdrawalOrders 获取提现订单列表
func (s *FinanceService) ListWithdrawalOrders(ctx context.Context, userID string, offset, limit int64) ([]*WithdrawalOrder, error) {
	userOrdersKey := fmt.Sprintf("withdrawal:user:%s", userID)
	orderNos, err := s.rdb.ZRevRange(ctx, userOrdersKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	var orders []*WithdrawalOrder
	for _, orderNo := range orderNos {
		order, err := s.GetWithdrawalOrder(ctx, orderNo)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}
	return orders, nil
}

// GetWithdrawalOrder 获取提现订单
func (s *FinanceService) GetWithdrawalOrder(ctx context.Context, orderNo string) (*WithdrawalOrder, error) {
	orderKey := fmt.Sprintf("withdrawal:order:%s", orderNo)
	data, err := s.rdb.HGetAll(ctx, orderKey).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("order not found: %s", orderNo)
	}
	return &WithdrawalOrder{
		ID:          data["id"],
		UserID:      data["user_id"],
		TenantID:    data["tenant_id"],
		OrderNo:     data["order_no"],
		AccountID:   data["account_id"],
		AccountType: AccountType(data["account_type"]),
		AccountName: data["account_name"],
		AccountNo:   data["account_no"],
		Amount:      parseInt64(data["amount"]),
		Currency:    data["currency"],
		FeeAmount:   parseInt64(data["fee_amount"]),
		ActualAmount: parseInt64(data["actual_amount"]),
		Status:      WithdrawalStatus(data["status"]),
		Remark:      data["remark"],
		RejectReason: data["reject_reason"],
	}, nil
}

// ReviewWithdrawalOrder 审核提现订单
func (s *FinanceService) ReviewWithdrawalOrder(ctx context.Context, orderNo, reviewerID string, approved bool, reason string) error {
	orderKey := fmt.Sprintf("withdrawal:order:%s", orderNo)
	now := time.Now().Unix()

	updates := map[string]interface{}{
		"reviewed_by": reviewerID,
		"reviewed_at": now,
	}

	if approved {
		updates["status"] = string(WithdrawalStatusApproved)
	} else {
		updates["status"] = string(WithdrawalStatusRejected)
		updates["reject_reason"] = reason

		// 拒绝时退还余额
		data, _ := s.rdb.HGetAll(ctx, orderKey).Result()
		if amount := parseInt64(data["amount"]); amount > 0 {
			balanceKey := fmt.Sprintf("balance:%s:%s", data["tenant_id"], data["user_id"])
			s.rdb.IncrBy(ctx, balanceKey, amount)
		}
	}

	s.rdb.HSet(ctx, orderKey, updates)
	return nil
}

// ==================== 辅助函数 ====================

func maskAccountNo(no string) string {
	if len(no) <= 4 {
		return no
	}
	return strings.Repeat("*", len(no)-4) + no[len(no)-4:]
}

func (s *FinanceService) clearOtherPrimary(ctx context.Context, userID, prefix, currentID string) {
	listKey := fmt.Sprintf("%s:list:%s", prefix, userID)
	ids, err := s.rdb.SMembers(ctx, listKey).Result()
	if err != nil {
		return
	}
	for _, id := range ids {
		if id == currentID {
			continue
		}
		key := fmt.Sprintf("%s:%s:%s", prefix, userID, id)
		s.rdb.HSet(ctx, key, "is_primary", false)
	}
}

func (s *FinanceService) calculateFee(amount int64, accountType AccountType) int64 {
	switch accountType {
	case AccountTypeBankCard:
		// 银行卡: 1元固定 + 0.1%
		return 100 + int64(float64(amount)*0.001)
	case AccountTypeAlipay, AccountTypeWeChat:
		// 支付宝/微信: 0.1%
		return int64(float64(amount) * 0.001)
	case AccountTypeCrypto:
		// 加密货币: 链上手续费固定5U
		return 500 // 5元
	default:
		// 其他: 0.5%
		return int64(float64(amount) * 0.005)
	}
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func generateFinanceShortID() string {
	b := make([]byte, 4)
	_, _ = strings.NewReader(uuid.New().String()).Read(b)
	return fmt.Sprintf("%x", b)
}
