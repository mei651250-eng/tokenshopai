package wallet

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/crypto/sha3"
	"go.uber.org/zap"
)

// WalletType 钱包类型
type WalletType string

const (
	WalletTypeMetaMask   WalletType = "metamask"
	WalletTypeTrustWallet WalletType = "trustwallet"
	WalletTypeWalletConnect WalletType = "walletconnect"
	WalletTypePhantom    WalletType = "phantom"    // Solana
	WalletTypeCoinbase   WalletType = "coinbase"
	WalletTypeOKX        WalletType = "okx_wallet"
	WalletTypeBitget     WalletType = "bitget"
	WalletTypeKeplr      WalletType = "keplr"       // Cosmos
	WalletTypePhantomEVM WalletType = "phantom_evm"
	WalletTypeRabby      WalletType = "rabby"
)

// SupportedWalletTypes 支持的钱包类型
var SupportedWalletTypes = []WalletType{
	WalletTypeMetaMask, WalletTypeTrustWallet, WalletTypeWalletConnect,
	WalletTypePhantom, WalletTypeCoinbase, WalletTypeOKX,
	WalletTypeBitget, WalletTypeKeplr, WalletTypePhantomEVM, WalletTypeRabby,
}

// ChainType 链类型
type ChainType string

const (
	ChainEthereum ChainType = "ethereum"
	ChainBSC      ChainType = "bsc"
	ChainPolygon  ChainType = "polygon"
	ChainArbitrum ChainType = "arbitrum"
	ChainOptimism ChainType = "optimism"
	ChainTron     ChainType = "tron"
	ChainSolana   ChainType = "solana"
	ChainAvalanche ChainType = "avalanche"
)

// CryptoCurrency 加密货币类型
type CryptoCurrency string

const (
	CryptoUSDT  CryptoCurrency = "USDT"
	CryptoUSDC  CryptoCurrency = "USDC"
	CryptoETH   CryptoCurrency = "ETH"
	CryptoBTC   CryptoCurrency = "BTC"
	CryptoSOL   CryptoCurrency = "SOL"
	CryptoBNB   CryptoCurrency = "BNB"
	CryptoTRX   CryptoCurrency = "TRX"
)

// SupportedCryptoCurrencies 支持的加密货币
var SupportedCryptoCurrencies = []CryptoCurrency{
	CryptoUSDT, CryptoUSDC, CryptoETH, CryptoBTC, CryptoSOL, CryptoBNB, CryptoTRX,
}

// USDT/USDC 支持的链网络
var StablecoinChainMap = map[CryptoCurrency][]ChainType{
	CryptoUSDT: {ChainEthereum, ChainBSC, ChainPolygon, ChainArbitrum, ChainOptimism, ChainTron, ChainAvalanche, ChainSolana},
	CryptoUSDC: {ChainEthereum, ChainBSC, ChainPolygon, ChainArbitrum, ChainOptimism, ChainAvalanche, ChainSolana},
}

// WalletBinding 钱包绑定记录
type WalletBinding struct {
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	TenantID   string      `json:"tenant_id"`
	WalletType WalletType  `json:"wallet_type"`
	Address    string      `json:"address"`
	ChainType  ChainType   `json:"chain_type"`
	IsPrimary  bool        `json:"is_primary"`
	Verified   bool        `json:"verified"`
	BindAt     int64       `json:"bind_at"`
	UnbindAt   *int64      `json:"unbind_at,omitempty"`
	Label      string      `json:"label,omitempty"`
}

// CryptoDepositOrder 加密货币充值订单
type CryptoDepositOrder struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	TenantID        string         `json:"tenant_id"`
	OrderNo         string         `json:"order_no"`
	Currency        CryptoCurrency `json:"currency"`
	ChainType       ChainType      `json:"chain_type"`
	Amount          string         `json:"amount"`            // 充值金额（字符串，避免精度问题）
	AmountReceived  string         `json:"amount_received"`   // 实际到账金额
	ToAddress       string         `json:"to_address"`        // 平台收款地址
	FromAddress     string         `json:"from_address"`       // 用户付款地址
	TxHash          string         `json:"tx_hash"`            // 链上交易哈希
	Status          DepositStatus  `json:"status"`
	Confirmations   int            `json:"confirmations"`
	RequiredConfirm int            `json:"required_confirm"`
	ExchangeRate    string         `json:"exchange_rate"`      // 成交汇率
	FiatAmount      int64          `json:"fiat_amount"`        // 法币等值金额（分）
	FiatCurrency    string         `json:"fiat_currency"`      // 法币类型
	CreatedAt       int64          `json:"created_at"`
	ConfirmedAt     *int64         `json:"confirmed_at,omitempty"`
	ExpiredAt       int64          `json:"expired_at"`
}

// DepositStatus 充值状态
type DepositStatus string

const (
	DepositStatusPending   DepositStatus = "pending"
	DepositStatusDetecting DepositStatus = "detecting"
	DepositStatusConfirming DepositStatus = "confirming"
	DepositStatusCompleted DepositStatus = "completed"
	DepositStatusFailed    DepositStatus = "failed"
	DepositStatusExpired   DepositStatus = "expired"
)

// VerifyChallenge 验证挑战（用于钱包签名验证）
type VerifyChallenge struct {
	Challenge string `json:"challenge"`
	ExpiresAt int64  `json:"expires_at"`
}

// WalletService 钱包服务
type WalletService struct {
	logger    *zap.Logger
	rdb       *redis.Client
	platformAddrs map[ChainType]string // 平台收款地址
}

// NewWalletService 创建钱包服务
func NewWalletService(logger *zap.Logger, rdb *redis.Client, platformAddrs map[ChainType]string) *WalletService {
	return &WalletService{
		logger:        logger,
		rdb:           rdb,
		platformAddrs: platformAddrs,
	}
}

// GenerateVerifyChallenge 生成钱包验证挑战
// 用户需要用钱包对该挑战消息进行签名，以证明地址所有权
func (s *WalletService) GenerateVerifyChallenge(ctx context.Context, address string) (*VerifyChallenge, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	challenge := fmt.Sprintf(
		"TokenHub Wallet Verification\n\nPlease sign this message to verify your wallet ownership.\n\nWallet: %s\nNonce: %s\nTimestamp: %d\n\nThis signature will not trigger any blockchain transaction or cost any gas fee.",
		address,
		hex.EncodeToString(nonce),
		time.Now().Unix(),
	)

	expiresAt := time.Now().Add(10 * time.Minute).Unix()

	// 存入Redis，10分钟过期
	key := fmt.Sprintf("wallet:challenge:%s", strings.ToLower(address))
	if err := s.rdb.Set(ctx, key, challenge, 10*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("store challenge: %w", err)
	}

	return &VerifyChallenge{
		Challenge: challenge,
		ExpiresAt: expiresAt,
	}, nil
}

// VerifySignature 验证钱包签名（EIP-191 personal_sign）
func (s *WalletService) VerifySignature(ctx context.Context, address, signature string) (bool, error) {
	key := fmt.Sprintf("wallet:challenge:%s", strings.ToLower(address))
	challenge, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, fmt.Errorf("challenge expired or not found")
	}
	if err != nil {
		return false, fmt.Errorf("get challenge: %w", err)
	}

	// 解码签名
	sigHex := signature
	if strings.HasPrefix(sigHex, "0x") {
		sigHex = sigHex[2:]
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature hex: %w", err)
	}
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65, got %d", len(sigBytes))
	}

	// 构造 EIP-191 个人签名消息
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(challenge), challenge)
	msgHash := keccak256Hash([]byte(prefixedMsg))

	// 提取 r, s, v
	r := new(big.Int).SetBytes(sigBytes[:32])
	sVal := new(big.Int).SetBytes(sigBytes[32:64])
	v := sigBytes[64]
	if v < 27 {
		v += 27
	}

	// 尝试恢复公钥并比对地址
	matched := false
	for _, recoveryID := range []byte{27, 28} {
		pubKey, err := recoverEthPubKey(msgHash, r, sVal, recoveryID)
		if err != nil {
			continue
		}
		recoveredAddr := pubKeyToAddress(pubKey)
		if strings.EqualFold(recoveredAddr, address) {
			matched = true
			break
		}
	}

	// 无论验证成功与否，删除 challenge（防重放）
	s.rdb.Del(ctx, key)

	if !matched {
		return false, nil
	}
	return true, nil
}

// VerifySignatureEVM EVM链签名验证
func (s *WalletService) VerifySignatureEVM(challenge, signature string, expectedAddr *ecdsa.PublicKey) bool {
	sigHex := signature
	if strings.HasPrefix(sigHex, "0x") {
		sigHex = sigHex[2:]
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != 65 {
		return false
	}

	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(challenge), challenge)
	msgHash := keccak256Hash([]byte(prefixedMsg))

	r := new(big.Int).SetBytes(sigBytes[:32])
	sVal := new(big.Int).SetBytes(sigBytes[32:64])
	v := sigBytes[64]
	if v < 27 {
		v += 27
	}

	expectedAddrHex := pubKeyToAddress(expectedAddr)

	for _, recoveryID := range []byte{27, 28} {
		pubKey, err := recoverEthPubKey(msgHash, r, sVal, recoveryID)
		if err != nil {
			continue
		}
		if strings.EqualFold(pubKeyToAddress(pubKey), expectedAddrHex) {
			return true
		}
	}
	return false
}

// keccak256Hash 计算 Keccak-256 哈希
func keccak256Hash(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// recoverEthPubKey 从签名恢复 secp256k1 公钥
func recoverEthPubKey(msgHash []byte, r, s *big.Int, v byte) (*ecdsa.PublicKey, error) {
	// 构造紧凑签名格式: [recoveryID + 27] [r 32字节] [s 32字节]
	recoveryID := v - 27
	if recoveryID > 3 {
		return nil, fmt.Errorf("invalid recovery id: %d", recoveryID)
	}

	// 将 r, s 编码为 32 字节大端
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	s.FillBytes(sBytes)

	// btcec.RecoverCompact 期望格式: [recovery flag byte] [r 32字节] [s 32字节]
	// recovery flag = 27 + recoveryID (0 or 1 for uncompressed)
	compactSig := make([]byte, 65)
	compactSig[0] = 27 + recoveryID
	copy(compactSig[1:33], rBytes)
	copy(compactSig[33:65], sBytes)

	pubKey, _, err := btcec.RecoverCompact(compactSig, msgHash)
	if err != nil {
		return nil, fmt.Errorf("recover pubkey: %w", err)
	}

	return pubKey.ToECDSA(), nil
}

// pubKeyToAddress 从公钥推导 Ethereum 地址
func pubKeyToAddress(pubKey *ecdsa.PublicKey) string {
	// 未压缩公钥: 04 + x(32) + y(32)
	pubBytes := make([]byte, 64)
	x := pubKey.X.Bytes()
	y := pubKey.Y.Bytes()
	copy(pubBytes[32-len(x):32], x)
	copy(pubBytes[64-len(y):64], y)

	hash := keccak256Hash(pubBytes)
	return hex.EncodeToString(hash[12:]) // 取后20字节
}

// BindWallet 绑定钱包
func (s *WalletService) BindWallet(ctx context.Context, binding *WalletBinding) error {
	// 检查是否已绑定
	bindKey := fmt.Sprintf("wallet:binding:%s:%s", binding.UserID, strings.ToLower(binding.Address))
	exists, err := s.rdb.Exists(ctx, bindKey).Result()
	if err != nil {
		return fmt.Errorf("check binding: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("wallet already bound to this user")
	}

	binding.ID = uuid.New().String()
	binding.BindAt = time.Now().Unix()
	binding.Verified = true

	// 存储绑定信息
	if err := s.rdb.HSet(ctx, bindKey, map[string]interface{}{
		"id":          binding.ID,
		"user_id":     binding.UserID,
		"tenant_id":   binding.TenantID,
		"wallet_type": string(binding.WalletType),
		"address":     binding.Address,
		"chain_type":  string(binding.ChainType),
		"is_primary":  binding.IsPrimary,
		"verified":    binding.Verified,
		"bind_at":     binding.BindAt,
		"label":       binding.Label,
	}).Err(); err != nil {
		return fmt.Errorf("store binding: %w", err)
	}

	// 用户钱包列表
	listKey := fmt.Sprintf("wallet:user:%s", binding.UserID)
	s.rdb.SAdd(ctx, listKey, strings.ToLower(binding.Address))

	s.logger.Info("wallet bound",
		zap.String("user_id", binding.UserID),
		zap.String("wallet_type", string(binding.WalletType)),
		zap.String("address", binding.Address),
	)

	return nil
}

// UnbindWallet 解绑钱包
func (s *WalletService) UnbindWallet(ctx context.Context, userID, address string) error {
	bindKey := fmt.Sprintf("wallet:binding:%s:%s", userID, strings.ToLower(address))
	if err := s.rdb.Del(ctx, bindKey).Err(); err != nil {
		return fmt.Errorf("delete binding: %w", err)
	}

	listKey := fmt.Sprintf("wallet:user:%s", userID)
	s.rdb.SRem(ctx, listKey, strings.ToLower(address))

	return nil
}

// ListUserWallets 获取用户绑定的钱包列表
func (s *WalletService) ListUserWallets(ctx context.Context, userID string) ([]*WalletBinding, error) {
	listKey := fmt.Sprintf("wallet:user:%s", userID)
	addresses, err := s.rdb.SMembers(ctx, listKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get wallet list: %w", err)
	}

	var wallets []*WalletBinding
	for _, addr := range addresses {
		bindKey := fmt.Sprintf("wallet:binding:%s:%s", userID, strings.ToLower(addr))
		data, err := s.rdb.HGetAll(ctx, bindKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		wallets = append(wallets, &WalletBinding{
			ID:         data["id"],
			UserID:     data["user_id"],
			TenantID:   data["tenant_id"],
			WalletType: WalletType(data["wallet_type"]),
			Address:    data["address"],
			ChainType:  ChainType(data["chain_type"]),
			IsPrimary:  data["is_primary"] == "1",
			Verified:   data["verified"] == "1",
			BindAt:     int64(0), // 从data解析
			Label:      data["label"],
		})
	}

	return wallets, nil
}

// CreateDepositOrder 创建加密货币充值订单
func (s *WalletService) CreateDepositOrder(ctx context.Context, userID, tenantID string, currency CryptoCurrency, chain ChainType, amount string, fiatCurrency string) (*CryptoDepositOrder, error) {
	// 验证支持的链
	chains, ok := StablecoinChainMap[currency]
	if !ok {
		return nil, fmt.Errorf("unsupported currency: %s", currency)
	}

	supported := false
	for _, c := range chains {
		if c == chain {
			supported = true
			break
		}
	}
	if !supported {
		return nil, fmt.Errorf("currency %s not supported on chain %s", currency, chain)
	}

	// 获取平台收款地址
	platformAddr, ok := s.platformAddrs[chain]
	if !ok {
		return nil, fmt.Errorf("no platform deposit address configured for chain %s", chain)
	}

	orderNo := fmt.Sprintf("DEP%s%s", time.Now().Format("20060102150405"), generateShortID())
	order := &CryptoDepositOrder{
		ID:              uuid.New().String(),
		UserID:          userID,
		TenantID:        tenantID,
		OrderNo:         orderNo,
		Currency:        currency,
		ChainType:       chain,
		Amount:          amount,
		ToAddress:       platformAddr,
		Status:          DepositStatusPending,
		RequiredConfirm: getRequiredConfirmations(chain),
		FiatCurrency:    fiatCurrency,
		CreatedAt:       time.Now().Unix(),
		ExpiredAt:       time.Now().Add(30 * time.Minute).Unix(),
	}

	// 存储订单
	orderKey := fmt.Sprintf("deposit:order:%s", order.OrderNo)
	if err := s.rdb.HSet(ctx, orderKey, map[string]interface{}{
		"id":               order.ID,
		"user_id":          order.UserID,
		"tenant_id":        order.TenantID,
		"order_no":         order.OrderNo,
		"currency":         string(order.Currency),
		"chain_type":       string(order.ChainType),
		"amount":           order.Amount,
		"to_address":       order.ToAddress,
		"status":           string(order.Status),
		"required_confirm": order.RequiredConfirm,
		"fiat_currency":    order.FiatCurrency,
		"created_at":       order.CreatedAt,
		"expired_at":       order.ExpiredAt,
	}).Err(); err != nil {
		return nil, fmt.Errorf("store deposit order: %w", err)
	}

	// 设置过期
	s.rdb.ExpireAt(ctx, orderKey, time.Unix(order.ExpiredAt, 0))

	// 用户订单索引
	userOrdersKey := fmt.Sprintf("deposit:user:%s", userID)
	s.rdb.ZAdd(ctx, userOrdersKey, &redis.Z{
		Score:  float64(order.CreatedAt),
		Member: order.OrderNo,
	})

	s.logger.Info("deposit order created",
		zap.String("order_no", order.OrderNo),
		zap.String("currency", string(currency)),
		zap.String("chain", string(chain)),
		zap.String("amount", amount),
	)

	return order, nil
}

// ConfirmDeposit 确认充值（由链上监控回调）
func (s *WalletService) ConfirmDeposit(ctx context.Context, orderNo, txHash, fromAddress, amountReceived string, confirmations int, exchangeRate float64) error {
	orderKey := fmt.Sprintf("deposit:order:%s", orderNo)

	// 检查订单状态
	status, err := s.rdb.HGet(ctx, orderKey, "status").Result()
	if err != nil {
		return fmt.Errorf("order not found: %s", orderNo)
	}
	if status != string(DepositStatusPending) && status != string(DepositStatusConfirming) {
		return fmt.Errorf("order status invalid: %s", status)
	}

	fiatCurrency, _ := s.rdb.HGet(ctx, orderKey, "fiat_currency").Result()
	_ = fiatCurrency // reserved for future fiat settlement
	userID, _ := s.rdb.HGet(ctx, orderKey, "user_id").Result()
	tenantID, _ := s.rdb.HGet(ctx, orderKey, "tenant_id").Result()

	requiredConfirm, _ := s.rdb.HGet(ctx, orderKey, "required_confirm").Int()

	updates := map[string]interface{}{
		"tx_hash":         txHash,
		"from_address":    fromAddress,
		"amount_received": amountReceived,
		"confirmations":   confirmations,
		"exchange_rate":    fmt.Sprintf("%.6f", exchangeRate),
	}

	if confirmations >= requiredConfirm {
		// 计算法币等值
		amountFloat := new(big.Float)
		amountFloat.SetString(amountReceived)
		rateFloat := new(big.Float).SetFloat64(exchangeRate)
		fiatFloat := new(big.Float).Mul(amountFloat, rateFloat)
		fiatFloat.Mul(fiatFloat, big.NewFloat(100)) // 转分为单位

		fiatCents, _ := fiatFloat.Int64()

		now := time.Now().Unix()
		updates["status"] = string(DepositStatusCompleted)
		updates["fiat_amount"] = fiatCents
		updates["confirmed_at"] = now

		// 充值到用户余额
		balanceKey := fmt.Sprintf("balance:%s:%s", tenantID, userID)
		s.rdb.IncrBy(ctx, balanceKey, fiatCents)

		s.logger.Info("deposit confirmed and credited",
			zap.String("order_no", orderNo),
			zap.String("amount", amountReceived),
			zap.Int64("fiat_cents", fiatCents),
		)
	} else {
		updates["status"] = string(DepositStatusConfirming)
	}

	s.rdb.HSet(ctx, orderKey, updates)
	return nil
}

// GetDepositOrder 获取充值订单
func (s *WalletService) GetDepositOrder(ctx context.Context, orderNo string) (*CryptoDepositOrder, error) {
	orderKey := fmt.Sprintf("deposit:order:%s", orderNo)
	data, err := s.rdb.HGetAll(ctx, orderKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("order not found: %s", orderNo)
	}

	return &CryptoDepositOrder{
		ID:             data["id"],
		UserID:         data["user_id"],
		TenantID:       data["tenant_id"],
		OrderNo:        data["order_no"],
		Currency:       CryptoCurrency(data["currency"]),
		ChainType:      ChainType(data["chain_type"]),
		Amount:         data["amount"],
		AmountReceived: data["amount_received"],
		ToAddress:      data["to_address"],
		FromAddress:    data["from_address"],
		TxHash:         data["tx_hash"],
		Status:         DepositStatus(data["status"]),
		FiatCurrency:   data["fiat_currency"],
	}, nil
}

// ListDepositOrders 获取用户充值订单列表
func (s *WalletService) ListDepositOrders(ctx context.Context, userID string, offset, limit int64) ([]*CryptoDepositOrder, error) {
	userOrdersKey := fmt.Sprintf("deposit:user:%s", userID)
	orderNos, err := s.rdb.ZRevRange(ctx, userOrdersKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	var orders []*CryptoDepositOrder
	for _, orderNo := range orderNos {
		order, err := s.GetDepositOrder(ctx, orderNo)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetCryptoExchangeRate 获取加密货币汇率（从外部API或缓存获取）
func (s *WalletService) GetCryptoExchangeRate(ctx context.Context, crypto CryptoCurrency, fiat string) (float64, error) {
	// 先从Redis缓存获取
	cacheKey := fmt.Sprintf("crypto_rate:%s:%s", crypto, fiat)
	if rate, err := s.rdb.Get(ctx, cacheKey).Float64(); err == nil && rate > 0 {
		return rate, nil
	}

	// 从CoinGecko免费API获取
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=%s", cryptoToCoinGeckoID(crypto), strings.ToLower(fiat))
	resp, err := http.Get(url)
	if err != nil {
		// 回退到静态汇率
		return s.getFallbackRate(crypto, fiat)
	}
	defer resp.Body.Close()

	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.getFallbackRate(crypto, fiat)
	}

	if coinData, ok := result[cryptoToCoinGeckoID(crypto)]; ok {
		if rate, ok := coinData[strings.ToLower(fiat)]; ok {
			// 缓存5分钟
			s.rdb.Set(ctx, cacheKey, rate, 5*time.Minute)
			return rate, nil
		}
	}

	return s.getFallbackRate(crypto, fiat)
}

// cryptoToCoinGeckoID 转换为CoinGecko ID
func cryptoToCoinGeckoID(crypto CryptoCurrency) string {
	switch crypto {
	case CryptoBTC:
		return "bitcoin"
	case CryptoETH:
		return "ethereum"
	case CryptoUSDT:
		return "tether"
	case CryptoUSDC:
		return "usd-coin"
	case CryptoSOL:
		return "solana"
	case CryptoBNB:
		return "binancecoin"
	case CryptoTRX:
		return "tron"
	default:
		return strings.ToLower(string(crypto))
	}
}

// getFallbackRate 回退到静态汇率
func (s *WalletService) getFallbackRate(crypto CryptoCurrency, fiat string) (float64, error) {
	rates := map[CryptoCurrency]map[string]float64{
		CryptoUSDT: {"CNY": 7.25, "USD": 1.0, "EUR": 0.92, "JPY": 155.0, "KRW": 1350.0},
		CryptoUSDC: {"CNY": 7.25, "USD": 1.0, "EUR": 0.92, "JPY": 155.0, "KRW": 1350.0},
		CryptoETH:  {"CNY": 25000.0, "USD": 3500.0, "EUR": 3200.0},
		CryptoBTC:  {"CNY": 500000.0, "USD": 70000.0, "EUR": 64000.0},
	}

	if rateMap, ok := rates[crypto]; ok {
		if rate, ok := rateMap[fiat]; ok {
			return rate, nil
		}
	}

	return 0, fmt.Errorf("exchange rate not found for %s/%s", crypto, fiat)
}

// getRequiredConfirmations 获取链所需确认数
func getRequiredConfirmations(chain ChainType) int {
	switch chain {
	case ChainEthereum:
		return 12
	case ChainBSC:
		return 15
	case ChainPolygon:
		return 128
	case ChainArbitrum:
		return 10
	case ChainOptimism:
		return 10
	case ChainTron:
		return 20
	case ChainSolana:
		return 32
	case ChainAvalanche:
		return 12
	default:
		return 12
	}
}

// generateShortID 生成短ID
func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
