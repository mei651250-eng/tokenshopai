package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tokenhub/backend/internal/gateway"
	"github.com/tokenhub/backend/internal/gateway/router"
	"go.uber.org/zap"
)

// AIProxy AI代理
type AIProxy struct {
	logger  *zap.Logger
	router  *router.ModelRouter
	client  *http.Client
	timeout time.Duration
	streamTimeout time.Duration
}

// NewAIProxy 创建AI代理
func NewAIProxy(
	logger *zap.Logger,
	router *router.ModelRouter,
	timeout, streamTimeout time.Duration,
) *AIProxy {
	return &AIProxy{
		logger:  logger,
		router:  router,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		timeout:       timeout,
		streamTimeout: streamTimeout,
	}
}

// ProxyResult 代理结果
type ProxyResult struct {
	Response *gateway.ChatResponse
	Usage    *gateway.Usage
	ModelID  string
	Provider gateway.ModelProvider
	Latency  time.Duration
	Attempt  int
}

// Proxy 执行代理请求（非流式）
func (p *AIProxy) Proxy(ctx context.Context, req *gateway.ChatRequest) (*ProxyResult, error) {
	if req.TraceID == "" {
		req.TraceID = uuid.New().String()
	}

	// 1. 路由选择
	routeResult, err := p.router.Route(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("route failed: %w", err)
	}

	start := time.Now()
	converter := routeResult.Converter
	modelCfg := routeResult.ModelConfig

	p.logger.Info("proxying request",
		zap.String("trace_id", req.TraceID),
		zap.String("model", req.Model),
		zap.String("provider", string(modelCfg.Provider)),
		zap.String("model_id", modelCfg.ID),
		zap.Int("attempt", routeResult.Attempt),
	)

	// 2. 协议转换
	reqBody, err := converter.ConvertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("convert request: %w", err)
	}

	// 3. 构建HTTP请求
	httpReq, err := converter.BuildHTTPRequest(modelCfg.Endpoint, modelCfg.APIKey, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build http request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	// 4. 发送请求
	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.router.RecordFailure(modelCfg.ID, err)
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	// 5. 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(body))
		p.router.RecordFailure(modelCfg.ID, err)
		return nil, err
	}

	// 6. 转换响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	chatResp, err := converter.ConvertResponse(body)
	if err != nil {
		return nil, fmt.Errorf("convert response: %w", err)
	}

	// 7. 记录成功
	p.router.RecordSuccess(modelCfg.ID)
	p.router.UpdateModelStats(modelCfg.ID, int(latency.Milliseconds()), true)

	return &ProxyResult{
		Response: chatResp,
		Usage:    &chatResp.Usage,
		ModelID:  modelCfg.ID,
		Provider: modelCfg.Provider,
		Latency:  latency,
		Attempt:  routeResult.Attempt,
	}, nil
}

// StreamResult 流式代理结果
type StreamChunkResult struct {
	Chunk  *gateway.StreamChunk
	Done   bool
	Usage  *gateway.Usage
}

// ProxyStream 执行流式代理请求
func (p *AIProxy) ProxyStream(
	ctx context.Context,
	req *gateway.ChatRequest,
	onChunk func(chunk *StreamChunkResult) error,
) (*ProxyResult, error) {
	if req.TraceID == "" {
		req.TraceID = uuid.New().String()
	}
	req.Stream = true

	// 1. 路由选择
	routeResult, err := p.router.Route(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("route failed: %w", err)
	}

	start := time.Now()
	converter := routeResult.Converter
	modelCfg := routeResult.ModelConfig

	// 2. 协议转换
	reqBody, err := converter.ConvertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("convert request: %w", err)
	}

	// 3. 构建HTTP请求（使用流式超时）
	streamClient := &http.Client{Timeout: p.streamTimeout}
	httpReq, err := converter.BuildHTTPRequest(modelCfg.Endpoint, modelCfg.APIKey, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build http request: %w", err)
	}
	httpReq = httpReq.WithContext(ctx)

	// 4. 发送请求
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		p.router.RecordFailure(modelCfg.ID, err)
		return nil, fmt.Errorf("send stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(body))
		p.router.RecordFailure(modelCfg.ID, err)
		return nil, err
	}

	// 5. 流式读取
	var totalUsage gateway.Usage
	totalOutputTokens := 0

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0), 10*1024*1024) // 10MB buffer

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		chunk, err := converter.ParseStreamChunk(line)
		if err != nil {
			if err == io.EOF {
				onChunk(&StreamChunkResult{Done: true})
				break
			}
			p.logger.Warn("parse stream chunk error", zap.Error(err))
			continue
		}
		if chunk == nil {
			continue
		}

		totalOutputTokens++

		if err := onChunk(&StreamChunkResult{
			Chunk: chunk,
			Done:  false,
		}); err != nil {
			return nil, err
		}
	}

	latency := time.Since(start)

	totalUsage.CompletionTokens = totalOutputTokens
	totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens

	p.router.RecordSuccess(modelCfg.ID)
	p.router.UpdateModelStats(modelCfg.ID, int(latency.Milliseconds()), true)

	return &ProxyResult{
		Usage:    &totalUsage,
		ModelID:  modelCfg.ID,
		Provider: modelCfg.Provider,
		Latency:  latency,
		Attempt:  routeResult.Attempt,
	}, nil
}
