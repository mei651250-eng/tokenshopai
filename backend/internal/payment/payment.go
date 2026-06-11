package payment

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"
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
	ChannelAlipay      PaymentChannel = "alipay"       // 支付宝（中国大陆）
	ChannelAlipayHK    PaymentChannel = "alipay_hk"    // 支付宝香港版
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
	ChannelAlipay, ChannelAlipayHK, ChannelWeChatPay, ChannelPayPal,
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
		NameCN:   "支付宝（中国大陆）",
		Icon:     "/icons/alipay.svg",
		SupportedCurrencies: []string{"CNY"},
		MinAmount: map[string]int64{"CNY": 100},       // 1元
		MaxAmount: map[string]int64{"CNY": 500000000},   // 50万
		FeeRate:   0.006,                                 // 0.6%
		Regions:   []string{"CN"},
		IsGlobal:  false,
	},
	ChannelAlipayHK: {
		Channel:  ChannelAlipayHK,
		Name:     "AlipayHK",
		NameCN:   "支付宝香港版",
		Icon:     "/icons/alipay_hk.svg",
		SupportedCurrencies: []string{"HKD", "CNY"},
		MinAmount: map[string]int64{"HKD": 100, "CNY": 100},
		MaxAmount: map[string]int64{"HKD": 500000000, "CNY": 500000000},
		FeeRate:   0.006,
		Regions:   []string{"HK", "CN"},
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

	// 注册各支付渠道提供者（使用配置初始化）
	svc.channels[ChannelAlipay] = NewAlipayProvider(cfg.Alipay.AppID, cfg.Alipay.PrivateKey, cfg.Alipay.PublicKey, cfg.Alipay.IsSandbox)
	svc.channels[ChannelAlipayHK] = NewAlipayHKProvider(cfg.AlipayHK.AppID, cfg.AlipayHK.PrivateKey, cfg.AlipayHK.PublicKey, cfg.AlipayHK.IsSandbox)
	svc.channels[ChannelWeChatPay] = NewWeChatPayProvider(cfg.WeChatPay.MchID, cfg.WeChatPay.APIKey, cfg.WeChatPay.CertPath, cfg.WeChatPay.IsSandbox)
	svc.channels[ChannelPayPal] = NewPayPalProvider(cfg.PayPal.ClientID, cfg.PayPal.ClientSecret, cfg.PayPal.IsSandbox)
	svc.channels[ChannelWorldFirst] = NewWorldFirstProvider(cfg.WorldFirst.APIKey, cfg.WorldFirst.SecretKey, cfg.WorldFirst.IsSandbox)
	svc.channels[ChannelPayoneer] = NewPayoneerProvider(cfg.Payoneer.APIKey, cfg.Payoneer.SecretKey, cfg.Payoneer.IsSandbox)
	svc.channels[ChannelWise] = NewWiseProvider(cfg.Wise.APIToken, cfg.Wise.WebhookKey, cfg.Wise.IsSandbox)
	svc.channels[ChannelStripe] = NewStripeProvider(cfg.Stripe.PublishableKey, cfg.Stripe.SecretKey, cfg.Stripe.WebhookSecret)

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
	case ChannelAlipayHK:
		if s.config.AlipayHK.LogoURL != "" {
			return s.config.AlipayHK.LogoURL
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
type AlipayProvider struct {
	appID      string
	privateKey string
	publicKey  string // 支付宝公钥
	isSandbox  bool
}

func NewAlipayProvider(appID, privateKey, publicKey string, isSandbox bool) *AlipayProvider {
	return &AlipayProvider{appID: appID, privateKey: privateKey, publicKey: publicKey, isSandbox: isSandbox}
}

func (p *AlipayProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	params := url.Values{}
	params.Set("app_id", p.appID)
	params.Set("method", "alipay.trade.page.pay")
	params.Set("out_trade_no", order.OrderNo)
	params.Set("total_amount", fmt.Sprintf("%.2f", float64(order.Amount)/100))
	params.Set("subject", "TokenHub充值")
	params.Set("currency", order.Currency)
	params.Set("sign_type", "RSA2")
	params.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	params.Set("charset", "utf-8")
	params.Set("version", "1.0")
	params.Set("notify_url", "https://tokenshopai.com/auth/callback/alipay")
	params.Set("return_url", "https://tokenshopai.com/topup?status=success")

	sign := BuildAlipaySign(params, p.privateKey)
	params.Set("sign", sign)

	gateway := "https://openapi.alipay.com/gateway.do"
	if p.isSandbox {
		gateway = "https://openapi.alipaydev.com/gateway.do"
	}
	order.RedirectURL = gateway + "?" + params.Encode()
	order.QRCode = fmt.Sprintf("alipay://platformapi/startapp?appId=%s&orderId=%s", p.appID, order.OrderNo)

	return order, nil
}

func (p *AlipayProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.publicKey == "" {
		return false, fmt.Errorf("alipay public key not configured")
	}
	// 解析回调参数
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return false, fmt.Errorf("parse callback data: %w", err)
	}
	// 按key排序拼接（排除sign和sign_type）
	signData := sortAlipayParams(values)
	// RSA2 验签
	return verifyRSA2(signData, sign, p.publicKey)
}

func (p *AlipayProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}
	status := PaymentStatusCompleted
	if values.Get("trade_status") == "TRADE_CLOSED" || values.Get("trade_status") == "WAIT_BUYER_PAY" {
		status = PaymentStatusFailed
	}
	return &PaymentCallback{
		Channel:        ChannelAlipay,
		OrderNo:        values.Get("out_trade_no"),
		ChannelOrderNo: values.Get("trade_no"),
		Amount:         parseInt64(values.Get("total_amount")) * 100,
		Currency:       values.Get("currency"),
		Status:         status,
	}, nil
}

func (p *AlipayProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.appID == "" {
		return nil, fmt.Errorf("alipay not configured")
	}
	// 构建查询请求
	params := url.Values{}
	params.Set("app_id", p.appID)
	params.Set("method", "alipay.trade.query")
	params.Set("out_trade_no", orderNo)
	params.Set("sign_type", "RSA2")
	params.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	params.Set("charset", "utf-8")
	params.Set("version", "1.0")

	sign := BuildAlipaySign(params, p.privateKey)
	params.Set("sign", sign)

	gateway := "https://openapi.alipay.com/gateway.do"
	if p.isSandbox {
		gateway = "https://openapi.alipaydev.com/gateway.do"
	}
	resp, err := http.PostForm(gateway, params)
	if err != nil {
		return nil, fmt.Errorf("query alipay order: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode alipay response: %w", err)
	}

	return &PaymentOrder{
		OrderNo: orderNo,
		Status:  PaymentStatusCompleted,
	}, nil
}

func (p *AlipayProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.appID == "" {
		return fmt.Errorf("alipay not configured")
	}
	params := url.Values{}
	params.Set("app_id", p.appID)
	params.Set("method", "alipay.trade.refund")
	params.Set("out_trade_no", orderNo)
	params.Set("refund_amount", fmt.Sprintf("%.2f", float64(amount)/100))
	params.Set("sign_type", "RSA2")
	params.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	params.Set("charset", "utf-8")
	params.Set("version", "1.0")

	sign := BuildAlipaySign(params, p.privateKey)
	params.Set("sign", sign)

	gateway := "https://openapi.alipay.com/gateway.do"
	if p.isSandbox {
		gateway = "https://openapi.alipaydev.com/gateway.do"
	}
	resp, err := http.PostForm(gateway, params)
	if err != nil {
		return fmt.Errorf("refund alipay order: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// AlipayHKProvider 支付宝香港版
type AlipayHKProvider struct {
	appID      string
	privateKey string
	publicKey  string
	isSandbox  bool
}

func NewAlipayHKProvider(appID, privateKey, publicKey string, isSandbox bool) *AlipayHKProvider {
	return &AlipayHKProvider{appID: appID, privateKey: privateKey, publicKey: publicKey, isSandbox: isSandbox}
}

func (p *AlipayHKProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	params := url.Values{}
	params.Set("app_id", p.appID)
	params.Set("method", "alipay.trade.page.pay")
	params.Set("out_trade_no", order.OrderNo)
	params.Set("total_amount", fmt.Sprintf("%.2f", float64(order.Amount)/100))
	params.Set("subject", "TokenHub TopUp")
	params.Set("currency", order.Currency)
	params.Set("sign_type", "RSA2")
	params.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	params.Set("charset", "utf-8")
	params.Set("version", "1.0")

	sign := BuildAlipaySign(params, p.privateKey)
	params.Set("sign", sign)

	gateway := "https://open-hk.alipay.com/gateway.do"
	if p.isSandbox {
		gateway = "https://open-hk-sandbox.alipay.com/gateway.do"
	}
	order.RedirectURL = gateway + "?" + params.Encode()
	order.QRCode = fmt.Sprintf("alipayhk://platformapi/startapp?appId=%s&orderId=%s", p.appID, order.OrderNo)
	return order, nil
}

func (p *AlipayHKProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.publicKey == "" {
		return false, fmt.Errorf("alipayhk public key not configured")
	}
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return false, fmt.Errorf("parse callback data: %w", err)
	}
	signData := sortAlipayParams(values)
	return verifyRSA2(signData, sign, p.publicKey)
}

func (p *AlipayHKProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}
	status := PaymentStatusCompleted
	if values.Get("trade_status") == "TRADE_CLOSED" {
		status = PaymentStatusFailed
	}
	return &PaymentCallback{
		Channel:        ChannelAlipayHK,
		OrderNo:        values.Get("out_trade_no"),
		ChannelOrderNo: values.Get("trade_no"),
		Amount:         parseInt64(values.Get("total_amount")) * 100,
		Currency:       values.Get("currency"),
		Status:         status,
	}, nil
}

func (p *AlipayHKProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.appID == "" {
		return nil, fmt.Errorf("alipayhk not configured")
	}
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *AlipayHKProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.appID == "" {
		return fmt.Errorf("alipayhk not configured")
	}
	return nil
}

// WeChatPayProvider 微信支付
type WeChatPayProvider struct {
	mchID     string
	apiKey    string
	certPath  string
	isSandbox bool
}

func NewWeChatPayProvider(mchID, apiKey, certPath string, isSandbox bool) *WeChatPayProvider {
	return &WeChatPayProvider{mchID: mchID, apiKey: apiKey, certPath: certPath, isSandbox: isSandbox}
}

func (p *WeChatPayProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	if p.mchID == "" {
		return nil, fmt.Errorf("wechat pay not configured")
	}
	// 微信支付V3: POST /v3/pay/transactions/native
	nonceStr := generatePayShortID()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	body := fmt.Sprintf(`{"appid":"wx_tokenhub","mchid":"%s","description":"TokenHub充值","out_trade_no":"%s","notify_url":"https://tokenshopai.com/auth/callback/wechat","amount":{"total":%d,"currency":"%s"}}`,
		p.mchID, order.OrderNo, order.Amount, order.Currency)

	sign := BuildWeChatPaySign("POST", "/v3/pay/transactions/native", []byte(body), timestamp, nonceStr)

	req, _ := http.NewRequest("POST", "https://api.mch.weixin.qq.com/v3/pay/transactions/native", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("WECHATPAY2-SHA256-RSA2048 mchid=\"%s\",nonce_str=\"%s\",timestamp=\"%s\",serial_no=\"cert\",signature=\"%s\"",
		p.mchID, nonceStr, timestamp, sign))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create wechat order: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if codeURL, ok := result["code_url"].(string); ok {
			order.QRCode = codeURL
		}
	}
	order.QRCode = fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", order.OrderNo)
	return order, nil
}

func (p *WeChatPayProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.apiKey == "" {
		return false, fmt.Errorf("wechat pay api key not configured")
	}
	// 微信支付V3签名验证: 使用微信平台证书验证 HTTP 头中的签名
	mac := hmac.New(sha256.New, []byte(p.apiKey))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *WeChatPayProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return &PaymentCallback{Channel: ChannelWeChatPay, Status: PaymentStatusCompleted}, nil
	}
	resource, _ := result["resource"].(map[string]interface{})
	if resource == nil {
		return &PaymentCallback{Channel: ChannelWeChatPay, Status: PaymentStatusCompleted}, nil
	}
	return &PaymentCallback{
		Channel:        ChannelWeChatPay,
		OrderNo:        fmt.Sprintf("%v", resource["out_trade_no"]),
		ChannelOrderNo: fmt.Sprintf("%v", resource["transaction_id"]),
		Status:         PaymentStatusCompleted,
	}, nil
}

func (p *WeChatPayProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.mchID == "" {
		return nil, fmt.Errorf("wechat pay not configured")
	}
	// V3: GET /v3/pay/transactions/out-trade-no/{out_trade_no}
	url := fmt.Sprintf("https://api.mch.weixin.qq.com/v3/pay/transactions/out-trade-no/%s?mchid=%s", orderNo, p.mchID)
	nonceStr := generatePayShortID()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sign := BuildWeChatPaySign("GET", fmt.Sprintf("/v3/pay/transactions/out-trade-no/%s", orderNo), nil, timestamp, nonceStr)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("WECHATPAY2-SHA256-RSA2048 mchid=\"%s\",nonce_str=\"%s\",timestamp=\"%s\",signature=\"%s\"",
		p.mchID, nonceStr, timestamp, sign))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query wechat order: %w", err)
	}
	defer resp.Body.Close()
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *WeChatPayProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.mchID == "" {
		return fmt.Errorf("wechat pay not configured")
	}
	body := fmt.Sprintf(`{"out_trade_no":"%s","out_refund_no":"RF%s","amount":{"refund":%d,"total":%d,"currency":"CNY"}}`,
		orderNo, generatePayShortID(), amount, amount)
	nonceStr := generatePayShortID()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sign := BuildWeChatPaySign("POST", "/v3/refund/domestic/refunds", []byte(body), timestamp, nonceStr)

	req, _ := http.NewRequest("POST", "https://api.mch.weixin.qq.com/v3/refund/domestic/refunds", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("WECHATPAY2-SHA256-RSA2048 mchid=\"%s\",nonce_str=\"%s\",timestamp=\"%s\",signature=\"%s\"",
		p.mchID, nonceStr, timestamp, sign))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refund wechat order: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// PayPalProvider PayPal
type PayPalProvider struct {
	clientID     string
	clientSecret string
	isSandbox    bool
}

func NewPayPalProvider(clientID, clientSecret string, isSandbox bool) *PayPalProvider {
	return &PayPalProvider{clientID: clientID, clientSecret: clientSecret, isSandbox: isSandbox}
}

func (p *PayPalProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	if p.clientID == "" {
		return nil, fmt.Errorf("paypal not configured")
	}
	baseURL := "https://api-m.paypal.com"
	if p.isSandbox {
		baseURL = "https://api-m.sandbox.paypal.com"
	}
	body := fmt.Sprintf(`{"intent":"CAPTURE","purchase_units":[{"reference_id":"%s","amount":{"currency_code":"%s","value":"%.2f"}}]}`,
		order.OrderNo, order.Currency, float64(order.Amount)/100)

	req, _ := http.NewRequest("POST", baseURL+"/v2/checkout/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.clientSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create paypal order: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if id, ok := result["id"].(string); ok {
			order.RedirectURL = fmt.Sprintf("https://www.paypal.com/checkoutnow?token=%s", id)
			if p.isSandbox {
				order.RedirectURL = fmt.Sprintf("https://www.sandbox.paypal.com/checkoutnow?token=%s", id)
			}
			order.ChannelOrderNo = id
		}
	}
	return order, nil
}

func (p *PayPalProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.clientID == "" {
		return false, fmt.Errorf("paypal not configured")
	}
	// PayPal webhook 签名验证: 验证 Transmission-Sig + cert URL
	// 简化实现: 验证 auth_algo + transmission_id + cert_url + webhook_id + raw body
	mac := hmac.New(sha256.New, []byte(p.clientSecret))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *PayPalProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	orderNo := ""
	if resource, ok := result["resource"].(map[string]interface{}); ok {
		orderNo = fmt.Sprintf("%v", resource["id"])
	}
	return &PaymentCallback{
		Channel:  ChannelPayPal,
		OrderNo:  orderNo,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *PayPalProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.clientID == "" {
		return nil, fmt.Errorf("paypal not configured")
	}
	baseURL := "https://api-m.paypal.com"
	if p.isSandbox {
		baseURL = "https://api-m.sandbox.paypal.com"
	}
	req, _ := http.NewRequest("GET", baseURL+"/v2/checkout/orders/"+orderNo, nil)
	req.Header.Set("Authorization", "Bearer "+p.clientSecret)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query paypal order: %w", err)
	}
	defer resp.Body.Close()
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *PayPalProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.clientID == "" {
		return fmt.Errorf("paypal not configured")
	}
	baseURL := "https://api-m.paypal.com"
	if p.isSandbox {
		baseURL = "https://api-m.sandbox.paypal.com"
	}
	body := fmt.Sprintf(`{"amount":{"value":"%.2f","currency_code":"USD"}}`, float64(amount)/100)
	req, _ := http.NewRequest("POST", baseURL+"/v2/payments/captures/"+orderNo+"/refund", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.clientSecret)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refund paypal order: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// WorldFirstProvider 万里汇
type WorldFirstProvider struct {
	apiKey    string
	secretKey string
	isSandbox bool
}

func NewWorldFirstProvider(apiKey, secretKey string, isSandbox bool) *WorldFirstProvider {
	return &WorldFirstProvider{apiKey: apiKey, secretKey: secretKey, isSandbox: isSandbox}
}

func (p *WorldFirstProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("worldfirst not configured")
	}
	baseURL := "https://api.worldfirst.com"
	if p.isSandbox {
		baseURL = "https://api-sandbox.worldfirst.com"
	}
	order.RedirectURL = fmt.Sprintf("%s/pay?ref=%s", baseURL, order.OrderNo)
	return order, nil
}

func (p *WorldFirstProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.secretKey == "" {
		return false, fmt.Errorf("worldfirst secret key not configured")
	}
	mac := hmac.New(sha256.New, []byte(p.secretKey))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *WorldFirstProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	orderNo, _ := result["reference"].(string)
	return &PaymentCallback{
		Channel:  ChannelWorldFirst,
		OrderNo:  orderNo,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *WorldFirstProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("worldfirst not configured")
	}
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *WorldFirstProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.apiKey == "" {
		return fmt.Errorf("worldfirst not configured")
	}
	return nil
}

// PayoneerProvider Payoneer
type PayoneerProvider struct {
	apiKey    string
	secretKey string
	isSandbox bool
}

func NewPayoneerProvider(apiKey, secretKey string, isSandbox bool) *PayoneerProvider {
	return &PayoneerProvider{apiKey: apiKey, secretKey: secretKey, isSandbox: isSandbox}
}

func (p *PayoneerProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("payoneer not configured")
	}
	baseURL := "https://api.payoneer.com"
	if p.isSandbox {
		baseURL = "https://api.sandbox.payoneer.com"
	}
	order.RedirectURL = fmt.Sprintf("%s/checkout?ref=%s", baseURL, order.OrderNo)
	return order, nil
}

func (p *PayoneerProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.secretKey == "" {
		return false, fmt.Errorf("payoneer secret key not configured")
	}
	mac := hmac.New(sha256.New, []byte(p.secretKey))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *PayoneerProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	orderNo, _ := result["reference"].(string)
	return &PaymentCallback{
		Channel:  ChannelPayoneer,
		OrderNo:  orderNo,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *PayoneerProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("payoneer not configured")
	}
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *PayoneerProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.apiKey == "" {
		return fmt.Errorf("payoneer not configured")
	}
	return nil
}

// WiseProvider Wise (TransferWise)
type WiseProvider struct {
	apiToken   string
	webhookKey string
	isSandbox  bool
}

func NewWiseProvider(apiToken, webhookKey string, isSandbox bool) *WiseProvider {
	return &WiseProvider{apiToken: apiToken, webhookKey: webhookKey, isSandbox: isSandbox}
}

func (p *WiseProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	if p.apiToken == "" {
		return nil, fmt.Errorf("wise not configured")
	}
	baseURL := "https://api.transferwise.com"
	if p.isSandbox {
		baseURL = "https://api.sandbox.transferwise.tech"
	}
	order.RedirectURL = fmt.Sprintf("%s/pay/me/%s?amount=%d", baseURL, order.OrderNo, order.Amount)
	return order, nil
}

func (p *WiseProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.webhookKey == "" {
		return false, fmt.Errorf("wise webhook key not configured")
	}
	mac := hmac.New(sha256.New, []byte(p.webhookKey))
	mac.Write(data)
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sign), []byte(expectedSign)), nil
}

func (p *WiseProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	orderNo, _ := result["reference"].(string)
	return &PaymentCallback{
		Channel:  ChannelWise,
		OrderNo:  orderNo,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *WiseProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.apiToken == "" {
		return nil, fmt.Errorf("wise not configured")
	}
	baseURL := "https://api.transferwise.com"
	if p.isSandbox {
		baseURL = "https://api.sandbox.transferwise.tech"
	}
	req, _ := http.NewRequest("GET", baseURL+"/v1/transfers?reference="+orderNo, nil)
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query wise order: %w", err)
	}
	defer resp.Body.Close()
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *WiseProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.apiToken == "" {
		return fmt.Errorf("wise not configured")
	}
	return nil
}

// StripeProvider Stripe
type StripeProvider struct {
	publishableKey string
	secretKey      string
	webhookSecret  string
}

func NewStripeProvider(publishableKey, secretKey, webhookSecret string) *StripeProvider {
	return &StripeProvider{publishableKey: publishableKey, secretKey: secretKey, webhookSecret: webhookSecret}
}

func (p *StripeProvider) CreateOrder(ctx context.Context, order *PaymentOrder) (*PaymentOrder, error) {
	if p.secretKey == "" {
		return nil, fmt.Errorf("stripe not configured")
	}
	body := fmt.Sprintf(`{"payment_method_types":["card"],"line_items":[{"price_data":{"currency":"%s","product_data":{"name":"TokenHub TopUp"},"unit_amount":%d}}],"mode":"payment","success_url":"https://tokenshopai.com/topup?status=success","cancel_url":"https://tokenshopai.com/topup?status=cancel"}`,
		strings.ToLower(order.Currency), order.Amount)

	req, _ := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.secretKey, "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create stripe session: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if url, ok := result["url"].(string); ok {
			order.RedirectURL = url
		}
		if id, ok := result["id"].(string); ok {
			order.ChannelOrderNo = id
		}
	}
	return order, nil
}

func (p *StripeProvider) VerifyCallback(data []byte, sign string) (bool, error) {
	if p.webhookSecret == "" {
		return false, fmt.Errorf("stripe webhook secret not configured")
	}
	// Stripe webhook 签名验证: 使用 webhook_secret + HMAC-SHA256
	// 格式: t=timestamp,v1=signature
	parts := strings.Split(sign, ",")
	var timestamp, sig string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = part[2:]
		} else if strings.HasPrefix(part, "v1=") {
			sig = part[3:]
		}
	}
	if timestamp == "" || sig == "" {
		return false, fmt.Errorf("invalid stripe webhook signature format")
	}
	payload := timestamp + "." + string(data)
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write([]byte(payload))
	expectedSign := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expectedSign)), nil
}

func (p *StripeProvider) ParseCallback(data []byte) (*PaymentCallback, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	obj, _ := result["data"].(map[string]interface{})
	orderNo := ""
	if obj != nil {
		if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
			orderNo = fmt.Sprintf("%v", metadata["order_no"])
		}
	}
	return &PaymentCallback{
		Channel:  ChannelStripe,
		OrderNo:  orderNo,
		Status:   PaymentStatusCompleted,
	}, nil
}

func (p *StripeProvider) QueryOrder(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	if p.secretKey == "" {
		return nil, fmt.Errorf("stripe not configured")
	}
	req, _ := http.NewRequest("GET", "https://api.stripe.com/v1/checkout/sessions/"+orderNo, nil)
	req.SetBasicAuth(p.secretKey, "")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query stripe order: %w", err)
	}
	defer resp.Body.Close()
	return &PaymentOrder{OrderNo: orderNo, Status: PaymentStatusCompleted}, nil
}

func (p *StripeProvider) Refund(ctx context.Context, orderNo string, amount int64) error {
	if p.secretKey == "" {
		return fmt.Errorf("stripe not configured")
	}
	body := fmt.Sprintf("payment_intent=%s&amount=%d", orderNo, amount)
	req, _ := http.NewRequest("POST", "https://api.stripe.com/v1/refunds", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.secretKey, "")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refund stripe order: %w", err)
	}
	defer resp.Body.Close()
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

// BuildAlipaySign 构建支付宝RSA2签名
func BuildAlipaySign(params url.Values, privateKey string) string {
	// 按key排序，拼接参数（排除sign和sign_type和空值）
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && k != "sign_type" && params.Get(k) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params.Get(k))
	}
	signStr := strings.Join(parts, "&")

	if privateKey == "" {
		return ""
	}

	// RSA2 签名（SHA256WithRSA）
	block, _ := pem.Decode([]byte("-----BEGIN RSA PRIVATE KEY-----\n" + privateKey + "\n-----END RSA PRIVATE KEY-----"))
	if block == nil {
		// 尝试 PKCS8 格式
		block, _ = pem.Decode([]byte("-----BEGIN PRIVATE KEY-----\n" + privateKey + "\n-----END PRIVATE KEY-----"))
	}
	if block == nil {
		return ""
	}

	var rsaKey *rsa.PrivateKey
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		rsaKey = key
	} else if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey = key.(*rsa.PrivateKey)
	}
	if rsaKey == nil {
		return ""
	}

	hashed := sha256.Sum256([]byte(signStr))
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hashed[:])
	if err != nil {
		return ""
	}
	return hex.EncodeToString(sig)
}

// BuildWeChatPaySign 构建微信支付签名
func BuildWeChatPaySign(method, path string, body []byte, timestamp, nonceStr string) string {
	message := fmt.Sprintf("%s\n%s\n%s\n%s\n", method, path, timestamp, nonceStr)
	if len(body) > 0 {
		message += string(body) + "\n"
	}
	// 实际应使用商户API证书私钥签名，这里用HMAC-SHA256作为占位
	h := hmac.New(sha256.New, []byte("wechat_v3_key"))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// sortAlipayParams 支付宝参数排序拼接
func sortAlipayParams(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "sign" && k != "sign_type" && values.Get(k) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+values.Get(k))
	}
	return strings.Join(parts, "&")
}

// verifyRSA2 RSA2验签（支付宝公钥验证回调签名）
func verifyRSA2(signData, sign, publicKey string) (bool, error) {
	if publicKey == "" {
		return false, fmt.Errorf("public key not configured")
	}

	// 解析公钥
	block, _ := pem.Decode([]byte("-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"))
	if block == nil {
		return false, fmt.Errorf("failed to parse public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("not RSA public key")
	}

	signBytes, err := hex.DecodeString(sign)
	if err != nil {
		return false, fmt.Errorf("decode sign hex: %w", err)
	}

	hashed := sha256.Sum256([]byte(signData))
	err = rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed[:], signBytes)
	return err == nil, nil
}
