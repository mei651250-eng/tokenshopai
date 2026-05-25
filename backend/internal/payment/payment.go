package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tokenhub/backend/internal/config"
)

// PaymentChannel 支付渠道
type PaymentChannel string

const (
	ChannelAlipay      PaymentChannel = "alipay"       // 支付宝
	ChannelWeChatPay   PaymentChannel = "wechat_pay"   // 微信支付
	ChannelPayPal      PaymentChannel = "paypal"        // PayPal
	ChannelWorldFirst  PaymentChannel = "worldfirst"    // 万里汇
	ChannelPayoneer    PaymentChannel = "payoneer"      // Payoneer
	ChannelWise        PaymentChannel = "wise"          // Wise
	ChannelAirwallex   PaymentChannel = "airwallex"    // Airwallex (万里汇同系)
	ChannelStripe      PaymentChannel = "stripe"        // Stripe (信用卡)
	ChannelCrypto      PaymentChannel = "crypto"        // 加密货币（USDT/USDC）
)

// SupportedChannels 支持的支付渠道
var SupportedChannels = []PaymentChannel{
	ChannelAlipay, ChannelWeChatPay, ChannelPayPal,
	ChannelWorldFirst, ChannelPayoneer, ChannelWise,
	ChannelAirwallex, ChannelStripe, ChannelCrypto,
}

// ChannelInfo 渠道信息
type ChannelInfo struct {
	Channel       PaymentChannel `json:"channel"`
	Name          string         `json:"name"`
	NameCN        string         `json:"name_cn,omitempty"`
	Icon          string         `json:"icon"`
	SupportedCurrencies []string `json:"supported_currencies"`
	MinAmount     map[string]int64 `json:"min_amount"` // 按币种最小金额（分）
	MaxAmount     map[string]int64 `json:"max_amount"`
	FeeRate       float64        `json:"fee_rate"`
	FeeFixed      map[string]int64 `json:"fee_fixed"` // 固定手续费
	Regions       []string       `json:"regions"`        // 支持的地区
	IsGlobal      bool           `json:"is_global"`
}

// ChannelRegistry 渠道注册表
var ChannelRegistry = map[PaymentChannel]ChannelInfo{
	ChannelAlipay: {
		Channel:  ChannelAlipay,
		Name:     "Alipay",
		NameCN:   "支付宝",
		Icon:     "/icons/alipay.svg",
		SupportedCurrencies: []string{"CNY"},
		MinAmount: map[string]int64{"CNY": 100},       // 1元
		MaxAmount: map[string]int64{"CNY": 500000000},   // 50万
		FeeRate:   0.006,                                 // 0.6%
		Regions:   []string{"CN"},
		IsGlobal:  false,
	},
	ChannelWeChatPay: {
		Channel:  ChannelWeChatPay,
		Name:     "WeChat Pay",
		NameCN:   "微信支付",
		Icon:     "/icons/wechat_pay.svg",
		SupportedCurrencies: []string{"CNY"},
		MinAmount: map[string]int64{"CNY": 100},
		MaxAmount: map[string]int64{"CNY": 500000000},
		FeeRate:   0.006,
		Regions:   []string{"CN"},
		IsGlobal:  false,
	},
	ChannelPayPal: {
		Channel:  ChannelPayPal,
		Name:     "PayPal",
		Icon:     "/icons/paypal.svg",
		SupportedCurrencies: []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD"},
		MinAmount: map[string]int64{"USD": 100, "EUR": 100},
		MaxAmount: map[string]int64{"USD": 100000000},
		FeeRate:   0.039,                                 // 3.9%
		FeeFixed:  map[string]int64{"USD": 30, "EUR": 35}, // 固定费用（分）
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
	ChannelWorldFirst: {
		Channel:  ChannelWorldFirst,
		Name:     "WorldFirst",
		NameCN:   "万里汇",
		Icon:     "/icons/worldfirst.svg",
		SupportedCurrencies: []string{"USD", "EUR", "GBP", "CNY", "JPY", "AUD", "CAD", "SGD", "HKD"},
		MinAmount: map[string]int64{"USD": 100, "CNY": 700},
		MaxAmount: map[string]int64{"USD": 10000000000},
		FeeRate:   0.01,
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
	ChannelPayoneer: {
		Channel:  ChannelPayoneer,
		Name:     "Payoneer",
		Icon:     "/icons/payoneer.svg",
		SupportedCurrencies: []string{"USD", "EUR", "GBP", "JPY", "AUD", "CAD", "MXN", "BRL"},
		MinAmount: map[string]int64{"USD": 100},
		MaxAmount: map[string]int64{"USD": 10000000000},
		FeeRate:   0.02,
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
	ChannelWise: {
		Channel:  ChannelWise,
		Name:     "Wise",
		Icon:     "/icons/wise.svg",
		SupportedCurrencies: []string{"USD", "EUR", "GBP", "JPY", "AUD", "CAD", "SGD", "HKD", "CHF", "NZD"},
		MinAmount: map[string]int64{"USD": 100, "EUR": 100},
		MaxAmount: map[string]int64{"USD": 10000000000},
		FeeRate:   0.007,
		FeeFixed:  map[string]int64{"USD": 40},
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
	ChannelAirwallex: {
		Channel:  ChannelAirwallex,
		Name:     "Airwallex",
		Icon:     "/icons/airwallex.svg",
		SupportedCurrencies: []string{"USD", "EUR", "GBP", "CNY", "JPY", "AUD", "CAD", "SGD", "HKD"},
		MinAmount: map[string]int64{"USD": 100},
		MaxAmount: map[string]int64{"USD": 10000000000},
		FeeRate:   0.012,
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
	ChannelStripe: {
		Channel:  ChannelStripe,
		Name:     "Stripe",
		Icon:     "/icons/stripe.svg",
		SupportedCurrencies: []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "SGD", "HKD"},
		MinAmount: map[string]int64{"USD": 50},
		MaxAmount: map[string]int64{"USD": 9999999999},
		FeeRate:   0.029,
		FeeFixed:  map[string]int64{"USD": 30},
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
	ChannelCrypto: {
		Channel:  ChannelCrypto,
		Name:     "Crypto (USDT/USDC)",
		NameCN:   "加密货币",
		Icon:     "/icons/crypto.svg",
		SupportedCurrencies: []string{"USDT", "USDC"},
		MinAmount: map[string]int64{"USDT": 1000},     // 10 USDT
		MaxAmount: map[string]int64{"USDT": 10000000000},
		FeeRate:   0.001,                                // 0.1% 网络手续费
		Regions:   []string{"GLOBAL"},
		IsGlobal:  true,
	},
}

// PaymentOrder 支付订单
type PaymentOrder struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	TenantID        string         `json:"tenant_id"`
	OrderNo         string         `json:"order_no"`
	Channel         PaymentChannel `json:"channel"`
	Amount          int64          `json:"amount"`            // 支付金额（分）
	Currency        string         `json:"currency"`
	FeeAmount       int64          `json:"fee_amount"`        // 手续费（分）
	ActualAmount    int64          `json:"actual_amount"`     // 实际到账（分）
	ToCurrency      string         `json:"to_currency"`       // 充值目标币种
	ExchangeRate    float64        `json:"exchange_rate"`     // 汇率
	Status          PaymentStatus  `json:"status"`
	ChannelOrderNo  string         `json:"channel_order_no,omitempty"` // 第三方订单号
	RedirectURL     string         `json:"redirect_url,omitempty"`     // 支付跳转URL
	QRCode          string         `json:"qr_code,omitempty"`          // 二维码内容
	CallbackData    string         `json:"callback_data,omitempty"`    // 回调原始数据
	Remark          string         `json:"remark,omitempty"`
	CreatedAt       int64          `json:"created_at"`
	PaidAt          *int64         `json:"paid_at,omitempty"`
	ExpiredAt       int64          `json:"expired_at"`
}

// PaymentStatus 支付状态
type PaymentStatus string

const (
	PaymentStatusCreated   PaymentStatus = "created"
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// PaymentCallback 支付回调
type PaymentCallback struct {
	Channel       PaymentChannel `json:"channel"`
	OrderNo       string         `json:"order_no"`
	ChannelOrderNo string        `json:"channel_order_no"`
	Amount        int64          `json:"amount"`
	Currency      string         `json:"currency"`
	Status        PaymentStatus  `json:"status"`
	Sign          string         `json:"sign"`
	RawData       string         `json:"raw_data"`
}

// PaymentService 支付服务
type PaymentService struct {
	logger   *zap.Logger
	rdb      *redis.Client
	channels map[PaymentChannel]PaymentProvider
	config   *config.PaymentConfig
}

// PaymentProvider 支付提供商接口
type PaymentProvider interface {
	CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error)
	VerifyCallback(data []byte, sign string) (bool, error)
	ParseCallback(data []byte) (*PaymentCallback, error)
	QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error)
	Refund(ctx context.Context, orderNo string, amount int64) error
}

// NewPaymentService 创建支付服务
func NewPaymentService(logger *zap.Logger, rdb *redis.Client, cfg *config.PaymentConfig) *PaymentService {
	svc := &PaymentService{
		logger:   logger,
		rdb:      rdb,
		channels: make(map[PaymentChannel]PaymentProvider),
		config:   cfg,
	}

	// 注册各支付渠道提供者
	svc.channels[ChannelAlipay] = &AlipayProvider{}
	svc.channels[ChannelWeChatPay] = &WeChatPayProvider{}
	svc.channels[ChannelPayPal] = &PayPalProvider{}
	svc.channels[ChannelWorldFirst] = &WorldFirstProvider{}
	svc.channels[ChannelPayoneer] = &PayoneerProvider{}
	svc.channels[ChannelWise] = &WiseProvider{}
	svc.channels[ChannelStripe] = &StripeProvider{}

	return svc
}

// CreatePaymentOrder 创建支付订单
func (s *PaymentService) CreatePaymentOrder(ctx context.Context, userID, tenantID string, channel PaymentChannel, amount int64, currency, toCurrency string) (*PaymentOrder, error) {
	// 验证渠道
	info, ok := ChannelRegistry[channel]
	if !ok {
		return nil, fmt.Errorf("unsupported payment channel: %s", channel)
	}

	// 验证币种
	currencySupported := false
	for _, c := range info.SupportedCurrencies {
		if c == currency {
			currencySupported = true
			break
		}
	}
	if !currencySupported {
		return nil, fmt.Errorf("currency %s not supported by channel %s", currency, channel)
	}

	// 验证金额
	if minAmt, ok := info.MinAmount[currency]; ok && amount < minAmt {
		return nil, fmt.Errorf("amount %d below minimum %d for %s", amount, minAmt, currency)
	}
	if maxAmt, ok := info.MaxAmount[currency]; ok && amount > maxAmt {
		return nil, fmt.Errorf("amount %d exceeds maximum %d for %s", amount, maxAmt, currency)
	}

	// 计算手续费
	feeAmount := int64(float64(amount)*info.FeeRate) + info.FeeFixed[currency]
	actualAmount := amount - feeAmount
	if actualAmount <= 0 {
		return nil, fmt.Errorf("amount too small after fee deduction")
	}

	// 计算汇率和到账金额
	exchangeRate := s.getExchangeRate(currency, toCurrency)
	creditAmount := int64(float64(actualAmount) * exchangeRate)

	orderNo := fmt.Sprintf("PAY%s%s", time.Now().Format("20060102150405"), generatePayShortID())

	order := &PaymentOrder{
		ID:           uuid.New().String(),
		UserID:       userID,
		TenantID:     tenantID,
		OrderNo:      orderNo,
		Channel:      channel,
		Amount:       amount,
		Currency:     currency,
		FeeAmount:    feeAmount,
		ActualAmount: creditAmount,
		ToCurrency:   toCurrency,
		ExchangeRate: exchangeRate,
		Status:       PaymentStatusCreated,
		CreatedAt:    time.Now().Unix(),
		ExpiredAt:    time.Now().Add(30 * time.Minute).Unix(),
	}

	// 调用渠道创建订单
	if provider, ok := s.channels[channel]; ok {
		var err error
		order, err = provider.CreateOrder(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("create channel order: %w", err)
		}
		order.Status = PaymentStatusPending
	}

	// 存储订单
	orderKey := fmt.Sprintf("payment:order:%s", order.OrderNo)
	s.storeOrder(ctx, orderKey, order)

	// 用户订单索引
	userOrdersKey := fmt.Sprintf("payment:user:%s", userID)
	s.rdb.ZAdd(ctx, userOrdersKey, &redis.Z{
		Score:  float64(order.CreatedAt),
		Member: order.OrderNo,
	})

	s.logger.Info("payment order created",
		zap.String("order_no", order.OrderNo),
		zap.String("channel", string(channel)),
		zap.Int64("amount", amount),
		zap.String("currency", currency),
	)

	return order, nil
}

// HandleCallback 处理支付回调
func (s *PaymentService) HandleCallback(ctx context.Context, channel PaymentChannel, data []byte, sign string) error {
	provider, ok := s.channels[channel]
	if !ok {
		return fmt.Errorf("unsupported channel: %s", channel)
	}

	// 验证签名
	valid, err := provider.VerifyCallback(data, sign)
	if err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid callback signature")
	}

	// 解析回调
	callback, err := provider.ParseCallback(data)
	if err != nil {
		return fmt.Errorf("parse callback: %w", err)
	}

	// 获取订单
	orderKey := fmt.Sprintf("payment:order:%s", callback.OrderNo)
	orderData, err := s.rdb.HGetAll(ctx, orderKey).Result()
	if err != nil || len(orderData) == 0 {
		return fmt.Errorf("order not found: %s", callback.OrderNo)
	}

	// 幂等检查
	currentStatus := orderData["status"]
	if currentStatus == string(PaymentStatusCompleted) {
		s.logger.Warn("duplicate callback ignored", zap.String("order_no", callback.OrderNo))
		return nil
	}

	// 更新订单状态
	now := time.Now().Unix()
	updates := map[string]interface{}{
		"status":          string(callback.Status),
		"channel_order_no": callback.ChannelOrderNo,
		"callback_data":   string(data),
		"paid_at":         now,
	}

	if callback.Status == PaymentStatusPaid || callback.Status == PaymentStatusCompleted {
		updates["status"] = string(PaymentStatusCompleted)

		// 充值到用户余额
		userID := orderData["user_id"]
		tenantID := orderData["tenant_id"]
		actualAmount := orderData["actual_amount"]

		balanceKey := fmt.Sprintf("balance:%s:%s", tenantID, userID)
		// 解析 actualAmount 并充值
		s.rdb.IncrBy(ctx, balanceKey, parseInt64(actualAmount))

		s.logger.Info("payment completed and credited",
			zap.String("order_no", callback.OrderNo),
			zap.String("channel", string(channel)),
		)
	}

	s.rdb.HSet(ctx, orderKey, updates)
	return nil
}

// GetPaymentOrder 获取支付订单
func (s *PaymentService) GetPaymentOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	orderKey := fmt.Sprintf("payment:order:%s", orderNo)
	data, err := s.rdb.HGetAll(ctx, orderKey).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("order not found: %s", orderNo)
	}

	return &PaymentOrder{
		ID:             data["id"],
		UserID:         data["user_id"],
		TenantID:       data["tenant_id"],
		OrderNo:        data["order_no"],
		Channel:        PaymentChannel(data["channel"]),
		Currency:       data["currency"],
		Status:         PaymentStatus(data["status"]),
		ChannelOrderNo: data["channel_order_no"],
		RedirectURL:    data["redirect_url"],
		QRCode:         data["qr_code"],
	}, nil
}

// ListPaymentOrders 获取用户支付订单
func (s *PaymentService) ListPaymentOrders(ctx context.Context, userID string, offset, limit int64) ([]*PaymentOrder, error) {
	userOrdersKey := fmt.Sprintf("payment:user:%s", userID)
	orderNos, err := s.rdb.ZRevRange(ctx, userOrdersKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	var orders []*PaymentOrder
	for _, orderNo := range orderNos {
		order, err := s.GetPaymentOrder(ctx, orderNo)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetAvailableChannels 获取可用支付渠道
func (s *PaymentService) GetAvailableChannels(currency string) []ChannelInfo {
	var available []ChannelInfo
	for _, ch := range SupportedChannels {
		info := ChannelRegistry[ch]
		for _, c := range info.SupportedCurrencies {
			if c == currency || currency == "" {
				// 用配置中的 LogoURL 覆盖默认 Icon
				info.Icon = s.getChannelLogo(ch, info.Icon)
				available = append(available, info)
				break
			}
		}
	}
	return available
}

// getChannelLogo 获取渠道 Logo，优先使用配置中的自定义 LogoURL
func (s *PaymentService) getChannelLogo(ch PaymentChannel, defaultIcon string) string {
	if s.config == nil {
		return defaultIcon
	}
	switch ch {
	case ChannelAlipay:
		if s.config.Alipay.LogoURL != "" {
			return s.config.Alipay.LogoURL
		}
	case ChannelWeChatPay:
		if s.config.WeChatPay.LogoURL != "" {
			return s.config.WeChatPay.LogoURL
		}
	case ChannelPayPal:
		if s.config.PayPal.LogoURL != "" {
			return s.config.PayPal.LogoURL
		}
	case ChannelWorldFirst:
		if s.config.WorldFirst.LogoURL != "" {
			return s.config.WorldFirst.LogoURL
		}
	case ChannelPayoneer:
		if s.config.Payoneer.LogoURL != "" {
			return s.config.Payoneer.LogoURL
		}
	case ChannelWise:
		if s.config.Wise.LogoURL != "" {
			return s.config.Wise.LogoURL
		}
	case ChannelStripe:
		if s.config.Stripe.LogoURL != "" {
			return s.config.Stripe.LogoURL
		}
	}
	return defaultIcon
}

// getExchangeRate 获取汇率（简化版）
func (s *PaymentService) getExchangeRate(from, to string) float64 {
	if from == to {
		return 1.0
	}
	rates := map[string]map[string]float64{
		"USD": {"CNY": 7.25, "EUR": 0.92, "JPY": 155.0, "GBP": 0.79, "KRW": 1350.0},
		"CNY": {"USD": 0.138, "EUR": 0.127, "JPY": 21.4},
		"EUR": {"USD": 1.087, "CNY": 7.88, "JPY": 168.5},
		"GBP": {"USD": 1.266, "CNY": 9.18, "EUR": 1.165},
	}
	if rateMap, ok := rates[from]; ok {
		if rate, ok := rateMap[to]; ok {
			return rate
		}
	}
	// 通过USD中转
	if r1, ok := rates[from]["USD"]; ok {
		if r2, ok := rates["USD"][to]; ok {
			return r1 * r2
		}
	}
	return 1.0
}

// storeOrder 存储订单到Redis
func (s *PaymentService) storeOrder(ctx context.Context, key string, order *PaymentOrder) {
	s.rdb.HSet(ctx, key, map[string]interface{}{
		"id":             order.ID,
		"user_id":        order.UserID,
		"tenant_id":      order.TenantID,
		"order_no":       order.OrderNo,
		"channel":        string(order.Channel),
		"amount":         order.Amount,
		"currency":       order.Currency,
		"fee_amount":     order.FeeAmount,
		"actual_amount":  order.ActualAmount,
		"to_currency":    order.ToCurrency,
		"exchange_rate":  order.ExchangeRate,
		"status":         string(order.Status),
		"redirect_url":   order.RedirectURL,
		"qr_code":        order.QRCode,
		"created_at":     order.CreatedAt,
		"expired_at":     order.ExpiredAt,
	})
}

// ==================== 支付渠道提供者实现 ====================

// AlipayProvider 支付宝
type AlipayProvider struct{}

func (p *AlipayProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// 实际实现：调用支付宝SDK alipay.trade.page.pay / alipay.trade.app.pay
	// 这里生成模拟的支付链接和二维码
	params := url.Values{}
	params.Set("app_id", "ALIPAY_APP_ID")
	params.Set("method", "alipay.trade.page.pay")
	params.Set("out_trade_no", order.OrderNo)
	params.Set("total_amount", fmt.Sprintf("%.2f", float64(order.Amount)/100))
	params.Set("subject", "TokenHub充值")
	params.Set("currency", order.Currency)

	order.RedirectURL = "https://openapi.alipay.com/gateway.do?" + params.Encode()
	order.QRCode = fmt.Sprintf("alipay://platformapi/startapp?orderId=%s", order.OrderNo)

	return order, nil
}

func (p *AlipayProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	// 支付宝RSA2签名验证
	// 实际：使用支付宝公钥验证签名
	// go-alipay SDK: alipay.VerifySign()
	return true, nil
}

func (p *AlipayProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}
	return &PaymentCallback{
		Channel:        ChannelAlipay,
		OrderNo:        values.Get("out_trade_no"),
		ChannelOrderNo: values.Get("trade_no"),
		Amount:         parseInt64(values.Get("total_amount")) * 100,
		Currency:       values.Get("currency"),
		Status:         PaymentStatusCompleted,
	}, nil
}

func (p *AlipayProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *AlipayProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// WeChatPayProvider 微信支付
type WeChatPayProvider struct{}

func (p *WeChatPayProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// 实际实现：调用微信支付V3 API /pay/transactions/native (扫码) 或 /pay/transactions/app (APP)
	// 使用 wechatpay-go SDK
	order.QRCode = fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", order.OrderNo)
	return order, nil
}

func (p *WeChatPayProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	// 微信支付V3签名验证: HTTP头中 Wechatpay-Signature
	// 使用微信平台证书验证
	return true, nil
}

func (p *WeChatPayProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	return &PaymentCallback{
		Channel:  ChannelWeChatPay,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *WeChatPayProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *WeChatPayProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// PayPalProvider PayPal
type PayPalProvider struct{}

func (p *PayPalProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// PayPal Orders V2 API: POST /v2/checkout/orders
	// 使用 paypal-go-sdk
	order.RedirectURL = fmt.Sprintf("https://www.paypal.com/checkoutnow?token=%s", order.OrderNo)
	return order, nil
}

func (p *PayPalProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	// PayPal webhook签名验证
	return true, nil
}

func (p *PayPalProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	return &PaymentCallback{
		Channel:  ChannelPayPal,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *PayPalProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *PayPalProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// WorldFirstProvider 万里汇
type WorldFirstProvider struct{}

func (p *WorldFirstProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// WorldFirst API (蚂蚁集团): 创建收款单
	// POST /api/v3/payments/create
	order.RedirectURL = fmt.Sprintf("https://www.worldfirst.com/pay?ref=%s", order.OrderNo)
	return order, nil
}

func (p *WorldFirstProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	// WorldFirst签名验证: HMAC-SHA256
	mac := hmac.New(sha256.New, []byte("worldfirst_secret"))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *WorldFirstProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	return &PaymentCallback{
		Channel:  ChannelWorldFirst,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *WorldFirstProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *WorldFirstProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// PayoneerProvider Payoneer
type PayoneerProvider struct{}

func (p *PayoneerProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// Payoneer API: POST /api/v4/charges
	order.RedirectURL = fmt.Sprintf("https://pay.payoneer.com/checkout?ref=%s", order.OrderNo)
	return order, nil
}

func (p *PayoneerProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	return true, nil
}

func (p *PayoneerProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	return &PaymentCallback{
		Channel:  ChannelPayoneer,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *PayoneerProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *PayoneerProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// WiseProvider Wise (TransferWise)
type WiseProvider struct{}

func (p *WiseProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// Wise API: POST /v1/quotes -> POST /v1/transfers
	order.RedirectURL = fmt.Sprintf("https://wise.com/pay/me/%s?amount=%d", order.OrderNo, order.Amount)
	return order, nil
}

func (p *WiseProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	// Wise webhook签名验证
	mac := hmac.New(sha256.New, []byte("wise_webhook_secret"))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *WiseProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	return &PaymentCallback{
		Channel:  ChannelWise,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *WiseProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *WiseProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// StripeProvider Stripe
type StripeProvider struct{}

func (p *StripeProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	// Stripe Checkout Session: POST /v1/checkout/sessions
	// 使用 stripe-go SDK
	order.RedirectURL = fmt.Sprintf("https://checkout.stripe.com/c/pay/%s", order.OrderNo)
	return order, nil
}

func (p *StripeProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	// Stripe webhook签名验证
	return true, nil
}

func (p *StripeProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	return &PaymentCallback{
		Channel:  ChannelStripe,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *StripeProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return nil, nil
}

func (p *StripeProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	return nil
}

// ==================== 辅助函数 ====================

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func generatePayShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// BuildAlipaySign 构建支付宝签名（简化）
func BuildAlipaySign(params url.Values, privateKey string) string {
	// 按key排序，拼接参数，RSA2签名
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && k != "sign_type" && params.Get(k) != "" {
			keys = append(keys, k)
		}
	}
	// sort.Strings(keys) // 实际需排序
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params.Get(k))
	}
	signStr := strings.Join(parts, "&")
	_ = signStr // 实际RSA2签名
	return "simulated_sign"
}

// BuildWeChatPaySign 构建微信支付签名
func BuildWeChatPaySign(method, path string, body []byte, timestamp, nonceStr string) string {
	message := fmt.Sprintf("%s\n%s\n%d\n%s\n", method, path, time.Now().Unix(), nonceStr)
	if len(body) > 0 {
		message += string(body) + "\n"
	}
	mac := hmac.New(sha256.New, []byte("wechat_v3_key"))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// Ensure unused import is referenced
var _ = http.MethodPost
var _ = url.Values{}
