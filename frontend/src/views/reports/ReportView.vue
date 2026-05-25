<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('reports.title') }}</h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1">{{ t('reports.subtitle') }}</p>
      </div>
      <div class="flex items-center gap-3">
        <!-- 时间范围选择 -->
        <el-radio-group v-model="timeRange" size="small" @change="onTimeRangeChange">
          <el-radio-button value="today">{{ t('reports.today') }}</el-radio-button>
          <el-radio-button value="7d">{{ t('reports.last7d') }}</el-radio-button>
          <el-radio-button value="30d">{{ t('reports.last30d') }}</el-radio-button>
          <el-radio-button value="90d">{{ t('reports.last90d') }}</el-radio-button>
        </el-radio-group>
        <!-- 导出按钮 -->
        <el-dropdown @command="onExport" trigger="click">
          <el-button type="primary" size="small">
            <el-icon class="mr-1"><Download /></el-icon>
            {{ t('common.export') }}
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="csv">CSV</el-dropdown-item>
              <el-dropdown-item command="excel">Excel</el-dropdown-item>
              <el-dropdown-item command="pdf">PDF</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- 概览统计卡片 -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <div
        v-for="stat in summaryStats"
        :key="stat.label"
        class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ stat.label }}</p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ stat.value }}</p>
          </div>
          <div class="w-12 h-12 rounded-xl flex items-center justify-center" :class="stat.bgClass">
            <el-icon :size="24" :class="stat.iconClass"><component :is="stat.icon" /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center text-sm">
          <span :class="stat.trend > 0 ? 'text-green-500' : 'text-red-500'">
            {{ stat.trend > 0 ? '↑' : '↓' }} {{ Math.abs(stat.trend) }}%
          </span>
          <span class="text-gray-400 dark:text-gray-500 ml-2">{{ t('reports.vsLastPeriod') }}</span>
        </div>
      </div>
    </div>

    <!-- 图表区域 - 第一行 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <!-- 请求趋势 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('reports.requestTrend') }}</h3>
          <el-radio-group v-model="requestTrendType" size="small">
            <el-radio-button value="qps">QPS</el-radio-button>
            <el-radio-button value="count">{{ t('reports.count') }}</el-radio-button>
          </el-radio-group>
        </div>
        <v-chart :option="requestTrendOption" autoresize style="height: 300px" />
      </div>

      <!-- Token 消耗趋势 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('reports.tokenTrend') }}</h3>
          <el-tag size="small" type="info">{{ t('reports.inputOutput') }}</el-tag>
        </div>
        <v-chart :option="tokenTrendOption" autoresize style="height: 300px" />
      </div>
    </div>

    <!-- 图表区域 - 第二行 -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
      <!-- 模型调用分布 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ t('reports.modelDistribution') }}</h3>
        <v-chart :option="modelPieOption" autoresize style="height: 300px" />
      </div>

      <!-- 费用分布 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ t('reports.costDistribution') }}</h3>
        <v-chart :option="costBarOption" autoresize style="height: 300px" />
      </div>

      <!-- 延迟分布 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ t('reports.latencyDistribution') }}</h3>
        <v-chart :option="latencyBarOption" autoresize style="height: 300px" />
      </div>
    </div>

    <!-- 错误分析 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <!-- 错误率趋势 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ t('reports.errorRateTrend') }}</h3>
        <v-chart :option="errorRateOption" autoresize style="height: 280px" />
      </div>

      <!-- 错误类型分布 -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ t('reports.errorTypeDistribution') }}</h3>
        <v-chart :option="errorPieOption" autoresize style="height: 280px" />
      </div>
    </div>

    <!-- 模型排行表格 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 mb-6">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('reports.modelRanking') }}</h3>
        <el-input
          v-model="modelSearch"
          :placeholder="t('reports.searchModel')"
          size="small"
          style="width: 220px"
          clearable
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      <el-table :data="filteredModelRanking" stripe style="width: 100%">
        <el-table-column type="index" width="60" :label="t('reports.rank')" />
        <el-table-column prop="model" :label="t('reports.modelName')" width="180">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full" :style="{ background: row.color }"></span>
              <span class="font-medium">{{ row.model }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="provider" :label="t('model.provider')" width="120" />
        <el-table-column prop="requests" :label="t('reports.requests')" width="120" sortable>
          <template #default="{ row }">
            {{ formatNumber(row.requests) }}
          </template>
        </el-table-column>
        <el-table-column prop="tokens" :label="t('billing.tokens')" width="120" sortable>
          <template #default="{ row }">
            {{ formatTokens(row.tokens) }}
          </template>
        </el-table-column>
        <el-table-column prop="successRate" :label="t('reports.successRate')" width="120" sortable>
          <template #default="{ row }">
            <span :class="row.successRate >= 99 ? 'text-green-500' : row.successRate >= 95 ? 'text-yellow-500' : 'text-red-500'">
              {{ row.successRate.toFixed(1) }}%
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="avgLatency" :label="t('reports.avgLatency')" width="120" sortable>
          <template #default="{ row }">
            <span :class="row.avgLatency <= 300 ? 'text-green-500' : row.avgLatency <= 600 ? 'text-yellow-500' : 'text-red-500'">
              {{ row.avgLatency }}ms
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="cost" :label="t('reports.cost')" width="120" sortable>
          <template #default="{ row }">
            ¥{{ row.cost.toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="errors" :label="t('reports.errors')" width="100" sortable>
          <template #default="{ row }">
            <el-tag v-if="row.errors > 0" type="danger" size="small">{{ row.errors }}</el-tag>
            <span v-else class="text-gray-400">0</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 租户用量排行 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('reports.tenantRanking') }}</h3>
      </div>
      <el-table :data="tenantRanking" stripe style="width: 100%">
        <el-table-column type="index" width="60" :label="t('reports.rank')" />
        <el-table-column prop="name" :label="t('tenant.name')" width="200" />
        <el-table-column prop="plan" :label="t('tenant.plan')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="planTagType(row.plan)">{{ row.plan }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="requests" :label="t('reports.requests')" width="120" sortable>
          <template #default="{ row }">
            {{ formatNumber(row.requests) }}
          </template>
        </el-table-column>
        <el-table-column prop="tokens" :label="t('billing.tokens')" width="120" sortable>
          <template #default="{ row }">
            {{ formatTokens(row.tokens) }}
          </template>
        </el-table-column>
        <el-table-column prop="cost" :label="t('reports.cost')" width="120" sortable>
          <template #default="{ row }">
            ¥{{ row.cost.toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="successRate" :label="t('reports.successRate')" width="120" sortable>
          <template #default="{ row }">
            {{ row.successRate.toFixed(1) }}%
          </template>
        </el-table-column>
        <el-table-column prop="quota" :label="t('reports.quotaUsage')" min-width="160">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-progress :percentage="row.quotaPercent" :stroke-width="6" :color="quotaColor(row.quotaPercent)" style="width: 80px" />
              <span class="text-xs text-gray-500">{{ row.quotaPercent }}%</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Download, Search } from '@element-plus/icons-vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent,
  GridComponent, DataZoomComponent,
} from 'echarts/components'

use([CanvasRenderer, LineChart, PieChart, BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, DataZoomComponent])

const { t } = useI18n()

// 时间范围
const timeRange = ref('7d')

function onTimeRangeChange() {
  // TODO: 重新加载数据
}

// 导出
function onExport(format: string) {
  ElMessage.success(`${t('reports.exporting')} ${format.toUpperCase()}...`)
}

// ============ 概览统计 ============
const summaryStats = computed(() => [
  {
    label: t('reports.totalRequests'),
    value: formatNumber(getTimeSeriesData().totalRequests),
    trend: 12.5,
    icon: 'TrendCharts',
    bgClass: 'bg-blue-50 dark:bg-blue-900/20',
    iconClass: 'text-blue-500',
  },
  {
    label: t('reports.totalTokens'),
    value: formatTokens(getTimeSeriesData().totalTokens),
    trend: 8.3,
    icon: 'Coin',
    bgClass: 'bg-purple-50 dark:bg-purple-900/20',
    iconClass: 'text-purple-500',
  },
  {
    label: t('reports.avgSuccessRate'),
    value: '99.2%',
    trend: 0.5,
    icon: 'SuccessFilled',
    bgClass: 'bg-green-50 dark:bg-green-900/20',
    iconClass: 'text-green-500',
  },
  {
    label: t('reports.totalCost'),
    value: '¥3,240.80',
    trend: -5.2,
    icon: 'Wallet',
    bgClass: 'bg-orange-50 dark:bg-orange-900/20',
    iconClass: 'text-orange-500',
  },
])

function getTimeSeriesData() {
  const map: Record<string, { totalRequests: number; totalTokens: number }> = {
    today: { totalRequests: 23456, totalTokens: 456789 },
    '7d': { totalRequests: 128456, totalTokens: 2400000 },
    '30d': { totalRequests: 523400, totalTokens: 10200000 },
    '90d': { totalRequests: 1568000, totalTokens: 31500000 },
  }
  return map[timeRange.value] || map['7d']
}

// ============ 请求趋势 ============
const requestTrendType = ref('qps')

const timeLabels: Record<string, string[]> = {
  today: ['00:00', '02:00', '04:00', '06:00', '08:00', '10:00', '12:00', '14:00', '16:00', '18:00', '20:00', '22:00'],
  '7d': ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
  '30d': ['1日', '3日', '5日', '7日', '9日', '11日', '13日', '15日', '17日', '19日', '21日', '23日', '25日', '27日', '29日'],
  '90d': ['1月', '2月', '3月'],
}

const requestDataMap: Record<string, { qps: number[]; count: number[] }> = {
  today: {
    qps: [45, 30, 25, 35, 180, 350, 520, 680, 750, 620, 480, 390],
    count: [162000, 108000, 90000, 126000, 648000, 1260000, 1872000, 2448000, 2700000, 2232000, 1728000, 1404000],
  },
  '7d': {
    qps: [520, 680, 750, 620, 780, 350, 280],
    count: [44928000, 58752000, 64800000, 53568000, 67392000, 30240000, 24192000],
  },
  '30d': {
    qps: [420, 510, 480, 620, 560, 780, 650, 720, 690, 750, 680, 710, 640, 590, 810],
    count: [36288000, 44064000, 41472000, 53568000, 48384000, 67392000, 56160000, 62208000, 59616000, 64800000, 58752000, 61344000, 55296000, 50976000, 69984000],
  },
  '90d': {
    qps: [580, 620, 710],
    count: [150336000, 160704000, 183888000],
  },
}

const requestTrendOption = computed(() => {
  const labels = timeLabels[timeRange.value] || timeLabels['7d']
  const data = requestDataMap[timeRange.value] || requestDataMap['7d']
  const seriesData = requestTrendType.value === 'qps' ? data.qps : data.count
  const seriesName = requestTrendType.value === 'qps' ? 'QPS' : t('reports.count')

  return {
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: labels, boundaryGap: false },
    yAxis: { type: 'value', name: seriesName },
    series: [{
      name: seriesName,
      type: 'line',
      smooth: true,
      areaStyle: { color: 'rgba(99,102,241,0.15)' },
      lineStyle: { color: '#6366f1', width: 2 },
      itemStyle: { color: '#6366f1' },
      data: seriesData,
    }],
  }
})

// ============ Token 消耗趋势 ============
const tokenTrendOption = computed(() => {
  const labels = timeLabels[timeRange.value] || timeLabels['7d']
  const isToday = timeRange.value === 'today'
  const inputMultiplier = isToday ? 1 : 100
  const outputMultiplier = isToday ? 1 : 80

  const inputData = labels.map((_, i) => Math.floor((320 + i * 40 + Math.sin(i) * 80) * inputMultiplier))
  const outputData = labels.map((_, i) => Math.floor((180 + i * 25 + Math.cos(i) * 50) * outputMultiplier))

  return {
    tooltip: { trigger: 'axis' },
    legend: { data: [t('reports.inputTokens'), t('reports.outputTokens')], top: 0 },
    grid: { left: '3%', right: '4%', bottom: '3%', top: 40, containLabel: true },
    xAxis: { type: 'category', data: labels },
    yAxis: { type: 'value', name: 'Token' },
    series: [
      {
        name: t('reports.inputTokens'),
        type: 'bar',
        stack: 'tokens',
        data: inputData,
        itemStyle: { color: '#6366f1' },
      },
      {
        name: t('reports.outputTokens'),
        type: 'bar',
        stack: 'tokens',
        data: outputData,
        itemStyle: { color: '#818cf8' },
      },
    ],
  }
})

// ============ 模型调用分布饼图 ============
const modelPieOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { orient: 'vertical', left: 'left', top: 'middle' },
  series: [{
    type: 'pie',
    radius: ['40%', '70%'],
    center: ['60%', '50%'],
    avoidLabelOverlap: false,
    itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
    label: { show: false },
    emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
    data: [
      { value: 45200, name: 'GPT-4o', itemStyle: { color: '#10b981' } },
      { value: 28300, name: 'Claude-3.5', itemStyle: { color: '#6366f1' } },
      { value: 18600, name: 'Qwen-Max', itemStyle: { color: '#f59e0b' } },
      { value: 12800, name: 'DeepSeek-V2', itemStyle: { color: '#ef4444' } },
      { value: 8500, name: 'GLM-4', itemStyle: { color: '#8b5cf6' } },
      { value: 4200, name: t('reports.other'), itemStyle: { color: '#94a3b8' } },
    ],
  }],
}))

// ============ 费用分布柱状图 ============
const costBarOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: ['GPT-4o', 'Claude-3.5', 'Qwen-Max', 'DeepSeek-V2', 'GLM-4'] },
  yAxis: { type: 'value', name: '¥' },
  series: [{
    type: 'bar',
    data: [
      { value: 1240, itemStyle: { color: '#10b981' } },
      { value: 860, itemStyle: { color: '#6366f1' } },
      { value: 420, itemStyle: { color: '#f59e0b' } },
      { value: 180, itemStyle: { color: '#ef4444' } },
      { value: 320, itemStyle: { color: '#8b5cf6' } },
    ],
    barWidth: '50%',
    itemStyle: { borderRadius: [4, 4, 0, 0] },
  }],
}))

// ============ 延迟分布柱状图 ============
const latencyBarOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: ['GPT-4o', 'Claude-3.5', 'Qwen-Max', 'DeepSeek-V2', 'GLM-4'] },
  yAxis: { type: 'value', name: 'ms' },
  series: [
    {
      name: 'P50',
      type: 'bar',
      data: [280, 320, 180, 210, 350],
      itemStyle: { color: '#6366f1', borderRadius: [4, 4, 0, 0] },
    },
    {
      name: 'P99',
      type: 'bar',
      data: [820, 950, 560, 640, 1050],
      itemStyle: { color: '#f59e0b', borderRadius: [4, 4, 0, 0] },
    },
  ],
}))

// ============ 错误率趋势 ============
const errorRateOption = computed(() => {
  const labels = timeLabels[timeRange.value] || timeLabels['7d']
  const errorData = labels.map(() => (Math.random() * 2 + 0.2).toFixed(2))

  return {
    tooltip: { trigger: 'axis', formatter: '{b}: {c}%' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: labels, boundaryGap: false },
    yAxis: { type: 'value', name: '%', max: 5 },
    series: [{
      type: 'line',
      smooth: true,
      areaStyle: { color: 'rgba(239,68,68,0.1)' },
      lineStyle: { color: '#ef4444', width: 2 },
      itemStyle: { color: '#ef4444' },
      data: errorData,
    }],
    visualMap: {
      show: false,
      pieces: [
        { lte: 1, color: '#10b981' },
        { gt: 1, lte: 3, color: '#f59e0b' },
        { gt: 3, color: '#ef4444' },
      ],
    },
  }
})

// ============ 错误类型分布 ============
const errorPieOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { orient: 'vertical', right: 10, top: 'middle' },
  series: [{
    type: 'pie',
    radius: ['35%', '65%'],
    center: ['40%', '50%'],
    avoidLabelOverlap: false,
    itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
    label: { show: false },
    emphasis: { label: { show: true, fontSize: 13, fontWeight: 'bold' } },
    data: [
      { value: 45, name: t('reports.errorTimeout'), itemStyle: { color: '#ef4444' } },
      { value: 25, name: t('reports.errorRateLimit'), itemStyle: { color: '#f59e0b' } },
      { value: 15, name: t('reports.errorAuth'), itemStyle: { color: '#8b5cf6' } },
      { value: 10, name: t('reports.errorModel'), itemStyle: { color: '#6366f1' } },
      { value: 5, name: t('reports.other'), itemStyle: { color: '#94a3b8' } },
    ],
  }],
}))

// ============ 模型排行 ============
const modelSearch = ref('')

const modelRanking = [
  { model: 'GPT-4o', provider: 'OpenAI', requests: 45200, tokens: 2450000, successRate: 99.5, avgLatency: 280, cost: 1240.50, errors: 12, color: '#10b981' },
  { model: 'Claude-3.5', provider: 'Anthropic', requests: 28300, tokens: 1860000, successRate: 99.3, avgLatency: 320, cost: 860.20, errors: 8, color: '#6366f1' },
  { model: 'Qwen-Max', provider: 'Alibaba', requests: 18600, tokens: 980000, successRate: 99.8, avgLatency: 180, cost: 420.30, errors: 2, color: '#f59e0b' },
  { model: 'DeepSeek-V2', provider: 'DeepSeek', requests: 12800, tokens: 720000, successRate: 99.1, avgLatency: 210, cost: 180.80, errors: 5, color: '#ef4444' },
  { model: 'GLM-4', provider: 'Zhipu', requests: 8500, tokens: 560000, successRate: 98.7, avgLatency: 350, cost: 320.40, errors: 4, color: '#8b5cf6' },
  { model: 'GPT-4o-mini', provider: 'OpenAI', requests: 6200, tokens: 380000, successRate: 99.6, avgLatency: 150, cost: 95.60, errors: 1, color: '#10b981' },
  { model: 'Claude-3-Haiku', provider: 'Anthropic', requests: 4800, tokens: 240000, successRate: 99.8, avgLatency: 120, cost: 62.40, errors: 0, color: '#6366f1' },
]

const filteredModelRanking = computed(() => {
  if (!modelSearch.value) return modelRanking
  const s = modelSearch.value.toLowerCase()
  return modelRanking.filter(r => r.model.toLowerCase().includes(s) || r.provider.toLowerCase().includes(s))
})

// ============ 租户排行 ============
const tenantRanking = [
  { name: '科技创新有限公司', plan: 'Enterprise', requests: 52340, tokens: 2800000, cost: 1560.80, successRate: 99.6, quotaPercent: 78 },
  { name: '数据智能科技', plan: 'Pro', requests: 31200, tokens: 1800000, cost: 820.40, successRate: 99.2, quotaPercent: 92 },
  { name: '未来AI实验室', plan: 'Pro', requests: 18600, tokens: 960000, cost: 540.60, successRate: 99.8, quotaPercent: 65 },
  { name: '智慧教育平台', plan: 'Starter', requests: 8400, tokens: 420000, cost: 186.20, successRate: 99.1, quotaPercent: 88 },
  { name: '金融科技集团', plan: 'Enterprise', requests: 62800, tokens: 3200000, cost: 1890.30, successRate: 99.4, quotaPercent: 45 },
  { name: '智能制造系统', plan: 'Starter', requests: 3200, tokens: 180000, cost: 68.40, successRate: 98.9, quotaPercent: 72 },
  { name: '健康医疗AI', plan: 'Free', requests: 1200, tokens: 68000, cost: 24.60, successRate: 99.7, quotaPercent: 96 },
  { name: '电商推荐系统', plan: 'Pro', requests: 22100, tokens: 1200000, cost: 680.90, successRate: 99.3, quotaPercent: 58 },
]

// ============ 工具函数 ============
function formatNumber(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

function formatTokens(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

function planTagType(plan: string): string {
  const map: Record<string, string> = {
    Enterprise: 'danger',
    Pro: 'warning',
    Starter: 'info',
    Free: 'success',
  }
  return map[plan] || 'info'
}

function quotaColor(pct: number): string {
  if (pct >= 90) return '#ef4444'
  if (pct >= 70) return '#f59e0b'
  return '#10b981'
}
</script>
