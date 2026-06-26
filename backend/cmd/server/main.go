package main

import (
	"bytes"
	"context"
	"encoding/json"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tokenhub/backend/internal/auth"
	"github.com/tokenhub/backend/internal/billing"
	"github.com/tokenhub/backend/internal/security/audit"
	"github.com/tokenhub/backend/internal/channel"
	tokencore "github.com/tokenhub/backend/internal/token"
	tokencache "github.com/tokenhub/backend/internal/cache"
	"github.com/tokenhub/backend/internal/common/middleware"
	"github.com/tokenhub/backend/internal/config"
	"github.com/tokenhub/backend/internal/distribution"
	"github.com/tokenhub/backend/internal/subscription"
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

	// 配置校验
	warnings, errors := cfg.Validate()
	for _, w := range warnings {
		fmt.Printf("[CONFIG WARNING] %s\n", w)
	}
	if len(errors) > 0 && cfg.Server.Mode == "release" {
		panic(fmt.Sprintf("Configuration errors (must fix for production):\n  %s", strings.Join(errors, "\n  ")))
	}
	for _, e := range errors {
		fmt.Printf("[CONFIG ERROR] %s\n", e)
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

	// 3.3 初始化订阅服务
	subSvc := subscription.NewSubscriptionService(logger, db, nil)
	if err := subSvc.SeedDefaultPlans(context.Background()); err != nil {
		logger.Warn("Failed to seed default plans", zap.Error(err))
	}

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
	_ = i18nService // 通过中间件或路由handler中使用

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

	// 密钥保险库（优先使用环境变量，其次使用配置文件）
	kvMasterKey := os.Getenv("KEYVAULT_MASTER_KEY")
	if kvMasterKey == "" {
		kvMasterKey = cfg.KeyVault.MasterKey
	}
	keyVault := keyvault.NewKeyVault(logger, rdb, kvMasterKey)

	// 分销服务
	distService := distribution.NewDistributionService(logger, rdb)

	// 订阅服务
	subService := subscription.NewSubscriptionService(logger, db, rdb)

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

	// 渠道服务（渠道管理+健康检查+Key轮换）
	channelService := channel.NewChannelService(logger, db)
	channelService.AutoMigrate()

	// 启动渠道健康检查（每5分钟自动检测所有启用渠道，失败率>50%自动禁用）
	healthChecker := channel.NewHealthChecker(channelService, logger, 5*time.Minute)
	healthChecker.Start()
	defer healthChecker.Stop()

	// 令牌服务（sk-xxx令牌二次分发）
	tokenService := tokencore.NewTokenService(logger, db)
	tokenService.AutoMigrate()

	// 租户服务
	tenantService := tenant.NewTenantService(logger, db)
	tenantService.AutoMigrate()

	// 审计服务
	auditService := audit.NewAuditService(logger, db)
	auditService.AutoMigrate()

	// 验证码服务（短信+邮箱，使用配置文件中的密钥）
	smsSender := auth.NewAliyunSMSSender(cfg.Verification.SMS.Aliyun.AccessKeyID, cfg.Verification.SMS.Aliyun.AccessKeySecret, cfg.Verification.SMS.Aliyun.SignName, cfg.Verification.SMS.Aliyun.TemplateCode)
	emailSender := auth.NewSMTPEmailSender(cfg.Verification.Email.SMTP.Host, cfg.Verification.Email.SMTP.Port, cfg.Verification.Email.SMTP.Username, cfg.Verification.Email.SMTP.Password, cfg.Verification.Email.SMTP.FromAddr, cfg.Verification.Email.SMTP.FromName)
	verificationService := auth.NewVerificationService(logger, rdb, smsSender, emailSender)

	// 人脸识别服务（WebAuthn）- 使用配置或环境变量设置 origin
	webAuthnOrigin := os.Getenv("WEBAUTHN_ORIGIN")
	if webAuthnOrigin == "" {
		webAuthnOrigin = "http://localhost:3001"
		if cfg.Server.Mode == "release" {
			webAuthnOrigin = "https://tokenshopai.com"
		}
	}
	faceAuthService := auth.NewFaceAuthService(db, rdb, logger, "localhost", "TokenHub", webAuthnOrigin)

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
	engine.Use(middleware.CORSMiddleware(cfg.Security.CORS.AllowedOrigins))
	engine.Use(middleware.RequestIDMiddleware())
	engine.Use(middleware.MetricsMiddleware(monitorService, logger))

	// 健康检查（含详细依赖状态）
	engine.GET("/health", func(c *gin.Context) {
		checks := gin.H{}
		overall := "healthy"

		// 检查数据库连接
		sqlDB, err := db.DB()
		dbStatus := "ok"
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "error"
			overall = "degraded"
		}
		// 数据库连接池统计
		dbStats := gin.H{"status": dbStatus}
		if sqlDB != nil {
			stats := sqlDB.Stats()
			dbStats["open_connections"] = stats.OpenConnections
			dbStats["idle_connections"] = stats.Idle
			dbStats["in_use"] = stats.InUse
		}
		checks["database"] = dbStats

		// 检查Redis连接
		redisStatus := "ok"
		redisLatency := int64(0)
		start := time.Now()
		if rdb.Ping(c.Request.Context()).Err() != nil {
			redisStatus = "error"
			overall = "degraded"
		}
		redisLatency = time.Since(start).Milliseconds()
		checks["redis"] = gin.H{
			"status":  redisStatus,
			"latency": fmt.Sprintf("%dms", redisLatency),
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  overall,
			"version": "0.1.0",
			"time":    time.Now().Format(time.RFC3339),
			"checks":  checks,
		})
	})

	// API v1 路由组
	v1 := engine.Group("/v1")
	{

	// Prometheus 指标端点（无需认证，供 Prometheus 抓取）
	engine.GET("/metrics", gin.WrapH(promMetrics))

	// AI 模型调用接口（兼容 OpenAI 格式）
	v1.POST("/chat/completions", middleware.APIKeyMiddleware(rdb), middleware.RateLimitMiddleware(rdb, cfg.Gateway.RateLimitPerSec, time.Second, "api"), func(c *gin.Context) {
		// 订阅检查：验证用户是否有活跃订阅
		userID := c.GetString("user_id")
		if userID != "" {
			active, sub := subService.CheckSubscriptionActive(c.Request.Context(), userID)
			if !active {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": gin.H{
						"message": "No active subscription. Please subscribe to a plan.",
						"type":    "subscription_required",
						"code":    "no_subscription",
					},
				})
				return
			}
			// 检查 Token 配额
			_, plan, err := subService.GetUserSubscriptionPlan(c.Request.Context(), userID)
			if err == nil && plan.TokenLimit > 0 && sub.TokenUsed >= plan.TokenLimit {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": gin.H{
						"message": "Token quota exceeded. Please upgrade your plan.",
						"type":    "quota_exceeded",
						"code":    "token_quota_exceeded",
					},
				})
				return
			}
		}
		// 余额熔断：检查用户余额是否充足
		tenantID := c.GetString("tenant_id")
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
		// 渠道优先路由：优先查找渠道做负载均衡，无渠道时 fallback 到 ModelRouter
		handleChatCompletionWithChannel(c, aiProxy, channelService, billingService, desensitizer, subService, logger)
	})
		v1.POST("/completions", middleware.APIKeyMiddleware(rdb), func(c *gin.Context) {
			handleCompletion(c, aiProxy, billingService, logger)
		})
		v1.GET("/models", middleware.APIKeyMiddleware(rdb), func(c *gin.Context) {
			handleListModels(c, modelRouter)
		})
		// 图像生成 API（兼容 OpenAI /v1/images/generations）
		v1.POST("/images/generations", middleware.APIKeyMiddleware(rdb), func(c *gin.Context) {
			handleImageGeneration(c, modelRouter, billingService, logger)
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

			models := make([]map[string]interface{}, 0, len(result.Data))
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

		// ==================== 订阅管理 ====================
		// 获取所有计划（含禁用）
		admin.GET("/subscription/plans", func(c *gin.Context) {
			planType := c.DefaultQuery("type", "all")
			plans, err := subService.ListPlans(c.Request.Context(), planType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": plans})
		})

		// 创建计划
		admin.POST("/subscription/plans", func(c *gin.Context) {
			var plan subscription.SubscriptionPlan
			if err := c.ShouldBindJSON(&plan); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := subService.CreatePlan(c.Request.Context(), &plan); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "plan created", "plan": plan})
		})

		// 更新计划
		admin.PUT("/subscription/plans/:id", func(c *gin.Context) {
			id := c.Param("id")
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := subService.UpdatePlan(c.Request.Context(), id, updates); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "plan updated"})
		})

		// 删除计划
		admin.DELETE("/subscription/plans/:id", func(c *gin.Context) {
			id := c.Param("id")
			if err := subService.DeletePlan(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "plan deleted"})
		})

		// 订阅统计
		admin.GET("/subscription/stats", func(c *gin.Context) {
			stats, err := subService.GetSubscriptionStats(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": stats})
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
			userID := c.Param("id")
			// 查找用户
			var user auth.User
			if err := db.First(&user, "id = ?", userID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			// 生成临时重置令牌
			resetToken := make([]byte, 32)
			rand.Read(resetToken)
			tokenStr := hex.EncodeToString(resetToken)
			// 存入Redis，1小时过期
			resetKey := fmt.Sprintf("password:reset:%s", tokenStr)
			rdb.Set(c.Request.Context(), resetKey, user.ID, 1*time.Hour)
			// 发送重置邮件
			resetURL := fmt.Sprintf("https://tokenshopai.com/reset-password?token=%s", tokenStr)
			emailBody := fmt.Sprintf(`<h2>密码重置</h2><p>您好 %s，</p><p>请点击以下链接重置您的密码：</p><p><a href="%s">%s</a></p><p>此链接1小时内有效。</p><p>如非本人操作，请忽略此邮件。</p>`, user.Username, resetURL, resetURL)
			if err := emailSender.Send(c.Request.Context(), user.Email, "TokenHub - 密码重置", emailBody); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send reset email"})
				return
			}
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
			// 从Redis时间序列数据中聚合请求趋势
			now := time.Now()
			hours := 24
			if c.Query("range") == "7d" {
				hours = 168
			}
			var data []map[string]interface{}
			for i := hours; i >= 0; i-- {
				t := now.Add(-time.Duration(i) * time.Hour)
				minuteKey := t.Format("200601021504")
				requests, _ := rdb.Get(c.Request.Context(), fmt.Sprintf("metrics:global:requests:%s", minuteKey)).Int64()
				tokens, _ := rdb.Get(c.Request.Context(), fmt.Sprintf("metrics:global:tokens:%s", minuteKey)).Int64()
				data = append(data, map[string]interface{}{
					"time":     t.Format("2006-01-02 15:04"),
					"requests": requests,
					"tokens":   tokens,
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": data})
		})

		admin.GET("/report/token-trend", func(c *gin.Context) {
			now := time.Now()
			hours := 24
			if c.Query("range") == "7d" {
				hours = 168
			}
			var data []map[string]interface{}
			for i := hours; i >= 0; i-- {
				t := now.Add(-time.Duration(i) * time.Hour)
				minuteKey := t.Format("200601021504")
				inputTokens, _ := rdb.Get(c.Request.Context(), fmt.Sprintf("metrics:global:tokens:%s", minuteKey)).Int64()
				outputTokens, _ := rdb.Get(c.Request.Context(), fmt.Sprintf("metrics:global:output_tokens:%s", minuteKey)).Int64()
				data = append(data, map[string]interface{}{
					"time":          t.Format("2006-01-02 15:04"),
					"input_tokens":  inputTokens,
					"output_tokens": outputTokens,
					"total_tokens":  inputTokens + outputTokens,
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": data})
		})

		admin.GET("/report/model-distribution", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, gin.H{"data": metrics.Models})
		})

		admin.GET("/report/cost-distribution", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			var data []map[string]interface{}
			for _, m := range metrics.Models {
				data = append(data, map[string]interface{}{
					"model_id":   m.ModelID,
					"model_name": m.ModelName,
					"provider":   m.Provider,
					"requests":   m.Requests,
					"tokens":     m.Tokens,
					"amount":     m.Tokens * 10, // 简化费用估算
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": data})
		})

		admin.GET("/report/latency-distribution", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			var data []map[string]interface{}
			for _, m := range metrics.Models {
				data = append(data, map[string]interface{}{
					"model_id":       m.ModelID,
					"model_name":     m.ModelName,
					"avg_latency_ms": m.AvgLatencyMs,
					"success_rate":   m.SuccessRate,
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": data})
		})

		admin.GET("/report/error-analysis", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			var data []map[string]interface{}
			for _, m := range metrics.Models {
				if m.Errors > 0 {
					data = append(data, map[string]interface{}{
						"model_id":    m.ModelID,
						"model_name":  m.ModelName,
						"provider":    m.Provider,
						"total_reqs":  m.Requests,
						"errors":      m.Errors,
						"error_rate":  float64(m.Errors) / float64(m.Requests) * 100,
						"circuit":     m.CircuitState,
					})
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": data})
		})

		admin.GET("/report/model-ranking", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, gin.H{"data": metrics.Models})
		})

		admin.GET("/report/tenant-ranking", func(c *gin.Context) {
			metrics := monitorService.GetMetrics()
			c.JSON(http.StatusOK, gin.H{"data": metrics.Tenants})
		})

		admin.GET("/report/export", func(c *gin.Context) {
			format := c.Query("format")
			if format == "" {
				format = "csv"
			}
			timeRange := c.DefaultQuery("time_range", "7d")
			metrics := monitorService.GetMetrics()

			if format == "json" {
				c.Header("Content-Type", "application/json")
				c.Header("Content-Disposition", "attachment; filename=report.json")
				data := map[string]interface{}{
					"export_time": time.Now().Format("2006-01-02 15:04:05"),
					"time_range":  timeRange,
					"summary": map[string]interface{}{
						"total_requests":  metrics.TotalRequests,
						"total_tokens":   metrics.TotalTokens,
						"total_amount":   metrics.TotalAmount,
						"success_rate":   metrics.SuccessRate,
						"avg_latency_ms": metrics.P50LatencyMs,
					},
					"models":  metrics.Models,
					"tenants": metrics.Tenants,
				}
				jsonData, _ := json.Marshal(data)
				c.Data(http.StatusOK, "application/json", jsonData)
				return
			}

			// CSV格式
			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Header("Content-Disposition", "attachment; filename=report.csv")
			var csv strings.Builder
			csv.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
			csv.WriteString("导出时间,时间范围\n")
			csv.WriteString(fmt.Sprintf("%s,%s\n\n", time.Now().Format("2006-01-02 15:04:05"), timeRange))
			csv.WriteString("总请求数,总Token数,总费用(分),成功率,P50延迟(ms)\n")
			csv.WriteString(fmt.Sprintf("%d,%d,%d,%.2f,%.1f\n\n", metrics.TotalRequests, metrics.TotalTokens, metrics.TotalAmount, metrics.SuccessRate, metrics.P50LatencyMs))
			csv.WriteString("模型ID,模型名称,供应商,请求数,Token数,平均延迟(ms)\n")
			for _, m := range metrics.Models {
				csv.WriteString(fmt.Sprintf("%s,%s,%s,%d,%d,%.1f\n", m.ModelID, m.ModelName, m.Provider, m.Requests, m.Tokens, m.AvgLatencyMs))
			}
			c.String(http.StatusOK, csv.String())
		})

		// ============ 租户管理 ============
		admin.GET("/tenants", func(c *gin.Context) {
			status := c.Query("status")
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
			tenants, total, err := tenantService.List(c.Request.Context(), tenant.TenantStatus(status), page, pageSize)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": tenants, "total": total})
		})

		admin.GET("/tenants/:id", func(c *gin.Context) {
			t, err := tenantService.GetByID(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": t})
		})

		admin.POST("/tenants", func(c *gin.Context) {
			var t tenant.Tenant
			if err := c.ShouldBindJSON(&t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := tenantService.Create(c.Request.Context(), &t); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"data": t})
		})

		admin.PUT("/tenants/:id", func(c *gin.Context) {
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := tenantService.Update(c.Request.Context(), c.Param("id"), updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "tenant updated"})
		})

		admin.DELETE("/tenants/:id", func(c *gin.Context) {
			if err := tenantService.Delete(c.Request.Context(), c.Param("id")); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "tenant suspended"})
		})

		admin.GET("/tenants/:id/quotas", func(c *gin.Context) {
			t, err := tenantService.GetByID(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			_ = t
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})

		admin.PUT("/tenants/:id/quotas", func(c *gin.Context) {
			var q tenant.TenantQuota
			if err := c.ShouldBindJSON(&q); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			q.TenantID = c.Param("id")
			if err := tenantService.SetQuota(c.Request.Context(), &q); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "quota updated"})
		})

		// ============ API Key 管理 ============
		// 列出当前用户的 API Keys
		admin.GET("/apikeys", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var keys []auth.APIKey
			query := db.Where("user_id = ? AND tenant_id = ?", userID, tenantID)
			if err := query.Order("created_at DESC").Find(&keys).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 脱敏：不返回 key_hash
			type APIKeySafe struct {
				ID          string           `json:"id"`
				Name        string           `json:"name"`
				KeyPrefix   string           `json:"key_prefix"`
				Permissions []auth.Permission `json:"permissions"`
				Models      []string         `json:"models"`
				RateLimit   int              `json:"rate_limit"`
				QuotaDaily  int64            `json:"quota_daily"`
				Status      auth.APIKeyStatus `json:"status"`
				ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
				CreatedAt   time.Time        `json:"created_at"`
				LastUsedAt  *time.Time       `json:"last_used_at,omitempty"`
			}
			safe := make([]APIKeySafe, 0, len(keys))
			for _, k := range keys {
				safe = append(safe, APIKeySafe{
					ID:          k.ID,
					Name:        k.Name,
					KeyPrefix:   k.KeyPrefix,
					Permissions: k.Permissions,
					Models:      k.Models,
					RateLimit:   k.RateLimit,
					QuotaDaily:  k.QuotaDaily,
					Status:      k.Status,
					ExpiresAt:   k.ExpiresAt,
					CreatedAt:   k.CreatedAt,
					LastUsedAt:  k.LastUsedAt,
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": safe})
		})

		// 创建 API Key
		admin.POST("/apikeys", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				Name        string           `json:"name" binding:"required"`
				Permissions []auth.Permission `json:"permissions"`
				Models      []string         `json:"models"`
				RateLimit   int              `json:"rate_limit"`
				QuotaDaily  int64            `json:"quota_daily"`
				ExpiresAt   *time.Time       `json:"expires_at"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// 生成 sk- 前缀的 API Key
			rawKey := "sk-" + generateSecureRandom(48)
			hash := sha256.Sum256([]byte(rawKey))
			keyHash := hex.EncodeToString(hash[:])
			prefix := rawKey[:8]

			if req.Permissions == nil {
				req.Permissions = []auth.Permission{auth.PermModelRoute}
			}
			if req.RateLimit == 0 {
				req.RateLimit = 60
			}
			if req.QuotaDaily == 0 {
				req.QuotaDaily = 1000000
			}

			apiKey := &auth.APIKey{
				ID:          uuid.New().String(),
				TenantID:    tenantID,
				UserID:      userID,
				Name:        req.Name,
				KeyHash:     keyHash,
				KeyPrefix:   prefix,
				Permissions: req.Permissions,
				Models:      req.Models,
				RateLimit:   req.RateLimit,
				QuotaDaily:  req.QuotaDaily,
				Status:      auth.APIKeyActive,
				ExpiresAt:   req.ExpiresAt,
				CreatedAt:   time.Now(),
			}
			if err := db.Create(apiKey).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 只在创建时返回完整 Key
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"id":         apiKey.ID,
					"name":       apiKey.Name,
					"key":        rawKey,
					"key_prefix": prefix,
					"status":     apiKey.Status,
					"created_at": apiKey.CreatedAt,
				},
				"warning": "请妥善保存此 API Key，系统不会再次显示完整密钥",
			})
		})

		// 吊销 API Key
		admin.PUT("/apikeys/:id/revoke", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			id := c.Param("id")
			result := db.Model(&auth.APIKey{}).Where("id = ? AND user_id = ? AND tenant_id = ?", id, userID, tenantID).Update("status", auth.APIKeyRevoked)
			if result.RowsAffected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "API Key not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "API Key revoked"})
		})

		// 删除 API Key
		admin.DELETE("/apikeys/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			id := c.Param("id")
			result := db.Where("id = ? AND user_id = ? AND tenant_id = ?", id, userID, tenantID).Delete(&auth.APIKey{})
			if result.RowsAffected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "API Key not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "API Key deleted"})
		})

		// ============ 兑换码系统 ============
		admin.GET("/redeem-codes", func(c *gin.Context) {
			var codes []RedeemCode
			if err := db.Order("created_at DESC").Find(&codes).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": codes})
		})

		admin.POST("/redeem-codes/batch", func(c *gin.Context) {
			var req struct {
				Count     int     `json:"count" binding:"required"`
				Amount    float64 `json:"amount" binding:"required"`
				Prefix    string  `json:"prefix"`
				ExpiresAt *string `json:"expires_at"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.Count > 1000 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "batch size too large, max 1000"})
				return
			}
			prefix := req.Prefix
			if prefix == "" {
				prefix = "TH"
			}
			codes := make([]RedeemCode, 0, req.Count)
			for i := 0; i < req.Count; i++ {
				code := prefix + "-" + generateSecureRandom(12)
				rc := RedeemCode{
					ID:        uuid.New().String(),
					Code:      code,
					Amount:    req.Amount,
					Status:    "active",
					CreatedAt: time.Now(),
				}
				if req.ExpiresAt != nil {
					t, _ := time.Parse(time.RFC3339, *req.ExpiresAt)
					rc.ExpiresAt = &t
				}
				codes = append(codes, rc)
			}
			if err := db.Create(&codes).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			rawCodes := make([]string, 0, len(codes))
			for _, rc := range codes {
				rawCodes = append(rawCodes, rc.Code)
			}
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("created %d codes", len(codes)),
				"codes":   rawCodes,
			})
		})

		admin.DELETE("/redeem-codes/:id", func(c *gin.Context) {
			id := c.Param("id")
			if err := db.Where("id = ?", id).Delete(&RedeemCode{}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		// 用户兑换
		admin.POST("/redeem", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				Code string `json:"code" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			var rc RedeemCode
			if err := db.Where("code = ? AND status = ?", req.Code, "active").First(&rc).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "兑换码无效或已使用"})
				return
			}
			if rc.ExpiresAt != nil && rc.ExpiresAt.Before(time.Now()) {
				db.Model(&rc).Update("status", "expired")
				c.JSON(http.StatusBadRequest, gin.H{"error": "兑换码已过期"})
				return
			}
			// 先充值
			if err := billingService.TopUp(c.Request.Context(), tenantID, userID, int64(rc.Amount*100)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 充值成功后再标记已使用
			db.Model(&rc).Updates(map[string]interface{}{"status": "used", "used_by": userID, "used_at": time.Now()})
			c.JSON(http.StatusOK, gin.H{"message": "兑换成功", "amount": rc.Amount})
		})

		// ============ 公告系统 ============
		admin.GET("/announcements", func(c *gin.Context) {
			var anns []Announcement
			query := db.Order("pinned DESC, created_at DESC")
			if c.Query("active") == "true" {
				query = query.Where("status = ?", "active")
			}
			if err := query.Find(&anns).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": anns})
		})

		admin.POST("/announcements", func(c *gin.Context) {
			var req struct {
				Title   string `json:"title" binding:"required"`
				Content string `json:"content" binding:"required"`
				Type    string `json:"type"`
				Pinned  bool   `json:"pinned"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			annType := req.Type
			if annType == "" {
				annType = "info"
			}
			ann := Announcement{
				ID:        uuid.New().String(),
				Title:     req.Title,
				Content:   req.Content,
				Type:      annType,
				Status:    "active",
				Pinned:    req.Pinned,
				CreatedAt: time.Now(),
			}
			if err := db.Create(&ann).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": ann})
		})

		admin.PUT("/announcements/:id", func(c *gin.Context) {
			id := c.Param("id")
			var req struct {
				Title   string `json:"title"`
				Content string `json:"content"`
				Type    string `json:"type"`
				Pinned  *bool  `json:"pinned"`
				Status  string `json:"status"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			updates := map[string]interface{}{}
			if req.Title != "" { updates["title"] = req.Title }
			if req.Content != "" { updates["content"] = req.Content }
			if req.Type != "" { updates["type"] = req.Type }
			if req.Pinned != nil { updates["pinned"] = *req.Pinned }
			if req.Status != "" { updates["status"] = req.Status }
			db.Model(&Announcement{}).Where("id = ?", id).Updates(updates)
			c.JSON(http.StatusOK, gin.H{"message": "updated"})
		})

		admin.DELETE("/announcements/:id", func(c *gin.Context) {
			id := c.Param("id")
			db.Where("id = ?", id).Delete(&Announcement{})
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		// ============ 模型映射 ============
		admin.GET("/model-mappings", func(c *gin.Context) {
			var mappings []ModelMapping
			if err := db.Order("created_at DESC").Find(&mappings).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": mappings})
		})

		admin.POST("/model-mappings", func(c *gin.Context) {
			var req struct {
				SourceModel string `json:"source_model" binding:"required"`
				TargetModel string `json:"target_model" binding:"required"`
				TenantID    string `json:"tenant_id"`
				Priority   int    `json:"priority"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			mapping := ModelMapping{
				ID:          uuid.New().String(),
				SourceModel: req.SourceModel,
				TargetModel: req.TargetModel,
				TenantID:    req.TenantID,
				Priority:    req.Priority,
				Enabled:     true,
				CreatedAt:   time.Now(),
			}
			if err := db.Create(&mapping).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": mapping})
		})

		admin.DELETE("/model-mappings/:id", func(c *gin.Context) {
			id := c.Param("id")
			db.Where("id = ?", id).Delete(&ModelMapping{})
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		admin.PUT("/model-mappings/:id/toggle", func(c *gin.Context) {
			id := c.Param("id")
			var mapping ModelMapping
			if err := db.Where("id = ?", id).First(&mapping).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			db.Model(&mapping).Update("enabled", !mapping.Enabled)
			c.JSON(http.StatusOK, gin.H{"enabled": !mapping.Enabled})
		})

		// ============ 用户分组+倍率 ============
		admin.GET("/user-groups", func(c *gin.Context) {
			var groups []UserGroup
			if err := db.Order("multiplier ASC").Find(&groups).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": groups})
		})

		admin.POST("/user-groups", func(c *gin.Context) {
			var req struct {
				Name       string  `json:"name" binding:"required"`
				Multiplier float64 `json:"multiplier" binding:"required"`
				RpmLimit   int     `json:"rpm_limit"`
				TpmLimit   int64   `json:"tpm_limit"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			group := UserGroup{
				ID:         uuid.New().String(),
				Name:       req.Name,
				Multiplier: req.Multiplier,
				RpmLimit:   req.RpmLimit,
				TpmLimit:   req.TpmLimit,
				CreatedAt:  time.Now(),
			}
			if err := db.Create(&group).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": group})
		})

		admin.PUT("/user-groups/:id", func(c *gin.Context) {
			id := c.Param("id")
			var req struct {
				Name       string  `json:"name"`
				Multiplier float64 `json:"multiplier"`
				RpmLimit   int     `json:"rpm_limit"`
				TpmLimit   int64   `json:"tpm_limit"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			updates := map[string]interface{}{}
			if req.Name != "" { updates["name"] = req.Name }
			if req.Multiplier > 0 { updates["multiplier"] = req.Multiplier }
			if req.RpmLimit > 0 { updates["rpm_limit"] = req.RpmLimit }
			if req.TpmLimit > 0 { updates["tpm_limit"] = req.TpmLimit }
			db.Model(&UserGroup{}).Where("id = ?", id).Updates(updates)
			c.JSON(http.StatusOK, gin.H{"message": "updated"})
		})

		admin.DELETE("/user-groups/:id", func(c *gin.Context) {
			id := c.Param("id")
			db.Where("id = ?", id).Delete(&UserGroup{})
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		// 邀请奖励管理
		admin.GET("/referrals", func(c *gin.Context) {
			var refs []ReferralInvite
			db.Order("created_at DESC").Find(&refs)
			c.JSON(http.StatusOK, gin.H{"data": refs})
		})
		admin.GET("/referrals/my-code", func(c *gin.Context) {
			userID := c.GetString("user_id")
			// 为用户生成唯一邀请码（基于用户ID的短码）
			code := fmt.Sprintf("INV-%s", userID[:8])
			c.JSON(http.StatusOK, gin.H{"code": code})
		})
		admin.POST("/referrals/settle/:id", func(c *gin.Context) {
			id := c.Param("id")
			var ref ReferralInvite
			if err := db.First(&ref, "id = ?", id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			now := time.Now()
			db.Model(&ref).Updates(map[string]interface{}{"status": "rewarded", "rewarded_at": &now})
			// 给邀请人加余额
			if ref.InviterID != "" {
				rewardCents := int64(ref.RewardAmount * 100) // 元→分
				billingService.TopUp(c.Request.Context(), c.GetString("tenant_id"), ref.InviterID, rewardCents)
			}
			c.JSON(http.StatusOK, gin.H{"message": "settled"})
		})

		// 用户引导进度
		admin.GET("/onboarding", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var prog OnboardingProgress
			if err := db.Where("user_id = ?", userID).First(&prog).Error; err != nil {
				c.JSON(http.StatusOK, gin.H{"current_step": 0, "completed": false, "skipped": false})
				return
			}
			c.JSON(http.StatusOK, prog)
		})
		admin.PUT("/onboarding", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var req struct {
				CurrentStep *int  `json:"current_step"`
				Completed   *bool `json:"completed"`
				Skipped     *bool `json:"skipped"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			var prog OnboardingProgress
			if err := db.Where("user_id = ?", userID).First(&prog).Error; err != nil {
				prog = OnboardingProgress{
					ID:      uuid.New().String(),
					UserID:  userID,
				}
			}
			if req.CurrentStep != nil {
				prog.CurrentStep = *req.CurrentStep
			}
			if req.Completed != nil {
				prog.Completed = *req.Completed
				if *req.Completed {
					now := time.Now()
					prog.CompletedAt = &now
				}
			}
			if req.Skipped != nil {
				prog.Skipped = *req.Skipped
			}
			prog.UpdatedAt = time.Now()
			db.Save(&prog)
			c.JSON(http.StatusOK, prog)
		})

		// ============ 渠道管理 ============
		// 列出渠道（支持筛选）
		admin.GET("/channels", func(c *gin.Context) {
			var enabled *bool
			if v := c.Query("enabled"); v == "true" {
				t := true
				enabled = &t
			} else if v == "false" {
				f := false
				enabled = &f
			}
			channels, err := channelService.ListChannels(c.Request.Context(), c.Query("provider"), c.Query("group"), c.Query("model_name"), enabled)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": channels})
		})

		// 获取渠道详情
		admin.GET("/channels/:id", func(c *gin.Context) {
			ch, err := channelService.GetChannel(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": ch})
		})

		// 创建渠道
		admin.POST("/channels", func(c *gin.Context) {
			var ch channel.Channel
			if err := c.ShouldBindJSON(&ch); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			tenantID := c.GetString("tenant_id")
			if tenantID != "" {
				ch.TenantID = tenantID
			}
			if err := channelService.CreateChannel(c.Request.Context(), &ch); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"data": ch})
		})

		// 更新渠道
		admin.PUT("/channels/:id", func(c *gin.Context) {
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := channelService.UpdateChannel(c.Request.Context(), c.Param("id"), updates); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "updated"})
		})

		// 删除渠道
		admin.DELETE("/channels/:id", func(c *gin.Context) {
			if err := channelService.DeleteChannel(c.Request.Context(), c.Param("id")); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		// 切换渠道启用/禁用
		admin.PUT("/channels/:id/toggle", func(c *gin.Context) {
			ch, err := channelService.GetChannel(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
				return
			}
			newEnabled := !ch.Enabled
			if err := channelService.UpdateChannel(c.Request.Context(), c.Param("id"), map[string]interface{}{"enabled": newEnabled}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"enabled": newEnabled})
		})

		// 测试渠道
		admin.POST("/channels/:id/test", func(c *gin.Context) {
			result := channelService.TestChannel(c.Request.Context(), c.Param("id"))
			c.JSON(http.StatusOK, gin.H{"data": result})
		})

		// 批量测试渠道
		admin.POST("/channels/batch-test", func(c *gin.Context) {
			results := channelService.BatchTestChannels(c.Request.Context())
			c.JSON(http.StatusOK, gin.H{"data": results})
		})

		// 渠道统计
		admin.GET("/channels/stats", func(c *gin.Context) {
			stats := channelService.GetChannelStats(c.Request.Context())
			c.JSON(http.StatusOK, gin.H{"data": stats})
		})

		// ============ 令牌管理 ============
		// 列出令牌
		admin.GET("/tokens", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			tokens, err := tokenService.ListTokens(c.Request.Context(), userID, tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": tokens})
		})

		// 获取令牌详情
		admin.GET("/tokens/:id", func(c *gin.Context) {
			t, err := tokenService.GetToken(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": t})
		})

		// 创建令牌
		admin.POST("/tokens", func(c *gin.Context) {
			var req tokencore.CreateTokenRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			t, err := tokenService.CreateToken(c.Request.Context(), userID, tenantID, &req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"data": t})
		})

		// 吊销令牌
		admin.PUT("/tokens/:id/revoke", func(c *gin.Context) {
			if err := tokenService.RevokeToken(c.Request.Context(), c.Param("id")); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "revoked"})
		})

		// 删除令牌
		admin.DELETE("/tokens/:id", func(c *gin.Context) {
			if err := tokenService.DeleteToken(c.Request.Context(), c.Param("id")); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		// 更新令牌
		admin.PUT("/tokens/:id", func(c *gin.Context) {
			var req struct {
				Name         string   `json:"name"`
				QuotaTotal   *int64   `json:"quota_total"`
				Models       []string `json:"models"`
				AllowedIPs   []string `json:"allowed_ips"`
				RateLimitRPM *int     `json:"rate_limit_rpm"`
				RateLimitTPM *int     `json:"rate_limit_tpm"`
				Group        string   `json:"group"`
				ExpiresAt    *int64   `json:"expires_at"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			updates := map[string]interface{}{}
			if req.Name != "" {
				updates["name"] = req.Name
			}
			if req.QuotaTotal != nil {
				updates["quota_total"] = *req.QuotaTotal
			}
			if req.Models != nil {
				updates["models"] = req.Models
			}
			if req.AllowedIPs != nil {
				updates["allowed_ips"] = req.AllowedIPs
			}
			if req.RateLimitRPM != nil {
				updates["rate_limit_rpm"] = *req.RateLimitRPM
			}
			if req.RateLimitTPM != nil {
				updates["rate_limit_tpm"] = *req.RateLimitTPM
			}
			if req.Group != "" {
				updates["group_name"] = req.Group
			}
			if req.ExpiresAt != nil {
				updates["expires_at"] = *req.ExpiresAt
			}
			if err := tokenService.UpdateToken(c.Request.Context(), c.Param("id"), updates); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "updated"})
		})
	}

	// ============ 用户端 API（/user/）============
	// 任何已认证用户均可访问，数据按 user_id 隔离
	userGroup := engine.Group("/user")
	userGroup.Use(middleware.AuthMiddleware(jwtManager))
	{
		// 用户余额
		userGroup.GET("/balance", func(c *gin.Context) {
			tenantID := c.GetString("tenant_id")
			userID := c.GetString("user_id")
			balance, err := billingService.GetBalance(c.Request.Context(), tenantID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"balance": balance, "currency": cfg.Billing.DefaultCurrency})
		})

		// 用户 API Key 管理
		userGroup.GET("/apikeys", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var keys []auth.APIKey
			query := db.Where("user_id = ? AND tenant_id = ?", userID, tenantID)
			if err := query.Order("created_at DESC").Find(&keys).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			type APIKeySafe struct {
				ID          string           `json:"id"`
				Name        string           `json:"name"`
				KeyPrefix   string           `json:"key_prefix"`
				Permissions []auth.Permission `json:"permissions"`
				Models      []string         `json:"models"`
				RateLimit   int              `json:"rate_limit"`
				QuotaDaily  int64            `json:"quota_daily"`
				Status      auth.APIKeyStatus `json:"status"`
				ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
				CreatedAt   time.Time        `json:"created_at"`
				LastUsedAt  *time.Time       `json:"last_used_at,omitempty"`
			}
			safe := make([]APIKeySafe, 0, len(keys))
			for _, k := range keys {
				safe = append(safe, APIKeySafe{
					ID: k.ID, Name: k.Name, KeyPrefix: k.KeyPrefix,
					Permissions: k.Permissions, Models: k.Models,
					RateLimit: k.RateLimit, QuotaDaily: k.QuotaDaily,
					Status: k.Status, ExpiresAt: k.ExpiresAt,
					CreatedAt: k.CreatedAt, LastUsedAt: k.LastUsedAt,
				})
			}
			c.JSON(http.StatusOK, gin.H{"data": safe})
		})

		userGroup.POST("/apikeys", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				Name        string           `json:"name" binding:"required"`
				Permissions []auth.Permission `json:"permissions"`
				Models      []string         `json:"models"`
				RateLimit   int              `json:"rate_limit"`
				QuotaDaily  int64            `json:"quota_daily"`
				ExpiresAt   *time.Time       `json:"expires_at"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			rawKey := "sk-" + generateSecureRandom(48)
			hash := sha256.Sum256([]byte(rawKey))
			keyHash := hex.EncodeToString(hash[:])
			prefix := rawKey[:8]
			if req.Permissions == nil {
				req.Permissions = []auth.Permission{auth.PermModelRoute}
			}
			if req.RateLimit == 0 {
				req.RateLimit = 60
			}
			if req.QuotaDaily == 0 {
				req.QuotaDaily = 1000000
			}
			apiKey := &auth.APIKey{
				ID: uuid.New().String(), TenantID: tenantID, UserID: userID,
				Name: req.Name, KeyHash: keyHash, KeyPrefix: prefix,
				Permissions: req.Permissions, Models: req.Models,
				RateLimit: req.RateLimit, QuotaDaily: req.QuotaDaily,
				Status: auth.APIKeyActive, ExpiresAt: req.ExpiresAt, CreatedAt: time.Now(),
			}
			if err := db.Create(apiKey).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{"id": apiKey.ID, "name": apiKey.Name, "key": rawKey, "key_prefix": prefix, "status": apiKey.Status, "created_at": apiKey.CreatedAt},
				"warning": "请妥善保存此 API Key，系统不会再次显示完整密钥",
			})
		})

		userGroup.PUT("/apikeys/:id/revoke", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			id := c.Param("id")
			result := db.Model(&auth.APIKey{}).Where("id = ? AND user_id = ? AND tenant_id = ?", id, userID, tenantID).Update("status", auth.APIKeyRevoked)
			if result.RowsAffected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "API Key not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "API Key revoked"})
		})

		userGroup.DELETE("/apikeys/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			id := c.Param("id")
			result := db.Where("id = ? AND user_id = ? AND tenant_id = ?", id, userID, tenantID).Delete(&auth.APIKey{})
			if result.RowsAffected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "API Key not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "API Key deleted"})
		})

		// 用户调用日志（最近 N 条计费记录）
		userGroup.GET("/usage-logs", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			limit := 50
			if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 && l <= 200 {
				limit = l
			}
			offset := 0
			if o, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && o >= 0 {
				offset = o
			}
			var records []billing.BillingRecord
			query := db.Where("user_id = ? AND tenant_id = ?", userID, tenantID)
			if model := c.Query("model"); model != "" {
				query = query.Where("model_name = ?", model)
			}
			var total int64
			query.Model(&billing.BillingRecord{}).Count(&total)
			if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": records, "total": total, "limit": limit, "offset": offset})
		})

		// 用户月度统计
		userGroup.GET("/stats/monthly", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			now := time.Now()
			startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			var result struct {
				TotalTokens  int64 `json:"total_tokens"`
				TotalAmount  int64 `json:"total_amount"`
				RequestCount int64 `json:"request_count"`
			}
			db.Model(&billing.BillingRecord{}).
				Where("user_id = ? AND tenant_id = ? AND created_at >= ?", userID, tenantID, startOfMonth.Unix()).
				Select("COALESCE(SUM(total_tokens),0) as total_tokens, COALESCE(SUM(amount),0) as total_amount, COUNT(*) as request_count").
				Scan(&result)
			c.JSON(http.StatusOK, gin.H{"data": result})
		})

		// 用户邀请码
		userGroup.GET("/referral-code", func(c *gin.Context) {
			userID := c.GetString("user_id")
			code := fmt.Sprintf("INV-%s", userID[:8])
			c.JSON(http.StatusOK, gin.H{"code": code})
		})

		// 用户邀请记录
		userGroup.GET("/referrals", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var refs []ReferralInvite
			db.Where("inviter_id = ?", userID).Order("created_at DESC").Find(&refs)
			c.JSON(http.StatusOK, gin.H{"data": refs})
		})

		// 兑换码兑换
		userGroup.POST("/redeem", func(c *gin.Context) {
			var req struct {
				Code string `json:"code" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			var code RedeemCode
			if err := db.Where("code = ? AND status = ?", req.Code, "active").First(&code).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "兑换码无效或已使用"})
				return
			}
			if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "兑换码已过期"})
				return
			}
			tenantID := c.GetString("tenant_id")
			userID := c.GetString("user_id")
			amountCents := int64(code.Amount * 100)
			if err := billingService.TopUp(c.Request.Context(), tenantID, userID, amountCents); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			db.Model(&code).Updates(map[string]interface{}{"status": "used", "used_by": userID, "used_at": time.Now().Unix()})
			c.JSON(http.StatusOK, gin.H{"message": "兑换成功", "amount": code.Amount})
		})

		// 订阅管理
		userGroup.GET("/subscription", func(c *gin.Context) {
			userID := c.GetString("user_id")
			sub, plan, err := subService.GetUserSubscriptionPlan(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"data": nil, "message": "no active subscription"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": gin.H{
				"subscription": sub,
				"plan":         plan,
			}})
		})

		// 可用计划列表（用户视角）
		userGroup.GET("/subscription/plans", func(c *gin.Context) {
			planType := c.DefaultQuery("type", "all")
			plans, err := subService.ListPlans(c.Request.Context(), planType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": plans})
		})

		// 订阅计划
		userGroup.POST("/subscription/subscribe", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				PlanID string `json:"plan_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			sub, err := subService.Subscribe(c.Request.Context(), userID, tenantID, req.PlanID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "subscribed successfully", "data": sub})
		})

		// 取消订阅
		userGroup.POST("/subscription/cancel", func(c *gin.Context) {
			userID := c.GetString("user_id")
			if err := subService.CancelSubscription(c.Request.Context(), userID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
		})

		// 订阅历史
		userGroup.GET("/subscription/history", func(c *gin.Context) {
			userID := c.GetString("user_id")
			limit := 20
			subs, err := subService.ListUserSubscriptions(c.Request.Context(), userID, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": subs})
		})

		// 用户个人资料
		userGroup.GET("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var user struct {
				ID          string `json:"id"`
				Email       string `json:"email"`
				Role        string `json:"role"`
				DisplayName string `json:"display_name"`
				AvatarURL   string `json:"avatar_url"`
				CreatedAt   string `json:"created_at"`
			}
			if err := db.Table("users").Where("id = ?", userID).Select("id, email, role, display_name, avatar_url, created_at").Scan(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": user})
		})

		userGroup.PUT("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var req struct {
				DisplayName string `json:"display_name"`
				Phone       string `json:"phone"`
				Company     string `json:"company"`
				Bio         string `json:"bio"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			updates := map[string]interface{}{}
			if req.DisplayName != "" {
				updates["display_name"] = req.DisplayName
			}
			if req.Phone != "" {
				updates["phone"] = req.Phone
			}
			if req.Company != "" {
				updates["company"] = req.Company
			}
			if req.Bio != "" {
				updates["bio"] = req.Bio
			}
			if len(updates) > 0 {
				db.Table("users").Where("id = ?", userID).Updates(updates)
			}
			c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
		})

		// 可用模型列表（用户视角，仅返回启用的模型）
		userGroup.GET("/models", func(c *gin.Context) {
			var models []gateway.ModelConfig
			query := db.Where("enabled = ?", true)
			if provider := c.Query("provider"); provider != "" {
				query = query.Where("provider = ?", provider)
			}
			if err := query.Select("id, name, provider, model_id, input_price, output_price, currency, max_tokens, tags, streamable").Order("provider, name").Find(&models).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": models})
		})
	}

	// ============ 分销商端 API（/distributor/）============
	// 角色为 agent/referrer/reseller/affiliate 的分销商可访问
	distGroup := engine.Group("/distributor")
	distGroup.Use(middleware.AuthMiddleware(jwtManager))
	{
		// ---------- 仪表盘 ----------
		distGroup.GET("/dashboard", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")

			// 获取该用户的分销商信息
			distIDs, _ := distService.GetDistributorByUserID(c.Request.Context(), userID)
			if len(distIDs) == 0 {
				c.JSON(http.StatusOK, gin.H{
					"stats": gin.H{
						"total_commission":     0,
						"available_balance":    0,
						"total_referrals":      0,
						"new_referrals_month":  0,
						"total_clicks":         0,
						"conversion_rate":      0,
					},
					"recent_commissions": []interface{}{},
					"recent_referrals":   []interface{}{},
					"referral_link":      "",
				})
				return
			}

			// 取第一个关联的分销商
			dist, err := distService.GetDistributor(c.Request.Context(), distIDs[0])
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 获取佣金记录
			commissions, _ := distService.ListCommissionRecords(c.Request.Context(), dist.ID, 0, 10)

			// 获取推荐用户
			referrals := distService.ListReferrals(c.Request.Context(), dist.ID, 0, 10)

			// 余额（从计费服务获取）
			balance, _ := billingService.GetBalance(c.Request.Context(), tenantID, userID)

			c.JSON(http.StatusOK, gin.H{
				"stats": gin.H{
					"total_commission":     float64(dist.TotalCommission) / 100,
					"available_balance":    balance,
					"total_referrals":      dist.TotalReferred,
					"new_referrals_month":  dist.NewReferralsThisMonth,
					"total_clicks":         dist.TotalClicks,
					"conversion_rate":      dist.ConversionRate,
				},
				"recent_commissions": commissions,
				"recent_referrals":   referrals,
				"referral_link":      fmt.Sprintf("https://tokenshopai.com/register?ref=%s", dist.ReferralCode),
			})
		})

		// ---------- 推广链接 ----------
		distGroup.GET("/links", func(c *gin.Context) {
			userID := c.GetString("user_id")
			distIDs, _ := distService.GetDistributorByUserID(c.Request.Context(), userID)
			if len(distIDs) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "distributor not found"})
				return
			}
			dist, err := distService.GetDistributor(c.Request.Context(), distIDs[0])
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"main_link":    fmt.Sprintf("https://tokenshopai.com/register?ref=%s", dist.ReferralCode),
				"custom_links": distService.ListCustomLinks(c.Request.Context(), dist.ID),
			})
		})

		distGroup.POST("/links", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var req struct {
				Name       string `json:"name" binding:"required"`
				TargetPage string `json:"target_page"`
				Note       string `json:"note"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			link, err := distService.CreateCustomLink(c.Request.Context(), userID, req.Name, req.TargetPage, req.Note)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"link": link})
		})

		distGroup.DELETE("/links/:id", func(c *gin.Context) {
			userID := c.GetString("user_id")
			linkID := c.Param("id")
			if err := distService.DeleteCustomLink(c.Request.Context(), userID, linkID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "deleted"})
		})

		// ---------- 推荐用户 ----------
		distGroup.GET("/referrals", func(c *gin.Context) {
			userID := c.GetString("user_id")
			distIDs, _ := distService.GetDistributorByUserID(c.Request.Context(), userID)
			if len(distIDs) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "distributor not found"})
				return
			}
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
			offset := (page - 1) * pageSize
			referrals, total := distService.ListReferralsWithCount(c.Request.Context(), distIDs[0], int64(offset), int64(pageSize))
			c.JSON(http.StatusOK, gin.H{"data": referrals, "total": total})
		})

		// ---------- 佣金记录 ----------
		distGroup.GET("/commissions", func(c *gin.Context) {
			userID := c.GetString("user_id")
			distIDs, _ := distService.GetDistributorByUserID(c.Request.Context(), userID)
			if len(distIDs) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "distributor not found"})
				return
			}
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
			offset := int64((page - 1) * pageSize)
			records, _ := distService.ListCommissionRecords(c.Request.Context(), distIDs[0], offset, int64(pageSize))
			c.JSON(http.StatusOK, gin.H{"data": records})
		})

		// ---------- 提现管理 ----------
		distGroup.GET("/withdraw", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			balance, _ := billingService.GetBalance(c.Request.Context(), tenantID, userID)
			records := distService.ListWithdrawRecords(c.Request.Context(), userID, 0, 50)
			c.JSON(http.StatusOK, gin.H{
				"available_balance": balance,
				"records":           records,
			})
		})

		distGroup.POST("/withdraw", func(c *gin.Context) {
			userID := c.GetString("user_id")
			tenantID := c.GetString("tenant_id")
			var req struct {
				Amount    float64 `json:"amount" binding:"required"`
				Method    string  `json:"method" binding:"required"`
				Account   string  `json:"account" binding:"required"`
				RealName  string  `json:"real_name" binding:"required"`
				Note      string  `json:"note"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// 检查余额（balance 单位为分，需转换为元比较）
			balance, _ := billingService.GetBalance(c.Request.Context(), tenantID, userID)
			if float64(balance)/100 < req.Amount {
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
				return
			}
			if err := distService.RequestWithdraw(c.Request.Context(), userID, req.Amount, req.Method, req.Account, req.RealName, req.Note); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "withdraw request submitted"})
		})

		distGroup.POST("/withdraw/:id/cancel", func(c *gin.Context) {
			userID := c.GetString("user_id")
			recordID := c.Param("id")
			if err := distService.CancelWithdraw(c.Request.Context(), userID, recordID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
		})

		// ---------- 推广素材 ----------
		distGroup.GET("/materials", func(c *gin.Context) {
			category := c.DefaultQuery("category", "all")
			materials := distService.ListPromotionalMaterials(c.Request.Context(), category)
			c.JSON(http.StatusOK, gin.H{"data": materials})
		})

		// ---------- 分销商个人资料 ----------
		distGroup.GET("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			distIDs, _ := distService.GetDistributorByUserID(c.Request.Context(), userID)
			if len(distIDs) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "distributor not found"})
				return
			}
			dist, _ := distService.GetDistributor(c.Request.Context(), distIDs[0])
			c.JSON(http.StatusOK, gin.H{"distributor": dist})
		})

		distGroup.PUT("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			var req struct {
				CommissionRate float64 `json:"commission_rate"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := distService.UpdateDistributorProfile(c.Request.Context(), userID, req.CommissionRate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "updated"})
		})
	}

	// 公开模型广场接口（无需认证）
	engine.GET("/public/models", func(c *gin.Context) {
		var models []gateway.ModelConfig
		query := db.Where("enabled = ?", true)
		if provider := c.Query("provider"); provider != "" {
			query = query.Where("provider = ?", provider)
		}
		if err := query.Select("id, name, provider, model_id, input_price, output_price, currency, max_tokens, tags, streamable, latency_ms, success_rate, created_at").Order("provider, name").Find(&models).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": models})
	})

	// 公告公开接口（无需认证）
	engine.GET("/public/announcements", func(c *gin.Context) {
		var anns []Announcement
		if err := db.Where("status = ?", "active").Order("pinned DESC, created_at DESC").Limit(10).Find(&anns).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": anns})
	})

	// 认证接口（无需JWT）
	authGroup := engine.Group("/auth")
	{
		authGroup.POST("/login", func(c *gin.Context) {
			handleLogin(c, jwtManager, db, logger)
		})
		authGroup.POST("/register", func(c *gin.Context) {
			handleRegister(c, db, logger)
		})

		// OAuth 登录（GitHub / Google）
		authGroup.GET("/oauth/:provider", func(c *gin.Context) {
			handleOAuthRedirect(c, cfg, logger)
		})
		authGroup.GET("/oauth/:provider/callback", func(c *gin.Context) {
			handleOAuthCallback(c, jwtManager, db, logger, cfg)
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

// generateSecureRandom 生成安全的随机十六进制字符串
func generateSecureRandom(length int) string {
	b := make([]byte, length/2+1)
	if _, err := rand.Read(b); err != nil {
		// fallback to uuid
		return uuid.New().String() + uuid.New().String()
	}
	return hex.EncodeToString(b)[:length]
}

// ============ P1+ 新增模型 ============

// RedeemCode 兑换码
type RedeemCode struct {
	ID        string     `json:"id" gorm:"primaryKey"`
	Code      string     `json:"code" gorm:"uniqueIndex"`
	Amount    float64    `json:"amount"`
	Status    string     `json:"status"` // active, used, expired
	UsedBy    string     `json:"used_by,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Announcement 公告
type Announcement struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // info, warning, success, error
	Status    string    `json:"status"` // active, archived
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
}

// ModelMapping 模型映射
type ModelMapping struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	SourceModel string    `json:"source_model"`
	TargetModel string    `json:"target_model"`
	TenantID    string    `json:"tenant_id"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserGroup 用户分组
type UserGroup struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name"`
	Multiplier float64   `json:"multiplier"` // 倍率，如 0.8 表示 8 折
	RpmLimit   int       `json:"rpm_limit"`
	TpmLimit   int64     `json:"tpm_limit"`
	CreatedAt  time.Time `json:"created_at"`
}

// ReferralInvite 邀请记录
type ReferralInvite struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	InviterID   string     `json:"inviter_id" gorm:"index"`
	InviteeID   string     `json:"invitee_id"`
	InviteCode  string     `json:"invite_code" gorm:"index"`
	RewardAmount float64   `json:"reward_amount"`
	Status      string     `json:"status"` // pending, rewarded, expired
	RewardedAt  *time.Time `json:"rewarded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// OAuthAccount 第三方 OAuth 绑定
type OAuthAccount struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id" gorm:"index"`
	Provider     string    `json:"provider"` // github, google
	ProviderID   string    `json:"provider_id"`
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	DisplayName  string    `json:"display_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// OnboardingProgress 用户引导进度
type OnboardingProgress struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id" gorm:"uniqueIndex"`
	CurrentStep  int       `json:"current_step"`
	Completed    bool      `json:"completed"`
	Skipped      bool      `json:"skipped"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
		&RedeemCode{},
		&Announcement{},
		&ModelMapping{},
		&UserGroup{},
		&ReferralInvite{},
		&OAuthAccount{},
		&OnboardingProgress{},
		&channel.Channel{},
		&tokencore.AccessToken{},
		&subscription.SubscriptionPlan{},
		&subscription.UserSubscription{},
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
		logger.Warn("using default admin password; set ADMIN_PASSWORD environment variable for production")
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

// handleChatCompletionWithChannel 渠道优先路由的Chat Completion处理
// 优先查找渠道做负载均衡/Key轮换/自动重试，无可用渠道时 fallback 到原有 ModelRouter
func handleChatCompletionWithChannel(
	c *gin.Context,
	aiProxy *proxy.AIProxy,
	channelSvc *channel.ChannelService,
	billingSvc *billing.BillingService,
	desensitizer *desensitize.Desensitizer,
	subSvc *subscription.SubscriptionService,
	logger *zap.Logger,
) {
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream      bool     `json:"stream"`
		MaxTokens   *int     `json:"max_tokens,omitempty"`
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

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	// 1. 尝试渠道路由
	channels, err := channelSvc.GetChannelsForModel(c.Request.Context(), req.Model, tenantID)
	if err == nil && len(channels) > 0 {
		// 渠道亲和性：同一用户尽量路由到同一渠道
		affinityID := channelSvc.GetAffinity(userID)
		var selected *channel.Channel
		for _, ch := range channels {
			if ch.ID == affinityID && ch.Enabled {
				selected = ch
				break
			}
		}
		if selected == nil {
			// 按优先级+权重选择第一个可用渠道
			selected = channels[0]
		}

		// Key轮换
		apiKey := channelSvc.GetCurrentAPIKey(selected)
		channelSvc.SetAffinity(userID, selected.ID)

		logger.Info("channel routed request",
			zap.String("model", req.Model),
			zap.String("channel_id", selected.ID),
			zap.String("provider", string(selected.Provider)),
		)

		// 构造 ChatRequest 使用渠道的 endpoint/apiKey
		chatReq := &gateway.ChatRequest{
			Model:    req.Model,
			Stream:   req.Stream,
			TenantID: tenantID,
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

		// 通过渠道路由的请求：直接使用 proxy 但覆盖路由结果中的 endpoint/apiKey
		// 这里用一个简化的方式：在 ChatRequest 上标记渠道路由信息
		chatReq.ChannelEndpoint = selected.Endpoint
		chatReq.ChannelAPIKey = apiKey
		chatReq.ChannelID = selected.ID

		if req.Stream {
			handleStreamResponseWithChannel(c, aiProxy, channelSvc, billingSvc, subSvc, chatReq, logger)
		} else {
			handleNonStreamResponseWithChannel(c, aiProxy, channelSvc, billingSvc, desensitizer, subSvc, chatReq, logger)
		}
		return
	}

	// 2. 无可用渠道，fallback 到原有 ModelRouter
	chatReq := &gateway.ChatRequest{
		Model:    req.Model,
		Stream:   req.Stream,
		TenantID: tenantID,
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
		handleStreamResponse(c, aiProxy, billingSvc, chatReq, logger)
	} else {
		handleNonStreamResponse(c, aiProxy, billingSvc, desensitizer, chatReq, logger)
	}
}

// handleNonStreamResponseWithChannel 使用渠道的非流式响应
func handleNonStreamResponseWithChannel(
	c *gin.Context,
	aiProxy *proxy.AIProxy,
	channelSvc *channel.ChannelService,
	billingSvc *billing.BillingService,
	desensitizer *desensitize.Desensitizer,
	subSvc *subscription.SubscriptionService,
	req *gateway.ChatRequest,
	logger *zap.Logger,
) {
	start := time.Now()

	// 使用渠道路由代理请求
	result, err := aiProxy.ProxyWithChannel(c.Request.Context(), req)
	if err != nil {
		channelSvc.RecordFailure(c.Request.Context(), req.ChannelID, err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"error": map[string]string{
				"code":    "upstream_error",
				"message": err.Error(),
			},
		})
		return
	}

	channelSvc.RecordSuccess(c.Request.Context(), req.ChannelID, int(time.Since(start).Milliseconds()))
	logger.Info("channel request completed",
		zap.String("channel_id", req.ChannelID),
		zap.Duration("latency", time.Since(start)),
	)

	// 脱敏处理响应
	if result.Response != nil && len(result.Response.Choices) > 0 && result.Response.Choices[0].Message != nil {
		result.Response.Choices[0].Message.Content = desensitizer.Desensitize(
			result.Response.Choices[0].Message.Content,
		)
	}

	// 扣费
	if result.Usage != nil {
		_, _ = billingSvc.DeductBalance(c.Request.Context(), &billing.DeductRequest{
			TenantID: req.TenantID,
			UserID:   req.User,
			Usage:    result.Usage,
			TraceID:  req.TraceID,
			Currency: "CNY",
		})
		// 追踪订阅用量
		if subSvc != nil && req.User != "" {
			_, sub, _ := subSvc.GetUserSubscriptionPlan(c.Request.Context(), req.User)
			if sub != nil {
				_ = subSvc.IncrementUsage(c.Request.Context(), sub.ID, int64(result.Usage.TotalTokens), 1)
			}
		}
	}

	c.JSON(http.StatusOK, result.Response)
}

// handleStreamResponseWithChannel 使用渠道的流式响应
func handleStreamResponseWithChannel(
	c *gin.Context,
	aiProxy *proxy.AIProxy,
	channelSvc *channel.ChannelService,
	billingSvc *billing.BillingService,
	subSvc *subscription.SubscriptionService,
	req *gateway.ChatRequest,
	logger *zap.Logger,
) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	start := time.Now()

	result, err := aiProxy.ProxyStreamWithChannel(c.Request.Context(), req, func(chunk *proxy.StreamChunkResult) error {
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
		channelSvc.RecordFailure(c.Request.Context(), req.ChannelID, err.Error())
		logger.Error("channel stream proxy failed", zap.Error(err))
		return
	}

	channelSvc.RecordSuccess(c.Request.Context(), req.ChannelID, int(time.Since(start).Milliseconds()))

	// 流式扣费
	if result != nil && result.Usage != nil {
		_, _ = billingSvc.DeductBalance(c.Request.Context(), &billing.DeductRequest{
			TenantID: req.TenantID,
			UserID:   req.User,
			Usage:    result.Usage,
			TraceID:  req.TraceID,
			Currency: "CNY",
		})
		// 追踪订阅用量
		if subSvc != nil && req.User != "" {
			_, sub, _ := subSvc.GetUserSubscriptionPlan(c.Request.Context(), req.User)
			if sub != nil {
				_ = subSvc.IncrementUsage(c.Request.Context(), sub.ID, int64(result.Usage.TotalTokens), 1)
			}
		}
	}

	c.Writer.Flush()
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
		Username    string `json:"username"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		TenantID    string `json:"tenant_id"`
		InviteCode  string `json:"invite_code"`
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

	// 处理邀请码
	if req.InviteCode != "" {
		var inviter auth.User
		if err := db.Where("id = ?", strings.TrimPrefix(req.InviteCode, "INV-")).First(&inviter).Error; err == nil {
			// 找到邀请人，创建邀请记录
			db.Create(&ReferralInvite{
				ID:           uuid.New().String(),
				InviterID:    inviter.ID,
				InviteeID:    user.ID,
				InviteCode:   req.InviteCode,
				RewardAmount: 5.0, // 默认奖励 5 元
				Status:       "pending",
				CreatedAt:    time.Now(),
			})
			logger.Info("referral recorded", zap.String("inviter", inviter.ID), zap.String("invitee", user.ID))
		}
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

// handleOAuthRedirect 发起 OAuth 重定向（GitHub / Google）
func handleOAuthRedirect(c *gin.Context, cfg *config.Config, logger *zap.Logger) {
	provider := c.Param("provider")
	var authURL, clientID string
	state := generateSecureRandom(16)

	switch provider {
	case "github":
		clientID = cfg.OAuth.GitHubClientID
		if clientID == "" {
			clientID = os.Getenv("GITHUB_CLIENT_ID")
		}
		if clientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub OAuth not configured"})
			return
		}
		authURL = fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user:email", clientID, state)
	case "google":
		clientID = cfg.OAuth.GoogleClientID
		if clientID == "" {
			clientID = os.Getenv("GOOGLE_CLIENT_ID")
		}
		if clientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Google OAuth not configured"})
			return
		}
		redirectURI := getOAuthRedirectBase(cfg, c.Request.Host) + "/auth/oauth/google/callback"
		authURL = fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&state=%s&scope=openid+email+profile&response_type=code", clientID, redirectURI, state)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// 将 state 存入 Redis 供回调校验（简化：直接用 cookie）
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// handleOAuthCallback 处理 OAuth 回调
func handleOAuthCallback(c *gin.Context, jwtManager *auth.JWTManager, db *gorm.DB, logger *zap.Logger, cfg *config.Config) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	// 校验 state
	savedState, _ := c.Cookie("oauth_state")
	if state == "" || state != savedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	var (
		accessToken  string
		email        string
		name         string
		avatarURL    string
		providerID   string
	)

	switch provider {
	case "github":
		var err error
		accessToken, email, name, avatarURL, providerID, err = exchangeGitHubCode(code, cfg)
		if err != nil {
			logger.Error("github oauth failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "google":
		var err error
		accessToken, email, name, avatarURL, providerID, err = exchangeGoogleCode(code, c.Request.Host, cfg)
		if err != nil {
			logger.Error("google oauth failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// 查找或创建用户
	var oauthAcc OAuthAccount
	var user auth.User

	if err := db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&oauthAcc).Error; err == nil {
		// 已绑定，直接登录
		db.First(&user, "id = ?", oauthAcc.UserID)
	} else {
		// 未绑定，按邮箱查找或创建
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			// 创建新用户
			var defaultTenant tenant.Tenant
			if err := db.Where("slug = ?", "default").First(&defaultTenant).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "default tenant not found"})
				return
			}
			user = auth.User{
				ID:           uuid.New().String(),
				TenantID:     defaultTenant.ID,
				Username:     name,
				Email:        email,
				PasswordHash: "", // OAuth 用户无密码
				Role:         auth.RoleDeveloper,
				Status:       auth.UserActive,
				Language:     "zh-CN",
				Timezone:     "Asia/Shanghai",
			}
			if err := db.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
		}

		// 绑定 OAuth
		db.Create(&OAuthAccount{
			ID:           uuid.New().String(),
			UserID:       user.ID,
			Provider:     provider,
			ProviderID:   providerID,
			AccessToken:  accessToken,
			AvatarURL:    avatarURL,
			DisplayName:  name,
			CreatedAt:    time.Now(),
		})
	}

	// 生成 JWT
	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, user.TenantID, user.Role, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重定向前端并携带 token
	frontendURL := cfg.Server.FrontendURL
	if frontendURL == "" {
		frontendURL = os.Getenv("FRONTEND_URL")
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3001"
	}
	redirectURL := fmt.Sprintf("%s/#/dashboard?token=%s&refresh_token=%s", frontendURL, tokenPair.AccessToken, tokenPair.RefreshToken)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// exchangeGitHubCode 用 code 换取 GitHub 用户信息
func exchangeGitHubCode(code string, cfg *config.Config) (accessToken, email, name, avatarURL, providerID string, err error) {
	clientID := cfg.OAuth.GitHubClientID
	if clientID == "" {
		clientID = os.Getenv("GITHUB_CLIENT_ID")
	}
	clientSecret := cfg.OAuth.GitHubClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	// 换取 access_token
	resp, err := http.Post(
		"https://github.com/login/oauth/access_token?client_id="+clientID+"&client_secret="+clientSecret+"&code="+code,
		"application/json",
		nil,
	)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	accessToken, _ = tokenResp["access_token"].(string)

	// 获取用户信息
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp2.Body.Close()

	var userInfo map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&userInfo)
	name, _ = userInfo["name"].(string)
	if name == "" {
		name, _ = userInfo["login"].(string)
	}
	avatarURL, _ = userInfo["avatar_url"].(string)
	pid, _ := userInfo["id"].(float64)
	providerID = fmt.Sprintf("%.0f", pid)

	// 获取邮箱
	req2, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	resp3, err := http.DefaultClient.Do(req2)
	if err == nil {
		defer resp3.Body.Close()
		var emails []map[string]interface{}
		json.NewDecoder(resp3.Body).Decode(&emails)
		for _, e := range emails {
			if primary, _ := e["primary"].(bool); primary {
				email, _ = e["email"].(string)
				break
			}
		}
	}
	return
}

// getOAuthRedirectBase 获取 OAuth 回调的基础 URL
// 优先使用配置的 FRONTEND_URL，确保与 Google/GitHub Console 中配置的回调地址一致
func getOAuthRedirectBase(cfg *config.Config, requestHost string) string {
	// 优先使用环境变量或配置文件中的 FRONTEND_URL
	frontendURL := cfg.Server.FrontendURL
	if frontendURL == "" {
		frontendURL = os.Getenv("FRONTEND_URL")
	}
	if frontendURL != "" {
		// 去掉尾部的 / （如果有的话）
		frontendURL = strings.TrimRight(frontendURL, "/")
		return frontendURL
	}

	// 开发模式：使用请求的 Host
	scheme := "http"
	if requestHost != "" && !strings.Contains(requestHost, "localhost") && !strings.Contains(requestHost, "127.0.0.1") {
		scheme = "https"
	}
	return scheme + "://" + requestHost
}

// exchangeGoogleCode 用 code 换取 Google 用户信息
func exchangeGoogleCode(code, host string, cfg *config.Config) (accessToken, email, name, avatarURL, providerID string, err error) {
	clientID := cfg.OAuth.GoogleClientID
	if clientID == "" {
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	clientSecret := cfg.OAuth.GoogleClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	redirectURI := getOAuthRedirectBase(cfg, host) + "/auth/oauth/google/callback"

	// 换取 token
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", map[string][]string{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	accessToken, _ = tokenResp["access_token"].(string)
	idToken, _ := tokenResp["id_token"].(string)
	_ = idToken

	// 获取用户信息
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp2.Body.Close()

	var userInfo map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&userInfo)
	email, _ = userInfo["email"].(string)
	name, _ = userInfo["name"].(string)
	avatarURL, _ = userInfo["picture"].(string)
	pid, _ := userInfo["id"].(string)
	providerID = pid
	return
}

// handleImageGeneration 处理图像生成请求（兼容 OpenAI /v1/images/generations）
func handleImageGeneration(c *gin.Context, modelRouter *smartrouter.ModelRouter, billingSvc *billing.BillingService, logger *zap.Logger) {
	var req struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt" binding:"required"`
		N              int    `json:"n"`
		Size           string `json:"size"`
		ResponseFormat string `json:"response_format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}
	if req.N <= 0 {
		req.N = 1
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}
	if req.Model == "" {
		req.Model = "dall-e-3"
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "url"
	}

	// 构造上游请求体
	payload := map[string]interface{}{
		"model":           req.Model,
		"prompt":          req.Prompt,
		"n":               req.N,
		"size":            req.Size,
		"response_format": req.ResponseFormat,
	}
	payloadBytes, _ := json.Marshal(payload)

	// 路由选择
	chatReq := &gateway.ChatRequest{
		Model:    req.Model,
		Messages: []gateway.ChatMessage{{Role: "user", Content: req.Prompt}},
		TraceID:  uuid.New().String(),
	}
	routeResult, err := modelRouter.Route(c.Request.Context(), chatReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": "no available upstream for model " + req.Model, "type": "route_error"}})
		return
	}

	mc := routeResult.ModelConfig

	// 向上游发送图片生成请求
	upstreamURL := strings.TrimSuffix(mc.Endpoint, "/") + "/v1/images/generations"
	httpReq, _ := http.NewRequestWithContext(c.Request.Context(), "POST", upstreamURL, bytes.NewReader(payloadBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+mc.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		logger.Error("image generation upstream failed", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": err.Error(), "type": "upstream_error"}})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	// 计费（简化：按图片数量计费）
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID != "" && userID != "" {
		costCents := int64(req.N * 400) // 每张约 4 元
		billingSvc.TopUp(c.Request.Context(), tenantID, userID, -costCents) // 负数=扣费
	}

	c.Data(resp.StatusCode, "application/json", bodyBytes)
}
