<template>
  <div class="docs-page">
    <!-- 页面头部 -->
    <div class="page-hero">
      <h1>API 文档</h1>
      <p>5 分钟快速接入 TokenHub API，完全兼容 OpenAI 协议标准</p>
    </div>

    <!-- 快速开始 -->
    <div class="doc-section">
      <h2>🚀 快速开始</h2>
      <div class="steps-grid">
        <div class="step-card">
          <div class="step-num">1</div>
          <h3>注册获取 API Key</h3>
          <p>注册 TokenHub 账号后，在 <router-link to="/apikeys">API 密钥</router-link> 页面创建密钥</p>
        </div>
        <div class="step-card">
          <div class="step-num">2</div>
          <h3>替换 Base URL</h3>
          <p>将代码中的 <code>api.openai.com</code> 替换为 <code>api.tokenhub.cc</code></p>
        </div>
        <div class="step-card">
          <div class="step-num">3</div>
          <h3>开始调用</h3>
          <p>无需修改其他代码，享受统一网关的智能路由和负载均衡</p>
        </div>
      </div>
    </div>

    <!-- Base URL 配置 -->
    <div class="doc-section">
      <h2>🌐 Base URL</h2>
      <div class="config-table">
        <div class="config-row">
          <span class="config-label">生产环境</span>
          <code class="config-value">https://api.tokenhub.cc/v1</code>
        </div>
        <div class="config-row">
          <span class="config-label">认证方式</span>
          <code class="config-value">Authorization: Bearer sk-your-api-key</code>
        </div>
      </div>
    </div>

    <!-- 代码示例 -->
    <div class="doc-section">
      <h2>💻 代码示例</h2>
      <div class="lang-tabs">
        <button v-for="l in langTabs" :key="l.key" class="lang-tab" :class="{ active: activeLang === l.key }" @click="activeLang = l.key">{{ l.label }}</button>
      </div>
      <div class="code-block">
        <div class="code-header">
          <span>{{ activeLangLabel }}</span>
          <button class="copy-btn" @click="copyCode(activeCode)">复制</button>
        </div>
        <pre><code>{{ activeCode }}</code></pre>
      </div>
    </div>

    <!-- API 端点 -->
    <div class="doc-section">
      <h2>📋 API 端点</h2>
      <div class="endpoint-list">
        <div v-for="ep in endpoints" :key="ep.path" class="endpoint-card">
          <div class="endpoint-header" @click="ep.expanded = !ep.expanded">
            <div class="endpoint-left">
              <span class="method-badge" :class="ep.method.toLowerCase()">{{ ep.method }}</span>
              <code class="endpoint-path">{{ ep.path }}</code>
              <span class="endpoint-desc">{{ ep.desc }}</span>
            </div>
            <svg class="expand-icon" :class="{ open: ep.expanded }" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
          </div>
          <div v-if="ep.expanded" class="endpoint-body">
            <div v-if="ep.params" class="params-section">
              <h4>请求参数</h4>
              <el-table :data="ep.params" size="small" border>
                <el-table-column prop="name" label="参数" width="140" />
                <el-table-column prop="type" label="类型" width="100" />
                <el-table-column prop="required" label="必填" width="70">
                  <template #default="{ row }"><el-tag :type="row.required ? 'danger' : 'info'" size="small">{{ row.required ? '是' : '否' }}</el-tag></template>
                </el-table-column>
                <el-table-column prop="desc" label="说明" />
              </el-table>
            </div>
            <div v-if="ep.example" class="example-section">
              <h4>请求示例</h4>
              <pre class="example-code">{{ ep.example }}</pre>
            </div>
            <div v-if="ep.response" class="response-section">
              <h4>响应示例</h4>
              <pre class="response-code">{{ ep.response }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 错误码 -->
    <div class="doc-section">
      <h2>⚠️ 错误码</h2>
      <el-table :data="errorCodes" stripe border>
        <el-table-column prop="code" label="状态码" width="100" />
        <el-table-column prop="type" label="错误类型" width="200" />
        <el-table-column prop="desc" label="说明" />
        <el-table-column prop="solution" label="解决方案" />
      </el-table>
    </div>

    <!-- 速率限制 -->
    <div class="doc-section">
      <h2>⏱️ 速率限制</h2>
      <el-table :data="rateLimits" stripe border>
        <el-table-column prop="plan" label="方案" width="150" />
        <el-table-column prop="rpm" label="RPM (每分钟请求)" width="180" />
        <el-table-column prop="tpm" label="TPM (每分钟Token)" width="180" />
        <el-table-column prop="concurrent" label="并发连接" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'

const activeLang = ref('python')
const langTabs = [
  { key: 'python', label: 'Python' },
  { key: 'nodejs', label: 'Node.js' },
  { key: 'curl', label: 'cURL' },
  { key: 'go', label: 'Go' },
]

const codeExamples: Record<string, string> = {
  python: `from openai import OpenAI

client = OpenAI(
    base_url="https://api.tokenhub.cc/v1",
    api_key="sk-your-tokenhub-key"
)

# 非流式请求
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "你是一个有帮助的助手。"},
        {"role": "user", "content": "你好！请介绍一下自己。"}
    ],
    temperature=0.7,
    max_tokens=1000
)
print(response.choices[0].message.content)

# 流式请求
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "写一首关于春天的诗"}],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")`,

  nodejs: `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'https://api.tokenhub.cc/v1',
  apiKey: 'sk-your-tokenhub-key',
});

// 非流式请求
const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [
    { role: 'system', content: '你是一个有帮助的助手。' },
    { role: 'user', content: '你好！请介绍一下自己。' },
  ],
  temperature: 0.7,
  max_tokens: 1000,
});
console.log(response.choices[0].message.content);

// 流式请求
const stream = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: '写一首关于春天的诗' }],
  stream: true,
});
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}`,

  curl: `# 非流式请求
curl https://api.tokenhub.cc/v1/chat/completions \\
  -H "Authorization: Bearer sk-your-tokenhub-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "system", "content": "你是一个有帮助的助手。"},
      {"role": "user", "content": "你好！"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'

# 流式请求
curl https://api.tokenhub.cc/v1/chat/completions \\
  -H "Authorization: Bearer sk-your-tokenhub-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "你好！"}],
    "stream": true
  }'`,

  go: `package main

import (
    "context"
    "fmt"
    "io"

    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("sk-your-tokenhub-key")
    config.BaseURL = "https://api.tokenhub.cc/v1"

    client := openai.NewClientWithConfig(config)

    // 非流式请求
    resp, _ := client.CreateChatCompletion(context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT4o,
            Messages: []openai.ChatCompletionMessage{
                {Role: openai.ChatMessageRoleSystem, Content: "你是一个有帮助的助手。"},
                {Role: openai.ChatMessageRoleUser, Content: "你好！"},
            },
        },
    )
    fmt.Println(resp.Choices[0].Message.Content)

    // 流式请求
    stream, _ := client.CreateChatCompletionStream(context.Background(),
        openai.ChatCompletionRequest{
            Model:    openai.GPT4o,
            Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "你好！"}},
            Stream:   true,
        },
    )
    defer stream.Close()
    for {
        resp, err := stream.Recv()
        if err == io.EOF { break }
        fmt.Print(resp.Choices[0].Delta.Content)
    }
}`,
}

const activeLangLabel = computed(() => langTabs.find(l => l.key === activeLang.value)?.label || '')
const activeCode = computed(() => codeExamples[activeLang.value] || '')

const endpoints = reactive([
  {
    method: 'POST', path: '/v1/chat/completions', desc: '对话补全', expanded: false,
    params: [
      { name: 'model', type: 'string', required: true, desc: '模型 ID，如 gpt-4o、claude-4-sonnet' },
      { name: 'messages', type: 'array', required: true, desc: '消息数组，每项含 role 和 content' },
      { name: 'temperature', type: 'number', required: false, desc: '采样温度 0-2，默认 1' },
      { name: 'max_tokens', type: 'integer', required: false, desc: '最大生成 Token 数' },
      { name: 'stream', type: 'boolean', required: false, desc: '是否启用流式输出' },
      { name: 'top_p', type: 'number', required: false, desc: '核采样参数 0-1' },
    ],
    example: `{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}`,
    response: `{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "gpt-4o",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "Hello! How can I help you?"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30}
}`,
  },
  {
    method: 'POST', path: '/v1/completions', desc: '文本补全（已废弃）', expanded: false,
    params: [
      { name: 'model', type: 'string', required: true, desc: '模型 ID' },
      { name: 'prompt', type: 'string', required: true, desc: '补全提示文本' },
      { name: 'max_tokens', type: 'integer', required: false, desc: '最大生成 Token 数' },
    ],
    example: '{ "model": "gpt-4o", "prompt": "Once upon a time", "max_tokens": 50 }',
    response: '{ "id": "cmpl-xxx", "choices": [{"text": "...", "finish_reason": "stop"}] }',
  },
  {
    method: 'GET', path: '/v1/models', desc: '获取可用模型列表', expanded: false,
    params: [],
    example: 'GET /v1/models\nAuthorization: Bearer sk-your-key',
    response: `{
  "object": "list",
  "data": [
    {"id": "gpt-4o", "object": "model", "owned_by": "openai"},
    {"id": "claude-4-sonnet", "object": "model", "owned_by": "anthropic"}
  ]
}`,
  },
  {
    method: 'POST', path: '/v1/embeddings', desc: '文本嵌入', expanded: false,
    params: [
      { name: 'model', type: 'string', required: true, desc: '嵌入模型，如 text-embedding-3-large' },
      { name: 'input', type: 'string/array', required: true, desc: '要嵌入的文本' },
    ],
    example: '{ "model": "text-embedding-3-large", "input": "Hello world" }',
    response: '{ "object": "list", "data": [{"embedding": [0.1, ...], "index": 0}], "model": "text-embedding-3-large" }',
  },
  {
    method: 'POST', path: '/v1/images/generations', desc: '图像生成', expanded: false,
    params: [
      { name: 'model', type: 'string', required: false, desc: '图像生成模型，如 dall-e-3' },
      { name: 'prompt', type: 'string', required: true, desc: '图像描述' },
      { name: 'n', type: 'integer', required: false, desc: '生成数量，默认 1' },
      { name: 'size', type: 'string', required: false, desc: '图像尺寸，如 1024x1024' },
    ],
    example: '{ "model": "dall-e-3", "prompt": "A cute cat", "n": 1, "size": "1024x1024" }',
    response: '{ "data": [{"url": "https://..."}] }',
  },
])

const errorCodes = [
  { code: '400', type: 'invalid_request_error', desc: '请求参数格式错误', solution: '检查请求体 JSON 格式和参数值' },
  { code: '401', type: 'authentication_error', desc: 'API Key 无效或已过期', solution: '检查 Authorization 头部是否正确' },
  { code: '402', type: 'insufficient_balance', desc: '账户余额不足', solution: '前往钱包充值' },
  { code: '403', type: 'permission_denied', desc: '无权访问该模型', solution: '检查 API Key 是否有该模型的访问权限' },
  { code: '404', type: 'model_not_found', desc: '模型不存在', solution: '通过 GET /v1/models 确认可用模型' },
  { code: '429', type: 'rate_limit_exceeded', desc: '请求频率超限', solution: '降低请求频率或升级方案' },
  { code: '500', type: 'server_error', desc: '服务器内部错误', solution: '稍后重试，如持续出现请联系支持' },
  { code: '503', type: 'service_unavailable', desc: '上游服务不可用', solution: '系统会自动切换到备用渠道' },
]

const rateLimits = [
  { plan: '免费版', rpm: '10', tpm: '40,000', concurrent: '2' },
  { plan: '开发者版', rpm: '100', tpm: '200,000', concurrent: '10' },
  { plan: '企业版', rpm: '无限', tpm: '无限', concurrent: '无限' },
]

function copyCode(code: string) {
  navigator.clipboard.writeText(code)
  ElMessage.success('已复制到剪贴板')
}
</script>

<style scoped>
.docs-page { max-width: 900px; padding: 0; }
.page-hero { margin-bottom: 32px; }
.page-hero h1 { font-size: 24px; font-weight: 800; margin: 0 0 8px; }
.page-hero p { color: #64748b; font-size: 14px; margin: 0; }

.doc-section { margin-bottom: 40px; }
.doc-section h2 { font-size: 18px; font-weight: 700; margin: 0 0 16px; }

.steps-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.step-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 24px;
  position: relative;
}
.step-num {
  position: absolute;
  top: -12px;
  left: 20px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #4f46e5);
  color: #fff;
  font-size: 14px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}
.step-card h3 { font-size: 15px; font-weight: 600; margin: 8px 0; }
.step-card p { font-size: 13px; color: #64748b; line-height: 1.6; margin: 0; }
.step-card code { background: #f1f5f9; padding: 1px 6px; border-radius: 4px; font-size: 12px; color: #6366f1; }

.config-table { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; overflow: hidden; }
.config-row { display: flex; align-items: center; padding: 14px 20px; border-bottom: 1px solid #f1f5f9; }
.config-row:last-child { border-bottom: none; }
.config-label { width: 120px; font-size: 13px; font-weight: 600; color: #64748b; }
.config-value { font-family: monospace; font-size: 13px; color: #334155; background: #f8fafc; padding: 4px 10px; border-radius: 6px; }

.lang-tabs { display: flex; gap: 4px; margin-bottom: 12px; }
.lang-tab {
  background: #fff;
  border: 1px solid #e5e7eb;
  padding: 6px 16px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.lang-tab:hover { border-color: #818cf8; }
.lang-tab.active { background: #6366f1; color: #fff; border-color: #6366f1; }

.code-block {
  background: #1e1e2e;
  border-radius: 12px;
  overflow: hidden;
}
.code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: rgba(0,0,0,0.2);
  font-size: 13px;
  color: #a0a0b0;
}
.copy-btn {
  background: rgba(255,255,255,0.1);
  border: none;
  color: #a0a0b0;
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
}
.copy-btn:hover { background: rgba(255,255,255,0.2); color: #fff; }
.code-block pre { padding: 20px; margin: 0; overflow-x: auto; }
.code-block code { color: #cdd6f4; font-family: 'Fira Code', Menlo, monospace; font-size: 13px; line-height: 1.7; white-space: pre; }

.endpoint-list { display: flex; flex-direction: column; gap: 8px; }
.endpoint-card { background: #fff; border: 1px solid #e5e7eb; border-radius: 10px; overflow: hidden; }
.endpoint-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  cursor: pointer;
  transition: background 0.2s;
}
.endpoint-header:hover { background: #f8fafc; }
.endpoint-left { display: flex; align-items: center; gap: 10px; }
.method-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 10px;
  border-radius: 4px;
  color: #fff;
  text-transform: uppercase;
}
.method-badge.get { background: #22c55e; }
.method-badge.post { background: #3b82f6; }
.endpoint-path { font-family: monospace; font-size: 13px; font-weight: 600; color: #334155; }
.endpoint-desc { font-size: 12px; color: #94a3b8; }
.expand-icon { width: 18px; height: 18px; color: #94a3b8; transition: transform 0.3s; }
.expand-icon.open { transform: rotate(180deg); }

.endpoint-body { padding: 0 18px 18px; border-top: 1px solid #f1f5f9; }
.params-section, .example-section, .response-section { margin-top: 14px; }
.params-section h4, .example-section h4, .response-section h4 { font-size: 13px; font-weight: 600; margin-bottom: 8px; color: #64748b; }
.example-code, .response-code {
  background: #1e1e2e;
  color: #cdd6f4;
  padding: 14px;
  border-radius: 8px;
  font-size: 12px;
  font-family: 'Fira Code', monospace;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre;
}
</style>
