<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('dashboard.title') }}</h1>
      <p class="text-gray-500 mt-1">{{ t('app.description') }}</p>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ stat.label }}</p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ stat.value }}</p>
          </div>
          <div
            class="w-12 h-12 rounded-xl flex items-center justify-center"
            :class="stat.bgClass"
          >
            <el-icon :size="24" :class="stat.iconClass"><component :is="stat.icon" /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center text-sm">
          <span :class="stat.trend > 0 ? 'text-green-500' : 'text-red-500'">
            {{ stat.trend > 0 ? '↑' : '↓' }} {{ Math.abs(stat.trend) }}%
          </span>
          <span class="text-gray-400 ml-2">vs 昨日</span>
        </div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <!-- QPS Trend -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">请求趋势</h3>
        <v-chart :option="qpsChartOption" autoresize style="height: 300px" />
      </div>

      <!-- Token Distribution -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">模型调用分布</h3>
        <v-chart :option="modelPieOption" autoresize style="height: 300px" />
      </div>
    </div>

    <!-- Recent Activity -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">最近调用</h3>
      <el-table :data="recentCalls" stripe>
        <el-table-column prop="model" label="模型" width="180" />
        <el-table-column prop="provider" label="供应商" width="120" />
        <el-table-column prop="tokens" label="Token" width="100" />
        <el-table-column prop="latency" label="延迟(ms)" width="100" />
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="time" label="时间" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent,
} from 'echarts/components'
import { adminApi } from '@/api'

use([CanvasRenderer, LineChart, PieChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

const { t } = useI18n()

// 从后端获取的真实指标数据
const metrics = ref<any>({})
const loading = ref(false)

async function loadMetrics() {
  loading.value = true
  try {
    const res = await adminApi.getMetrics()
    metrics.value = res.data || res
  } catch {
    // 后端不可用时使用默认值
    metrics.value = {}
  } finally {
    loading.value = false
  }
}

const stats = computed(() => {
  const m = metrics.value
  return [
    {
      label: t('dashboard.totalRequests'),
      value: m.total_requests != null ? m.total_requests.toLocaleString() : '0',
      trend: 12.5,
      icon: 'TrendCharts',
      bgClass: 'bg-blue-50 dark:bg-blue-900/20',
      iconClass: 'text-blue-500',
    },
    {
      label: t('dashboard.totalTokens'),
      value: m.total_tokens != null ? (m.total_tokens > 1000000 ? `${(m.total_tokens / 1000000).toFixed(1)}M` : m.total_tokens.toLocaleString()) : '0',
      trend: 8.3,
      icon: 'Coin',
      bgClass: 'bg-purple-50 dark:bg-purple-900/20',
      iconClass: 'text-purple-500',
    },
    {
      label: t('dashboard.successRate'),
      value: m.success_rate != null ? `${(m.success_rate * 100).toFixed(1)}%` : '0%',
      trend: 0.5,
      icon: 'SuccessFilled',
      bgClass: 'bg-green-50 dark:bg-green-900/20',
      iconClass: 'text-green-500',
    },
    {
      label: t('dashboard.avgLatency'),
      value: m.avg_latency != null ? `${m.avg_latency}ms` : '0ms',
      trend: -5.2,
      icon: 'Timer',
      bgClass: 'bg-orange-50 dark:bg-orange-900/20',
      iconClass: 'text-orange-500',
    },
  ]
})

const recentCalls = computed(() => {
  const models = metrics.value?.models || []
  if (models.length === 0) {
    return [
      { model: 'gpt-4o', provider: 'OpenAI', tokens: 2048, latency: 450, status: 'success', time: '2分钟前' },
      { model: 'claude-3.5', provider: 'Anthropic', tokens: 1536, latency: 380, status: 'success', time: '5分钟前' },
      { model: 'qwen-max', provider: '阿里', tokens: 1024, latency: 210, status: 'success', time: '8分钟前' },
      { model: 'deepseek-v2', provider: 'DeepSeek', tokens: 3072, latency: 290, status: 'success', time: '12分钟前' },
      { model: 'gpt-4o', provider: 'OpenAI', tokens: 512, latency: 0, status: 'error', time: '15分钟前' },
    ]
  }
  return models.slice(0, 5).map((m: any) => ({
    model: m.name || m.id,
    provider: m.provider || '-',
    tokens: m.total_tokens || 0,
    latency: m.avg_latency || 0,
    status: m.error_rate && m.error_rate < 0.1 ? 'success' : 'error',
    time: '最近',
  }))
})

const qpsChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: {
    type: 'category',
    data: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '现在'],
  },
  yAxis: { type: 'value', name: 'QPS' },
  series: [{
    name: 'QPS',
    type: 'line',
    smooth: true,
    areaStyle: { color: 'rgba(99,102,241,0.1)' },
    lineStyle: { color: '#6366f1', width: 2 },
    itemStyle: { color: '#6366f1' },
    data: [120, 80, 340, 560, 780, 650, 820],
  }],
}))

const modelPieOption = computed(() => {
  const models = metrics.value?.models || []
  const pieData = models.length > 0
    ? models.map((m: any) => ({ value: m.total_requests || 0, name: m.name || m.id }))
    : [
        { value: 45, name: 'GPT-4o', itemStyle: { color: '#10b981' } },
        { value: 25, name: 'Claude-3.5', itemStyle: { color: '#6366f1' } },
        { value: 15, name: 'Qwen-Max', itemStyle: { color: '#f59e0b' } },
        { value: 10, name: 'DeepSeek-V2', itemStyle: { color: '#ef4444' } },
        { value: 5, name: '其他', itemStyle: { color: '#8b5cf6' } },
      ]
  return {
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', left: 'left' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 16, fontWeight: 'bold' } },
      data: pieData,
    }],
  }
})

onMounted(() => {
  loadMetrics()
})
</script>
