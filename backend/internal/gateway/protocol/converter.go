package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tokenhub/backend/internal/gateway"
)

// ProtocolConverter 协议转换器接口
type ProtocolConverter interface {
	// ConvertRequest 将统一格式转换为供应商原始格式
	ConvertRequest(req *gateway.ChatRequest) ([]byte, error)
	// ConvertResponse 将供应商原始响应转换为统一格式
	ConvertResponse(body []byte) (*gateway.ChatResponse, error)
	// BuildHTTPRequest 构建供应商API请求
	BuildHTTPRequest(endpoint, apiKey string, body []byte) (*http.Request, error)
	// ParseStreamChunk 解析流式响应块
	ParseStreamChunk(line string) (*gateway.StreamChunk, error)
	// Provider 返回供应商标识
	Provider() gateway.ModelProvider
}

// --- OpenAI 协议转换器 ---

type OpenAIConverter struct{}

func (c *OpenAIConverter) Provider() gateway.ModelProvider {
	return gateway.ProviderOpenAI
}

func (c *OpenAIConverter) ConvertRequest(req *gateway.ChatRequest) ([]byte, error) {
	// OpenAI 兼容格式直接使用
	return json.Marshal(req)
}

func (c *OpenAIConverter) ConvertResponse(body []byte) (*gateway.ChatResponse, error) {
	var resp gateway.ChatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal openai response: %w", err)
	}
	return &resp, nil
}

func (c *OpenAIConverter) BuildHTTPRequest(endpoint, apiKey string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest("POST", endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return req, nil
}

func (c *OpenAIConverter) ParseStreamChunk(line string) (*gateway.StreamChunk, error) {
	if len(line) < 6 || line[:6] != "data: " {
		return nil, nil
	}
	data := line[6:]
	if data == "[DONE]" {
		return nil, io.EOF
	}
	var chunk gateway.StreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, fmt.Errorf("parse stream chunk: %w", err)
	}
	return &chunk, nil
}

// --- Claude 协议转换器 ---

type ClaudeConverter struct{}

type claudeRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []claudeMessage    `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model   string `json:"model"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *ClaudeConverter) Provider() gateway.ModelProvider {
	return gateway.ProviderClaude
}

func (c *ClaudeConverter) ConvertRequest(req *gateway.ChatRequest) ([]byte, error) {
	claudeReq := claudeRequest{
		Model:     req.Model,
		MaxTokens: 4096,
		Stream:    req.Stream,
	}
	if req.MaxTokens != nil {
		claudeReq.MaxTokens = *req.MaxTokens
	}

	// 提取 system 消息并转换格式
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			claudeReq.System = msg.Content
		} else {
			claudeReq.Messages = append(claudeReq.Messages, claudeMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return json.Marshal(claudeReq)
}

func (c *ClaudeConverter) ConvertResponse(body []byte) (*gateway.ChatResponse, error) {
	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("unmarshal claude response: %w", err)
	}

	content := ""
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	finishReason := "stop"
	return &gateway.ChatResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Model:   claudeResp.Model,
		Choices: []gateway.Choice{{
			Index:        0,
			Message:      &gateway.ChatMessage{Role: "assistant", Content: content},
			FinishReason: &finishReason,
		}},
		Usage: gateway.Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}, nil
}

func (c *ClaudeConverter) BuildHTTPRequest(endpoint, apiKey string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest("POST", endpoint+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return req, nil
}

func (c *ClaudeConverter) ParseStreamChunk(line string) (*gateway.StreamChunk, error) {
	if len(line) < 6 || line[:6] != "data: " {
		return nil, nil
	}
	data := line[6:]
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil, err
	}
	// 简化处理：将 Claude 事件转换为 OpenAI 兼容 chunk
	chunk := &gateway.StreamChunk{
		Object: "chat.completion.chunk",
	}
	if eventType, ok := raw["type"].(string); ok && eventType == "content_block_delta" {
		if delta, ok := raw["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok {
				chunk.Choices = []gateway.Choice{{
					Index: 0,
					Delta: &gateway.ChatMessage{Role: "assistant", Content: text},
				}}
			}
		}
	}
	return chunk, nil
}

// --- 文心 协议转换器 ---

type WenxinConverter struct{}

type wenxinRequest struct {
	Messages []wenxinMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
}

type wenxinMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type wenxinResponse struct {
	ID      string `json:"id"`
	Result  string `json:"result"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *WenxinConverter) Provider() gateway.ModelProvider {
	return gateway.ProviderWenxin
}

func (c *WenxinConverter) ConvertRequest(req *gateway.ChatRequest) ([]byte, error) {
	wenxinReq := wenxinRequest{Stream: req.Stream}
	for _, msg := range req.Messages {
		wenxinReq.Messages = append(wenxinReq.Messages, wenxinMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return json.Marshal(wenxinReq)
}

func (c *WenxinConverter) ConvertResponse(body []byte) (*gateway.ChatResponse, error) {
	var wenxinResp wenxinResponse
	if err := json.Unmarshal(body, &wenxinResp); err != nil {
		return nil, fmt.Errorf("unmarshal wenxin response: %w", err)
	}
	finishReason := "stop"
	return &gateway.ChatResponse{
		ID:     wenxinResp.ID,
		Object: "chat.completion",
		Choices: []gateway.Choice{{
			Index:        0,
			Message:      &gateway.ChatMessage{Role: "assistant", Content: wenxinResp.Result},
			FinishReason: &finishReason,
		}},
		Usage: gateway.Usage{
			PromptTokens:     wenxinResp.Usage.PromptTokens,
			CompletionTokens: wenxinResp.Usage.CompletionTokens,
			TotalTokens:      wenxinResp.Usage.TotalTokens,
		},
	}, nil
}

func (c *WenxinConverter) BuildHTTPRequest(endpoint, apiKey string, body []byte) (*http.Request, error) {
	// 文心使用 access_token 而非 API Key
	req, err := http.NewRequest("POST", endpoint+"?access_token="+apiKey, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *WenxinConverter) ParseStreamChunk(line string) (*gateway.StreamChunk, error) {
	if len(line) < 6 || line[:6] != "data: " {
		return nil, nil
	}
	data := line[6:]
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil, err
	}
	chunk := &gateway.StreamChunk{Object: "chat.completion.chunk"}
	if result, ok := raw["result"].(string); ok {
		chunk.Choices = []gateway.Choice{{
			Index: 0,
			Delta: &gateway.ChatMessage{Role: "assistant", Content: result},
		}}
	}
	return chunk, nil
}

// ConverterRegistry 转换器注册表
type ConverterRegistry struct {
	converters map[gateway.ModelProvider]ProtocolConverter
}

func NewConverterRegistry() *ConverterRegistry {
	r := &ConverterRegistry{
		converters: make(map[gateway.ModelProvider]ProtocolConverter),
	}
	// 注册内置转换器
	r.Register(&OpenAIConverter{})
	r.Register(&ClaudeConverter{})
	r.Register(&WenxinConverter{})
	return r
}

func (r *ConverterRegistry) Register(c ProtocolConverter) {
	r.converters[c.Provider()] = c
}

func (r *ConverterRegistry) Get(provider gateway.ModelProvider) (ProtocolConverter, error) {
	c, ok := r.converters[provider]
	if !ok {
		return nil, fmt.Errorf("no converter for provider: %s", provider)
	}
	return c, nil
}
