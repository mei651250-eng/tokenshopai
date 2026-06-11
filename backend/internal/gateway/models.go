package gateway

// ModelProvider 大模型供应商
type ModelProvider string

const (
	ProviderOpenAI    ModelProvider = "openai"
	ProviderAzure     ModelProvider = "azure"
	ProviderClaude    ModelProvider = "claude"
	ProviderGemini    ModelProvider = "gemini"
	ProviderWenxin    ModelProvider = "wenxin"     // 百度文心
	ProviderQwen      ModelProvider = "qwen"       // 阿里通义
	ProviderSpark     ModelProvider = "spark"       // 讯飞星火
	ProviderDoubao    ModelProvider = "doubao"      // 字节豆包
	ProviderDeepSeek  ModelProvider = "deepseek"   // DeepSeek
	ProviderMoonshot  ModelProvider = "moonshot"    // 月之暗面
	ProviderZhipu     ModelProvider = "zhipu"      // 智谱
	ProviderCustom    ModelProvider = "custom"      // 自定义
)

// ModelConfig 模型配置
type ModelConfig struct {
	ID          string        `json:"id" gorm:"primaryKey"`
	Name        string        `json:"name"`
	Provider    ModelProvider `json:"provider"`
	ModelID     string        `json:"model_id"`     // 供应商侧的模型ID
	Endpoint    string        `json:"endpoint"`     // API 端点
	APIKey      string        `json:"-" gorm:"-"`   // 运行时从加密存储读取
	APIKeyEnc   string        `json:"api_key_enc" gorm:"column:api_key_enc"` // 加密后的API Key
	MaxTokens   int           `json:"max_tokens"`
	InputPrice  float64       `json:"input_price"`  // 每1K Token输入价格
	OutputPrice float64       `json:"output_price"` // 每1K Token输出价格
	Currency    string        `json:"currency"`
	Weight      int           `json:"weight"`       // 路由权重 1-100
	Priority    int           `json:"priority"`      // 优先级，数字越小越优先
	Enabled     bool          `json:"enabled"`
	Streamable  bool          `json:"streamable"`   // 是否支持流式
	TenantID    string        `json:"tenant_id" gorm:"index"` // 所属租户，空为公共模型
	Tags        []string      `json:"tags" gorm:"serializer:json"`
	LatencyMs   int           `json:"latency_ms"`   // 最近平均延迟
	SuccessRate float64       `json:"success_rate"`  // 最近成功率 0-1
	CreatedAt   int64         `json:"created_at"`
	UpdatedAt   int64         `json:"updated_at"`
}

// ChatRequest 统一聊天请求（OpenAI兼容格式）
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	N           *int          `json:"n,omitempty"`
	User        string        `json:"user,omitempty"`
	// 扩展字段
	TenantID    string `json:"-"` // 由中间件注入
	APIKeyID    string `json:"-"` // 由中间件注入
	TraceID     string `json:"-"` // 链路追踪ID
	// 渠道路由扩展字段
	ChannelEndpoint string `json:"-"` // 渠道指定的端点
	ChannelAPIKey   string `json:"-"` // 渠道指定的API Key
	ChannelID       string `json:"-"` // 渠道ID，用于记录成功/失败
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatResponse 统一聊天响应（OpenAI兼容格式）
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason *string `json:"finish_reason"`
}

// Usage Token使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}
