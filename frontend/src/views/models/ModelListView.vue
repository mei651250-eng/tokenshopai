<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('model.list') }}</h1>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon class="mr-1"><Plus /></el-icon> {{ t('model.create') }}
      </el-button>
    </div>

    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="models" stripe v-loading="loading">
        <el-table-column prop="name" :label="t('model.name')" width="160" />
        <el-table-column prop="provider" :label="t('model.provider')" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.provider }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="model_id" label="Model ID" width="160" />
        <el-table-column :label="t('model.inputPrice')" width="120">
          <template #default="{ row }">{{ row.input_price }}/{{ row.currency || 'CNY' }}/1K</template>
        </el-table-column>
        <el-table-column :label="t('model.outputPrice')" width="120">
          <template #default="{ row }">{{ row.output_price }}/{{ row.currency || 'CNY' }}/1K</template>
        </el-table-column>
        <el-table-column :label="t('model.latency')" width="100">
          <template #default="{ row }">{{ row.latency_ms }}ms</template>
        </el-table-column>
        <el-table-column :label="t('model.successRate')" width="100">
          <template #default="{ row }">{{ (row.success_rate * 100).toFixed(1) }}%</template>
        </el-table-column>
        <el-table-column :label="t('model.weight')" width="80">
          <template #default="{ row }">{{ row.weight }}</template>
        </el-table-column>
        <el-table-column :label="t('common.status')" width="90">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" @change="handleToggle(row)" />
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="150">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Create/Edit Dialog -->
    <el-dialog v-model="showDialog" :title="isEdit ? t('model.edit') : t('model.create')" width="600px">
      <el-form :model="formData" label-width="120px">
        <el-form-item :label="t('model.name')">
          <el-input v-model="formData.name" placeholder="e.g., gpt-4o" />
        </el-form-item>
        <el-form-item :label="t('model.provider')">
          <el-select v-model="formData.provider" class="w-full">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Azure OpenAI" value="azure" />
            <el-option label="Anthropic (Claude)" value="claude" />
            <el-option label="Google (Gemini)" value="gemini" />
            <el-option label="百度文心" value="wenxin" />
            <el-option label="阿里通义" value="qwen" />
            <el-option label="讯飞星火" value="spark" />
            <el-option label="字节豆包" value="doubao" />
            <el-option label="DeepSeek" value="deepseek" />
            <el-option label="月之暗面" value="moonshot" />
            <el-option label="智谱" value="zhipu" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="Model ID">
          <el-input v-model="formData.model_id" placeholder="e.g., gpt-4o (供应商侧模型ID)" />
        </el-form-item>
        <el-form-item :label="t('model.endpoint')">
          <el-input v-model="formData.endpoint" placeholder="https://api.openai.com" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="formData.api_key" type="password" show-password :placeholder="isEdit ? '留空则不修改' : '输入API Key'" />
        </el-form-item>
        <el-form-item :label="t('model.inputPrice')">
          <el-input-number v-model="formData.input_price" :min="0" :precision="4" />
        </el-form-item>
        <el-form-item :label="t('model.outputPrice')">
          <el-input-number v-model="formData.output_price" :min="0" :precision="4" />
        </el-form-item>
        <el-form-item label="Max Tokens">
          <el-input-number v-model="formData.max_tokens" :min="1" :max="1000000" />
        </el-form-item>
        <el-form-item :label="t('model.weight')">
          <el-slider v-model="formData.weight" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="formData.priority" :min="0" :max="100" />
        </el-form-item>
        <el-form-item label="支持流式">
          <el-switch v-model="formData.streamable" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi, type ModelConfig } from '@/api'

const { t } = useI18n()
const loading = ref(false)
const submitting = ref(false)
const showDialog = ref(false)
const isEdit = ref(false)
const editingId = ref('')
const models = ref<ModelConfig[]>([])

const defaultForm = (): Partial<ModelConfig> => ({
  name: '',
  provider: 'openai',
  model_id: '',
  endpoint: '',
  api_key: '',
  input_price: 0,
  output_price: 0,
  max_tokens: 4096,
  weight: 50,
  priority: 0,
  streamable: true,
  enabled: true,
})

const formData = reactive<Partial<ModelConfig>>(defaultForm())

async function loadModels() {
  loading.value = true
  try {
    const res: any = await adminApi.getModels()
    models.value = res.models || []
  } catch (e) {
    console.error('Failed to load models', e)
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  isEdit.value = false
  editingId.value = ''
  Object.assign(formData, defaultForm())
  showDialog.value = true
}

function openEditDialog(row: ModelConfig) {
  isEdit.value = true
  editingId.value = row.id
  Object.assign(formData, {
    name: row.name,
    provider: row.provider,
    model_id: row.model_id,
    endpoint: row.endpoint,
    api_key: '',
    input_price: row.input_price,
    output_price: row.output_price,
    max_tokens: row.max_tokens,
    weight: row.weight,
    priority: row.priority,
    streamable: row.streamable,
    enabled: row.enabled,
  })
  showDialog.value = true
}

async function handleSubmit() {
  if (!formData.name || !formData.provider || !formData.model_id || !formData.endpoint) {
    ElMessage.warning('请填写必填字段')
    return
  }

  submitting.value = true
  try {
    if (isEdit.value) {
      await adminApi.updateModel(editingId.value, formData)
      ElMessage.success('模型更新成功')
    } else {
      await adminApi.createModel(formData)
      ElMessage.success('模型创建成功')
    }
    showDialog.value = false
    await loadModels()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleToggle(row: ModelConfig) {
  try {
    await adminApi.toggleModel(row.id)
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error('操作失败')
  }
}

async function handleDelete(row: ModelConfig) {
  try {
    await ElMessageBox.confirm(`确定删除模型 "${row.name}" 吗？`, '确认删除', {
      type: 'warning',
    })
    await adminApi.deleteModel(row.id)
    ElMessage.success('模型已删除')
    await loadModels()
  } catch (e) {
    // cancelled or error
  }
}

onMounted(() => {
  loadModels()
})
</script>
