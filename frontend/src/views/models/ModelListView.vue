<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('model.list') }}</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon class="mr-1"><Plus /></el-icon> {{ t('model.create') }}
      </el-button>
    </div>

    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="models" stripe>
        <el-table-column prop="name" :label="t('model.name')" width="160" />
        <el-table-column prop="provider" :label="t('model.provider')" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.provider }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('model.inputPrice')" width="120">
          <template #default="{ row }">¥{{ row.input_price }}/1K</template>
        </el-table-column>
        <el-table-column :label="t('model.outputPrice')" width="120">
          <template #default="{ row }">¥{{ row.output_price }}/1K</template>
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
            <el-switch v-model="row.enabled" size="small" />
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="150">
          <template #default="{ row }">
            <el-button size="small" text type="primary">编辑</el-button>
            <el-button size="small" text type="danger">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Create Dialog -->
    <el-dialog v-model="showCreateDialog" :title="t('model.create')" width="600px">
      <el-form :model="newModel" label-width="100px">
        <el-form-item :label="t('model.name')">
          <el-input v-model="newModel.name" placeholder="e.g., gpt-4o" />
        </el-form-item>
        <el-form-item :label="t('model.provider')">
          <el-select v-model="newModel.provider" class="w-full">
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
        <el-form-item :label="t('model.endpoint')">
          <el-input v-model="newModel.endpoint" placeholder="https://api.openai.com" />
        </el-form-item>
        <el-form-item :label="t('model.inputPrice')">
          <el-input-number v-model="newModel.input_price" :min="0" :precision="4" />
        </el-form-item>
        <el-form-item :label="t('model.outputPrice')">
          <el-input-number v-model="newModel.output_price" :min="0" :precision="4" />
        </el-form-item>
        <el-form-item :label="t('model.weight')">
          <el-slider v-model="newModel.weight" :min="1" :max="100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus } from '@element-plus/icons-vue'

const { t } = useI18n()
const showCreateDialog = ref(false)

const models = ref([
  { name: 'gpt-4o', provider: 'OpenAI', input_price: 0.03, output_price: 0.06, latency_ms: 450, success_rate: 0.995, weight: 35, enabled: true },
  { name: 'gpt-4o-mini', provider: 'OpenAI', input_price: 0.00015, output_price: 0.0006, latency_ms: 320, success_rate: 0.998, weight: 25, enabled: true },
  { name: 'claude-3.5-sonnet', provider: 'Anthropic', input_price: 0.003, output_price: 0.015, latency_ms: 380, success_rate: 0.998, weight: 20, enabled: true },
  { name: 'qwen-max', provider: '阿里', input_price: 0.02, output_price: 0.06, latency_ms: 210, success_rate: 0.991, weight: 10, enabled: true },
  { name: 'deepseek-v2', provider: 'DeepSeek', input_price: 0.001, output_price: 0.002, latency_ms: 290, success_rate: 0.985, weight: 10, enabled: true },
])

const newModel = reactive({
  name: '',
  provider: 'openai',
  endpoint: '',
  input_price: 0,
  output_price: 0,
  weight: 50,
})

function handleCreate() {
  showCreateDialog.value = false
}
</script>
