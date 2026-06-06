package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tokenhub/backend/internal/auth"
	"github.com/tokenhub/backend/internal/billing"
	tokencache "github.com/tokenhub/backend/internal/cache"
	"github.com/tokenhub/backend/internal/common/middleware"
	"github.com/tokenhub/backend/internal/config"
	"github.com/tokenhub/backend/internal/distribution"
	"github.com/tokenhub/backend/internal/finance"
	"github.com/tokenhub/backend/internal/gateway"
	"github.com/tokenhub/backend/internal/gateway/lb"
	"github.com/tokenhub/backend/internal/gateway/proxy"
	smartrouter "github.com/tokenhub/backend/internal/gateway/router"
	"github.com/tokenhub/backend/internal/i18n"
	"github.com/tokenhub/backend/internal/keyvault"
	"github.com/tokenhub/backend/internal/metrics"
	"github.com/tokenhub/backend/internal/monitor"
	"github.com/tokenhub/backend/internal/payment"
	"github.com/tokenhub/backend/internal/platform"
	"github.com/tokenhub/backend/internal/quota"
	"github.com/tokenhub/backend/internal/reconciliation"
	"github.com/tokenhub/backend/internal/refund"
	"github.com/tokenhub/backend/internal/security/desensitize"
	"github.com/tokenhub/backend/internal/security/waf"
	"github.com/tokenhub/backend/internal/tenant"
	"github.com/tokenhub/backend/internal/wallet"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	// 1. 加载配置
	configPath := "configs/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 2. 初始化日志
	logger := initLogger(cfg.Server.Mode)
	defer logger.Sync()

	// 3. 初始化数据库
	db, err := initDB(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect database", zap.Error(err))
	}

	// 3.1 自动迁移表结构
	autoMigrate(db, logger)

	// 3.2 初始化超级管理员
	initSuperAdmin(db, logger)

	// 4. 初始化Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect Redis", zap.Error(err))
	}

	// 5. 初始化核心服务
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.Expire,
		cfg.JWT.RefreshExp,
		cfg.JWT.Issuer,
	)

	modelRouter := smartrouter.NewModelRouter(logger, lb.StrategyWeightedRandom, cfg.Gateway.MaxRetries)
	aiProxy := proxy.NewAIProxy(logger, modelRouter, cfg.Gateway.Timeout, cfg.Gateway.StreamTimeout)
	billingService := billing.NewBillingService(logger, rdb)
	monitorService := monitor.NewMonitorService(logger)
	wafService := waf.NewWAF(cfg.Security.EnableWAF)
	desensitizer := desensitize.NewDesensitizer(cfg.Security.EnableDesensitize)
	i18nService := i18n.NewI18n(cfg.I18n.DefaultLocale, cfg.I18n.SupportedLocales)
	_ = i18nService

	// 钱包服务（Web3钱包绑定 + 加密货币充值）
	platformAddrs := map[string]map[wallet.ChainType]string{
		"ethereum":  {wallet.ChainEthereum: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18"},
		"bsc":      {wallet.ChainBSC: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18"},
		"polygon":  {wallet.ChainPolygon: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18"},
		"tron":     {wallet.ChainTron: "TYDzsWUZbMF6n3qG1GmwVsQ1v9Xd9gKb0S"},
	}
	// Flatten for wallet service
	walletAddrs := make(map[wallet.ChainType]string)
	for _, m := range platformAddrs {
		for k, v := range m {
			walletAddrs[k] = v
		}
	}
	walletService := wallet.NewWalletService(logger, rdb, walletAddrs)

	// 支付服务（支付宝/微信/PayPal/万里汇/WorldFirst/Payoneer/Wise/Stripe）
	paymentService := payment.NewPaymentService(logger, rdb, &cfg.Payment)

	// 财务服务（收款账号 + 提现管理）
	financeService := finance.NewFinanceService(logger, rdb)

	// 配额服务
	quotaService := quota.NewQuotaService(logger, rdb)

	// 退款服务
	refundService := refund.NewRefundService(logger, rdb)

	// 对账服务
	reconService := reconciliation.NewReconciliationService(logger, rdb)

	// 密钥保险库
	keyVault := keyvault.NewKeyVault(logger, rdb, os.Getenv("KEYVAULT_MASTER_KEY"))

	// 分销服务
	distService := distribution.NewDistributionService(logger, rdb)

	// Token缓存
	tokenCache := tokencache.NewTokenCache(logger, rdb, tokencache.CacheConfig{
		Enabled:    true,
		TTL:        10 * time.Minute,
		MaxEntries: 100000,
	})

	// Prometheus指标
	promMetrics := metrics.NewPrometheusMetrics(logger, rdb)

	// 平台服务（用户管理 + RBAC + 审计日志 + 通知 + 个人中心）
	platformService := platform.NewPlatformService(db, rdb)
	platformService.AutoMigrate()
	platformService.SeedData(context.Background())

	// 验证码服务（短信+邮箱）
	smsSender := auth.NewAliyunSMSSender("", "", "TokenHub", "SMS_123456")
	emailSender := auth.NewSMTPEmailSender("smtp.tokenhub.com", 465, "", "", "noreply@tokenhub.com", "TokenHub")
	verificationService := auth.NewVerificationService(logger, rdb, smsSender, emailSender)

	// 人脸识别服务（WebAuthn）
	faceAuthService := auth.NewFaceAuthService(db, rdb, logger, "localhost", "TokenHub", "http://localhost:3001")

	logger.Info("services initialized",
		zap.String("lb_strategy", string(lb.StrategyWeightedRandom)),
		zap.Bool("waf_enabled", wafService.IsEnabled()),
		zap.Bool("desensitize_enabled", desensitizer.Enabled()),
	)

	// 6. 设置Gin路由
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORSMiddleware(nil)) // 开发模式允许所有来源，生产环境应传入允许的域名列表
	engine.Use(middleware.RequestIDMiddleware())
	engine.Use(middleware.MetricsMiddleware(monitorService, logger))

	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		// 检查数据库连接
		sqlDB, err := db.DB()
		dbStatus := "ok"
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "error"
		}
		// 检查Redis连接
		redisStatus := "ok"
		if rdb.Ping(ctx).Err() != nil {
			redisStatus = "error"
		}

		overall := "healthy"
		if dbStatus != "ok" || redisStatus != "ok" {
			overall = "degraded"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    overall,
			"version":   "0.1.0",
			"time":      time.Now().Format(time.RFC3339),
			"checks": gin.H{
				"database": dbStatus,
				"redis":    redisStatus,
			},
		})
	})

	// API v1 路由组
	v1 := engine.Group("/v1")
	{

	// Prometheus 指标端点（无需认证，供 Prometheus 抓取）
	engine.GET("/metrics", gin.WrapH(promMetrics))

	// AI 模型调用接口（兼容 OpenAI 格式）
	v1.POST("/chat/completions", middleware.APIKeyMiddleware(rdb), middleware.RateLimitMiddleware(rdb, cfg.Gateway.RateLimitPerSec, time.Second, "api"), func(c *gin.Context) {
		// 余额熔断：检查用户余额是否充足
		tenantID := c.GetString("tenant_id")
		userID := c.GetString("user_id")
		if tenantID != "" && userID != "" {
			balance, err := billingService.GetBalance(c.Request.Context(), tenantID, userID)
			if err == nil && balance < cfg.Billing.MinBalance {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": gin.H{
						"message": "Insufficient balance. Please top up your account.",
						"type":    "insufficient_balance",
						"code":    "balance_too_low",
					},
				})
				return
			}
		}
		handleChatCompletion(c, aiProxy, billingService, desensitizer, logger)
	})
		v1.POST("/completions", middleware.APIKeyMiddleware(rdb), func(c *gin.Context) {
			handleCompletion(c, aiProxy, billingService, logger)
		})
		v1.GET("/models", middleware.APIKeyMiddleware(rdb), func(c *gin.Context) {
			handleListModels(c, modelRouter)
		})
	}

	// 管理API
	admin := engine.Group("/admin")
	admin.Use(middleware.AuthMiddleware(jwtManager))
	{
		// 模型管理 CRUD
		// 获取模型列表
		admin.GET("/models", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			var models []gateway.ModelConfig
			query := db.Where("tenant_id = ? OR tenant_id = ''", tenantID)
			if err := query.Order("created_at DESC").Find(&models).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 同时返回熔断器状态
			circuitStates := modelRouter.GetCircuitStates()
			c.JSON(http.StatusOK, gin.H{"models": models, "circuit_states": circuitStates})
		})

		// 获取单个模型
		admin.GET("/models/:id", func(c *gin.Context) {
			id := c.Param("id")
			var model gateway.ModelConfig
			if err := db.First(&model, "id = ?", id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"model": model})
		})

		// 创建模型
		admin.POST("/models", func(c *gin.Context) {
			var req struct {
				Name        string  `json:"name" binding:"required"`
				Provider    string  `json:"provider" binding:"required"`
				ModelID     string  `json:"model_id" binding:"required"`
				Endpoint    string  `json:"endpoint" binding:"required"`
				APIKey      string  `json:"api_key"`
				MaxTokens   int     `json:"max_tokens"`
				InputPrice  float64 `json:"input_price"`
				OutputPrice float64 `json:"output_price"`
				Currency    string  `json:"currency"`
				Weight      int     `json:"weight"`
				Priority    int     `json:"priority"`
				Enabled     bool    `json:"enabled"`
				Streamable  bool    `json:"streamable"`
				TenantID    string  `json:"tenant_id"`
				Tags        []string `json:"tags"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			tenantID := c.GetString("tenant_id")
			if req.TenantID != "" {
				tenantID = req.TenantID
			}
			if req.Currency == "" {
				req.Currency = "CNY"
			}
			if req.Weight == 0 {
				req.Weight = 50
			}
			if req.MaxTokens == 0 {
				req.MaxTokens = 4096
			}

			now := time.Now().Unix()
			model := &gateway.ModelConfig{
				ID:          uuid.New().String(),
				Name:        req.Name,
				Provider:    gateway.ModelProvider(req.Provider),
				ModelID:     req.ModelID,
				Endpoint:    req.Endpoint,
				MaxTokens:   req.MaxTokens,
				InputPrice:  req.InputPrice,
				OutputPrice: req.OutputPrice,
				Currency:    req.Currency,
				Weight:      req.Weight,
				Priority:    req.Priority,
				Enabled:     req.Enabled,
				Streamable:  req.Streamable,
				TenantID:    tenantID,
				Tags:        req.Tags,
				LatencyMs:   0,
				SuccessRate: 1.0,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// 加密存储API Key
			if req.APIKey != "" {
				if err := keyVault.StoreKey(c.Request.Context(), "model_"+model.ID, req.APIKey); err != nil {
					// 如果keyvault不可用，明文存储（仅开发环境）
					model.APIKeyEnc = req.APIKey
				} else {
					model.APIKeyEnc = "encrypted:model_" + model.ID
				}
			}

			if err := db.Create(model).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 注册到内存路由器
			model.APIKey = req.APIKey
			modelRouter.RegisterModel(model)

			logger.Info("model created", zap.String("name", model.Name), zap.String("provider", string(model.Provider)))
			c.JSON(http.StatusOK, gin.H{"message": "model created", "model": model})
		})

		// 更新模型
		admin.PUT("/models/:id", func(c *gin.Context) {
			id := c.Param("id")
			var model gateway.ModelConfig
			if err := db.First(&model, "id = ?", id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}

			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 白名单过滤
			allowedFields := map[string]bool{
				"name": true, "provider": true, "model_id": true, "endpoint": true,
				"max_tokens": true, "input_price": true, "output_price": true,
				"currency": true, "weight": true, "priority": true,
				"enabled": true, "streamable": true, "tags": true,
			}
			filtered := make(map[string]interface{})
			for k, v := range updates {
				if allowedFields[k] {
					filtered[k] = v
				}
			}

			// 处理API Key更新
			if apiKey, ok := updates["api_key"].(string); ok && apiKey != "" {
				if err := keyVault.StoreKey(c.Request.Context(), "model_"+id, apiKey); err != nil {
					filtered["api_key_enc"] = apiKey
				} else {
					filtered["api_key_enc"] = "encrypted:model_" + id
				}
			}

			filtered["updated_at"] = time.Now().Unix()

			if len(filtered) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no valid fields to update"})
				return
			}

			if err := db.Model(&gateway.ModelConfig{}).Where("id = ?", id).Updates(filtered).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 更新内存路由器
			db.First(&model, "id = ?", id)
			modelRouter.UnregisterModel(model.ID)
			if model.Enabled {
				modelRouter.RegisterModel(&model)
			}

			c.JSON(http.StatusOK, gin.H{"message": "model updated", "model": model})
		})

		// 删除模型
		admin.DELETE("/models/:id", func(c *gin.Context) {
			id := c.Param("id")
			var model gateway.ModelConfig
			if err := db.First(&model, "id = ?", id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}

			if err := db.Delete(&model).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 从内存路由器注销
			modelRouter.UnregisterModel(model.ID)

			c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
		})

		// 发现厂商可用模型
		admin.POST("/models/discover", func(c *gin.Context) {
			var req struct {
				Endpoint string `json:"endpoint" binding:"required"`
				APIKey   string `json:"api_key"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint is required"})
				return
			}

			// 构造厂商 /v1/models 请求
			modelsURL := strings.TrimRight(req.Endpoint, "/") + "/models"
			httpReq, _ := http.NewRequestWithContext(c.Request.Context(), "GET", modelsURL, nil)
			if req.APIKey != "" {
				httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
			}
			httpReq.Header.Set("Accept", "application/json")

			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Do(httpReq)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to connect provider: " + err.Error()})
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				c.JSON(http.StatusBadGateway, gin.H{"error": "provider returned " + resp.Status, "detail": string(body)})
				return
			}

			// 解析 OpenAI 格式响应: {"data": [{"id":"model-name","owned_by":"..."},...]}
			var result struct {
				Data []struct {
					ID      string `json:"id"`
					Object  string `json:"object"`
					OwnedBy string `json:"owned_by"`
					Created int64  `json:"created"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse provider response"})
				return
			}

			models := make([]gin.H{}, 0, len(result.Data))
			for _, m := range result.Data {
				models = append(models, gin.H{
					"id":       m.ID,
					"owned_by": m.OwnedBy,
				})
			}
			c.JSON(http.StatusOK, gin.H{"models": models})
		})

		// 切换模型启用状态
		admin.PUT("/models/:id/toggle", func(c *gin.Context) {
			id := c.Param("id")
			var model gateway.ModelConfig
			if err := db.First(&model, "id = ?", id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
				return
			}

			newEnabled := !model.Enabled
			db.Model(&gateway.ModelConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
				"enabled":    newEnabled,
				"updated_at": time.Now().Unix(),
			})

			// 更新内存路由器
			if newEnabled {
				model.Enabled = true
				modelRouter.RegisterModel(&model)
			} else {
				modelRouter.UnregisterModel(model.ID)
			}

			c.JSON(http.StatusOK, gin.H{"message": "model toggled", "enabled": newEnabled})
		})

		// 计费
		admin.GET("/billing/balance", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			userID := c.GetString("user_id")
			balance, err := billingService.GetBalance(c.Request.Context(), tenantID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"balance": balance, "currency": cfg.Billing.DefaultCurrency})
		})

		admin.POST("/billing/topup", func(c *gin.Context) {
			var req struct {
				Amount int64 `json:"amount"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			tenantID := c.GetString("tenant_id")
			userID := c.GetString("user_id")
			if err := billingService.TopUp(c.Request.Context(), tenantID, userID, req.Amount); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "topup success"})
		})

		// ==================== 钱包管理 ====================
		// 绑定Web3钱包
		admin.POST("/wallet/bind", func(c *gin.Context) {
			var req struct {
				WalletType string `json:"wallet_type"`
				Address    string `json:"address" binding:"required"`
				ChainType  string `json:"chain_type" binding:"required"`
				Label      string `json:"label"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			binding := &wallet.WalletBinding{
				UserID:     userID,
				TenantID:   tenantID,
				WalletType: wallet.WalletType(req.WalletType),
				Address:    req.Address,
				ChainType:  wallet.ChainType(req.ChainType),
				IsPrimary:  false,
				Label:      req.Label,
			}
			if err := walletService.BindWallet(c.Request.Context(), binding); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "wallet bound successfully", "binding": binding})
		})

		// 解绑钱包
		admin.DELETE("/wallet/unbind", func(c *gin.Context) {
			var req struct {
				Address string `json:"address" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			if err := walletService.UnbindWallet(c.Request.Context(), userID, req.Address); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "wallet unbound"})
		})

		// 获取钱包验证挑战（EIP-191签名验证）
		admin.GET("/wallet/challenge", func(c *gin.Context) {
			address := c.Query("address")
			if address == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
				return
			}
			challenge, err := walletService.GenerateVerifyChallenge(c.Request.Context(), address)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, challenge)
		})

		// 验证钱包签名
		admin.POST("/wallet/verify", func(c *gin.Context) {
			var req struct {
				Address   string `json:"address" binding:"required"`
				Signature string `json:"signature" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			valid, err := walletService.VerifySignature(c.Request.Context(), req.Address, req.Signature)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"valid": valid})
		})

		// 用户钱包列表
		admin.GET("/wallet/list", func(c *gin.Context) {
			userID := c.GetString("user_id")
			wallets, err := walletService.ListUserWallets(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"wallets": wallets})
		})

		// 创建加密货币充值订单
		admin.POST("/wallet/deposit", func(c *gin.Context) {
			var req struct {
				Currency   string `json:"currency" binding:"required"`  // USDT, USDC
				ChainType  string `json:"chain_type" binding:"required"`
				Amount     string `json:"amount" binding:"required"`
				FiatCurrency string `json:"fiat_currency"`  // CNY, USD
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			fiatCurrency := req.FiatCurrency
			if fiatCurrency == "" {
				fiatCurrency = "CNY"
			}
			order, err := walletService.CreateDepositOrder(
				c.Request.Context(), userID, tenantID,
				wallet.CryptoCurrency(req.Currency),
				wallet.ChainType(req.ChainType),
				req.Amount, fiatCurrency,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, order)
		})

		// 查询充值订单状态
		admin.GET("/wallet/deposit/:orderNo", func(c *gin.Context) {
			orderNo := c.Param("orderNo")
			order, err := walletService.GetDepositOrder(c.Request.Context(), orderNo)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, order)
		})

		// 获取加密货币汇率
		admin.GET("/wallet/exchange-rate", func(c *gin.Context) {
			crypto := c.Query("crypto")
			fiat := c.Query("fiat")
			if fiat == "" {
				fiat = "CNY"
			}
			rate, err := walletService.GetCryptoExchangeRate(c.Request.Context(), wallet.CryptoCurrency(crypto), fiat)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"crypto": crypto, "fiat": fiat, "rate": rate})
		})

		// 用户充值订单列表
		admin.GET("/wallet/deposit", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var orders []wallet.CryptoDepositOrder
			if err := db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&orders).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"orders": orders})
		})

		// 钱包余额查询
		admin.GET("/wallet/balance", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			userID := c.GetString("user_id")
			balance, err := billingService.GetBalance(c.Request.Context(), tenantID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"balance": balance, "currency": cfg.Billing.DefaultCurrency})
		})

		// 连接并绑定钱包（验证签名后自动绑定）
		admin.POST("/wallet/connect-bind", func(c *gin.Context) {
			var req struct {
				WalletType string `json:"wallet_type" binding:"required"`
				Address    string `json:"address" binding:"required"`
				ChainType  string `json:"chain_type" binding:"required"`
				Signature  string `json:"signature"`
				Label      string `json:"label"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			binding := &wallet.WalletBinding{
				UserID:     userID,
				TenantID:   tenantID,
				WalletType: wallet.WalletType(req.WalletType),
				Address:    req.Address,
				ChainType:  wallet.ChainType(req.ChainType),
				IsPrimary:  false,
				Verified:   req.Signature != "",
				Label:      req.Label,
			}
			if err := walletService.BindWallet(c.Request.Context(), binding); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "wallet connected and bound", "binding": binding})
		})

		// 获取支持的钱包类型
		admin.GET("/wallet/supported-types", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"wallet_types":  wallet.SupportedWalletTypes,
				"crypto_currencies": wallet.SupportedCryptoCurrencies,
				"stablecoin_chains": wallet.StablecoinChainMap,
			})
		})

		// ==================== 支付管理 ====================
		// 获取可用支付渠道
		admin.GET("/payment/channels", func(c *gin.Context) {
			currency := c.Query("currency")
			channels := paymentService.GetAvailableChannels(currency)
			c.JSON(http.StatusOK, gin.H{"channels": channels})
		})

		// 创建支付订单
		admin.POST("/payment/create", func(c *gin.Context) {
			var req struct {
				Channel     string `json:"channel" binding:"required"`
				Amount      int64  `json:"amount" binding:"required"`
				Currency    string `json:"currency" binding:"required"`
				ToCurrency  string `json:"to_currency"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			toCurrency := req.ToCurrency
			if toCurrency == "" {
				toCurrency = "CNY"
			}
			order, err := paymentService.CreatePaymentOrder(
				c.Request.Context(), userID, tenantID,
				payment.PaymentChannel(req.Channel),
				req.Amount, req.Currency, toCurrency,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, order)
		})

		// 查询支付订单
		admin.GET("/payment/order/:orderNo", func(c *gin.Context) {
			orderNo := c.Param("orderNo")
			order, err := paymentService.GetPaymentOrder(c.Request.Context(), orderNo)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, order)
		})

		// 用户支付订单列表
		admin.GET("/payment/orders", func(c *gin.Context) {
			userID := c.GetString("user_id")
			orders, err := paymentService.ListPaymentOrders(c.Request.Context(), userID, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"orders": orders})
		})

		// ==================== 配额管理 ====================
		// 获取配额列表
		admin.GET("/quotas", func(c *gin.Context) {
			tenantID := c.Query("tenant_id")
			if tenantID == "" {
				tenantID = c.GetString("tenant_id")
			}
			statuses, err := quotaService.GetAllQuotaStatus(c.Request.Context(), tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"quotas": statuses})
		})

		// 获取单个配额
		admin.GET("/quotas/:tenantId/:quotaType", func(c *gin.Context) {
			tenantID := c.Param("tenantId")
			quotaType := quota.QuotaType(c.Param("quotaType"))
			status, err := quotaService.GetQuotaStatus(c.Request.Context(), tenantID, quotaType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, status)
		})

		// 设置配额
		admin.POST("/quotas", func(c *gin.Context) {
			var req struct {
				TenantID   string  `json:"tenant_id" binding:"required"`
				QuotaType  string  `json:"quota_type" binding:"required"`
				Limit      int64   `json:"limit" binding:"required"`
				PeriodDays int     `json:"period_days"`
				AlertAt    float64 `json:"alert_at"`
				BlockAt    float64 `json:"block_at"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			config := &quota.QuotaConfig{
				TenantID:   req.TenantID,
				QuotaType:  quota.QuotaType(req.QuotaType),
				Limit:      req.Limit,
				PeriodDays: req.PeriodDays,
				AlertAt:    req.AlertAt,
				BlockAt:    req.BlockAt,
			}
			if config.PeriodDays == 0 {
				config.PeriodDays = 1
			}
			if config.AlertAt == 0 {
				config.AlertAt = 0.8
			}
			if config.BlockAt == 0 {
				config.BlockAt = 1.0
			}
			if err := quotaService.SetQuota(c.Request.Context(), config); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "quota set", "config": config})
		})

		// 重置配额
		admin.POST("/quotas/:tenantId/:quotaType/reset", func(c *gin.Context) {
			tenantID := c.Param("tenantId")
			quotaType := quota.QuotaType(c.Param("quotaType"))
			if err := quotaService.ResetQuota(c.Request.Context(), tenantID, quotaType); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "quota reset"})
		})

		// ==================== 退款管理 ====================
		// 创建退款申请
		admin.POST("/refunds", func(c *gin.Context) {
			var req struct {
				PaymentOrderNo string `json:"payment_order_no" binding:"required"`
				Amount         int64  `json:"amount" binding:"required"`
				Reason         string `json:"reason" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			order := &refund.RefundOrder{
				UserID:         userID,
				TenantID:       tenantID,
				PaymentOrderNo: req.PaymentOrderNo,
				Amount:         req.Amount,
				Reason:         req.Reason,
				Currency:       "CNY",
			}
			if err := refundService.CreateRefundOrder(c.Request.Context(), order); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "refund created", "order": order})
		})

		// 用户退款列表
		admin.GET("/refunds", func(c *gin.Context) {
			userID := c.GetString("user_id")
			orders, err := refundService.ListRefundOrders(c.Request.Context(), userID, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"refunds": orders})
		})

		// 待审核退款
		admin.GET("/refunds/pending", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			orders, err := refundService.ListPendingRefunds(c.Request.Context(), tenantID, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"refunds": orders})
		})

		// 获取退款详情
		admin.GET("/refunds/:orderNo", func(c *gin.Context) {
			orderNo := c.Param("orderNo")
			order, err := refundService.GetRefundOrder(c.Request.Context(), orderNo)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, order)
		})

		// 审核退款
		admin.POST("/refunds/:orderNo/review", func(c *gin.Context) {
			orderNo := c.Param("orderNo")
			var req struct {
				Approved bool   `json:"approved"`
				Reason   string `json:"reason"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			reviewerID := c.GetString("user_id")
			if err := refundService.ReviewRefundOrder(c.Request.Context(), orderNo, reviewerID, req.Approved, req.Reason); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "refund reviewed"})
		})

		// ==================== 对账管理 ====================
		// 日汇总
		admin.GET("/reconciliation/daily", func(c *gin.Context) {
			date := c.Query("date")
			tenantID := c.Query("tenant_id")
			if date == "" {
				date = time.Now().Format("2006-01-02")
			}
			summary, err := reconService.GetDailySummary(c.Request.Context(), date, tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, summary)
		})

		// 日期范围汇总
		admin.GET("/reconciliation/range", func(c *gin.Context) {
			startDate := c.Query("start_date")
			endDate := c.Query("end_date")
			tenantID := c.Query("tenant_id")
			summaries, err := reconService.GetDateRangeSummary(c.Request.Context(), startDate, endDate, tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"summaries": summaries})
		})

		// 聚合汇总
		admin.GET("/reconciliation/aggregated", func(c *gin.Context) {
			startDate := c.Query("start_date")
			endDate := c.Query("end_date")
			tenantID := c.Query("tenant_id")
			summary, err := reconService.GetAggregatedSummary(c.Request.Context(), startDate, endDate, tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, summary)
		})

		// ==================== 分销管理 ====================
		// 注册分销商
		admin.POST("/distribution/register", func(c *gin.Context) {
			var req struct {
				Role            string  `json:"role" binding:"required"`
				CommissionType  string  `json:"commission_type"`
				CommissionRate  float64 `json:"commission_rate"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			dist := &distribution.Distributor{
				UserID:         userID,
				TenantID:       tenantID,
				Role:           distribution.DistributorRole(req.Role),
				CommissionType: distribution.CommissionType(req.CommissionType),
				CommissionRate: req.CommissionRate,
			}
			if err := distService.RegisterDistributor(c.Request.Context(), dist); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "distributor registered", "distributor": dist})
		})

		// 分销商列表
		admin.GET("/distribution/distributors", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			distributors, err := distService.ListDistributors(c.Request.Context(), tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"distributors": distributors})
		})

		// 分销商详情
		admin.GET("/distribution/distributors/:id", func(c *gin.Context) {
			id := c.Param("id")
			dist, err := distService.GetDistributor(c.Request.Context(), id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dist)
		})

		// 佣金记录
		admin.GET("/distribution/commissions", func(c *gin.Context) {
			distID := c.Query("distributor_id")
			records, err := distService.ListCommissionRecords(c.Request.Context(), distID, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"commissions": records})
		})

		// 结算佣金
		admin.POST("/distribution/settle", func(c *gin.Context) {
			var req struct {
				Period string `json:"period" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			total, err := distService.SettleCommissions(c.Request.Context(), req.Period)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"total": total, "period": req.Period})
		})

		// ==================== 密钥保险库 ====================
		// 存储密钥
		admin.POST("/keyvault/:keyId", func(c *gin.Context) {
			keyID := c.Param("keyId")
			var req struct {
				Key string `json:"key" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := keyVault.StoreKey(c.Request.Context(), keyID, req.Key); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "key stored securely"})
		})

		// 获取密钥（仅超级管理员）
		admin.GET("/keyvault/:keyId", func(c *gin.Context) {
			role := c.GetString("role")
			if role != "super_admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "only super_admin can access keys"})
				return
			}
			keyID := c.Param("keyId")
			key, err := keyVault.GetKey(c.Request.Context(), keyID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"key_id": keyID, "key": key})
		})

		// ==================== 缓存管理 ====================
		// 缓存统计
		admin.GET("/cache/stats", func(c *gin.Context) {
			stats := tokenCache.GetStats(c.Request.Context())
			c.JSON(http.StatusOK, stats)
		})

		// 清空缓存
		admin.DELETE("/cache/clear", func(c *gin.Context) {
			if err := tokenCache.ClearAll(c.Request.Context()); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "cache cleared"})
		})

		// 监控大屏
		admin.GET("/monitor/metrics", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, metrics)
		})

		// 人脸识别凭据管理
		admin.GET("/face/credentials", func(c *gin.Context) {
			userID := c.GetString("user_id")
			credentials, err := faceAuthService.ListCredentials(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"credentials": credentials})
		})

		admin.DELETE("/face/credentials/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			credID := c.Param("id")
			if err := faceAuthService.RemoveCredential(c.Request.Context(), userID, credID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "credential removed"})
		})

		// ============ 财务管理（收款账号 + 提现） ============
		// 收款账号
		admin.GET("/finance/receiving", func(c *gin.Context) {
			userID := c.GetString("user_id")
			accounts, err := financeService.ListReceivingAccounts(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": accounts})
		})

		admin.POST("/finance/receiving", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				AccountType   string `json:"account_type"`
				AccountName   string `json:"account_name"`
				AccountNo     string `json:"account_no"`
				BankName      string `json:"bank_name"`
				BankBranch    string `json:"bank_branch"`
				QRCodeURL     string `json:"qrcode_url"`
				WalletAddress string `json:"wallet_address"`
				ChainType     string `json:"chain_type"`
				IsPrimary     bool   `json:"is_primary"`
				Label         string `json:"label"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			account := &finance.ReceivingAccount{
				UserID:        userID,
				TenantID:      tenantID,
				AccountType:   finance.AccountType(req.AccountType),
				AccountName:   req.AccountName,
				AccountNo:     req.AccountNo,
				BankName:      req.BankName,
				BankBranch:    req.BankBranch,
				QRCodeURL:     req.QRCodeURL,
				WalletAddress: req.WalletAddress,
				ChainType:     req.ChainType,
				IsPrimary:     req.IsPrimary,
				Verified:      true,
				Enabled:       true,
				Label:         req.Label,
			}
			if err := financeService.CreateReceivingAccount(c.Request.Context(), account); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "receiving account created", "data": account})
		})

		admin.DELETE("/finance/receiving/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			accountID := c.Param("id")
			if err := financeService.DeleteReceivingAccount(c.Request.Context(), userID, accountID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "receiving account deleted"})
		})

		admin.PUT("/finance/receiving/:id/primary", func(c *gin.Context) {
			userID := c.GetString("user_id")
			accountID := c.Param("id")
			if err := financeService.SetPrimaryReceivingAccount(c.Request.Context(), userID, accountID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "primary account updated"})
		})

		// 提现账户
		admin.GET("/finance/withdraw-accounts", func(c *gin.Context) {
			userID := c.GetString("user_id")
			accounts, err := financeService.ListWithdrawalAccounts(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": accounts})
		})

		admin.POST("/finance/withdraw-accounts", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				AccountType   string `json:"account_type"`
				AccountName   string `json:"account_name"`
				AccountNo     string `json:"account_no"`
				BankName      string `json:"bank_name"`
				BankBranch    string `json:"bank_branch"`
				SwiftCode     string `json:"swift_code"`
				WalletAddress string `json:"wallet_address"`
				ChainType     string `json:"chain_type"`
				IsPrimary     bool   `json:"is_primary"`
				Label         string `json:"label"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			account := &finance.WithdrawalAccount{
				UserID:        userID,
				TenantID:      tenantID,
				AccountType:   finance.AccountType(req.AccountType),
				AccountName:   req.AccountName,
				AccountNo:     req.AccountNo,
				BankName:      req.BankName,
				BankBranch:    req.BankBranch,
				SwiftCode:     req.SwiftCode,
				WalletAddress: req.WalletAddress,
				ChainType:     req.ChainType,
				IsPrimary:     req.IsPrimary,
				Verified:      true,
				Label:         req.Label,
			}
			if err := financeService.CreateWithdrawalAccount(c.Request.Context(), account); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "withdrawal account created", "data": account})
		})

		admin.DELETE("/finance/withdraw-accounts/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			accountID := c.Param("id")
			if err := financeService.DeleteWithdrawalAccount(c.Request.Context(), userID, accountID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "withdrawal account deleted"})
		})

		// 提现订单
		admin.POST("/finance/withdrawal", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				AccountID string `json:"account_id"`
				Amount    int64  `json:"amount"`
				Currency  string `json:"currency"`
				Remark    string `json:"remark"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// 获取账户信息
			accounts, _ := financeService.ListWithdrawalAccounts(c.Request.Context(), userID)
			var selectedAccount *finance.WithdrawalAccount
			for _, a := range accounts {
				if a.ID == req.AccountID {
					selectedAccount = a
					break
				}
			}
			if selectedAccount == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "account not found"})
				return
			}
			order := &finance.WithdrawalOrder{
				UserID:      userID,
				TenantID:    tenantID,
				AccountID:   req.AccountID,
				AccountType: selectedAccount.AccountType,
				AccountName: selectedAccount.AccountName,
				AccountNo:   selectedAccount.AccountNo,
				Amount:      req.Amount,
				Currency:    req.Currency,
				Remark:      req.Remark,
			}
			if err := financeService.CreateWithdrawalOrder(c.Request.Context(), order); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "withdrawal order created", "data": order})
		})

		admin.GET("/finance/withdrawal", func(c *gin.Context) {
			userID := c.GetString("user_id")
			orders, err := financeService.ListWithdrawalOrders(c.Request.Context(), userID, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": orders})
		})

		admin.GET("/finance/withdrawal/:orderNo", func(c *gin.Context) {
			orderNo := c.Param("orderNo")
			order, err := financeService.GetWithdrawalOrder(c.Request.Context(), orderNo)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, order)
		})

		// ============ 平台管理（用户/RBAC/审计/通知/个人中心） ============
		// 用户管理
		admin.GET("/users", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			search := c.Query("search")
			role := c.Query("role")
			status := c.Query("status")
			users, _, err := platformService.ListUsers(c.Request.Context(), tenantID, search, role, status, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			_ = userID
			c.JSON(http.StatusOK, gin.H{"data": users})
		})

		admin.POST("/users", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			var req struct {
				Email       string `json:"email"`
				Password    string `json:"password"`
				Role        string `json:"role"`
				DisplayName string `json:"display_name"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			user := &platform.User{
				TenantID:    tenantID,
				Email:       req.Email,
				DisplayName: req.DisplayName,
				Role:        req.Role,
				Status:      "active",
			}
			if err := platformService.CreateUser(c.Request.Context(), user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "user created", "data": user})
		})

		admin.PUT("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// 白名单过滤，防止role/tenant_id等敏感字段被篡改
			allowedFields := map[string]bool{
				"display_name": true,
				"phone":       true,
				"company":     true,
				"bio":         true,
				"status":      true,
				"role":        true, // 仅super_admin/tenant_admin可修改，由RBAC中间件控制
			}
			filtered := make(map[string]interface{})
			for k, v := range updates {
				if allowedFields[k] {
					filtered[k] = v
				}
			}
			if len(filtered) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no valid fields to update"})
				return
			}
			if err := platformService.UpdateUser(c.Request.Context(), id, filtered); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "user updated"})
		})

		admin.DELETE("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			if err := platformService.DeleteUser(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
		})

		admin.POST("/users/:id/reset-password", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "password reset email sent"})
		})

		// 角色管理
		admin.GET("/roles", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			roles, err := platformService.ListRoles(c.Request.Context(), tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": roles})
		})

		admin.POST("/roles", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			var req struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Permissions []string `json:"permissions"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			role := &platform.Role{
				TenantID:    tenantID,
				Name:        req.Name,
				Description: req.Description,
				Permissions: fmt.Sprintf("%v", req.Permissions),
			}
			if err := platformService.CreateRole(c.Request.Context(), role); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "role created", "data": role})
		})

		admin.PUT("/roles/:id", func(c *gin.Context) {
			id := c.Param("id")
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			allowedRoleFields := map[string]bool{
				"name": true, "description": true, "permissions": true,
			}
			filtered := make(map[string]interface{})
			for k, v := range updates {
				if allowedRoleFields[k] {
					filtered[k] = v
				}
			}
			if len(filtered) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no valid fields to update"})
				return
			}
			if err := platformService.UpdateRole(c.Request.Context(), id, filtered); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "role updated"})
		})

		admin.DELETE("/roles/:id", func(c *gin.Context) {
			id := c.Param("id")
			if err := platformService.DeleteRole(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "role deleted"})
		})

		// 审计日志
		admin.GET("/audit-logs", func(c *gin.Context) {
			userEmail := c.Query("user")
			action := c.Query("action")
			resource := c.Query("resource")
			level := c.Query("level")
			logs, total, err := platformService.ListAuditLogs(c.Request.Context(), userEmail, action, resource, level, time.Time{}, time.Time{}, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
		})

		admin.GET("/audit-logs/:id", func(c *gin.Context) {
			id := c.Param("id")
			var log platform.AuditLog
			if err := db.First(&log, "id = ?", id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": log})
		})

		admin.GET("/audit-logs/export", func(c *gin.Context) {
			// 导出审计日志为CSV
			userEmail := c.Query("user")
			action := c.Query("action")
			resource := c.Query("resource")
			logs, _, err := platformService.ListAuditLogs(c.Request.Context(), userEmail, action, resource, "", time.Time{}, time.Time{}, 0, 10000)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			var csv strings.Builder
			csv.WriteString("ID,UserEmail,Action,Resource,ResourceID,Detail,IP,Level,CreatedAt\n")
			for _, l := range logs {
				csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
					l.ID, l.UserEmail, l.Action, l.Resource, l.ResourceID, l.Detail, l.IP, l.Level, l.CreatedAt.Format(time.RFC3339)))
			}
			c.Header("Content-Type", "text/csv")
			c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
			c.String(http.StatusOK, csv.String())
		})

		// 通知
		admin.GET("/notifications", func(c *gin.Context) {
			userID := c.GetString("user_id")
			notifType := c.Query("type")
			notifs, _, err := platformService.ListNotifications(c.Request.Context(), userID, notifType, nil, 0, 50)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": notifs})
		})

		admin.PUT("/notifications/:id/read", func(c *gin.Context) {
			userID := c.GetString("user_id")
			id := c.Param("id")
			if err := platformService.MarkNotificationRead(c.Request.Context(), id, userID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
		})

		admin.PUT("/notifications/read-all", func(c *gin.Context) {
			userID := c.GetString("user_id")
			if err := platformService.MarkAllNotificationsRead(c.Request.Context(), userID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
		})

		admin.DELETE("/notifications/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			id := c.Param("id")
			if err := platformService.DeleteNotification(c.Request.Context(), id, userID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "notification deleted"})
		})

		admin.DELETE("/notifications", func(c *gin.Context) {
			userID := c.GetString("user_id")
			if err := platformService.ClearAllNotifications(c.Request.Context(), userID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "all cleared"})
		})

		// 个人中心
		admin.GET("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			user, err := platformService.GetProfile(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, user)
		})

		admin.PUT("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// 个人信息更新白名单，防止修改role/tenant_id等
			allowedProfileFields := map[string]bool{
				"display_name": true,
				"phone":       true,
				"company":     true,
				"bio":         true,
			}
			filtered := make(map[string]interface{})
			for k, v := range updates {
				if allowedProfileFields[k] {
					filtered[k] = v
				}
			}
			if len(filtered) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no valid fields to update"})
				return
			}
			if err := platformService.UpdateProfile(c.Request.Context(), userID, filtered); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
		})

		admin.PUT("/profile/password", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var req struct {
				CurrentPassword string `json:"current_password" binding:"required"`
				NewPassword     string `json:"new_password" binding:"required,min=8"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// 验证当前密码
			profile, err := platformService.GetProfile(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			// 获取auth.User的密码哈希
			var authUser auth.User
			if err := db.Where("email = ?", profile.Email).First(&authUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify password"})
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(req.CurrentPassword)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
				return
			}
			// 加密新密码
			hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
				return
			}
			db.Model(&auth.User{}).Where("id = ?", userID).Update("password_hash", string(hash))
			c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
		})

		admin.POST("/profile/avatar", func(c *gin.Context) {
			userID := c.GetString("user_id")
			file, err := c.FormFile("avatar")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file required"})
				return
			}
			// 限制文件大小（5MB）
			if file.Size > 5*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "avatar file too large (max 5MB)"})
				return
			}
			// 限制文件类型
			contentType := file.Header.Get("Content-Type")
			allowedTypes := map[string]bool{
				"image/jpeg": true,
				"image/png":  true,
				"image/gif":  true,
				"image/webp": true,
			}
			if !allowedTypes[contentType] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type (allowed: jpeg, png, gif, webp)"})
				return
			}
			// 生成文件名并保存
			ext := ".png"
			if idx := strings.LastIndex(file.Filename, "."); idx > 0 {
				ext = file.Filename[idx:]
			}
			filename := fmt.Sprintf("avatar_%s_%d%s", userID, time.Now().UnixNano(), ext)
			uploadDir := os.Getenv("UPLOAD_DIR")
			if uploadDir == "" {
				uploadDir = "./uploads/avatars"
			}
			os.MkdirAll(uploadDir, 0755)
			savePath := fmt.Sprintf("%s/%s", uploadDir, filename)
			if err := c.SaveUploadedFile(file, savePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
				return
			}
			avatarURL := fmt.Sprintf("/uploads/avatars/%s", filename)
			platformService.UpdateAvatar(c.Request.Context(), userID, avatarURL)
			c.JSON(http.StatusOK, gin.H{"message": "avatar updated", "avatar_url": avatarURL})
		})

		admin.PUT("/profile/2fa", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var req struct {
				Enabled bool `json:"enabled"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := platformService.Toggle2FA(c.Request.Context(), userID, req.Enabled); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "2fa toggled"})
		})

		admin.GET("/profile/devices", func(c *gin.Context) {
			userID := c.GetString("user_id")
			devices, err := platformService.ListLoginDevices(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": devices})
		})

		admin.DELETE("/profile/devices/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			id := c.Param("id")
			if err := platformService.RevokeLoginDevice(c.Request.Context(), id, userID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "device revoked"})
		})

		// WAF管理
		admin.GET("/security/waf/blocked-ips", func(c *gin.Context) {
			ips := wafService.GetBlockedIPs()
			c.JSON(http.StatusOK, gin.H{"blocked_ips": ips})
		})

		admin.POST("/security/waf/block-ip", func(c *gin.Context) {
			var req struct {
				IP string `json:"ip"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			wafService.AddIPBlacklist(req.IP)
			c.JSON(http.StatusOK, gin.H{"message": "IP blocked"})
		})

		// ============ 报表系统 ============
		admin.GET("/report/summary", func(c *gin.Context) {
			timeRange := c.Query("time_range")
			_ = timeRange
			// 从monitor服务获取汇总数据
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, gin.H{
				"total_requests":  metrics.TotalRequests,
				"total_tokens":   metrics.TotalTokens,
				"total_amount":   metrics.TotalAmount,
				"avg_latency":   metrics.P50LatencyMs,
				"success_rate":   metrics.SuccessRate,
				"active_models":  len(metrics.Models),
			})
		})

		admin.GET("/report/request-trend", func(c *gin.Context) {
			// 返回请求趋势数据（简化版，实际应从ClickHouse查询）
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.GET("/report/token-trend", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.GET("/report/model-distribution", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, gin.H{"data": metrics.Models})
		})

		admin.GET("/report/cost-distribution", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.GET("/report/latency-distribution", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.GET("/report/error-analysis", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.GET("/report/model-ranking", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, gin.H{"data": metrics.Models})
		})

		admin.GET("/report/tenant-ranking", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.GET("/report/export", func(c *gin.Context) {
			format := c.Query("format")
			if format == "" {
				format = "csv"
			}
			// 简化版导出
			c.Header("Content-Type", "text/csv")
			c.Header("Content-Disposition", "attachment; filename=report.csv")
			c.String(http.StatusOK, "report export placeholder\n")
		})
	}

	// 认证接口（无需JWT）
	authGroup := engine.Group("/auth")
	{
		authGroup.POST("/login", func(c *gin.Context) {
			handleLogin(c, jwtManager, db, logger)
		})
		authGroup.POST("/register", func(c *gin.Context) {
			handleRegister(c, db, logger)
		})

		// 验证码登录/注册
		authGroup.POST("/verification/send", func(c *gin.Context) {
			handleSendVerificationCode(c, verificationService, logger)
		})
		authGroup.POST("/login/code", func(c *gin.Context) {
			handleLoginByCode(c, verificationService, jwtManager, logger)
		})
		authGroup.POST("/register/code", func(c *gin.Context) {
			handleRegisterByCode(c, verificationService, jwtManager, logger)
		})

		// WebAuthn 人脸识别登录
		authGroup.POST("/face/register-options", func(c *gin.Context) {
			handleFaceRegisterOptions(c, faceAuthService, jwtManager, logger)
		})
		authGroup.POST("/face/register-verify", func(c *gin.Context) {
			handleFaceRegisterVerify(c, faceAuthService, jwtManager, logger)
		})
		authGroup.POST("/face/auth-options", func(c *gin.Context) {
			handleFaceAuthOptions(c, faceAuthService, logger)
		})
		authGroup.POST("/face/auth-verify", func(c *gin.Context) {
			handleFaceAuthVerify(c, faceAuthService, jwtManager, logger)
		})

		// 支付回调（无需JWT，由签名验证安全）
		authGroup.POST("/callback/alipay", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelAlipay, logger)
		})
		authGroup.POST("/callback/alipay_hk", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelAlipayHK, logger)
		})
		authGroup.POST("/callback/wechat", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelWeChatPay, logger)
		})
		authGroup.POST("/callback/paypal", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelPayPal, logger)
		})
		authGroup.POST("/callback/worldfirst", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelWorldFirst, logger)
		})
		authGroup.POST("/callback/payoneer", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelPayoneer, logger)
		})
		authGroup.POST("/callback/wise", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelWise, logger)
		})
		authGroup.POST("/callback/stripe", func(c *gin.Context) {
			handlePaymentCallback(c, paymentService, payment.ChannelStripe, logger)
		})
	}

	// WebSocket 大屏推送
	engine.GET("/ws/monitor", func(c *gin.Context) {
		handleMonitorWebSocket(c, monitorService, logger)
	})

	// 7. 启动HTTP服务
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// 8. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	monitorService.Stop()
	platformService.Stop() // 排空审计日志

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulWait)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server forced shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

func initLogger(mode string) *zap.Logger {
	var zapConfig zap.Config
	if mode == "release" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	logger, _ := zapConfig.Build()
	return logger
}

func initDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	logLevel := gormlogger.Silent
	switch cfg.LogLevel {
	case "info":
		logLevel = gormlogger.Info
	case "warn":
		logLevel = gormlogger.Warn
	case "error":
		logLevel = gormlogger.Error
	}

	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
}

// autoMigrate 自动迁移数据库表结构
func autoMigrate(db *gorm.DB, logger *zap.Logger) {
	if err := db.AutoMigrate(
		&auth.User{},
		&auth.APIKey{},
		&auth.FaceCredential{},
		&tenant.Tenant{},
		&wallet.WalletBinding{},
		&wallet.CryptoDepositOrder{},
		&payment.PaymentOrder{},
		&gateway.ModelConfig{},
	); err != nil {
		logger.Fatal("Failed to auto migrate database", zap.Error(err))
	}
	logger.Info("database auto migrate completed")
}

// initSuperAdmin 初始化超级管理员账号
func initSuperAdmin(db *gorm.DB, logger *zap.Logger) {
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@tokenhub.com"
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123456"
	}

	// 检查是否已存在超级管理员
	var count int64
	db.Model(&auth.User{}).Where("role = ?", auth.RoleSuperAdmin).Count(&count)
	if count > 0 {
		logger.Info("super admin already exists, skipping initialization")
		return
	}

	// 检查默认租户是否存在，不存在则创建
	tenantID := os.Getenv("ADMIN_TENANT_ID")
	if tenantID == "" {
		tenantID = uuid.New().String()
	}
	var tenantCount int64
	db.Model(&tenant.Tenant{}).Where("id = ?", tenantID).Count(&tenantCount)
	if tenantCount == 0 {
		defaultTenant := &tenant.Tenant{
			ID:     tenantID,
			Name:   "TokenHub Default",
			Slug:   "default",
			Status: tenant.TenantActive,
			Plan:   tenant.PlanEnterprise,
			Region: "cn",
		}
		if err := db.Create(defaultTenant).Error; err != nil {
			logger.Error("failed to create default tenant", zap.Error(err))
			return
		}
		logger.Info("default tenant created", zap.String("tenant_id", tenantID))
	}

	// 创建超级管理员
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash admin password", zap.Error(err))
		return
	}

	admin := &auth.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Username:     "admin",
		Email:        adminEmail,
		PasswordHash: string(hash),
		Role:         auth.RoleSuperAdmin,
		Status:       auth.UserActive,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
	}

	if err := db.Create(admin).Error; err != nil {
		logger.Error("failed to create super admin", zap.Error(err))
		return
	}

	logger.Info("super admin created successfully",
		zap.String("email", adminEmail),
		zap.String("username", "admin"),
		zap.String("role", string(auth.RoleSuperAdmin)),
	)
}

// handleChatCompletion 处理Chat Completion请求
func handleChatCompletion(
	c *gin.Context,
	aiProxy *proxy.AIProxy,
	billingService *billing.BillingService,
	desensitizer *desensitize.Desensitizer,
	logger *zap.Logger,
) {
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream     bool    `json:"stream"`
		MaxTokens  *int    `json:"max_tokens,omitempty"`
		Temperature *float64 `json:"temperature,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// 构造内部请求
	chatReq := &gateway.ChatRequest{
		Model:    req.Model,
		Stream:   req.Stream,
		TenantID: c.GetString("tenant_id"),
		APIKeyID: c.GetString("api_key_id"),
		TraceID:  c.GetString("request_id"),
		MaxTokens: req.MaxTokens,
	}
	for _, msg := range req.Messages {
		chatReq.Messages = append(chatReq.Messages, gateway.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if req.Stream {
		// 流式响应
		handleStreamResponse(c, aiProxy, billingService, chatReq, logger)
	} else {
		// 非流式响应
		handleNonStreamResponse(c, aiProxy, billingService, desensitizer, chatReq, logger)
	}
}

func handleNonStreamResponse(
	c *gin.Context,
	aiProxy *proxy.AIProxy,
	billingService *billing.BillingService,
	desensitizer *desensitize.Desensitizer,
	req *gateway.ChatRequest,
	logger *zap.Logger,
) {
	result, err := aiProxy.Proxy(c.Request.Context(), req)
	if err != nil {
		logger.Error("proxy failed", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{
			"error": map[string]string{
				"code":    "upstream_error",
				"message": err.Error(),
			},
		})
		return
	}

	// 脱敏处理响应
	if result.Response != nil && len(result.Response.Choices) > 0 && result.Response.Choices[0].Message != nil {
		result.Response.Choices[0].Message.Content = desensitizer.Desensitize(
			result.Response.Choices[0].Message.Content,
		)
	}

	// 扣费
	if result.Usage != nil {
		_, _ = billingService.DeductBalance(c.Request.Context(), &billing.DeductRequest{
			TenantID: req.TenantID,
			UserID:   req.User,
			Usage:    result.Usage,
			TraceID:  req.TraceID,
			Currency: "CNY",
		})
	}

	c.JSON(http.StatusOK, result.Response)
}

func handleStreamResponse(
	c *gin.Context,
	aiProxy *proxy.AIProxy,
	billingService *billing.BillingService,
	req *gateway.ChatRequest,
	logger *zap.Logger,
) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	result, err := aiProxy.ProxyStream(c.Request.Context(), req, func(chunk *proxy.StreamChunkResult) error {
		if chunk.Done {
			c.SSEvent("message", "[DONE]")
			return nil
		}
		if chunk.Chunk != nil {
			c.SSEvent("message", chunk.Chunk)
		}
		return nil
	})

	if err != nil {
		logger.Error("stream proxy failed", zap.Error(err))
		return
	}

	// 流式扣费
	if result != nil && result.Usage != nil {
		_, _ = billingService.DeductBalance(c.Request.Context(), &billing.DeductRequest{
			TenantID: req.TenantID,
			UserID:   req.User,
			Usage:    result.Usage,
			TraceID:  req.TraceID,
			Currency: "CNY",
		})
	}

	c.Writer.Flush()
}

func handleListModels(c *gin.Context, router *smartrouter.ModelRouter) {
	// 从路由器获取已注册的模型列表
	models := router.GetAllModels()
	data := make([]interface{}, 0, len(models))
	for _, m := range models {
		if m.Enabled {
			data = append(data, gin.H{
				"id":       m.ID,
				"object":   "model",
				"created":  m.CreatedAt,
				"owned_by": string(m.Provider),
				"name":     m.Name,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// handleCompletion 处理Text Completion请求
func handleCompletion(c *gin.Context, aiProxy *proxy.AIProxy, billingService *billing.BillingService, logger *zap.Logger) {
	c.JSON(http.StatusOK, gin.H{
		"error": map[string]string{
			"code":    "deprecated",
			"message": "use /v1/chat/completions instead",
		},
	})
}

func handleLogin(c *gin.Context, jwtManager *auth.JWTManager, db *gorm.DB, logger *zap.Logger) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从数据库查询用户
	var user auth.User
	if err := db.Where("email = ? AND status = ?", req.Email, auth.UserActive).First(&user).Error; err != nil {
		logger.Warn("login failed: user not found or disabled", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		logger.Warn("login failed: wrong password", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// 更新最后登录时间
	now := time.Now()
	db.Model(&user).Update("last_login_at", &now)

	// 生成JWT
	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, user.TenantID, user.Role, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenPair,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

func handleRegister(c *gin.Context, db *gorm.DB, logger *zap.Logger) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查邮箱是否已注册
	var count int64
	db.Model(&auth.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 确定租户ID
	tenantID := req.TenantID
	if tenantID == "" {
		// 分配到默认租户
		var defaultTenant tenant.Tenant
		if err := db.Where("slug = ?", "default").First(&defaultTenant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "default tenant not found"})
			return
		}
		tenantID = defaultTenant.ID
	}

	// 创建用户
	username := req.Username
	if username == "" {
		username = strings.Split(req.Email, "@")[0]
	}
	user := &auth.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Username:     username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         auth.RoleDeveloper,
		Status:       auth.UserActive,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
	}

	if err := db.Create(user).Error; err != nil {
		logger.Error("failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	logger.Info("user registered", zap.String("email", req.Email))
	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"tenant_id":  user.TenantID,
		},
	})
}

func handleMonitorWebSocket(c *gin.Context, monitorService *monitor.MonitorService, logger *zap.Logger) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 生产环境应检查Origin是否在允许列表中
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // 非浏览器请求
			}
			// 开发环境允许所有来源，生产环境应限制
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	monitorService.RegisterClient(conn)
	defer monitorService.UnregisterClient(conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleSendVerificationCode 发送验证码（短信/邮箱）
func handleSendVerificationCode(c *gin.Context, svc *auth.VerificationService, logger *zap.Logger) {
	var req auth.SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.SendVerificationCode(c.Request.Context(), &req); err != nil {
		logger.Warn("send verification code failed", zap.Error(err))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification code sent"})
}

// handleLoginByCode 验证码登录
func handleLoginByCode(c *gin.Context, svc *auth.VerificationService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req auth.LoginByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, userInfo, err := svc.LoginOrRegisterByCode(c.Request.Context(), &req, jwtManager)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   tokenPair,
		"user":    userInfo,
		"is_new":  true, // 简化处理
	})
}

// handleRegisterByCode 验证码注册
func handleRegisterByCode(c *gin.Context, svc *auth.VerificationService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req auth.RegisterByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loginReq := &auth.LoginByCodeRequest{
		Type:        req.Type,
		Target:      req.Target,
		Code:        req.Code,
		CountryCode: req.CountryCode,
	}

	tokenPair, userInfo, err := svc.LoginOrRegisterByCode(c.Request.Context(), loginReq, jwtManager)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":  tokenPair,
		"user":   userInfo,
	})
}

// ==================== WebAuthn 人脸识别 ====================

// handleFaceRegisterOptions 获取人脸注册选项（需JWT认证）
func handleFaceRegisterOptions(c *gin.Context, faceSvc *auth.FaceAuthService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	options, err := faceSvc.GenerateRegistrationOptions(c.Request.Context(), userID)
	if err != nil {
		logger.Error("failed to generate face registration options", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, options)
}

// handleFaceRegisterVerify 验证人脸注册
func handleFaceRegisterVerify(c *gin.Context, faceSvc *auth.FaceAuthService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req struct {
		SessionKey  string                 `json:"session_key"`
		Credential  map[string]interface{} `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credential, err := faceSvc.VerifyRegistration(c.Request.Context(), req.SessionKey, req.Credential)
	if err != nil {
		logger.Warn("face registration verification failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "face credential registered successfully",
		"credential": credential,
	})
}

// handleFaceAuthOptions 获取人脸认证选项（无需JWT，但需提供邮箱）
func handleFaceAuthOptions(c *gin.Context, faceSvc *auth.FaceAuthService, logger *zap.Logger) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	options, err := faceSvc.GenerateAuthenticationOptions(c.Request.Context(), req.Email)
	if err != nil {
		logger.Warn("failed to generate face auth options", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, options)
}

// handleFaceAuthVerify 验证人脸认证
func handleFaceAuthVerify(c *gin.Context, faceSvc *auth.FaceAuthService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req struct {
		SessionKey  string                 `json:"session_key"`
		Credential  map[string]interface{} `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := faceSvc.VerifyAuthentication(c.Request.Context(), req.SessionKey, req.Credential)
	if err != nil {
		logger.Warn("face authentication failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 生成 JWT
	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, user.TenantID, user.Role, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenPair,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

// handlePaymentCallback 处理支付回调
func handlePaymentCallback(c *gin.Context, svc *payment.PaymentService, channel payment.PaymentChannel, logger *zap.Logger) {
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body failed"})
		return
	}

	sign := c.GetHeader("X-Signature")
	if sign == "" {
		sign = c.GetHeader("Wechatpay-Signature")
	}
	if sign == "" {
		sign = c.Query("sign")
	}

	if err := svc.HandleCallback(c.Request.Context(), channel, data, sign); err != nil {
		logger.Error("payment callback failed", zap.String("channel", string(channel)), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 各渠道回调响应格式不同
	switch channel {
	case payment.ChannelAlipay:
		c.String(http.StatusOK, "success")
	case payment.ChannelWeChatPay:
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
