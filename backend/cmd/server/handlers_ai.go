package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tokenhub/backend/internal/billing"
	"github.com/tokenhub/backend/internal/channel"
	"github.com/tokenhub/backend/internal/gateway"
	"github.com/tokenhub/backend/internal/gateway/proxy"
	smartrouter "github.com/tokenhub/backend/internal/gateway/router"
	"github.com/tokenhub/backend/internal/security/desensitize"
	"github.com/tokenhub/backend/internal/subscription"
	"go.uber.org/zap"
)

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
			Model:     req.Model,
			Stream:    req.Stream,
			TenantID:  tenantID,
			APIKeyID:  c.GetString("api_key_id"),
			TraceID:   c.GetString("request_id"),
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
		Model:     req.Model,
		Stream:    req.Stream,
		TenantID:  tenantID,
		APIKeyID:  c.GetString("api_key_id"),
		TraceID:   c.GetString("request_id"),
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

	// 构造内部请求
	chatReq := &gateway.ChatRequest{
		Model:     req.Model,
		Stream:    req.Stream,
		TenantID:  c.GetString("tenant_id"),
		APIKeyID:  c.GetString("api_key_id"),
		TraceID:   c.GetString("request_id"),
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
		costCents := int64(req.N * 400)                                     // 每张约 4 元
		billingSvc.TopUp(c.Request.Context(), tenantID, userID, -costCents) // 负数=扣费
	}

	c.Data(resp.StatusCode, "application/json", bodyBytes)
}
