package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/tokenhub/backend/internal/auth"
	"github.com/tokenhub/backend/internal/monitor"
	"github.com/tokenhub/backend/internal/security/waf"
	"go.uber.org/zap"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "unauthorized",
					"message": "Authorization header is required",
				},
			})
			return
		}

		// 去除 Bearer 前缀
		token = strings.TrimPrefix(token, "Bearer ")
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "invalid_token",
					"message": "Invalid or expired token",
				},
			})
			return
		}

		// 注入用户信息
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("role", string(claims.Role))
		c.Set("email", claims.Email)
		c.Next()
	}
}

// APIKeyMiddleware API Key认证中间件
// 从Redis缓存中验证API Key的合法性，防止未授权访问
func APIKeyMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// 也从 Authorization: Bearer sk-xxx 获取
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer sk-") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "missing_api_key",
					"message": "API key is required",
				},
			})
			return
		}

		// 从Redis缓存验证API Key
		// Key格式: apikey:{key_prefix} -> 存储key_hash, tenant_id, user_id, permissions, status等
		keyPrefix := apiKey
		if len(apiKey) > 8 {
			keyPrefix = apiKey[:8]
		}
		cacheKey := fmt.Sprintf("apikey:%s", keyPrefix)
		val, err := rdb.HGetAll(c.Request.Context(), cacheKey).Result()
		if err != nil || len(val) == 0 {
			// Key不在缓存中，可能无效或未加载
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "invalid_api_key",
					"message": "Invalid or unrecognized API key",
				},
			})
			return
		}

		// 检查Key状态
		if status, ok := val["status"]; ok && status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "api_key_revoked",
					"message": "API key has been revoked or expired",
				},
			})
			return
		}

		// 注入Key信息到上下文
		c.Set("api_key", apiKey)
		if tenantID, ok := val["tenant_id"]; ok {
			c.Set("tenant_id", tenantID)
		}
		if userID, ok := val["user_id"]; ok {
			c.Set("api_key_user_id", userID)
		}
		c.Next()
	}
}

// TenantMiddleware 租户中间件（从请求头或Token中提取租户ID）
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从JWT中获取
		if tenantID, exists := c.Get("tenant_id"); exists {
			c.Set("x-tenant-id", tenantID)
			c.Next()
			return
		}

		// 从请求头获取
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "missing_tenant",
					"message": "Tenant ID is required",
				},
			})
			return
		}

		c.Set("x-tenant-id", tenantID)
		c.Next()
	}
}

// WAFMiddleware WAF中间件
func WAFMiddleware(wafEngine *waf.WAF) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !wafEngine.IsEnabled() {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		// 检查 URL + 请求体
		var content strings.Builder
		content.WriteString(c.Request.URL.String())

		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				// 限制检查的 body 大小，防止内存溢出
				if len(bodyBytes) > 65536 {
					bodyBytes = bodyBytes[:65536]
				}
				content.WriteString(" ")
				content.Write(bodyBytes)
				// 恢复 body 供后续 handler 使用
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		result := wafEngine.CheckRequest(clientIP, content.String())
		if result.Blocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "waf_blocked",
					"message": result.Reason,
				},
			})
			return
		}

		c.Next()
	}
}

// CORSMiddleware 跨域中间件
// allowedOrigins: 允许的域名列表。空列表表示开发模式允许所有来源（不携带凭证）。
// 生产环境必须配置具体域名列表，此时支持携带凭证。
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0

	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowAll {
			// 开发模式：允许所有来源，但不设置 Credentials（CORS 规范不允许 * + credentials）
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && originSet[origin] {
			// 生产模式：仅允许配置列表中的 Origin，支持携带凭证
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			// Origin 不在白名单中，不设置 CORS 头，浏览器将阻止跨域请求
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-API-Key, X-Tenant-ID, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// MetricsMiddleware 指标采集中间件
func MetricsMiddleware(monitor *monitor.MonitorService, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		success := statusCode < 400

		// 采集指标
		if monitor != nil {
			monitor.RecordRequest(
				c.GetString("model_id"),
				c.GetString("model_name"),
				c.GetString("provider"),
				c.GetString("tenant_id"),
				int(latency.Milliseconds()),
				success,
				c.GetInt("total_tokens"),
				c.GetInt64("amount"),
			)
		}

		// 访问日志
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("request_id", c.GetString("request_id")),
		)
	}
}

// RBACMiddleware RBAC权限检查中间件
// 检查当前用户角色是否拥有指定权限
func RBACMiddleware(requiredPerm auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "forbidden",
					"message": "Access denied: role not found",
				},
			})
			return
		}

		role := auth.Role(fmt.Sprintf("%v", roleStr))
		if !auth.HasPermission(role, requiredPerm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "permission_denied",
					"message": fmt.Sprintf("Permission '%s' required", requiredPerm),
				},
			})
			return
		}

		c.Next()
	}
}

// UserUpdateWhitelistMiddleware 用户更新字段白名单中间件
// 防止通过API修改敏感字段（如role、tenant_id等），防止权限提升攻击
func UserUpdateWhitelistMiddleware() gin.HandlerFunc {
	// 允许用户通过API更新的安全字段白名单
	allowedFields := map[string]bool{
		"display_name": true,
		"phone":       true,
		"company":     true,
		"bio":         true,
		"status":      true,
	}

	return func(c *gin.Context) {
		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 过滤掉不在白名单中的字段
		filtered := make(map[string]interface{})
		for k, v := range updates {
			if allowedFields[k] {
				filtered[k] = v
			}
		}

		if len(filtered) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "no valid fields to update (allowed: display_name, phone, company, bio, status)",
			})
			return
		}

		// 将过滤后的数据存入上下文，供后续handler使用
		c.Set("filtered_updates", filtered)
		c.Next()
	}
}

// RateLimitMiddleware API限流中间件
// 基于Redis实现滑动窗口限流
func RateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration, keyPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用API Key，其次使用用户ID，最后使用IP
		key := c.GetString("api_key")
		if key == "" {
			key = c.GetString("user_id")
		}
		if key == "" {
			key = c.ClientIP()
		}

		cacheKey := fmt.Sprintf("ratelimit:%s:%s", keyPrefix, key)
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, cacheKey).Result()
		if err != nil {
			// Redis错误时放行，避免影响正常请求
			c.Next()
			return
		}

		if count == 1 {
			rdb.Expire(ctx, cacheKey, window)
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": map[string]string{
					"code":    "rate_limit_exceeded",
					"message": fmt.Sprintf("Rate limit exceeded: %d requests per %v", limit, window),
				},
			})
			return
		}

		// 设置限流头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(limit)-count))

		c.Next()
	}
}
