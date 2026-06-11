<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">渠道管理</h1>
      <div class="flex gap-2">
        <el-button @click="handleBatchTest" :loading="batchTesting">
          <el-icon class="mr-1"><Monitor /></el-icon> 批量测试
        </el-button>
        <el-button type="primary" @click="openCreateDialog">
          <el-icon class="mr-1"><Plus /></el-icon> 添加渠道
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-4 gap-4 mb-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">总渠道</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total || 0 }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-green-200 dark:border-green-800">
        <p class="text-sm text-green-600">正常</p>
        <p class="text-2xl font-bold text-green-600">{{ stats.active || 0 }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-red-200 dark:border-red-800">
        <p class="text-sm text-red-600">异常</p>
        <p class="text-2xl font-bold text-red-600">{{ stats.error || 0 }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">已禁用</p>
        <p class="text-2xl font-bold text-gray-400">{{ stats.disabled || 0 }}</p>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="flex gap-3 mb-4 flex-wrap">
      <el-select v-model="filterProvider" placeholder="供应商" clearable style="width: 160px" @change="loadChannels">
        <el-option v-for="(label, key) in providerLabels" :key="key" :label="label" :value="key" />
      </el-select>
      <el-input v-model="filterModel" placeholder="模型名称" clearable style="width: 180px" @clear="loadChannels" @keyup.enter="loadChannels" />
      <el-select v-model="filterStatus" placeholder="状态" clearable style="width: 120px" @change="loadChannels">
        <el-option label="正常" value="active" />
        <el-option label="异常" value="error" />
        <el-option label="已禁用" value="disabled" />
      </el-select>
      <el-button type="primary" @click="loadChannels" :icon="Search">搜索</el-button>
    </div>

    <!-- 渠道表格 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="channels" stripe v-loading="loading" row-key="id">
        <el-table-column prop="name" label="名称" width="140" show-overflow-tooltip />
        <el-table-column prop="provider" label="供应商" width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="providerTagType(row.provider)">{{ providerLabels[row.provider] || row.provider }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="model_name" label="对外模型名" width="160" show-overflow-tooltip />
        <el-table-column prop="model_id" label="供应商模型ID" width="180" show-overflow-tooltip />
        <el-table-column prop="endpoint" label="端点" width="200" show-overflow-tooltip />
        <el-table-column label="Key数量" width="80" align="center">
          <template #default="{ row }">{{ row.api_keys?.length || 1 }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ statusLabels[row.status] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="成功率" width="90" align="center">
          <template #default="{ row }">
            <span :class="row.success_rate >= 0.95 ? 'text-green-600' : row.success_rate >= 0.8 ? 'text-yellow-600' : 'text-red-600'">
              {{ (row.success_rate * 100).toFixed(1) }}%
            </span>
          </template>
        </el-table-column>
        <el-table-column label="延迟" width="80" align="center">
          <template #default="{ row }">{{ row.latency_ms }}ms</template>
        </el-table-column>
        <el-table-column label="倍率" width="70" align="center">
          <template #default="{ row }">{{ row.multiplier }}x</template>
        </el-table-column>
        <el-table-column label="权重" width="60" align="center">
          <template #default="{ row }">{{ row.weight }}</template>
        </el-table-column>
        <el-table-column label="启用" width="70" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" @change="handleToggle(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="handleTest(row)" :loading="row._testing">测试</el-button>
            <el-button size="small" text type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="showDialog" :title="isEdit ? '编辑渠道' : '添加渠道'" width="680px" top="5vh">
      <el-form :model="formData" label-width="120px">
        <el-form-item label="渠道名称">
          <el-input v-model="formData.name" placeholder="如：OpenAI主渠道" />
        </el-form-item>
        <el-form-item label="供应商">
          <el-select v-model="formData.provider" class="w-full" @change="onProviderChange" :disabled="isEdit">
            <el-option v-for="(label, key) in providerLabels" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="对外模型名">
          <el-input v-model="formData.model_name" placeholder="用户调用的统一模型名，如 gpt-4o" />
        </el-form-item>
        <el-form-item label="供应商模型ID">
          <el-input v-model="formData.model_id" placeholder="供应商侧的模型ID，通常与对外模型名相同" />
        </el-form-item>
        <el-form-item label="API 端点">
          <el-input v-model="formData.endpoint" placeholder="API端点URL" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="formData.api_key" type="textarea" :rows="2" :placeholder="isEdit ? '留空则不修改' : '输入API Key，支持多个Key用逗号分隔，自动轮换'" show-password />
          <p class="text-xs text-gray-400 mt-1">支持多个Key用英文逗号分隔，系统会自动轮换使用</p>
        </el-form-item>
        <el-form-item label="渠道分组">
          <el-input v-model="formData.group" placeholder="分组名，不同分组可设不同倍率" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="权重">
              <el-slider v-model="formData.weight" :min="1" :max="100" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="优先级">
              <el-input-number v-model="formData.priority" :min="0" :max="100" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="输入价格">
              <el-input-number v-model="formData.input_price" :min="0" :precision="4" />
              <span class="ml-1 text-xs text-gray-400">/1K Token</span>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="输出价格">
              <el-input-number v-model="formData.output_price" :min="0" :precision="4" />
              <span class="ml-1 text-xs text-gray-400">/1K Token</span>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="渠道倍率">
              <el-input-number v-model="formData.multiplier" :min="0.01" :max="100" :precision="2" :step="0.1" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max Tokens">
              <el-input-number v-model="formData.max_tokens" :min="1" :max="2000000" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="支持流式">
              <el-switch v-model="formData.streamable" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

    <!-- 批量测试结果 -->
    <el-dialog v-model="showTestDialog" title="批量测试结果" width="700px">
      <el-table :data="testResults" stripe>
        <el-table-column prop="channel_id" label="渠道ID" width="120" show-overflow-tooltip />
        <el-table-column label="结果" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="延迟" width="100" align="center">
          <template #default="{ row }">{{ row.latency_ms }}ms</template>
        </el-table-column>
        <el-table-column prop="error" label="错误信息" show-overflow-tooltip />
        <el-table-column label="可用模型" width="100" align="center">
          <template #default="{ row }">{{ row.model_list?.length || 0 }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus, Monitor, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api'

const loading = ref(false)
const submitting = ref(false)
const showDialog = ref(false)
const isEdit = ref(false)
const editingId = ref('')
const channels = ref<any[]>([])
const stats = ref<any>({})
const filterProvider = ref('')
const filterModel = ref('')
const filterStatus = ref('')
const batchTesting = ref(false)
const showTestDialog = ref(false)
const testResults = ref<any[]>([])

const providerLabels: Record<string, string> = {
  openai: 'OpenAI', azure: 'Azure OpenAI', claude: 'Anthropic (Claude)',
  gemini: 'Google (Gemini)', deepseek: 'DeepSeek', qwen: '阿里通义 (Qwen)',
  doubao: '字节豆包 (Doubao)', zhipu: '智谱 (GLM)', moonshot: '月之暗面 (Kimi)',
  wenxin: '百度文心 (ERNIE)', spark: '讯飞星火 (Spark)', minimax: 'MiniMax',
  baichuan: '百川 (Baichuan)', yi: '零一万物 (Yi)', stepfun: '阶跃星辰 (Step)',
  hunyuan: '腾讯混元', cohere: 'Cohere', mistral: 'Mistral AI',
  meta: 'Meta (Llama)', xai: 'xAI (Grok)', custom: '自定义',
}

const providerEndpoints: Record<string, string> = {
  openai: 'https://api.openai.com/v1', azure: 'https://{resource}.openai.azure.com/openai',
  claude: 'https://api.anthropic.com/v1', gemini: 'https://generativelanguage.googleapis.com/v1beta',
  deepseek: 'https://api.deepseek.com/v1', qwen: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  doubao: 'https://ark.cn-beijing.volces.com/api/v3', zhipu: 'https://open.bigmodel.cn/api/paas/v4',
  moonshot: 'https://api.moonshot.cn/v1', wenxin: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop',
  spark: 'https://spark-api-open.xf-yun.com/v1', minimax: 'https://api.minimax.chat/v1',
  baichuan: 'https://api.baichuan-ai.com/v1', yi: 'https://api.lingyiwanwu.com/v1',
  stepfun: 'https://api.stepfun.com/v1', hunyuan: 'https://hunyuan.tencentcloudapi.com',
  cohere: 'https://api.cohere.com/v2', mistral: 'https://api.mistral.ai/v1',
  xai: 'https://api.x.ai/v1', meta: 'https://api.together.xyz/v1',
}

const providerTagTypes: Record<string, string> = {
  openai: 'success', claude: 'warning', gemini: 'primary', deepseek: '', qwen: 'danger',
  custom: 'info',
}

const statusLabels: Record<string, string> = {
  active: '正常', error: '异常', disabled: '已禁用', testing: '测试中',
}

function providerTagType(p: string) { return providerTagTypes[p] || '' }
function statusTagType(s: string) {
  const m: Record<string, string> = { active: 'success', error: 'danger', disabled: 'info', testing: 'warning' }
  return m[s] || ''
}

const defaultForm = () => ({
  name: '', provider: 'openai', model_name: '', model_id: '', endpoint: 'https://api.openai.com/v1',
  api_key: '', group: '', weight: 50, priority: 0, max_tokens: 4096,
  input_price: 0, output_price: 0, currency: 'CNY', multiplier: 1.0,
  streamable: true, enabled: true,
})

const formData = reactive(defaultForm())

async function loadChannels() {
  loading.value = true
  try {
    const res: any = await adminApi.listChannels({ provider: filterProvider.value, model_name: filterModel.value, status: filterStatus.value })
    channels.value = res.channels || res.data || []
  } catch (e) {
    console.error('Failed to load channels', e)
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const res: any = await adminApi.getChannelStats()
    stats.value = res.stats || res.data || res
  } catch (e) {
    console.error('Failed to load stats', e)
  }
}

function onProviderChange(provider: string) {
  formData.endpoint = providerEndpoints[provider] || ''
}

function openCreateDialog() {
  isEdit.value = false
  editingId.value = ''
  Object.assign(formData, defaultForm())
  showDialog.value = true
}

function openEditDialog(row: any) {
  isEdit.value = true
  editingId.value = row.id
  Object.assign(formData, {
    name: row.name, provider: row.provider, model_name: row.model_name, model_id: row.model_id,
    endpoint: row.endpoint, api_key: '', group: row.group || '', weight: row.weight,
    priority: row.priority, max_tokens: row.max_tokens, input_price: row.input_price,
    output_price: row.output_price, currency: row.currency, multiplier: row.multiplier,
    streamable: row.streamable, enabled: row.enabled,
  })
  showDialog.value = true
}

async function handleSubmit() {
  if (!formData.name || !formData.provider || !formData.model_name || !formData.endpoint) {
    ElMessage.warning('请填写必填字段')
    return
  }
  submitting.value = true
  try {
    if (isEdit.value) {
      const updates: any = { ...formData }
      if (!updates.api_key) delete updates.api_key
      await adminApi.updateChannel(editingId.value, updates)
      ElMessage.success('渠道更新成功')
    } else {
      if (!formData.api_key) {
        ElMessage.warning('请输入API Key')
        submitting.value = false
        return
      }
      await adminApi.createChannel(formData)
      ElMessage.success('渠道创建成功')
    }
    showDialog.value = false
    await loadChannels()
    await loadStats()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleToggle(row: any) {
  try {
    await adminApi.toggleChannel(row.id)
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
    await loadStats()
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error('操作失败')
  }
}

async function handleTest(row: any) {
  row._testing = true
  try {
    const res: any = await adminApi.testChannel(row.id)
    const result = res.result || res.data || res
    if (result.success) {
      ElMessage.success(`测试成功，延迟 ${result.latency_ms}ms，可用模型 ${result.model_list?.length || 0} 个`)
    } else {
      ElMessage.error(`测试失败：${result.error}`)
    }
    await loadChannels()
    await loadStats()
  } catch (e: any) {
    ElMessage.error('测试请求失败')
  } finally {
    row._testing = false
  }
}

async function handleBatchTest() {
  batchTesting.value = true
  try {
    const res: any = await adminApi.batchTestChannels()
    testResults.value = res.results || res.data || []
    showTestDialog.value = true
    await loadChannels()
    await loadStats()
  } catch (e) {
    ElMessage.error('批量测试失败')
  } finally {
    batchTesting.value = false
  }
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`确定删除渠道 "${row.name}" 吗？`, '确认删除', { type: 'warning' })
    await adminApi.deleteChannel(row.id)
    ElMessage.success('渠道已删除')
    await loadChannels()
    await loadStats()
  } catch (e) {
    // cancelled
  }
}

onMounted(() => {
  loadChannels()
  loadStats()
})
</script>
