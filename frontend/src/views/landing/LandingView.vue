<template>
  <div class="landing-page">
    <!-- 导航栏 -->
    <header class="landing-nav" :class="{ scrolled: isScrolled }">
      <div class="nav-container">
        <div class="nav-brand" @click="scrollToTop">
          <div class="brand-logo">T</div>
          <span class="brand-name">TokenHub</span>
        </div>
        <nav class="nav-links">
          <a href="#features" @click.prevent="scrollTo('features')">功能特性</a>
          <a href="#models" @click.prevent="scrollTo('models')">模型广场</a>
          <a href="#pricing" @click.prevent="scrollTo('pricing')">价格方案</a>
          <a href="#docs" @click.prevent="scrollTo('docs')">API 文档</a>
          <a href="#faq" @click.prevent="scrollTo('faq')">常见问题</a>
        </nav>
        <div class="nav-actions">
          <button class="btn-ghost" @click="$router.push('/login')">登录</button>
          <button class="btn-primary" @click="$router.push('/login')">免费开始</button>
        </div>
      </div>
    </header>

    <!-- Hero 区域 -->
    <section class="hero-section">
      <div class="hero-bg">
        <div class="hero-glow glow-1"></div>
        <div class="hero-glow glow-2"></div>
        <div class="hero-glow glow-3"></div>
        <div class="hero-grid"></div>
      </div>
      <div class="hero-container">
        <div class="hero-badge">
          <span class="badge-dot"></span>
          企业级 AI API 网关平台
        </div>
        <h1 class="hero-title">
          一个接口，<br />
          <span class="gradient-text">连接所有 AI 模型</span>
        </h1>
        <p class="hero-desc">
          统一管理 OpenAI、Claude、Gemini、DeepSeek、通义千问等 100+ 模型，
          兼容 OpenAI 协议标准，毫秒级智能路由，按量计费更省钱。
        </p>
        <div class="hero-actions">
          <button class="btn-primary btn-lg" @click="$router.push('/login')">
            <svg class="icon-sm" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
            立即开始使用
          </button>
          <button class="btn-outline btn-lg" @click="scrollTo('docs')">
            <svg class="icon-sm" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            查看 API 文档
          </button>
        </div>
        <div class="hero-stats">
          <div class="stat-item">
            <span class="stat-value">100+</span>
            <span class="stat-label">支持模型</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat-item">
            <span class="stat-value">99.9%</span>
            <span class="stat-label">服务可用性</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat-item">
            <span class="stat-value">&lt;100ms</span>
            <span class="stat-label">路由延迟</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat-item">
            <span class="stat-value">10亿+</span>
            <span class="stat-label">日处理 Token</span>
          </div>
        </div>
        <!-- 代码示例 -->
        <div class="code-preview">
          <div class="code-header">
            <div class="code-dots">
              <span class="dot red"></span>
              <span class="dot yellow"></span>
              <span class="dot green"></span>
            </div>
            <span class="code-title">quick-start.py</span>
            <button class="code-copy" @click="copyCode">复制</button>
          </div>
          <pre class="code-body"><code><span class="code-keyword">from</span> openai <span class="code-keyword">import</span> OpenAI

<span class="code-comment"># 只需替换 base_url 和 api_key</span>
client = OpenAI(
    base_url=<span class="code-string">"https://api.tokenhub.cc/v1"</span>,
    api_key=<span class="code-string">"sk-your-tokenhub-key"</span>
)

response = client.chat.completions.create(
    model=<span class="code-string">"gpt-4o"</span>,
    messages=[{<span class="code-string">"role"</span>: <span class="code-string">"user"</span>, <span class="code-string">"content"</span>: <span class="code-string">"你好！"</span>}]
)
<span class="code-keyword">print</span>(response.choices[<span class="code-number">0</span>].message.content)</code></pre>
        </div>
      </div>
    </section>

    <!-- 品牌墙 -->
    <section class="brands-section">
      <div class="container">
        <p class="brands-title">支持主流 AI 厂商</p>
        <div class="brands-grid">
          <div class="brand-item" v-for="b in supportedBrands" :key="b.name">
            <span class="brand-emoji">{{ b.icon }}</span>
            <span class="brand-text">{{ b.name }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 功能特性 -->
    <section id="features" class="features-section">
      <div class="container">
        <div class="section-header">
          <span class="section-tag">核心能力</span>
          <h2 class="section-title">为什么选择 TokenHub？</h2>
          <p class="section-desc">为企业级 AI 应用提供稳定、安全、高性价比的 API 中转服务</p>
        </div>
        <div class="features-grid">
          <div class="feature-card" v-for="(f, i) in features" :key="i">
            <div class="feature-icon" :style="{ background: f.bg }">
              <span class="feature-emoji">{{ f.icon }}</span>
            </div>
            <h3 class="feature-title">{{ f.title }}</h3>
            <p class="feature-desc">{{ f.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- 模型广场 -->
    <section id="models" class="models-section">
      <div class="container">
        <div class="section-header">
          <span class="section-tag">模型广场</span>
          <h2 class="section-title">丰富的模型选择</h2>
          <p class="section-desc">一个 API Key 即可调用 100+ 主流 AI 模型，覆盖对话、代码、图像、嵌入等场景</p>
        </div>
        <div class="model-categories">
          <button
            v-for="cat in modelCategories"
            :key="cat.key"
            class="category-btn"
            :class="{ active: activeCategory === cat.key }"
            @click="activeCategory = cat.key"
          >
            {{ cat.icon }} {{ cat.label }}
          </button>
        </div>
        <div class="models-grid">
          <div
            class="model-card"
            v-for="m in filteredModels"
            :key="m.id"
          >
            <div class="model-header">
              <span class="model-provider-badge">{{ m.provider }}</span>
              <span class="model-type-badge" :class="m.type">{{ m.typeLabel }}</span>
            </div>
            <h4 class="model-name">{{ m.name }}</h4>
            <p class="model-id">{{ m.id }}</p>
            <div class="model-meta">
              <span class="meta-item">↑ {{ m.context }}K 上下文</span>
              <span class="meta-item">¥ {{ m.price }}/1M tokens</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 价格方案 -->
    <section id="pricing" class="pricing-section">
      <div class="container">
        <div class="section-header">
          <span class="section-tag">灵活定价</span>
          <h2 class="section-title">选择适合你的方案</h2>
          <p class="section-desc">按量计费，用多少付多少，无最低消费</p>
        </div>
        <div class="pricing-grid">
          <div class="pricing-card" v-for="p in pricingPlans" :key="p.name" :class="{ featured: p.featured }">
            <div v-if="p.featured" class="pricing-badge">推荐</div>
            <h3 class="pricing-name">{{ p.name }}</h3>
            <div class="pricing-price">
              <span class="price-amount">{{ p.price }}</span>
              <span class="price-unit">{{ p.unit }}</span>
            </div>
            <p class="pricing-desc">{{ p.desc }}</p>
            <ul class="pricing-features">
              <li v-for="(f, i) in p.features" :key="i">
                <svg class="check-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                {{ f }}
              </li>
            </ul>
            <button
              class="btn-primary btn-block"
              :class="{ 'btn-outline-dark': p.featured }"
              @click="$router.push('/login')"
            >
              {{ p.cta }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- API 文档预览 -->
    <section id="docs" class="docs-section">
      <div class="container">
        <div class="section-header">
          <span class="section-tag">开发者友好</span>
          <h2 class="section-title">5 分钟快速接入</h2>
          <p class="section-desc">完全兼容 OpenAI API 协议，无需修改代码即可切换</p>
        </div>
        <div class="docs-grid">
          <div class="doc-card" v-for="(d, i) in docSteps" :key="i">
            <div class="doc-step">{{ i + 1 }}</div>
            <h4 class="doc-title">{{ d.title }}</h4>
            <p class="doc-desc">{{ d.desc }}</p>
            <pre v-if="d.code" class="doc-code"><code>{{ d.code }}</code></pre>
          </div>
        </div>
        <div class="docs-endpoints">
          <h3 class="endpoints-title">支持的 API 端点</h3>
          <div class="endpoints-grid">
            <div class="endpoint-item" v-for="e in apiEndpoints" :key="e.path">
              <span class="endpoint-method" :class="e.method.toLowerCase()">{{ e.method }}</span>
              <code class="endpoint-path">{{ e.path }}</code>
              <span class="endpoint-desc">{{ e.desc }}</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 常见问题 -->
    <section id="faq" class="faq-section">
      <div class="container">
        <div class="section-header">
          <span class="section-tag">帮助中心</span>
          <h2 class="section-title">常见问题</h2>
        </div>
        <div class="faq-list">
          <div
            class="faq-item"
            v-for="(f, i) in faqs"
            :key="i"
            :class="{ open: openFaq === i }"
            @click="openFaq = openFaq === i ? -1 : i"
          >
            <div class="faq-question">
              <span>{{ f.q }}</span>
              <svg class="faq-arrow" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
            </div>
            <div class="faq-answer" v-if="openFaq === i">{{ f.a }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA 区域 -->
    <section class="cta-section">
      <div class="container">
        <div class="cta-card">
          <h2 class="cta-title">准备好开始了吗？</h2>
          <p class="cta-desc">注册即送 1 元体验额度，0 门槛上手 AI API 服务</p>
          <div class="cta-actions">
            <button class="btn-white btn-lg" @click="$router.push('/login')">
              免费注册
              <svg class="icon-sm" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- 页脚 -->
    <footer class="landing-footer">
      <div class="container">
        <div class="footer-grid">
          <div class="footer-brand">
            <div class="nav-brand">
              <div class="brand-logo">T</div>
              <span class="brand-name">TokenHub</span>
            </div>
            <p class="footer-slogan">企业级 AI API 网关平台</p>
            <p class="footer-copy">© 2026 TokenHub. All rights reserved.</p>
          </div>
          <div class="footer-links">
            <h4>产品</h4>
            <a href="#features">功能特性</a>
            <a href="#models">模型广场</a>
            <a href="#pricing">价格方案</a>
            <a href="#docs">API 文档</a>
          </div>
          <div class="footer-links">
            <h4>支持</h4>
            <a href="#faq">常见问题</a>
            <a href="#">工单系统</a>
            <a href="#">技术支持</a>
            <a href="#">状态监控</a>
          </div>
          <div class="footer-links">
            <h4>公司</h4>
            <a href="#">关于我们</a>
            <router-link to="/terms">服务条款</router-link>
            <router-link to="/privacy">隐私政策</router-link>
            <a href="#">联系我们</a>
          </div>
        </div>
        <div class="footer-bottom">
          <span>兼容 OpenAI API 协议</span>
          <span>|</span>
          <span>安全稳定 · 低延迟 · 高并发</span>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const isScrolled = ref(false)
const activeCategory = ref('chat')
const openFaq = ref(-1)

const supportedBrands = [
  { name: 'OpenAI', icon: '🟢' },
  { name: 'Anthropic', icon: '🟤' },
  { name: 'Google', icon: '🔵' },
  { name: 'DeepSeek', icon: '🐋' },
  { name: '通义千问', icon: '🟣' },
  { name: '智谱 GLM', icon: '✨' },
  { name: '豆包', icon: '🌋' },
  { name: 'Moonshot', icon: '🌙' },
  { name: '文心一言', icon: '🔵' },
  { name: '讯飞星火', icon: '⭐' },
  { name: 'MiniMax', icon: '🎯' },
  { name: 'Cohere', icon: '🟠' },
  { name: 'Mistral', icon: '💨' },
  { name: 'xAI', icon: '🚀' },
  { name: 'Meta Llama', icon: '🦙' },
  { name: '百川', icon: '🏔️' },
]

const features = [
  {
    icon: '⚡',
    title: '智能路由',
    desc: '多渠道负载均衡与故障自动切换，确保服务高可用，延迟低至 50ms',
    bg: 'linear-gradient(135deg, #fef3c7, #fde68a)',
  },
  {
    icon: '🔒',
    title: '安全合规',
    desc: '数据传输端到端加密，API Key 权限精细管控，操作审计全程可追溯',
    bg: 'linear-gradient(135deg, #dbeafe, #bfdbfe)',
  },
  {
    icon: '💰',
    title: '按量计费',
    desc: '精确到 Token 级别计费，用多少付多少，透明定价无隐藏费用',
    bg: 'linear-gradient(135deg, #d1fae5, #a7f3d0)',
  },
  {
    icon: '🔗',
    title: '协议兼容',
    desc: '完全兼容 OpenAI API 格式，现有代码只需替换 base_url 即可接入',
    bg: 'linear-gradient(135deg, #ede9fe, #ddd6fe)',
  },
  {
    icon: '📊',
    title: '实时监控',
    desc: '请求量、延迟、错误率、Token 消耗实时可视化，异常自动告警',
    bg: 'linear-gradient(135deg, #fce7f3, #fbcfe8)',
  },
  {
    icon: '🏢',
    title: '多租户架构',
    desc: '支持企业多部门独立管理，配额隔离、权限分离、统一结算',
    bg: 'linear-gradient(135deg, #e0e7ff, #c7d2fe)',
  },
]

const modelCategories = [
  { key: 'chat', label: '对话模型', icon: '💬' },
  { key: 'code', label: '代码模型', icon: '💻' },
  { key: 'vision', label: '视觉模型', icon: '👁️' },
  { key: 'embedding', label: '嵌入模型', icon: '🔗' },
  { key: 'image', label: '图像生成', icon: '🎨' },
]

const allModels = [
  // 对话模型
  { id: 'gpt-4o', name: 'GPT-4o', provider: 'OpenAI', type: 'chat', typeLabel: '对话', context: 128, price: '60' },
  { id: 'gpt-4o-mini', name: 'GPT-4o Mini', provider: 'OpenAI', type: 'chat', typeLabel: '对话', context: 128, price: '8' },
  { id: 'o3', name: 'o3', provider: 'OpenAI', type: 'chat', typeLabel: '推理', context: 200, price: '120' },
  { id: 'claude-4-sonnet', name: 'Claude 4 Sonnet', provider: 'Anthropic', type: 'chat', typeLabel: '对话', context: 200, price: '21' },
  { id: 'claude-4-opus', name: 'Claude 4 Opus', provider: 'Anthropic', type: 'chat', typeLabel: '推理', context: 200, price: '75' },
  { id: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro', provider: 'Google', type: 'chat', typeLabel: '对话', context: 1000, price: '10' },
  { id: 'deepseek-r1', name: 'DeepSeek R1', provider: 'DeepSeek', type: 'chat', typeLabel: '推理', context: 128, price: '4' },
  { id: 'deepseek-v3', name: 'DeepSeek V3', provider: 'DeepSeek', type: 'chat', typeLabel: '对话', context: 128, price: '1' },
  { id: 'qwen-max', name: '通义千问 Max', provider: '阿里云', type: 'chat', typeLabel: '对话', context: 128, price: '8' },
  { id: 'glm-4-plus', name: 'GLM-4 Plus', provider: '智谱', type: 'chat', typeLabel: '对话', context: 128, price: '25' },
  { id: 'doubao-pro-32k', name: '豆包 Pro', provider: '字节跳动', type: 'chat', typeLabel: '对话', context: 32, price: '0.5' },
  { id: 'moonshot-v1-128k', name: 'Moonshot v1', provider: '月之暗面', type: 'chat', typeLabel: '对话', context: 128, price: '14' },
  // 代码模型
  { id: 'o3-mini', name: 'o3 Mini', provider: 'OpenAI', type: 'code', typeLabel: '代码', context: 200, price: '12' },
  { id: 'claude-4-sonnet', name: 'Claude 4 Sonnet', provider: 'Anthropic', type: 'code', typeLabel: '代码', context: 200, price: '21' },
  { id: 'deepseek-coder-v2', name: 'DeepSeek Coder V2', provider: 'DeepSeek', type: 'code', typeLabel: '代码', context: 128, price: '1' },
  { id: 'qwen-coder-plus', name: '通义千问 Coder', provider: '阿里云', type: 'code', typeLabel: '代码', context: 128, price: '4' },
  // 视觉模型
  { id: 'gpt-4o', name: 'GPT-4o Vision', provider: 'OpenAI', type: 'vision', typeLabel: '视觉', context: 128, price: '60' },
  { id: 'claude-4-sonnet', name: 'Claude 4 Vision', provider: 'Anthropic', type: 'vision', typeLabel: '视觉', context: 200, price: '21' },
  { id: 'gemini-2.5-pro', name: 'Gemini 2.5 Pro Vision', provider: 'Google', type: 'vision', typeLabel: '视觉', context: 1000, price: '10' },
  { id: 'qwen-vl-max', name: '通义千问 VL', provider: '阿里云', type: 'vision', typeLabel: '视觉', context: 32, price: '8' },
  // 嵌入模型
  { id: 'text-embedding-3-large', name: 'Embedding 3 Large', provider: 'OpenAI', type: 'embedding', typeLabel: '嵌入', context: 8, price: '0.8' },
  { id: 'text-embedding-3-small', name: 'Embedding 3 Small', provider: 'OpenAI', type: 'embedding', typeLabel: '嵌入', context: 8, price: '0.1' },
  { id: 'text-embedding-v3', name: '通义 Embedding V3', provider: '阿里云', type: 'embedding', typeLabel: '嵌入', context: 8, price: '0.5' },
  // 图像生成
  { id: 'dall-e-3', name: 'DALL·E 3', provider: 'OpenAI', type: 'image', typeLabel: '图像', context: 0, price: '0.4/张' },
  { id: 'stable-diffusion-3', name: 'Stable Diffusion 3', provider: 'Stability', type: 'image', typeLabel: '图像', context: 0, price: '0.2/张' },
  { id: 'flux-pro', name: 'FLUX.1 Pro', provider: 'BlackForest', type: 'image', typeLabel: '图像', context: 0, price: '0.3/张' },
]

const filteredModels = computed(() => {
  return allModels.filter(m => m.type === activeCategory.value)
})

const pricingPlans = [
  {
    name: '免费体验',
    price: '¥0',
    unit: '',
    desc: '适合个人开发者试用和评估',
    featured: false,
    cta: '免费注册',
    features: [
      '注册送 1 元体验额度',
      '支持所有对话模型',
      '10 RPM 速率限制',
      '社区技术支持',
    ],
  },
  {
    name: '开发者版',
    price: '按量',
    unit: '计费',
    desc: '适合独立开发者和中小项目',
    featured: true,
    cta: '立即开通',
    features: [
      '按 Token 用量计费',
      '支持所有 100+ 模型',
      '100 RPM 速率限制',
      'API Key 管理',
      '用量实时监控',
      '工单技术支持',
    ],
  },
  {
    name: '企业版',
    price: '定制',
    unit: '方案',
    desc: '适合大规模企业级应用',
    featured: false,
    cta: '联系我们',
    features: [
      '专属高可用通道',
      '无限速率限制',
      '多租户架构支持',
      'SLA 服务保障 99.99%',
      '专属客户经理',
      '定制化部署方案',
    ],
  },
]

const docSteps = [
  {
    title: '注册获取 API Key',
    desc: '注册账号后，在控制台创建 API Key，即可开始调用',
    code: 'sk-xxxxxxxxxxxxxxxx',
  },
  {
    title: '替换 base_url',
    desc: '将你现有代码中的 OpenAI base_url 替换为 TokenHub 地址',
    code: 'base_url="https://api.tokenhub.cc/v1"',
  },
  {
    title: '开始调用',
    desc: '无需修改其他代码，直接享受统一网关服务',
    code: `curl https://api.tokenhub.cc/v1/chat/completions \\
  -H "Authorization: Bearer sk-xxx" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hi"}]}'`,
  },
]

const apiEndpoints = [
  { method: 'POST', path: '/v1/chat/completions', desc: '对话补全' },
  { method: 'POST', path: '/v1/completions', desc: '文本补全' },
  { method: 'POST', path: '/v1/embeddings', desc: '文本嵌入' },
  { method: 'GET', path: '/v1/models', desc: '模型列表' },
  { method: 'POST', path: '/v1/images/generations', desc: '图像生成' },
  { method: 'POST', path: '/v1/audio/transcriptions', desc: '语音转文字' },
]

const faqs = [
  {
    q: 'TokenHub 和直接调用 OpenAI 有什么区别？',
    a: 'TokenHub 提供统一的 API 网关，一个 API Key 即可调用 100+ 模型（OpenAI、Claude、Gemini、国产模型等），无需分别注册和付费。同时提供智能路由、故障切换、用量监控等企业级能力。',
  },
  {
    q: '需要修改现有代码吗？',
    a: '几乎不需要。TokenHub 完全兼容 OpenAI API 协议，你只需要将 base_url 从 https://api.openai.com/v1 替换为 https://api.tokenhub.cc/v1，并将 API Key 替换为 TokenHub 的 Key 即可。',
  },
  {
    q: '数据安全吗？',
    a: 'TokenHub 不存储任何请求和响应内容，数据仅做转发处理。所有通信采用 TLS 加密，API Key 支持权限隔离和过期设置，操作全程审计可追溯。',
  },
  {
    q: '计费方式是怎样的？',
    a: '按实际使用的 Token 数量计费，不同模型单价不同，透明定价无隐藏费用。充值后按量扣费，余额不足时自动暂停服务，不会产生欠费。',
  },
  {
    q: '服务可用性如何保障？',
    a: '多渠道负载均衡 + 自动故障切换，当某个上游 API 不可用时自动路由到备用渠道，SLA 承诺 99.9% 可用性。企业版提供 99.99% SLA 保障。',
  },
  {
    q: '支持哪些国产模型？',
    a: '支持通义千问、智谱 GLM、豆包、Moonshot、文心一言、讯飞星火、MiniMax、百川、零一万物、阶跃星辰等主流国产模型，且价格通常比官方更优惠。',
  },
]

function scrollTo(id: string) {
  const el = document.getElementById(id)
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function copyCode() {
  const code = `from openai import OpenAI

client = OpenAI(
    base_url="https://api.tokenhub.cc/v1",
    api_key="sk-your-tokenhub-key"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "你好！"}]
)
print(response.choices[0].message.content)`
  navigator.clipboard.writeText(code)
}

function handleScroll() {
  isScrolled.value = window.scrollY > 20
}

onMounted(() => window.addEventListener('scroll', handleScroll))
onUnmounted(() => window.removeEventListener('scroll', handleScroll))
</script>

<style scoped>
/* ====== 基础变量 ====== */
.landing-page {
  --primary: #6366f1;
  --primary-light: #818cf8;
  --primary-dark: #4f46e5;
  --bg: #ffffff;
  --bg-soft: #f8fafc;
  --bg-muted: #f1f5f9;
  --text: #0f172a;
  --text-secondary: #475569;
  --text-muted: #94a3b8;
  --border: #e2e8f0;
  --radius: 12px;
  --radius-lg: 16px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif;
  color: var(--text);
  overflow-x: hidden;
}

/* ====== 导航栏 ====== */
.landing-nav {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  height: 64px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  transition: all 0.3s;
  border-bottom: 1px solid transparent;
}
.landing-nav.scrolled {
  background: rgba(255, 255, 255, 0.95);
  border-bottom-color: var(--border);
  box-shadow: 0 1px 3px rgba(0,0,0,0.05);
}
.nav-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.nav-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}
.brand-logo {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary), var(--primary-dark));
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 800;
  font-size: 18px;
}
.brand-name {
  font-size: 20px;
  font-weight: 700;
  color: var(--text);
}
.nav-links {
  display: flex;
  gap: 32px;
}
.nav-links a {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s;
}
.nav-links a:hover {
  color: var(--primary);
}
.nav-actions {
  display: flex;
  gap: 12px;
}

/* ====== 按钮 ====== */
.btn-primary {
  background: linear-gradient(135deg, var(--primary), var(--primary-dark));
  color: #fff;
  border: none;
  padding: 8px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}
.btn-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}
.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.2s;
}
.btn-ghost:hover {
  color: var(--primary);
}
.btn-outline {
  background: transparent;
  color: var(--primary);
  border: 1.5px solid var(--primary);
  padding: 8px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}
.btn-outline:hover {
  background: var(--primary);
  color: #fff;
}
.btn-lg {
  padding: 12px 28px;
  font-size: 16px;
}
.btn-block {
  width: 100%;
  justify-content: center;
}
.btn-white {
  background: #fff;
  color: var(--primary);
  border: none;
  padding: 12px 28px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s;
}
.btn-white:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.15);
}
.btn-outline-dark {
  background: var(--primary);
  color: #fff;
}
.btn-outline-dark:hover {
  background: var(--primary-dark);
  color: #fff;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
}
.icon-sm {
  width: 18px;
  height: 18px;
}

/* ====== Hero ====== */
.hero-section {
  position: relative;
  padding: 140px 24px 80px;
  text-align: center;
  overflow: hidden;
}
.hero-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.hero-glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.3;
}
.glow-1 {
  width: 600px;
  height: 400px;
  top: -100px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--primary);
}
.glow-2 {
  width: 400px;
  height: 400px;
  top: 100px;
  right: -100px;
  background: #a78bfa;
}
.glow-3 {
  width: 400px;
  height: 400px;
  top: 50px;
  left: -100px;
  background: #60a5fa;
}
.hero-grid {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(circle, #e2e8f0 1px, transparent 1px);
  background-size: 40px 40px;
  opacity: 0.5;
}
.hero-container {
  position: relative;
  max-width: 800px;
  margin: 0 auto;
}
.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: rgba(99, 102, 241, 0.08);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 100px;
  padding: 6px 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--primary);
  margin-bottom: 24px;
}
.badge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  animation: pulse-dot 2s ease-in-out infinite;
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
.hero-title {
  font-size: 56px;
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.02em;
  margin-bottom: 24px;
}
.gradient-text {
  background: linear-gradient(135deg, var(--primary), #a78bfa, #60a5fa);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.hero-desc {
  font-size: 18px;
  line-height: 1.7;
  color: var(--text-secondary);
  max-width: 600px;
  margin: 0 auto 36px;
}
.hero-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
  margin-bottom: 48px;
}
.hero-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 32px;
  margin-bottom: 48px;
}
.stat-item {
  text-align: center;
}
.stat-value {
  display: block;
  font-size: 28px;
  font-weight: 800;
  color: var(--text);
}
.stat-label {
  display: block;
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
}
.stat-divider {
  width: 1px;
  height: 40px;
  background: var(--border);
}

/* ====== 代码预览 ====== */
.code-preview {
  background: #1e1e2e;
  border-radius: var(--radius-lg);
  overflow: hidden;
  text-align: left;
  box-shadow: 0 25px 50px -12px rgba(0,0,0,0.15);
}
.code-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: rgba(0,0,0,0.3);
  gap: 12px;
}
.code-dots {
  display: flex;
  gap: 6px;
}
.code-dots .dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}
.dot.red { background: #ff5f57; }
.dot.yellow { background: #febc2e; }
.dot.green { background: #28c840; }
.code-title {
  font-size: 13px;
  color: #a0a0b0;
  flex: 1;
}
.code-copy {
  background: rgba(255,255,255,0.1);
  border: none;
  color: #a0a0b0;
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.code-copy:hover {
  background: rgba(255,255,255,0.2);
  color: #fff;
}
.code-body {
  padding: 20px;
  margin: 0;
  overflow-x: auto;
  font-size: 14px;
  line-height: 1.8;
  color: #cdd6f4;
  font-family: 'Fira Code', 'Cascadia Code', 'JetBrains Mono', Menlo, monospace;
}
.code-keyword { color: #cba6f7; }
.code-string { color: #a6e3a1; }
.code-comment { color: #6c7086; }
.code-number { color: #fab387; }

/* ====== 品牌墙 ====== */
.brands-section {
  padding: 48px 24px;
  background: var(--bg-soft);
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}
.brands-title {
  text-align: center;
  font-size: 14px;
  color: var(--text-muted);
  margin-bottom: 24px;
  font-weight: 500;
}
.brands-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 16px;
  max-width: 900px;
  margin: 0 auto;
}
.brand-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #fff;
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all 0.2s;
}
.brand-item:hover {
  border-color: var(--primary-light);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.1);
}
.brand-emoji {
  font-size: 16px;
}

/* ====== 通用 Section ====== */
.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
}
.section-header {
  text-align: center;
  margin-bottom: 48px;
}
.section-tag {
  display: inline-block;
  background: rgba(99, 102, 241, 0.08);
  color: var(--primary);
  font-size: 13px;
  font-weight: 600;
  padding: 4px 14px;
  border-radius: 100px;
  margin-bottom: 16px;
}
.section-title {
  font-size: 36px;
  font-weight: 800;
  letter-spacing: -0.02em;
  margin-bottom: 12px;
}
.section-desc {
  font-size: 16px;
  color: var(--text-secondary);
  max-width: 560px;
  margin: 0 auto;
}

/* ====== 功能特性 ====== */
.features-section {
  padding: 96px 24px;
}
.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}
.feature-card {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 32px;
  transition: all 0.3s;
}
.feature-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 32px rgba(0,0,0,0.08);
  border-color: var(--primary-light);
}
.feature-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}
.feature-emoji {
  font-size: 24px;
}
.feature-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 8px;
}
.feature-desc {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-secondary);
}

/* ====== 模型广场 ====== */
.models-section {
  padding: 96px 24px;
  background: var(--bg-soft);
}
.model-categories {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-bottom: 32px;
}
.category-btn {
  background: #fff;
  border: 1px solid var(--border);
  padding: 8px 18px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}
.category-btn:hover {
  border-color: var(--primary-light);
  color: var(--primary);
}
.category-btn.active {
  background: var(--primary);
  color: #fff;
  border-color: var(--primary);
}
.models-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}
.model-card {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 20px;
  transition: all 0.2s;
}
.model-card:hover {
  border-color: var(--primary-light);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.08);
}
.model-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.model-provider-badge {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.model-type-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 100px;
  background: #ede9fe;
  color: #7c3aed;
}
.model-type-badge.chat { background: #dbeafe; color: #2563eb; }
.model-type-badge.code { background: #d1fae5; color: #059669; }
.model-type-badge.vision { background: #fce7f3; color: #db2777; }
.model-type-badge.embedding { background: #fef3c7; color: #d97706; }
.model-type-badge.image { background: #ede9fe; color: #7c3aed; }
.model-name {
  font-size: 15px;
  font-weight: 700;
  margin-bottom: 2px;
}
.model-id {
  font-size: 12px;
  color: var(--text-muted);
  font-family: monospace;
  margin-bottom: 12px;
}
.model-meta {
  display: flex;
  gap: 12px;
}
.meta-item {
  font-size: 12px;
  color: var(--text-secondary);
}

/* ====== 价格方案 ====== */
.pricing-section {
  padding: 96px 24px;
}
.pricing-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  align-items: start;
}
.pricing-card {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 36px;
  position: relative;
  transition: all 0.3s;
}
.pricing-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 32px rgba(0,0,0,0.08);
}
.pricing-card.featured {
  border-color: var(--primary);
  box-shadow: 0 8px 32px rgba(99, 102, 241, 0.15);
  transform: scale(1.03);
}
.pricing-card.featured:hover {
  transform: scale(1.03) translateY(-4px);
}
.pricing-badge {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: linear-gradient(135deg, var(--primary), var(--primary-dark));
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  padding: 4px 16px;
  border-radius: 100px;
}
.pricing-name {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 12px;
}
.pricing-price {
  margin-bottom: 8px;
}
.price-amount {
  font-size: 40px;
  font-weight: 800;
}
.price-unit {
  font-size: 16px;
  color: var(--text-muted);
  margin-left: 4px;
}
.pricing-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}
.pricing-features {
  list-style: none;
  padding: 0;
  margin: 0 0 28px;
}
.pricing-features li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
  padding: 6px 0;
}
.check-icon {
  width: 16px;
  height: 16px;
  color: #22c55e;
  flex-shrink: 0;
}

/* ====== 文档预览 ====== */
.docs-section {
  padding: 96px 24px;
  background: var(--bg-soft);
}
.docs-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  margin-bottom: 48px;
}
.doc-card {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 28px;
}
.doc-step {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary), var(--primary-dark));
  color: #fff;
  font-size: 14px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
}
.doc-title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 8px;
}
.doc-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 16px;
  line-height: 1.6;
}
.doc-code {
  background: #1e1e2e;
  color: #cdd6f4;
  padding: 12px;
  border-radius: 8px;
  font-size: 12px;
  overflow-x: auto;
  margin: 0;
  font-family: 'Fira Code', Menlo, monospace;
  line-height: 1.6;
}
.docs-endpoints {
  background: #fff;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 32px;
}
.endpoints-title {
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 20px;
}
.endpoints-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}
.endpoint-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--bg-soft);
  border-radius: 8px;
  font-size: 13px;
}
.endpoint-method {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
  text-transform: uppercase;
}
.endpoint-method.get {
  background: #d1fae5;
  color: #059669;
}
.endpoint-method.post {
  background: #dbeafe;
  color: #2563eb;
}
.endpoint-path {
  font-family: monospace;
  color: var(--text);
  font-weight: 600;
}
.endpoint-desc {
  color: var(--text-muted);
}

/* ====== FAQ ====== */
.faq-section {
  padding: 96px 24px;
}
.faq-list {
  max-width: 720px;
  margin: 0 auto;
}
.faq-item {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  margin-bottom: 8px;
  cursor: pointer;
  transition: all 0.2s;
  overflow: hidden;
}
.faq-item:hover {
  border-color: var(--primary-light);
}
.faq-item.open {
  border-color: var(--primary);
}
.faq-question {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 20px;
  font-size: 15px;
  font-weight: 600;
}
.faq-arrow {
  width: 20px;
  height: 20px;
  color: var(--text-muted);
  transition: transform 0.3s;
  flex-shrink: 0;
}
.faq-item.open .faq-arrow {
  transform: rotate(180deg);
}
.faq-answer {
  padding: 0 20px 18px;
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-secondary);
}

/* ====== CTA ====== */
.cta-section {
  padding: 96px 24px;
  background: var(--bg-soft);
}
.cta-card {
  background: linear-gradient(135deg, var(--primary), var(--primary-dark));
  border-radius: 24px;
  padding: 64px;
  text-align: center;
  color: #fff;
}
.cta-title {
  font-size: 36px;
  font-weight: 800;
  margin-bottom: 12px;
}
.cta-desc {
  font-size: 16px;
  opacity: 0.9;
  margin-bottom: 32px;
}
.cta-actions {
  display: flex;
  justify-content: center;
}

/* ====== 页脚 ====== */
.landing-footer {
  padding: 64px 24px 32px;
  background: #0f172a;
  color: #94a3b8;
}
.landing-footer .brand-name {
  color: #fff;
}
.footer-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 48px;
  margin-bottom: 48px;
}
.footer-slogan {
  font-size: 14px;
  margin-top: 12px;
  color: #64748b;
}
.footer-copy {
  font-size: 12px;
  margin-top: 16px;
  color: #475569;
}
.footer-links h4 {
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 16px;
}
.footer-links a {
  display: block;
  color: #64748b;
  font-size: 13px;
  text-decoration: none;
  padding: 4px 0;
  transition: color 0.2s;
}
.footer-links a:hover {
  color: #94a3b8;
}
.footer-bottom {
  border-top: 1px solid #1e293b;
  padding-top: 24px;
  text-align: center;
  font-size: 13px;
  color: #475569;
  display: flex;
  justify-content: center;
  gap: 8px;
}

/* ====== 响应式 ====== */
@media (max-width: 1024px) {
  .features-grid { grid-template-columns: repeat(2, 1fr); }
  .models-grid { grid-template-columns: repeat(3, 1fr); }
  .pricing-grid { grid-template-columns: repeat(3, 1fr); }
  .footer-grid { grid-template-columns: repeat(2, 1fr); gap: 32px; }
}
@media (max-width: 768px) {
  .nav-links { display: none; }
  .hero-title { font-size: 36px; }
  .hero-desc { font-size: 16px; }
  .hero-actions { flex-direction: column; align-items: center; }
  .hero-stats { flex-wrap: wrap; gap: 16px; }
  .stat-divider { display: none; }
  .section-title { font-size: 28px; }
  .features-grid { grid-template-columns: 1fr; }
  .models-grid { grid-template-columns: repeat(2, 1fr); }
  .pricing-grid { grid-template-columns: 1fr; max-width: 400px; margin: 0 auto; }
  .docs-grid { grid-template-columns: 1fr; }
  .endpoints-grid { grid-template-columns: 1fr; }
  .footer-grid { grid-template-columns: 1fr; gap: 24px; }
  .cta-card { padding: 40px 24px; }
  .cta-title { font-size: 28px; }
}
</style>
