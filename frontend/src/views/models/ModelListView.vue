<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('model.list') }}</h1>
      <div class="flex gap-2">
        <el-button type="success" @click="showBatchDialog = true">
          <el-icon class="mr-1"><Download /></el-icon> 厂商一键导入
        </el-button>
        <el-button type="primary" @click="openCreateDialog">
          <el-icon class="mr-1"><Plus /></el-icon> {{ t('model.create') }}
        </el-button>
      </div>
    </div>

    <!-- 厂商分组 Tabs -->
    <el-tabs v-model="activeProvider" type="border-card" class="mb-4" @tab-change="onProviderTabChange">
      <el-tab-pane label="全部" name="all">
        <template #label>
          <span>全部 <el-badge :value="models.length" type="info" class="ml-1" /></span>
        </template>
      </el-tab-pane>
      <el-tab-pane
        v-for="prov in usedProviders"
        :key="prov"
        :label="providerLabels[prov] || prov"
        :name="prov"
      >
        <template #label>
          <span>{{ providerLabels[prov] || prov }} <el-badge :value="providerModelCount(prov)" type="info" class="ml-1" /></span>
        </template>
      </el-tab-pane>
    </el-tabs>

    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="filteredModels" stripe v-loading="loading">
        <el-table-column prop="name" :label="t('model.name')" width="160" />
        <el-table-column prop="provider" :label="t('model.provider')" width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="providerTagType(row.provider)">{{ providerLabels[row.provider] || row.provider }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="model_id" label="Model ID" width="200" />
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
    <el-dialog v-model="showDialog" :title="isEdit ? t('model.edit') : t('model.create')" width="620px">
      <el-form :model="formData" label-width="120px">
        <el-form-item :label="t('model.provider')">
          <el-select v-model="formData.provider" class="w-full" @change="onProviderChange" :disabled="isEdit">
            <el-option v-for="(label, key) in providerLabels" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型型号">
          <div class="flex gap-2 w-full">
            <el-select
              v-model="formData.model_id"
              class="flex-1"
              filterable
              allow-create
              :disabled="isEdit"
              placeholder="选择或输入模型ID"
              @change="onModelSelect"
            >
              <el-option-group v-for="group in currentModelGroups" :key="group.label" :label="group.label">
                <el-option v-for="m in group.models" :key="m.id" :label="m.id + (m.note ? ' (' + m.note + ')' : '')" :value="m.id" />
              </el-option-group>
            </el-select>
            <el-button type="success" :loading="discovering" :disabled="isEdit || !formData.endpoint" @click="discoverModels">
              {{ discovering ? '发现中...' : '自动发现' }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item :label="t('model.name')">
          <el-input v-model="formData.name" placeholder="显示名称，默认与模型ID相同" />
        </el-form-item>
        <el-form-item :label="t('model.endpoint')">
          <el-input v-model="formData.endpoint" placeholder="API端点URL" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="formData.api_key" type="password" show-password :placeholder="isEdit ? '留空则不修改' : '输入API Key后可自动发现模型'" @change="onApiKeyChange" />
        </el-form-item>
        <el-form-item :label="t('model.inputPrice')">
          <el-input-number v-model="formData.input_price" :min="0" :precision="4" />
          <span class="ml-2 text-gray-400 text-xs">每1K Token</span>
        </el-form-item>
        <el-form-item :label="t('model.outputPrice')">
          <el-input-number v-model="formData.output_price" :min="0" :precision="4" />
          <span class="ml-2 text-gray-400 text-xs">每1K Token</span>
        </el-form-item>
        <el-form-item label="Max Tokens">
          <el-input-number v-model="formData.max_tokens" :min="1" :max="2000000" />
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

    <!-- 厂商一键导入 Dialog -->
    <el-dialog v-model="showBatchDialog" title="厂商一键导入模型" width="700px" top="5vh">
      <el-alert type="info" :closable="false" class="mb-4">
        选择厂商后，可批量导入该厂商的所有模型。已存在的模型会自动跳过。
      </el-alert>

      <el-form label-width="100px">
        <el-form-item label="选择厂商">
          <el-select v-model="batchProvider" class="w-full" @change="onBatchProviderChange" placeholder="选择要导入的厂商">
            <el-option v-for="(label, key) in providerLabels" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="batchProvider" label="API 端点">
          <el-input v-model="batchEndpoint" placeholder="API端点URL" />
        </el-form-item>
        <el-form-item v-if="batchProvider" label="API Key">
          <el-input v-model="batchApiKey" type="password" show-password placeholder="输入API Key" />
        </el-form-item>
      </el-form>

      <!-- 厂商模型预览 -->
      <div v-if="batchProvider && batchModelList.length > 0" class="mt-4">
        <div class="flex items-center justify-between mb-2">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ providerLabels[batchProvider] }} 可用模型 ({{ batchModelList.length }}个)
          </h3>
          <div class="flex gap-2">
            <el-button size="small" @click="batchSelectAll">全选</el-button>
            <el-button size="small" @click="batchDeselectAll">全不选</el-button>
            <el-button size="small" @click="batchSelectNew">仅选新增</el-button>
          </div>
        </div>
        <div class="max-h-64 overflow-y-auto border rounded-lg p-2 dark:border-gray-600">
          <div v-for="group in batchModelGroups" :key="group.label" class="mb-3">
            <div class="text-xs font-semibold text-gray-500 mb-1 px-1">{{ group.label }}</div>
            <el-checkbox-group v-model="batchSelected">
              <div class="grid grid-cols-1 gap-1">
                <el-checkbox
                  v-for="m in group.models"
                  :key="m.id"
                  :label="m.id"
                  :value="m.id"
                  :disabled="isModelExist(m.id)"
                  class="mx-1"
                >
                  <span>{{ m.id }}</span>
                  <span v-if="m.note" class="text-gray-400 text-xs ml-1">({{ m.note }})</span>
                  <el-tag v-if="isModelExist(m.id)" size="small" type="info" class="ml-1">已存在</el-tag>
                </el-checkbox>
              </div>
            </el-checkbox-group>
          </div>
        </div>
      </div>

      <!-- 导入进度 -->
      <div v-if="batchImporting" class="mt-4">
        <el-progress :percentage="batchProgress" :format="() => `${batchDone}/${batchSelected.length}`" />
        <p class="text-xs text-gray-500 mt-1">{{ batchStatusText }}</p>
      </div>

      <template #footer>
        <el-button @click="showBatchDialog = false" :disabled="batchImporting">取消</el-button>
        <el-button
          type="primary"
          @click="handleBatchImport"
          :loading="batchImporting"
          :disabled="batchSelected.length === 0 || !batchApiKey"
        >
          导入 {{ batchSelected.length > 0 ? `(${batchSelected.length}个模型)` : '' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Download } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi, type ModelConfig } from '@/api'

const { t } = useI18n()
const loading = ref(false)
const submitting = ref(false)
const showDialog = ref(false)
const isEdit = ref(false)
const editingId = ref('')
const models = ref<ModelConfig[]>([])
const activeProvider = ref('all')

// ==================== 供应商与模型数据 ====================

interface ModelItem {
  id: string
  note?: string
  input_price?: number
  output_price?: number
  max_tokens?: number
  streamable?: boolean
}

interface ModelGroup {
  label: string
  models: ModelItem[]
}

interface ProviderConfig {
  label: string
  endpoint: string
  models: ModelGroup[]
}

const providerLabels: Record<string, string> = {
  openai: 'OpenAI',
  azure: 'Azure OpenAI',
  claude: 'Anthropic (Claude)',
  gemini: 'Google (Gemini)',
  deepseek: 'DeepSeek',
  qwen: '阿里通义 (Qwen)',
  doubao: '字节豆包 (Doubao)',
  zhipu: '智谱 (GLM)',
  moonshot: '月之暗面 (Kimi)',
  wenxin: '百度文心 (ERNIE)',
  spark: '讯飞星火 (Spark)',
  minimax: 'MiniMax',
  baichuan: '百川 (Baichuan)',
  yi: '零一万物 (Yi)',
  stepfun: '阶跃星辰 (Step)',
  hunyuan: '腾讯混元',
  cohere: 'Cohere',
  mistral: 'Mistral AI',
  meta: 'Meta (Llama)',
  xai: 'xAI (Grok)',
  custom: '自定义',
}

const providerTagTypes: Record<string, string> = {
  openai: 'success',
  azure: '',
  claude: 'warning',
  gemini: 'primary',
  deepseek: '',
  qwen: 'danger',
  doubao: '',
  zhipu: 'success',
  moonshot: '',
  wenxin: 'warning',
  spark: '',
  custom: 'info',
}

function providerTagType(provider: string): string {
  return providerTagTypes[provider] || ''
}

const providerConfigs: Record<string, ProviderConfig> = {
  openai: {
    label: 'OpenAI',
    endpoint: 'https://api.openai.com/v1',
    models: [
      { label: 'GPT-4o 系列', models: [
        { id: 'gpt-4o', note: '最新旗舰', input_price: 0.0025, output_price: 0.01, max_tokens: 16384 },
        { id: 'gpt-4o-mini', note: '高性价比', input_price: 0.00015, output_price: 0.0006, max_tokens: 16384 },
        { id: 'gpt-4o-audio-preview', note: '音频', input_price: 0.0025, output_price: 0.01 },
        { id: 'chatgpt-4o-latest', note: '最新动态版', input_price: 0.005, output_price: 0.015 },
      ]},
      { label: 'GPT-4 系列', models: [
        { id: 'gpt-4-turbo', note: '128K', input_price: 0.01, output_price: 0.03, max_tokens: 128000 },
        { id: 'gpt-4', note: '8K', input_price: 0.03, output_price: 0.06, max_tokens: 8192 },
        { id: 'gpt-4-32k', note: '32K', input_price: 0.06, output_price: 0.12, max_tokens: 32768 },
      ]},
      { label: 'GPT-3.5 系列', models: [
        { id: 'gpt-3.5-turbo', note: '经典', input_price: 0.0005, output_price: 0.0015, max_tokens: 16384 },
        { id: 'gpt-3.5-turbo-16k', input_price: 0.003, output_price: 0.004, max_tokens: 16384 },
      ]},
      { label: 'o 系列 (推理)', models: [
        { id: 'o3', note: '最新推理', input_price: 0.01, output_price: 0.04 },
        { id: 'o3-mini', note: '推理轻量', input_price: 0.0011, output_price: 0.0044 },
        { id: 'o4-mini', note: '推理最新', input_price: 0.0011, output_price: 0.0044 },
        { id: 'o1', note: '推理旗舰', input_price: 0.015, output_price: 0.06 },
        { id: 'o1-mini', input_price: 0.003, output_price: 0.012 },
      ]},
      { label: 'Embedding', models: [
        { id: 'text-embedding-3-large', note: '3072维', input_price: 0.00013, output_price: 0 },
        { id: 'text-embedding-3-small', note: '1536维', input_price: 0.00002, output_price: 0 },
        { id: 'text-embedding-ada-002', note: '经典', input_price: 0.0001, output_price: 0 },
      ]},
      { label: 'TTS/Whisper', models: [
        { id: 'tts-1', note: '文本转语音' },
        { id: 'tts-1-hd', note: '高清语音' },
        { id: 'whisper-1', note: '语音识别' },
      ]},
      { label: 'DALL·E', models: [
        { id: 'dall-e-3', note: '图像生成' },
        { id: 'gpt-image-1', note: '最新图像' },
      ]},
    ],
  },
  azure: {
    label: 'Azure OpenAI',
    endpoint: 'https://{resource}.openai.azure.com/openai',
    models: [
      { label: 'GPT-4o', models: [
        { id: 'gpt-4o', input_price: 0.0025, output_price: 0.01 },
        { id: 'gpt-4o-mini', input_price: 0.00015, output_price: 0.0006 },
      ]},
      { label: 'GPT-4', models: [
        { id: 'gpt-4-turbo', input_price: 0.01, output_price: 0.03 },
        { id: 'gpt-4', input_price: 0.03, output_price: 0.06 },
      ]},
      { label: 'GPT-3.5', models: [
        { id: 'gpt-35-turbo', input_price: 0.0005, output_price: 0.0015 },
        { id: 'gpt-35-turbo-16k', input_price: 0.003, output_price: 0.004 },
      ]},
      { label: 'o 系列', models: [
        { id: 'o1', input_price: 0.015, output_price: 0.06 },
        { id: 'o1-mini', input_price: 0.003, output_price: 0.012 },
      ]},
      { label: 'Embedding', models: [
        { id: 'text-embedding-3-large', input_price: 0.00013, output_price: 0 },
        { id: 'text-embedding-ada-002', input_price: 0.0001, output_price: 0 },
      ]},
      { label: 'DALL·E', models: [
        { id: 'dall-e-3' },
      ]},
    ],
  },
  claude: {
    label: 'Anthropic (Claude)',
    endpoint: 'https://api.anthropic.com/v1',
    models: [
      { label: 'Claude 4 系列 (最新)', models: [
        { id: 'claude-sonnet-4-20250514', note: 'Sonnet 4', input_price: 0.003, output_price: 0.015, max_tokens: 64000 },
        { id: 'claude-opus-4-20250514', note: 'Opus 4 旗舰', input_price: 0.015, output_price: 0.075, max_tokens: 32000 },
      ]},
      { label: 'Claude 3.5 系列', models: [
        { id: 'claude-3-5-sonnet-20241022', note: 'Sonnet v2', input_price: 0.003, output_price: 0.015, max_tokens: 8192 },
        { id: 'claude-3-5-haiku-20241022', note: 'Haiku 轻量', input_price: 0.0008, output_price: 0.004, max_tokens: 8192 },
      ]},
      { label: 'Claude 3 系列', models: [
        { id: 'claude-3-opus-20240229', note: 'Opus 旗舰', input_price: 0.015, output_price: 0.075, max_tokens: 4096 },
        { id: 'claude-3-sonnet-20240229', note: 'Sonnet 均衡', input_price: 0.003, output_price: 0.015, max_tokens: 4096 },
        { id: 'claude-3-haiku-20240307', note: 'Haiku 极速', input_price: 0.00025, output_price: 0.00125, max_tokens: 4096 },
      ]},
    ],
  },
  gemini: {
    label: 'Google (Gemini)',
    endpoint: 'https://generativelanguage.googleapis.com/v1beta',
    models: [
      { label: 'Gemini 2.5 系列 (最新)', models: [
        { id: 'gemini-2.5-pro-preview-06-05', note: 'Pro 最新', max_tokens: 1048576 },
        { id: 'gemini-2.5-flash-preview-05-20', note: 'Flash 最新', max_tokens: 1048576 },
      ]},
      { label: 'Gemini 2.0 系列', models: [
        { id: 'gemini-2.0-flash', note: 'Flash', input_price: 0.0001, output_price: 0.0004 },
        { id: 'gemini-2.0-flash-lite', note: 'Flash Lite', input_price: 0.000075, output_price: 0.0003 },
      ]},
      { label: 'Gemini 1.5 系列', models: [
        { id: 'gemini-1.5-pro', note: 'Pro 128K', input_price: 0.00125, output_price: 0.005 },
        { id: 'gemini-1.5-flash', note: 'Flash 轻量', input_price: 0.000075, output_price: 0.0003 },
        { id: 'gemini-1.5-flash-8b', note: '8B 极速' },
      ]},
      { label: 'Embedding', models: [
        { id: 'text-embedding-004' },
        { id: 'embedding-001' },
      ]},
      { label: '图像生成', models: [
        { id: 'imagen-3.0-generate-002', note: 'Imagen 3' },
      ]},
    ],
  },
  deepseek: {
    label: 'DeepSeek',
    endpoint: 'https://api.deepseek.com/v1',
    models: [
      { label: 'DeepSeek-V3 系列', models: [
        { id: 'deepseek-chat', note: 'V3 通用对话', input_price: 0.00027, output_price: 0.0011, max_tokens: 65536 },
        { id: 'deepseek-reasoner', note: 'R1 推理', input_price: 0.00055, output_price: 0.00219, max_tokens: 65536 },
      ]},
      { label: 'DeepSeek-V2 系列', models: [
        { id: 'deepseek-v2', note: 'V2', input_price: 0.00014, output_price: 0.00028 },
        { id: 'deepseek-v2.5', note: 'V2.5', input_price: 0.00014, output_price: 0.00028 },
      ]},
      { label: 'Coder 系列', models: [
        { id: 'deepseek-coder', note: 'V2 Coder', input_price: 0.00014, output_price: 0.00028 },
      ]},
    ],
  },
  qwen: {
    label: '阿里通义 (Qwen)',
    endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    models: [
      { label: 'Qwen3 系列 (最新)', models: [
        { id: 'qwen3-235b-a22b', note: 'MoE 235B', max_tokens: 32768 },
        { id: 'qwen3-32b', note: 'Dense 32B', max_tokens: 32768 },
        { id: 'qwen3-30b-a3b', note: 'MoE 30B', max_tokens: 32768 },
        { id: 'qwen3-14b', max_tokens: 32768 },
        { id: 'qwen3-8b', max_tokens: 32768 },
        { id: 'qwen3-4b', max_tokens: 32768 },
        { id: 'qwen3-1.7b', max_tokens: 32768 },
        { id: 'qwen3-0.6b', max_tokens: 32768 },
      ]},
      { label: 'Qwen2.5 系列', models: [
        { id: 'qwen-max', note: '最强', input_price: 0.004, output_price: 0.012, max_tokens: 32768 },
        { id: 'qwen-max-latest', note: '最新版', input_price: 0.004, output_price: 0.012 },
        { id: 'qwen-plus', note: '均衡', input_price: 0.0008, output_price: 0.002, max_tokens: 131072 },
        { id: 'qwen-plus-latest', input_price: 0.0008, output_price: 0.002 },
        { id: 'qwen-turbo', note: '极速', input_price: 0.0003, output_price: 0.0006, max_tokens: 131072 },
        { id: 'qwen-turbo-latest', input_price: 0.0003, output_price: 0.0006 },
        { id: 'qwen-long', note: '长文本1M', input_price: 0.0005, output_price: 0.002 },
      ]},
      { label: 'Qwen2.5 开源', models: [
        { id: 'qwen2.5-72b-instruct' },
        { id: 'qwen2.5-32b-instruct' },
        { id: 'qwen2.5-14b-instruct' },
        { id: 'qwen2.5-7b-instruct' },
        { id: 'qwen2.5-3b-instruct' },
        { id: 'qwen2.5-1.5b-instruct' },
        { id: 'qwen2.5-0.5b-instruct' },
      ]},
      { label: 'QVQ/QwQ 推理', models: [
        { id: 'qvq-max', note: '视觉推理' },
        { id: 'qwq-32b', note: '文本推理' },
      ]},
      { label: 'Embedding', models: [
        { id: 'text-embedding-v3' },
        { id: 'text-embedding-v2' },
      ]},
    ],
  },
  doubao: {
    label: '字节豆包 (Doubao)',
    endpoint: 'https://ark.cn-beijing.volces.com/api/v3',
    models: [
      { label: 'Doubao 系列', models: [
        { id: 'doubao-1.5-pro-256k', note: 'Pro 256K', input_price: 0.0005, output_price: 0.001, max_tokens: 256000 },
        { id: 'doubao-1.5-pro-32k', note: 'Pro 32K', input_price: 0.0005, output_price: 0.001 },
        { id: 'doubao-1.5-lite-32k', note: 'Lite 轻量', input_price: 0.00003, output_price: 0.00006 },
        { id: 'doubao-1.5-lite-128k', note: 'Lite 128K', input_price: 0.00003, output_price: 0.00006 },
      ]},
      { label: 'Doubao-2 系列', models: [
        { id: 'doubao-2-pro-256k', note: 'Pro 2代 256K' },
        { id: 'doubao-2-pro-128k', note: 'Pro 2代 128K' },
        { id: 'doubao-2-lite-32k', note: 'Lite 2代' },
      ]},
      { label: 'Seed 系列 (推理)', models: [
        { id: 'seed-1.6-flash', note: 'Flash' },
        { id: 'seed-1.6-thinking', note: '思考' },
      ]},
    ],
  },
  zhipu: {
    label: '智谱 (GLM)',
    endpoint: 'https://open.bigmodel.cn/api/paas/v4',
    models: [
      { label: 'GLM-4 系列', models: [
        { id: 'glm-4-plus', note: 'Plus 旗舰', input_price: 0.05, output_price: 0.05, max_tokens: 128000 },
        { id: 'glm-4-0520', note: '最新', input_price: 0.1, output_price: 0.1 },
        { id: 'glm-4-air', note: 'Air 均衡', input_price: 0.001, output_price: 0.001, max_tokens: 128000 },
        { id: 'glm-4-airx', note: 'AirX 极速', input_price: 0.001, output_price: 0.001 },
        { id: 'glm-4-flash', note: 'Flash 免费', input_price: 0, output_price: 0, max_tokens: 128000 },
        { id: 'glm-4-flashx', note: 'FlashX', input_price: 0, output_price: 0 },
        { id: 'glm-4-long', note: '长文本1M', input_price: 0.001, output_price: 0.001 },
      ]},
      { label: 'GLM-Z1 推理', models: [
        { id: 'glm-z1-air', note: 'Air 推理' },
        { id: 'glm-z1-airx', note: 'AirX 推理' },
        { id: 'glm-z1-flash', note: 'Flash 推理' },
      ]},
      { label: 'CogView 图像', models: [
        { id: 'cogview-4', note: '图像生成' },
      ]},
      { label: 'Embedding', models: [
        { id: 'embedding-3' },
        { id: 'embedding-2' },
      ]},
    ],
  },
  moonshot: {
    label: '月之暗面 (Kimi)',
    endpoint: 'https://api.moonshot.cn/v1',
    models: [
      { label: 'Kimi 系列', models: [
        { id: 'moonshot-v1-128k', note: '128K 长文本', input_price: 0.014, output_price: 0.014, max_tokens: 128000 },
        { id: 'moonshot-v1-32k', note: '32K', input_price: 0.012, output_price: 0.012, max_tokens: 32000 },
        { id: 'moonshot-v1-8k', note: '8K', input_price: 0.012, output_price: 0.012, max_tokens: 8192 },
        { id: 'kimi-latest', note: '最新版' },
      ]},
    ],
  },
  wenxin: {
    label: '百度文心 (ERNIE)',
    endpoint: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop',
    models: [
      { label: 'ERNIE 4.0 系列', models: [
        { id: 'ernie-4.0-8k', note: '旗舰 8K', input_price: 0.12, output_price: 0.12 },
        { id: 'ernie-4.0-8k-latest', note: '最新版' },
        { id: 'ernie-4.0-turbo-8k', note: 'Turbo' },
      ]},
      { label: 'ERNIE 3.5 系列', models: [
        { id: 'ernie-3.5-8k', note: '3.5 8K', input_price: 0.012, output_price: 0.012 },
        { id: 'ernie-3.5-128k', note: '3.5 128K', input_price: 0.012, output_price: 0.012 },
      ]},
      { label: 'ERNIE Speed', models: [
        { id: 'ernie-speed-128k', note: 'Speed 128K' },
        { id: 'ernie-speed-8k', note: 'Speed 8K' },
      ]},
      { label: 'ERNIE Lite', models: [
        { id: 'ernie-lite-8k', note: 'Lite 8K' },
        { id: 'ernie-lite-pro-128k', note: 'Lite Pro 128K' },
      ]},
      { label: 'ERNIE Character', models: [
        { id: 'ernie-char-8k', note: '角色扮演' },
      ]},
      { label: 'Embedding', models: [
        { id: 'bge-large-zh', note: 'BGE 中文' },
        { id: 'bge-large-en', note: 'BGE 英文' },
      ]},
    ],
  },
  spark: {
    label: '讯飞星火 (Spark)',
    endpoint: 'https://spark-api-open.xf-yun.com/v1',
    models: [
      { label: 'Spark Max 系列', models: [
        { id: 'generalv3.5', note: 'Max 最新', input_price: 0.004, output_price: 0.004 },
        { id: '4.0Ultra', note: 'Ultra 旗舰', input_price: 0.036, output_price: 0.036 },
      ]},
      { label: 'Spark Pro', models: [
        { id: 'generalv3', note: 'Pro', input_price: 0.004, output_price: 0.004 },
      ]},
      { label: 'Spark Lite', models: [
        { id: 'generalv2', note: 'Lite 免费', input_price: 0, output_price: 0 },
      ]},
    ],
  },
  minimax: {
    label: 'MiniMax',
    endpoint: 'https://api.minimax.chat/v1',
    models: [
      { label: 'MiniMax 系列', models: [
        { id: 'MiniMax-Text-01', note: '文本旗舰', max_tokens: 1048576 },
        { id: 'abab6.5s-chat', note: '6.5s', input_price: 0.001, output_price: 0.001 },
        { id: 'abab6.5-chat', note: '6.5', input_price: 0.004, output_price: 0.004 },
        { id: 'abab6.5t-chat', note: '6.5t' },
        { id: 'abab5.5-chat', note: '5.5' },
      ]},
    ],
  },
  baichuan: {
    label: '百川 (Baichuan)',
    endpoint: 'https://api.baichuan-ai.com/v1',
    models: [
      { label: 'Baichuan4 系列', models: [
        { id: 'Baichuan4', note: '旗舰', input_price: 0.1, output_price: 0.1 },
      ]},
      { label: 'Baichuan3 系列', models: [
        { id: 'Baichuan3-Turbo', note: 'Turbo', input_price: 0.004, output_price: 0.004 },
        { id: 'Baichuan3-Turbo-128k', note: 'Turbo 128K', input_price: 0.004, output_price: 0.004 },
      ]},
    ],
  },
  yi: {
    label: '零一万物 (Yi)',
    endpoint: 'https://api.lingyiwanwu.com/v1',
    models: [
      { label: 'Yi Lightning', models: [
        { id: 'yi-lightning', note: 'Lightning', input_price: 0.00099, output_price: 0.00099 },
      ]},
      { label: 'Yi Vision', models: [
        { id: 'yi-vision', note: '视觉', input_price: 0.006, output_price: 0.006 },
      ]},
      { label: 'Yi Large', models: [
        { id: 'yi-large', note: '旗舰', input_price: 0.02, output_price: 0.02 },
        { id: 'yi-large-turbo', note: 'Turbo' },
        { id: 'yi-large-rag', note: 'RAG' },
      ]},
      { label: 'Yi Medium', models: [
        { id: 'yi-medium', note: '均衡', input_price: 0.0025, output_price: 0.0025 },
        { id: 'yi-spark', note: 'Spark', input_price: 0.0004, output_price: 0.0004 },
      ]},
    ],
  },
  stepfun: {
    label: '阶跃星辰 (Step)',
    endpoint: 'https://api.stepfun.com/v1',
    models: [
      { label: 'Step-2 系列', models: [
        { id: 'step-2-16k', note: '旗舰', input_price: 0.038, output_price: 0.038 },
      ]},
      { label: 'Step-1 系列', models: [
        { id: 'step-1-8k', note: '8K', input_price: 0.005, output_price: 0.005 },
        { id: 'step-1-32k', note: '32K', input_price: 0.005, output_price: 0.005 },
      ]},
      { label: 'Step-1V 视觉', models: [
        { id: 'step-1v-8k', note: '视觉8K', input_price: 0.005, output_price: 0.005 },
      ]},
    ],
  },
  hunyuan: {
    label: '腾讯混元',
    endpoint: 'https://hunyuan.tencentcloudapi.com',
    models: [
      { label: '混元 Turbo 系列', models: [
        { id: 'hunyuan-turbos-latest', note: 'Turbo S 最新' },
        { id: 'hunyuan-turbo', note: 'Turbo' },
      ]},
      { label: '混元 Pro', models: [
        { id: 'hunyuan-pro', note: 'Pro 旗舰' },
        { id: 'hunyuan-standard', note: 'Standard 均衡' },
        { id: 'hunyuan-lite', note: 'Lite 免费' },
      ]},
      { label: '混元 Long', models: [
        { id: 'hunyuan-long', note: '长文本' },
      ]},
    ],
  },
  cohere: {
    label: 'Cohere',
    endpoint: 'https://api.cohere.com/v2',
    models: [
      { label: 'Command 系列', models: [
        { id: 'command-r-plus', note: 'R+ 旗舰', input_price: 0.0025, output_price: 0.01 },
        { id: 'command-r', note: 'R 均衡', input_price: 0.0005, output_price: 0.0015 },
        { id: 'command-r-plus-08-2024', note: 'R+ 8月版' },
      ]},
      { label: 'Embed', models: [
        { id: 'embed-v4', note: 'Embed V4' },
        { id: 'embed-multilingual-v3', note: '多语言' },
      ]},
    ],
  },
  mistral: {
    label: 'Mistral AI',
    endpoint: 'https://api.mistral.ai/v1',
    models: [
      { label: 'Mistral 系列', models: [
        { id: 'mistral-large-latest', note: 'Large 旗舰', input_price: 0.002, output_price: 0.006 },
        { id: 'mistral-medium-latest', note: 'Medium', input_price: 0.0008, output_price: 0.0024 },
        { id: 'mistral-small-latest', note: 'Small 轻量', input_price: 0.0001, output_price: 0.0003 },
        { id: 'open-mistral-nemo', note: 'Nemo 开源', input_price: 0.00003, output_price: 0.0001 },
      ]},
      { label: 'Codestral', models: [
        { id: 'codestral-latest', note: '代码专用', input_price: 0.0001, output_price: 0.0003 },
      ]},
      { label: 'Pixtral', models: [
        { id: 'pixtral-large-latest', note: '视觉' },
        { id: 'pixtral-12b', note: '12B 视觉' },
      ]},
      { label: 'Embed', models: [
        { id: 'mistral-embed', note: 'Embedding', input_price: 0.0001, output_price: 0 },
      ]},
    ],
  },
  xai: {
    label: 'xAI (Grok)',
    endpoint: 'https://api.x.ai/v1',
    models: [
      { label: 'Grok 系列', models: [
        { id: 'grok-3', note: 'Grok 3', input_price: 0.003, output_price: 0.015 },
        { id: 'grok-3-fast', note: 'Grok 3 Fast', input_price: 0.005, output_price: 0.025 },
        { id: 'grok-3-mini', note: 'Mini 推理', input_price: 0.0003, output_price: 0.0005 },
        { id: 'grok-3-mini-fast', note: 'Mini Fast', input_price: 0.0006, output_price: 0.001 },
        { id: 'grok-2', note: 'Grok 2', input_price: 0.002, output_price: 0.01 },
        { id: 'grok-2-mini', note: 'Grok 2 Mini', input_price: 0.0003, output_price: 0.0005 },
      ]},
    ],
  },
  meta: {
    label: 'Meta (Llama)',
    endpoint: 'https://api.together.xyz/v1',
    models: [
      { label: 'Llama 4 系列 (最新)', models: [
        { id: 'meta-llama/Llama-4-Maverick-17B-128E-Instruct', note: 'Maverick MoE' },
        { id: 'meta-llama/Llama-4-Scout-17B-16E-Instruct', note: 'Scout MoE' },
      ]},
      { label: 'Llama 3.3 系列', models: [
        { id: 'meta-llama/Llama-3.3-70B-Instruct-Turbo', note: '70B' },
      ]},
      { label: 'Llama 3.1 系列', models: [
        { id: 'meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo', note: '405B 旗舰' },
        { id: 'meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo', note: '70B' },
        { id: 'meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo', note: '8B' },
      ]},
      { label: 'Llama 3.2 Vision', models: [
        { id: 'meta-llama/Llama-3.2-90B-Vision-Instruct-Turbo', note: '90B 视觉' },
        { id: 'meta-llama/Llama-3.2-11B-Vision-Instruct-Turbo', note: '11B 视觉' },
      ]},
    ],
  },
  custom: {
    label: '自定义',
    endpoint: '',
    models: [
      { label: '自定义模型', models: [
        { id: '' },
      ]},
    ],
  },
}

// 当前供应商的模型分组（优先使用动态发现的结果，否则用静态配置）
const discoveredModels = ref<string[]>([])
const discovering = ref(false)

const currentModelGroups = computed<ModelGroup[]>(() => {
  if (discoveredModels.value.length > 0) {
    const staticConfig = providerConfigs[formData.provider]
    const staticIds = new Set<string>()
    if (staticConfig) {
      for (const g of staticConfig.models) {
        for (const m of g.models) staticIds.add(m.id)
      }
    }
    const newIds = discoveredModels.value.filter(id => !staticIds.has(id))
    const groups: ModelGroup[] = staticConfig ? [...staticConfig.models] : []
    if (newIds.length > 0) {
      groups.push({ label: '🔌 自动发现的新模型', models: newIds.map(id => ({ id })) })
    }
    return groups
  }
  const config = providerConfigs[formData.provider]
  return config ? config.models : []
})

// 从厂商 API 自动发现模型
async function discoverModels() {
  if (!formData.endpoint) {
    ElMessage.warning('请先选择厂商或填写 API 端点')
    return
  }
  discovering.value = true
  try {
    const res: any = await adminApi.discoverModels(formData.endpoint, formData.api_key)
    const models: Array<{ id: string; owned_by?: string }> = res.models || []
    discoveredModels.value = models.map(m => m.id)
    if (models.length === 0) {
      ElMessage.info('该厂商未返回可用模型，请手动输入')
    } else {
      ElMessage.success(`发现 ${models.length} 个可用模型`)
    }
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '无法连接厂商 API，请检查端点和密钥')
    discoveredModels.value = []
  } finally {
    discovering.value = false
  }
}

// 供应商变更时自动填充
async function onProviderChange(provider: string) {
  const config = providerConfigs[provider]
  if (config) {
    formData.endpoint = config.endpoint
  }
  formData.model_id = ''
  formData.name = ''
  discoveredModels.value = []
  if (formData.api_key && formData.endpoint && provider !== 'custom') {
    await discoverModels()
  }
}

// 选择模型时自动填充
function onModelSelect(modelId: string) {
  if (!modelId) return
  formData.name = modelId

  const config = providerConfigs[formData.provider]
  if (config) {
    for (const group of config.models) {
      for (const m of group.models) {
        if (m.id === modelId) {
          if (m.input_price !== undefined) formData.input_price = m.input_price
          if (m.output_price !== undefined) formData.output_price = m.output_price
          if (m.max_tokens !== undefined) formData.max_tokens = m.max_tokens
          if (m.streamable !== undefined) formData.streamable = m.streamable
          return
        }
      }
    }
  }
}

// ==================== 厂商分组与过滤 ====================

const usedProviders = computed(() => {
  const provSet = new Set<string>()
  models.value.forEach(m => { if (m.provider) provSet.add(m.provider) })
  return Array.from(provSet).sort()
})

function providerModelCount(provider: string): number {
  return models.value.filter(m => m.provider === provider).length
}

const filteredModels = computed(() => {
  if (activeProvider.value === 'all') return models.value
  return models.value.filter(m => m.provider === activeProvider.value)
})

function onProviderTabChange() {
  // 切换tab时不需要额外操作
}

// ==================== 表单与API逻辑 ====================

const defaultForm = (): Partial<ModelConfig> => ({
  name: '',
  provider: 'openai',
  model_id: '',
  endpoint: 'https://api.openai.com/v1',
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
  discoveredModels.value = []
  showDialog.value = true
}

// API Key 变化时自动发现模型
async function onApiKeyChange() {
  if (formData.api_key && formData.endpoint && formData.provider !== 'custom' && !isEdit.value) {
    await discoverModels()
  }
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

// ==================== 厂商一键导入 ====================

const showBatchDialog = ref(false)
const batchProvider = ref('')
const batchEndpoint = ref('')
const batchApiKey = ref('')
const batchSelected = ref<string[]>([])
const batchImporting = ref(false)
const batchDone = ref(0)
const batchProgress = computed(() =>
  batchSelected.value.length > 0 ? Math.round((batchDone.value / batchSelected.value.length) * 100) : 0
)
const batchStatusText = ref('')

const batchModelList = computed<ModelItem[]>(() => {
  const config = providerConfigs[batchProvider.value]
  if (!config) return []
  const list: ModelItem[] = []
  for (const group of config.models) {
    for (const m of group.models) {
      if (m.id) list.push(m)
    }
  }
  return list
})

const batchModelGroups = computed<ModelGroup[]>(() => {
  const config = providerConfigs[batchProvider.value]
  return config ? config.models.filter(g => g.models.some(m => m.id)) : []
})

function isModelExist(modelId: string): boolean {
  return models.value.some(m => m.model_id === modelId)
}

function onBatchProviderChange(provider: string) {
  const config = providerConfigs[provider]
  batchEndpoint.value = config ? config.endpoint : ''
  batchApiKey.value = ''
  batchSelected.value = []
  batchDone.value = 0
  batchStatusText.value = ''
  // 默认选中所有不存在的模型
  if (config) {
    const newIds: string[] = []
    for (const group of config.models) {
      for (const m of group.models) {
        if (m.id && !isModelExist(m.id)) {
          newIds.push(m.id)
        }
      }
    }
    batchSelected.value = newIds
  }
}

function batchSelectAll() {
  batchSelected.value = batchModelList.value.map(m => m.id)
}

function batchDeselectAll() {
  batchSelected.value = []
}

function batchSelectNew() {
  batchSelected.value = batchModelList.value.filter(m => !isModelExist(m.id)).map(m => m.id)
}

function findModelItem(provider: string, modelId: string): ModelItem | undefined {
  const config = providerConfigs[provider]
  if (!config) return undefined
  for (const group of config.models) {
    for (const m of group.models) {
      if (m.id === modelId) return m
    }
  }
  return undefined
}

async function handleBatchImport() {
  if (batchSelected.value.length === 0) {
    ElMessage.warning('请选择要导入的模型')
    return
  }
  if (!batchApiKey.value) {
    ElMessage.warning('请输入 API Key')
    return
  }

  batchImporting.value = true
  batchDone.value = 0
  let successCount = 0
  let skipCount = 0
  let errorCount = 0

  for (const modelId of batchSelected.value) {
    batchStatusText.value = `正在导入 ${modelId}...`
    const item = findModelItem(batchProvider.value, modelId)
    try {
      await adminApi.createModel({
        name: modelId,
        provider: batchProvider.value,
        model_id: modelId,
        endpoint: batchEndpoint.value,
        api_key: batchApiKey.value,
        input_price: item?.input_price ?? 0,
        output_price: item?.output_price ?? 0,
        max_tokens: item?.max_tokens ?? 4096,
        weight: 50,
        priority: 0,
        streamable: true,
        enabled: true,
        currency: 'CNY',
      })
      successCount++
    } catch (e: any) {
      if (e?.response?.data?.error?.includes('already exist') || e?.response?.data?.error?.includes('重复')) {
        skipCount++
      } else {
        errorCount++
        console.error(`Failed to import ${modelId}:`, e)
      }
    }
    batchDone.value++
  }

  batchImporting.value = false
  batchStatusText.value = `导入完成：成功 ${successCount} 个，跳过 ${skipCount} 个，失败 ${errorCount} 个`
  ElMessage.success(`批量导入完成！成功 ${successCount} 个，跳过 ${skipCount} 个，失败 ${errorCount} 个`)

  await loadModels()
  // 重置选择
  batchSelected.value = []
  batchDone.value = 0
}

onMounted(() => {
  loadModels()
})
</script>
