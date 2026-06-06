<template>
  <div class="marketplace-page">
    <!-- 页面头部 -->
    <div class="page-hero">
      <h1>模型广场</h1>
      <p>浏览所有可用 AI 模型，查看定价与性能指标，选择最适合您的模型</p>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-input
        v-model="searchQuery"
        placeholder="搜索模型名称..."
        prefix-icon="Search"
        clearable
        class="search-input"
      />
      <div class="provider-filters">
        <button
          class="provider-btn"
          :class="{ active: activeProvider === '' }"
          @click="activeProvider = ''"
        >全部</button>
        <button
          v-for="p in providers"
          :key="p.key"
          class="provider-btn"
          :class="{ active: activeProvider === p.key }"
          @click="activeProvider = p.key"
        >{{ p.icon }} {{ p.label }}</button>
      </div>
    </div>

    <!-- 模型统计 -->
    <div class="model-stats">
      <div class="stats-bar">
        <span class="stats-item">
          <strong>{{ filteredModels.length }}</strong> 个模型可用
        </span>
        <span class="stats-divider">|</span>
        <span class="stats-item">
          <strong>{{ providers.length }}</strong> 个供应商
        </span>
      </div>
      <div class="view-toggle">
        <button :class="{ active: viewMode === 'grid' }" @click="viewMode = 'grid'">⊞ 网格</button>
        <button :class="{ active: viewMode === 'table' }" @click="viewMode = 'table'">☰ 表格</button>
      </div>
    </div>

    <!-- 网格视图 -->
    <div v-if="viewMode === 'grid'" class="models-grid">
      <div class="model-card" v-for="m in filteredModels" :key="m.id" @click="showDetail(m)">
        <div class="card-header">
          <span class="provider-badge" :style="{ background: getProviderColor(m.provider) }">{{ getProviderLabel(m.provider) }}</span>
          <span v-if="m.streamable" class="stream-badge">流式</span>
        </div>
        <h3 class="model-name">{{ m.name }}</h3>
        <code class="model-id">{{ m.model_id }}</code>
        <div class="model-specs">
          <div class="spec-row">
            <span class="spec-label">上下文</span>
            <span class="spec-value">{{ m.max_tokens ? (m.max_tokens > 1000 ? (m.max_tokens / 1000).toFixed(0) + 'K' : m.max_tokens) : '-' }}</span>
          </div>
          <div class="spec-row">
            <span class="spec-label">输入价格</span>
            <span class="spec-value price">¥{{ m.input_price || 0 }}/1K</span>
          </div>
          <div class="spec-row">
            <span class="spec-label">输出价格</span>
            <span class="spec-value price">¥{{ m.output_price || 0 }}/1K</span>
          </div>
          <div class="spec-row">
            <span class="spec-label">延迟</span>
            <span class="spec-value">{{ m.latency_ms ? m.latency_ms + 'ms' : '-' }}</span>
          </div>
        </div>
        <div class="model-tags" v-if="m.tags && m.tags.length > 0">
          <el-tag v-for="t in m.tags.slice(0, 3)" :key="t" size="small" type="info">{{ t }}</el-tag>
        </div>
      </div>
    </div>

    <!-- 表格视图 -->
    <div v-else class="models-table-wrap">
      <el-table :data="filteredModels" stripe style="width: 100%">
        <el-table-column prop="name" label="模型" min-width="160">
          <template #default="{ row }">
            <div>
              <strong>{{ row.name }}</strong>
              <div class="text-xs text-gray-400 font-mono">{{ row.model_id }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="provider" label="供应商" width="120">
          <template #default="{ row }">
            <span class="provider-badge-sm" :style="{ background: getProviderColor(row.provider) }">{{ getProviderLabel(row.provider) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="上下文" width="100" align="center">
          <template #default="{ row }">{{ row.max_tokens ? (row.max_tokens > 1000 ? (row.max_tokens / 1000).toFixed(0) + 'K' : row.max_tokens) : '-' }}</template>
        </el-table-column>
        <el-table-column label="输入 ¥/1K" width="110" align="right">
          <template #default="{ row }">{{ row.input_price || 0 }}</template>
        </el-table-column>
        <el-table-column label="输出 ¥/1K" width="110" align="right">
          <template #default="{ row }">{{ row.output_price || 0 }}</template>
        </el-table-column>
        <el-table-column label="延迟" width="90" align="center">
          <template #default="{ row }">{{ row.latency_ms ? row.latency_ms + 'ms' : '-' }}</template>
        </el-table-column>
        <el-table-column label="流式" width="70" align="center">
          <template #default="{ row }"><el-tag v-if="row.streamable" type="success" size="small">✓</el-tag></template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 模型详情抽屉 -->
    <el-drawer v-model="detailVisible" :title="detailModel?.name || '模型详情'" size="480px">
      <div v-if="detailModel" class="detail-content">
        <div class="detail-header">
          <span class="provider-badge" :style="{ background: getProviderColor(detailModel.provider) }">{{ getProviderLabel(detailModel.provider) }}</span>
          <code class="detail-model-id">{{ detailModel.model_id }}</code>
        </div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="模型名称">{{ detailModel.name }}</el-descriptions-item>
          <el-descriptions-item label="供应商">{{ getProviderLabel(detailModel.provider) }}</el-descriptions-item>
          <el-descriptions-item label="上下文窗口">{{ detailModel.max_tokens ? detailModel.max_tokens.toLocaleString() : '-' }} tokens</el-descriptions-item>
          <el-descriptions-item label="输入价格">¥{{ detailModel.input_price || 0 }} / 1K tokens</el-descriptions-item>
          <el-descriptions-item label="输出价格">¥{{ detailModel.output_price || 0 }} / 1K tokens</el-descriptions-item>
          <el-descriptions-item label="延迟">{{ detailModel.latency_ms ? detailModel.latency_ms + 'ms' : '-' }}</el-descriptions-item>
          <el-descriptions-item label="成功率">{{ detailModel.success_rate ? (detailModel.success_rate * 100).toFixed(1) + '%' : '-' }}</el-descriptions-item>
          <el-descriptions-item label="流式支持">{{ detailModel.streamable ? '✓ 支持' : '✗ 不支持' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detailModel.tags && detailModel.tags.length > 0" class="detail-tags">
          <span class="detail-tag-label">标签：</span>
          <el-tag v-for="t in detailModel.tags" :key="t" size="small" type="info" class="mr-1">{{ t }}</el-tag>
        </div>
        <div class="detail-usage">
          <h4>调用示例</h4>
          <pre class="usage-code">curl https://api.tokenhub.cc/v1/chat/completions \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "{{ detailModel.model_id }}",
    "messages": [{"role": "user", "content": "你好"}]
  }'</pre>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { publicApi, adminApi } from '@/api'

interface ModelItem {
  id: string
  name: string
  provider: string
  model_id: string
  input_price: number
  output_price: number
  currency: string
  max_tokens: number
  streamable: boolean
  latency_ms: number
  success_rate: number
  tags: string[]
  created_at: string
}

const models = ref<ModelItem[]>([])
const loading = ref(false)
const searchQuery = ref('')
const activeProvider = ref('')
const viewMode = ref<'grid' | 'table'>('grid')
const detailVisible = ref(false)
const detailModel = ref<ModelItem | null>(null)

const providers = [
  { key: 'openai', label: 'OpenAI', icon: '🟢' },
  { key: 'claude', label: 'Anthropic', icon: '🟤' },
  { key: 'gemini', label: 'Google', icon: '🔵' },
  { key: 'deepseek', label: 'DeepSeek', icon: '🐋' },
  { key: 'qwen', label: '通义千问', icon: '🟣' },
  { key: 'zhipu', label: '智谱', icon: '✨' },
  { key: 'doubao', label: '豆包', icon: '🌋' },
  { key: 'moonshot', label: 'Moonshot', icon: '🌙' },
  { key: 'wenxin', label: '文心', icon: '🔵' },
  { key: 'spark', label: '星火', icon: '⭐' },
]

const filteredModels = computed(() => {
  let result = models.value
  if (activeProvider.value) {
    result = result.filter(m => m.provider === activeProvider.value)
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(m =>
      m.name.toLowerCase().includes(q) ||
      m.model_id.toLowerCase().includes(q) ||
      m.provider.toLowerCase().includes(q)
    )
  }
  return result
})

function getProviderLabel(p: string): string {
  const map: Record<string, string> = {
    openai: 'OpenAI', claude: 'Anthropic', gemini: 'Google',
    deepseek: 'DeepSeek', qwen: '通义', zhipu: '智谱',
    doubao: '豆包', moonshot: 'Moonshot', wenxin: '文心',
    spark: '星火', azure: 'Azure', custom: '自定义',
  }
  return map[p] || p
}

function getProviderColor(p: string): string {
  const map: Record<string, string> = {
    openai: '#10a37f', claude: '#d97706', gemini: '#4285f4',
    deepseek: '#2563eb', qwen: '#7c3aed', zhipu: '#3b82f6',
    doubao: '#ef4444', moonshot: '#6366f1', wenxin: '#2563eb',
    spark: '#f59e0b', azure: '#0078d4', custom: '#6b7280',
  }
  return map[p] || '#6b7280'
}

function showDetail(m: ModelItem) {
  detailModel.value = m
  detailVisible.value = true
}

async function loadModels() {
  loading.value = true
  try {
    // 先尝试公开接口，失败则用管理接口
    try {
      const res: any = await publicApi.getModels(activeProvider.value || undefined)
      models.value = res.data || []
    } catch {
      const res: any = await adminApi.getModels()
      models.value = (res.models || []).filter((m: any) => m.enabled)
    }
  } catch (e) {
    console.error('加载模型失败', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadModels)
</script>

<style scoped>
.marketplace-page { padding: 0; }
.page-hero { margin-bottom: 24px; }
.page-hero h1 { font-size: 24px; font-weight: 800; margin: 0 0 8px; }
.page-hero p { color: #64748b; font-size: 14px; margin: 0; }

.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 20px;
}
.search-input { width: 280px; }
.provider-filters { display: flex; flex-wrap: wrap; gap: 6px; }
.provider-btn {
  background: #fff;
  border: 1px solid #e5e7eb;
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.provider-btn:hover { border-color: #818cf8; }
.provider-btn.active { background: #6366f1; color: #fff; border-color: #6366f1; }

.model-stats {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.stats-bar { font-size: 13px; color: #94a3b8; display: flex; align-items: center; gap: 6px; }
.stats-bar strong { color: #334155; }
.stats-divider { color: #e5e7eb; }
.view-toggle { display: flex; gap: 4px; }
.view-toggle button {
  background: #fff;
  border: 1px solid #e5e7eb;
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.view-toggle button.active { background: #6366f1; color: #fff; border-color: #6366f1; }

.models-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
.model-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
}
.model-card:hover { border-color: #818cf8; transform: translateY(-2px); box-shadow: 0 4px 12px rgba(99,102,241,0.1); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.provider-badge {
  font-size: 11px; font-weight: 600; color: #fff;
  padding: 2px 10px; border-radius: 100px;
}
.provider-badge-sm {
  font-size: 11px; font-weight: 600; color: #fff;
  padding: 1px 8px; border-radius: 100px;
}
.stream-badge { font-size: 11px; color: #22c55e; font-weight: 600; }
.model-name { font-size: 15px; font-weight: 700; margin: 0 0 2px; }
.model-id { font-size: 11px; color: #94a3b8; font-family: monospace; }
.model-specs { margin-top: 12px; }
.spec-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 12px; }
.spec-label { color: #94a3b8; }
.spec-value { font-weight: 600; color: #334155; }
.spec-value.price { color: #6366f1; }
.model-tags { margin-top: 10px; display: flex; gap: 4px; flex-wrap: wrap; }

.detail-content { padding: 0 4px; }
.detail-header { display: flex; align-items: center; gap: 10px; margin-bottom: 20px; }
.detail-model-id { font-size: 13px; color: #94a3b8; font-family: monospace; }
.detail-tags { margin-top: 16px; }
.detail-tag-label { font-size: 13px; color: #94a3b8; margin-right: 8px; }
.detail-usage { margin-top: 24px; }
.detail-usage h4 { font-size: 14px; font-weight: 600; margin-bottom: 8px; }
.usage-code {
  background: #1e1e2e;
  color: #cdd6f4;
  padding: 16px;
  border-radius: 8px;
  font-size: 12px;
  font-family: 'Fira Code', monospace;
  line-height: 1.6;
  overflow-x: auto;
  white-space: pre;
}
.mr-1 { margin-right: 4px; }
.text-xs { font-size: 11px; }
.text-gray-400 { color: #94a3b8; }
.font-mono { font-family: monospace; }
.models-table-wrap { background: #fff; border-radius: 12px; border: 1px solid #e5e7eb; padding: 16px; }
</style>
