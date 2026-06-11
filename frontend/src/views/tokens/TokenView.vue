<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">令牌管理</h1>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon class="mr-1"><Plus /></el-icon> 创建令牌
      </el-button>
    </div>

    <el-alert type="info" :closable="false" class="mb-4">
      令牌以 <code>sk-</code> 格式生成，可分发给终端用户调用 API。支持额度限制、模型白名单、IP 限制等安全控制。
    </el-alert>

    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="tokens" stripe v-loading="loading">
        <el-table-column prop="name" label="名称" width="150" />
        <el-table-column label="Key" width="220">
          <template #default="{ row }">
            <span class="font-mono text-xs">{{ row.key_preview || 'sk-••••••••' }}</span>
            <el-button size="small" text type="primary" @click="copyKey(row)" class="ml-1">复制</el-button>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="tokenStatusType(row.status)">{{ tokenStatusLabel[row.status] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="额度" width="160">
          <template #default="{ row }">
            <template v-if="row.quota_total < 0">
              <span class="text-green-600">无限</span>
            </template>
            <template v-else>
              <el-progress :percentage="row.quota_total > 0 ? Math.round(row.quota_used / row.quota_total * 100) : 0" :stroke-width="6" :color="quotaColor(row)" />
              <span class="text-xs text-gray-400">{{ (row.quota_used / 100).toFixed(2) }} / {{ (row.quota_total / 100).toFixed(2) }} 元</span>
            </template>
          </template>
        </el-table-column>
        <el-table-column label="模型限制" width="120">
          <template #default="{ row }">
            <span v-if="!row.models?.length" class="text-gray-400">全部</span>
            <el-tag v-else size="small">{{ row.models.length }} 个</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="IP限制" width="100">
          <template #default="{ row }">
            <span v-if="!row.allowed_ips?.length" class="text-gray-400">无限制</span>
            <el-tag v-else size="small">{{ row.allowed_ips.length }} 个</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="RPM/TPM" width="100" align="center">
          <template #default="{ row }">{{ row.rate_limit_rpm || '∞' }} / {{ row.rate_limit_tpm || '∞' }}</template>
        </el-table-column>
        <el-table-column label="分组" width="80">
          <template #default="{ row }">{{ row.group || '-' }}</template>
        </el-table-column>
        <el-table-column label="过期时间" width="160">
          <template #default="{ row }">
            <span v-if="!row.expires_at" class="text-gray-400">永不过期</span>
            <span v-else>{{ new Date(row.expires_at * 1000).toLocaleDateString() }}</span>
          </template>
        </el-table-column>
        <el-table-column label="请求数" width="80" align="center">
          <template #default="{ row }">{{ row.total_requests || 0 }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" text type="warning" @click="handleRevoke(row)" :disabled="row.status !== 'active'">吊销</el-button>
            <el-button size="small" text type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建/编辑令牌对话框 -->
    <el-dialog v-model="showDialog" :title="editingTokenId ? '编辑令牌' : '创建令牌'" width="600px">
      <el-form :model="formData" label-width="120px">
        <el-form-item label="令牌名称">
          <el-input v-model="formData.name" placeholder="如：生产环境令牌" />
        </el-form-item>
        <el-form-item label="总额度">
          <el-input-number v-model="formData.quota_display" :min="-1" :precision="2" />
          <span class="ml-2 text-xs text-gray-400">元（-1 为无限额度）</span>
        </el-form-item>
        <el-form-item label="允许模型">
          <el-select v-model="formData.models" multiple filterable allow-create placeholder="留空则允许所有模型" class="w-full">
            <el-option v-for="m in modelOptions" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="IP 白名单">
          <el-select v-model="formData.allowed_ips" multiple filterable allow-create placeholder="留空则不限制IP" class="w-full">
          </el-select>
          <p class="text-xs text-gray-400 mt-1">支持 CIDR 格式，如 192.168.1.0/24</p>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="RPM 限制">
              <el-input-number v-model="formData.rate_limit_rpm" :min="0" :max="100000" :step="10" />
              <span class="ml-1 text-xs text-gray-400">/分钟</span>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="TPM 限制">
              <el-input-number v-model="formData.rate_limit_tpm" :min="0" :max="10000000" :step="1000" />
              <span class="ml-1 text-xs text-gray-400">/分钟</span>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="令牌分组">
          <el-input v-model="formData.group" placeholder="不同分组可设不同倍率" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="formData.expires_at_date" type="datetime" placeholder="留空则永不过期" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="editingTokenId ? handleUpdate() : handleCreate()" :loading="creating">
          {{ editingTokenId ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 显示新创建的 Key -->
    <el-dialog v-model="showKeyDialog" title="令牌创建成功" width="500px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" class="mb-4">
        请立即复制令牌 Key！关闭后将无法再次查看。
      </el-alert>
      <div class="bg-gray-100 dark:bg-gray-900 rounded-lg p-4 font-mono text-sm break-all">
        {{ newTokenKey }}
      </div>
      <template #footer>
        <el-button type="primary" @click="copyNewKey">复制 Key</el-button>
        <el-button @click="showKeyDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi, userApi } from '@/api'

const loading = ref(false)
const creating = ref(false)
const showDialog = ref(false)
const showKeyDialog = ref(false)
const newTokenKey = ref('')
const tokens = ref<any[]>([])
const editingTokenId = ref('')
const modelOptions = ref<string[]>([])

const tokenStatusLabel: Record<string, string> = { active: '有效', revoked: '已吊销', expired: '已过期' }
const tokenStatusType = (s: string) => ({ active: 'success', revoked: 'danger', expired: 'info' }[s] || '')
const quotaColor = (row: any) => {
  if (row.quota_total < 0) return '#67c23a'
  const pct = row.quota_used / row.quota_total
  if (pct > 0.9) return '#f56c6c'
  if (pct > 0.7) return '#e6a23c'
  return '#67c23a'
}

const formData = reactive({
  name: '', quota_display: -1, models: [] as string[], allowed_ips: [] as string[],
  rate_limit_rpm: 0, rate_limit_tpm: 0, group: '', expires_at_date: null as any,
})

async function loadTokens() {
  loading.value = true
  try {
    const res: any = await adminApi.listTokens()
    tokens.value = res.tokens || res.data || []
  } catch (e) {
    console.error('Failed to load tokens', e)
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingTokenId.value = ''
  formData.name = ''
  formData.quota_display = -1
  formData.models = []
  formData.allowed_ips = []
  formData.rate_limit_rpm = 0
  formData.rate_limit_tpm = 0
  formData.group = ''
  formData.expires_at_date = null
  showDialog.value = true
}

function openEditDialog(row: any) {
  editingTokenId.value = row.id
  formData.name = row.name || ''
  formData.quota_display = row.quota_total < 0 ? -1 : row.quota_total / 100
  formData.models = row.models || []
  formData.allowed_ips = row.allowed_ips || []
  formData.rate_limit_rpm = row.rate_limit_rpm || 0
  formData.rate_limit_tpm = row.rate_limit_tpm || 0
  formData.group = row.group || ''
  formData.expires_at_date = row.expires_at ? new Date(row.expires_at * 1000) : null
  showDialog.value = true
}

async function handleCreate() {
  if (!formData.name) {
    ElMessage.warning('请输入令牌名称')
    return
  }
  creating.value = true
  try {
    const quota_total = formData.quota_display < 0 ? -1 : Math.round(formData.quota_display * 100)
    const expires_at = formData.expires_at_date ? Math.floor(new Date(formData.expires_at_date).getTime() / 1000) : 0
    const res: any = await adminApi.createToken({
      name: formData.name,
      quota_total,
      models: formData.models,
      allowed_ips: formData.allowed_ips,
      rate_limit_rpm: formData.rate_limit_rpm,
      rate_limit_tpm: formData.rate_limit_tpm,
      group: formData.group,
      expires_at,
    })
    const token = res.token || res.data || res
    if (token.key) {
      newTokenKey.value = token.key
      showKeyDialog.value = true
    }
    showDialog.value = false
    await loadTokens()
    ElMessage.success('令牌创建成功')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

async function handleRevoke(row: any) {
  try {
    await ElMessageBox.confirm(`确定吊销令牌 "${row.name}" 吗？吊销后该令牌将无法使用。`, '确认吊销', { type: 'warning' })
    await adminApi.revokeToken(row.id)
    ElMessage.success('令牌已吊销')
    await loadTokens()
  } catch (e) { /* cancelled */ }
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除令牌 "${row.name}" 吗？`, '确认删除', { type: 'warning' })
    await adminApi.deleteToken(row.id)
    ElMessage.success('令牌已删除')
    await loadTokens()
  } catch (e) { /* cancelled */ }
}

function copyKey(row: any) {
  if (row.key_preview) {
    navigator.clipboard.writeText(row.key_preview)
    ElMessage.success('已复制 Key 前缀')
  } else {
    ElMessage.warning('出于安全考虑，令牌 Key 仅在创建时显示一次')
  }
}

async function handleUpdate() {
  if (!editingTokenId.value) return
  creating.value = true
  try {
    const quota_total = formData.quota_display < 0 ? -1 : Math.round(formData.quota_display * 100)
    const expires_at = formData.expires_at_date ? Math.floor(new Date(formData.expires_at_date).getTime() / 1000) : 0
    await adminApi.updateToken(editingTokenId.value, {
      name: formData.name,
      quota_total,
      models: formData.models,
      allowed_ips: formData.allowed_ips,
      rate_limit_rpm: formData.rate_limit_rpm,
      rate_limit_tpm: formData.rate_limit_tpm,
      group: formData.group,
      expires_at,
    })
    showDialog.value = false
    await loadTokens()
    ElMessage.success('令牌已更新')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '更新失败')
  } finally {
    creating.value = false
  }
}

async function copyNewKey() {
  try {
    await navigator.clipboard.writeText(newTokenKey.value)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败，请手动复制')
  }
}

async function loadModelOptions() {
  try {
    const res: any = await userApi.getModels()
    const models = res.data || []
    modelOptions.value = models.map((m: any) => m.model_id || m.name).filter(Boolean)
  } catch { /* fallback */ }
  // 如果获取失败，提供默认列表
  if (modelOptions.value.length === 0) {
    modelOptions.value = ['gpt-4o', 'gpt-4o-mini', 'o3', 'o3-mini', 'claude-4-sonnet', 'claude-4-opus', 'gemini-2.5-pro', 'deepseek-r1', 'deepseek-v3', 'qwen-max', 'glm-4-plus']
  }
}

onMounted(() => {
  loadTokens()
  loadModelOptions()
})
</script>
