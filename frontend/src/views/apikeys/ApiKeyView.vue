<template>
  <div class="apikey-page">
    <div class="page-header">
      <div>
        <h2>API 密钥管理</h2>
        <p class="text-gray-500 text-sm mt-1">创建和管理您的 API 密钥，用于调用 TokenHub 接口</p>
      </div>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon> 创建密钥
      </el-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stat-cards">
      <div class="stat-card">
        <div class="stat-icon active-icon">🔑</div>
        <div>
          <div class="stat-value">{{ keys.filter(k => k.status === 'active').length }}</div>
          <div class="stat-label">活跃密钥</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon revoked-icon">🚫</div>
        <div>
          <div class="stat-value">{{ keys.filter(k => k.status === 'revoked').length }}</div>
          <div class="stat-label">已吊销</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon total-icon">📊</div>
        <div>
          <div class="stat-value">{{ keys.length }}</div>
          <div class="stat-label">总计</div>
        </div>
      </div>
    </div>

    <!-- 密钥列表 -->
    <div class="key-list" v-loading="loading">
      <el-empty v-if="!loading && keys.length === 0" description="暂无 API 密钥，点击上方按钮创建">
        <el-button type="primary" @click="showCreateDialog = true">创建密钥</el-button>
      </el-empty>
      <div v-for="key in keys" :key="key.id" class="key-card" :class="{ revoked: key.status === 'revoked' }">
        <div class="key-main">
          <div class="key-info">
            <div class="key-name-row">
              <span class="key-name">{{ key.name }}</span>
              <el-tag :type="key.status === 'active' ? 'success' : 'danger'" size="small">
                {{ key.status === 'active' ? '活跃' : '已吊销' }}
              </el-tag>
            </div>
            <div class="key-meta">
              <span class="key-prefix">
                <el-icon><Key /></el-icon>
                {{ key.key_prefix }}••••••••••••••••••••••••••••••
              </span>
              <span class="meta-divider">|</span>
              <span>限速: {{ key.rate_limit }} RPM</span>
              <span class="meta-divider">|</span>
              <span>日配额: {{ formatQuota(key.quota_daily) }} tokens</span>
              <span v-if="key.expires_at" class="meta-divider">|</span>
              <span v-if="key.expires_at">过期: {{ formatDate(key.expires_at) }}</span>
            </div>
            <div class="key-meta" v-if="key.models && key.models.length > 0">
              <span>允许模型: </span>
              <el-tag v-for="m in key.models.slice(0, 5)" :key="m" size="small" type="info" class="model-tag">{{ m }}</el-tag>
              <el-tag v-if="key.models.length > 5" size="small" type="info">+{{ key.models.length - 5 }}</el-tag>
            </div>
          </div>
          <div class="key-actions">
            <el-button v-if="key.status === 'active'" type="warning" size="small" plain @click="handleRevoke(key)">
              吊销
            </el-button>
            <el-button type="danger" size="small" plain @click="handleDelete(key)">
              删除
            </el-button>
          </div>
        </div>
        <div class="key-footer">
          <span>创建于 {{ formatDate(key.created_at) }}</span>
          <span v-if="key.last_used_at"> · 最后使用 {{ formatDate(key.last_used_at) }}</span>
        </div>
      </div>
    </div>

    <!-- 创建密钥对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建 API 密钥" width="520px" :close-on-click-modal="false">
      <el-form :model="createForm" label-width="100px" label-position="top">
        <el-form-item label="密钥名称" required>
          <el-input v-model="createForm.name" placeholder="例如：生产环境密钥" />
        </el-form-item>
        <el-form-item label="速率限制 (RPM)">
          <el-input-number v-model="createForm.rate_limit" :min="1" :max="10000" :step="10" />
        </el-form-item>
        <el-form-item label="每日 Token 配额">
          <el-input-number v-model="createForm.quota_daily" :min="1000" :step="100000" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="createForm.expires_at" type="datetime" placeholder="留空表示永不过期" />
        </el-form-item>
        <el-form-item label="允许的模型">
          <el-select v-model="createForm.models" multiple filterable allow-create placeholder="留空表示允许所有模型">
            <el-option v-for="m in commonModels" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 新密钥展示对话框 -->
    <el-dialog v-model="showKeyDialog" title="API 密钥已创建" width="560px" :close-on-click-modal="false" :show-close="false">
      <el-alert type="warning" :closable="false" class="mb-4" show-icon>
        <template #title>
          <strong>请立即复制并妥善保存此密钥！</strong>
        </template>
        关闭此对话框后将无法再次查看完整密钥。
      </el-alert>
      <div class="new-key-display">
        <code class="key-value">{{ newKeyValue }}</code>
        <el-button type="primary" size="small" @click="copyKey">
          <el-icon><DocumentCopy /></el-icon> 复制
        </el-button>
      </div>
      <div class="key-usage mt-4">
        <p class="text-sm text-gray-500 font-medium mb-2">快速使用：</p>
        <pre class="usage-code">base_url = "https://api.tokenhub.cc/v1"
api_key  = "{{ newKeyValue }}"</pre>
      </div>
      <template #footer>
        <el-button type="primary" @click="showKeyDialog = false; newKeyValue = ''">我已保存密钥</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api'

interface ApiKeyItem {
  id: string
  name: string
  key_prefix: string
  permissions: string[]
  models: string[]
  rate_limit: number
  quota_daily: number
  status: string
  expires_at?: string
  created_at: string
  last_used_at?: string
}

const keys = ref<ApiKeyItem[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showKeyDialog = ref(false)
const creating = ref(false)
const newKeyValue = ref('')

const commonModels = [
  'gpt-4o', 'gpt-4o-mini', 'o3', 'o3-mini',
  'claude-4-sonnet', 'claude-4-opus',
  'gemini-2.5-pro', 'deepseek-r1', 'deepseek-v3',
  'qwen-max', 'glm-4-plus', 'doubao-pro-32k',
  'moonshot-v1-128k', 'text-embedding-3-large',
]

const createForm = reactive({
  name: '',
  rate_limit: 60,
  quota_daily: 1000000,
  expires_at: null as string | null,
  models: [] as string[],
})

function formatQuota(q: number): string {
  if (q >= 1000000) return (q / 1000000).toFixed(1) + 'M'
  if (q >= 1000) return (q / 1000).toFixed(0) + 'K'
  return String(q)
}

function formatDate(d: string): string {
  return new Date(d).toLocaleString('zh-CN')
}

async function loadKeys() {
  loading.value = true
  try {
    const res: any = await adminApi.getApiKeys()
    keys.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载密钥列表失败')
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.name.trim()) {
    ElMessage.warning('请输入密钥名称')
    return
  }
  creating.value = true
  try {
    const res: any = await adminApi.createApiKey({
      name: createForm.name,
      rate_limit: createForm.rate_limit,
      quota_daily: createForm.quota_daily,
      expires_at: createForm.expires_at || undefined,
      models: createForm.models.length > 0 ? createForm.models : undefined,
    })
    if (res.data?.key) {
      newKeyValue.value = res.data.key
      showCreateDialog.value = false
      showKeyDialog.value = true
      // Reset form
      createForm.name = ''
      createForm.rate_limit = 60
      createForm.quota_daily = 1000000
      createForm.expires_at = null
      createForm.models = []
      loadKeys()
    }
  } catch (e: any) {
    ElMessage.error(e.message || '创建密钥失败')
  } finally {
    creating.value = false
  }
}

async function handleRevoke(key: ApiKeyItem) {
  try {
    await ElMessageBox.confirm(`确定要吊销密钥「${key.name}」吗？吊销后使用此密钥的请求将被拒绝。`, '吊销密钥', { type: 'warning' })
    await adminApi.revokeApiKey(key.id)
    ElMessage.success('密钥已吊销')
    loadKeys()
  } catch {}
}

async function handleDelete(key: ApiKeyItem) {
  try {
    await ElMessageBox.confirm(`确定要删除密钥「${key.name}」吗？此操作不可恢复。`, '删除密钥', { type: 'error' })
    await adminApi.deleteApiKey(key.id)
    ElMessage.success('密钥已删除')
    loadKeys()
  } catch {}
}

function copyKey() {
  navigator.clipboard.writeText(newKeyValue.value)
  ElMessage.success('已复制到剪贴板')
}

onMounted(loadKeys)
</script>

<style scoped>
.apikey-page { padding: 0; }
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}
.page-header h2 { font-size: 20px; font-weight: 700; margin: 0; }
.stat-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
}
.stat-icon { font-size: 28px; width: 44px; height: 44px; display: flex; align-items: center; justify-content: center; border-radius: 10px; }
.active-icon { background: #d1fae5; }
.revoked-icon { background: #fee2e2; }
.total-icon { background: #dbeafe; }
.stat-value { font-size: 24px; font-weight: 800; }
.stat-label { font-size: 13px; color: #94a3b8; }
.key-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 12px;
  transition: all 0.2s;
}
.key-card:hover { border-color: #818cf8; }
.key-card.revoked { opacity: 0.65; }
.key-main { display: flex; justify-content: space-between; align-items: flex-start; }
.key-name-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.key-name { font-size: 15px; font-weight: 600; }
.key-prefix { font-family: monospace; font-size: 13px; color: #64748b; }
.key-meta { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; font-size: 12px; color: #94a3b8; margin-bottom: 4px; }
.meta-divider { color: #e5e7eb; margin: 0 2px; }
.model-tag { margin: 2px; }
.key-actions { display: flex; gap: 8px; flex-shrink: 0; }
.key-footer { margin-top: 12px; padding-top: 12px; border-top: 1px solid #f1f5f9; font-size: 12px; color: #94a3b8; }
.new-key-display {
  background: #1e1e2e;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.key-value {
  flex: 1;
  color: #a6e3a1;
  font-family: 'Fira Code', monospace;
  font-size: 14px;
  word-break: break-all;
}
.usage-code {
  background: #f8fafc;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  color: #334155;
  line-height: 1.6;
}
.mb-4 { margin-bottom: 16px; }
.mt-4 { margin-top: 16px; }
</style>
