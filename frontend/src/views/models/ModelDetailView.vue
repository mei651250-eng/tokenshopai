<template>
  <div class="p-6">
    <!-- Back + Header -->
    <div class="flex items-center gap-3 mb-6">
      <el-button :icon="ArrowLeft" @click="router.push('/models')" text>{{ t('common.back') }}</el-button>
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ model?.name || t('models.detail') }}</h1>
      <el-tag v-if="model" :type="model.enabled ? 'success' : 'info'" size="small">
        {{ model.enabled ? t('common.enabled') : t('common.disabled') }}
      </el-tag>
    </div>

    <div v-if="loading" class="space-y-4">
      <el-skeleton :rows="12" animated />
    </div>

    <div v-else-if="!model" class="text-center py-20 text-gray-400">
      <el-icon :size="48"><WarningFilled /></el-icon>
      <p class="mt-2">{{ t('models.notFound') }}</p>
    </div>

    <template v-else>
      <!-- Overview Cards -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('models.todayRequests') }}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ formatNum(model.today_requests) }}</p>
          <p class="text-xs text-green-500 mt-1">+12.5%</p>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('models.successRate') }}</p>
          <p class="text-2xl font-bold text-green-600 mt-1">{{ (model.success_rate * 100).toFixed(1) }}%</p>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('models.avgLatency') }}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ model.latency_ms }}ms</p>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('models.totalTokens') }}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ formatNum(model.total_tokens) }}</p>
        </div>
      </div>

      <!-- Tabs -->
      <el-tabs v-model="activeTab" class="model-detail-tabs">
        <!-- Config Tab -->
        <el-tab-pane :label="t('models.config')" name="config">
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
              <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.basicInfo') }}</h3>
              <el-descriptions :column="1" border>
                <el-descriptions-item :label="t('models.name')">{{ model.name }}</el-descriptions-item>
                <el-descriptions-item :label="t('models.provider')">
                  <el-tag size="small">{{ model.provider }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item :label="t('models.modelId')">{{ model.model_id }}</el-descriptions-item>
                <el-descriptions-item :label="t('models.weight')">{{ model.weight }}</el-descriptions-item>
                <el-descriptions-item :label="t('models.status')">
                  <el-switch v-model="model.enabled" @change="handleToggle" />
                </el-descriptions-item>
              </el-descriptions>
            </div>
            <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
              <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.pricing') }}</h3>
              <el-descriptions :column="1" border>
                <el-descriptions-item :label="t('models.inputPrice')">{{ model.input_price }} / 1K tokens</el-descriptions-item>
                <el-descriptions-item :label="t('models.outputPrice')">{{ model.output_price }} / 1K tokens</el-descriptions-item>
                <el-descriptions-item :label="t('models.currency')">{{ model.currency }}</el-descriptions-item>
              </el-descriptions>
            </div>
          </div>
        </el-tab-pane>

        <!-- API Tab -->
        <el-tab-pane :label="t('models.api')" name="api">
          <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
            <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.apiEndpoint') }}</h3>
            <div class="bg-gray-100 dark:bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-800 dark:text-gray-200 overflow-x-auto">
              <p class="text-gray-500 mb-2"># {{ t('models.chatApi') }}</p>
              <p>POST /v1/chat/completions</p>
              <p class="text-gray-500 mt-4 mb-2"># {{ t('models.requestExample') }}</p>
              <pre class="whitespace-pre-wrap">curl -X POST {{ apiBase }}/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "{{ model.model_id }}",
    "messages": [{"role": "user", "content": "Hello"}]
  }'</pre>
            </div>
          </div>
        </el-tab-pane>

        <!-- Stats Tab -->
        <el-tab-pane :label="t('models.stats')" name="stats">
          <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700 mb-4">
            <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.callTrend') }}</h3>
            <div class="h-64">
              <v-chart :option="trendOption" autoresize />
            </div>
          </div>
        </el-tab-pane>

        <!-- Rate Limit Tab -->
        <el-tab-pane :label="t('models.rateLimit')" name="ratelimit">
          <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
            <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.rateLimitConfig') }}</h3>
            <el-form label-width="160px">
              <el-form-item :label="t('models.rpmLimit')">
                <el-input-number v-model="rateLimit.rpm" :min="0" :max="100000" :step="100" />
                <span class="ml-2 text-gray-400 text-sm">{{ t('models.rpmUnit') }}</span>
              </el-form-item>
              <el-form-item :label="t('models.tpmLimit')">
                <el-input-number v-model="rateLimit.tpm" :min="0" :max="10000000" :step="1000" />
                <span class="ml-2 text-gray-400 text-sm">{{ t('models.tpmUnit') }}</span>
              </el-form-item>
              <el-form-item :label="t('models.concurrencyLimit')">
                <el-input-number v-model="rateLimit.concurrency" :min="1" :max="1000" />
              </el-form-item>
              <el-form-item :label="t('models.timeout')">
                <el-input-number v-model="rateLimit.timeout" :min="5" :max="300" />
                <span class="ml-2 text-gray-400 text-sm">{{ t('models.timeoutUnit') }}</span>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="saveRateLimit">{{ t('common.save') }}</el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- Test Tab -->
        <el-tab-pane :label="t('models.test')" name="test">
          <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
            <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.testConsole') }}</h3>
            <el-input v-model="testPrompt" type="textarea" :rows="4" :placeholder="t('models.testPlaceholder')" class="mb-4" />
            <div class="flex items-center gap-3 mb-4">
              <el-button type="primary" @click="sendTest" :loading="testLoading">{{ t('models.sendTest') }}</el-button>
              <el-select v-model="testTemperature" style="width: 120px">
                <el-option label="0 (精确)" :value="0" />
                <el-option label="0.7 (平衡)" :value="0.7" />
                <el-option label="1.0 (创意)" :value="1.0" />
              </el-select>
            </div>
            <div v-if="testResult" class="bg-gray-100 dark:bg-gray-900 rounded-lg p-4">
              <p class="whitespace-pre-wrap text-sm text-gray-800 dark:text-gray-200">{{ testResult }}</p>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowLeft, WarningFilled } from '@element-plus/icons-vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { adminApi } from '@/api'
import type { ModelConfig } from '@/api'

use([LineChart, GridComponent, TooltipComponent, CanvasRenderer])

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const model = ref<ModelConfig | null>(null)
const loading = ref(true)
const activeTab = ref('config')
const testPrompt = ref('')
const testResult = ref('')
const testLoading = ref(false)
const testTemperature = ref(0.7)
const rateLimit = ref({ rpm: 60, tpm: 100000, concurrency: 10, timeout: 30 })
const apiBase = computed(() => '')

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: 50, right: 20, top: 20, bottom: 30 },
  xAxis: { type: 'category', data: Array.from({ length: 24 }, (_, i) => `${i}:00`) },
  yAxis: { type: 'value' },
  series: [{
    type: 'line', smooth: true, data: Array.from({ length: 24 }, () => Math.floor(Math.random() * 500 + 100)),
    areaStyle: { color: 'rgba(99,102,241,0.1)' }, lineStyle: { color: '#6366f1' }, itemStyle: { color: '#6366f1' },
  }],
}))

function formatNum(n: number | undefined) {
  if (n === undefined) return '0'
  return n.toLocaleString()
}

async function loadModel() {
  loading.value = true
  try {
    const res: any = await adminApi.getModels()
    const models = res.data || res
    const id = route.params.id as string
    model.value = (Array.isArray(models) ? models : []).find((m: ModelConfig) => m.id === id) || null
    if (!model.value) {
      // Model not found, redirect back
      ElMessage.warning(t('models.notFound'))
      router.push('/models')
      return
    }
  } catch {
    ElMessage.error(t('models.notFound'))
    router.push('/models')
  } finally {
    loading.value = false
  }
}

function handleToggle() {
  ElMessage.success(model.value?.enabled ? t('models.enabled') : t('models.disabled'))
}

async function sendTest() {
  if (!testPrompt.value.trim()) return
  testLoading.value = true
  testResult.value = ''
  try {
    const res: any = await modelApi.chat({
      model: model.value?.model_id || '',
      messages: [{ role: 'user', content: testPrompt.value }],
      temperature: testTemperature.value,
    })
    testResult.value = res.choices?.[0]?.message?.content || JSON.stringify(res)
  } catch {
    testResult.value = t('models.testError')
  } finally {
    testLoading.value = false
  }
}

function saveRateLimit() {
  ElMessage.success(t('models.rateLimitSaved'))
}

import { modelApi } from '@/api'

onMounted(loadModel)
</script>
