<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('monitor.dashboard') }}</h1>
        <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium"
          :class="wsConnected ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400' : 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400'"
        >
          <span class="relative flex h-2 w-2">
            <span v-if="wsConnected" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-2 w-2"
              :class="wsConnected ? 'bg-green-500' : 'bg-red-500'"
            ></span>
          </span>
          {{ wsConnected ? t('monitor.realtime') : t('monitor.disconnected') }}
        </div>
      </div>
      <el-button type="primary" @click="$router.push('/monitor/screen')">
        <el-icon class="mr-1"><Monitor /></el-icon>
        {{ t('monitor.fullscreen') }}
      </el-button>
    </div>

    <!-- Realtime Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
      <div v-for="stat in statCards" :key="stat.label"
        class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700 relative overflow-hidden"
      >
        <div class="flex items-center justify-between mb-2">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ stat.label }}</p>
          <span class="text-lg" v-html="stat.icon"></span>
        </div>
        <p class="text-xl font-bold text-gray-900 dark:text-white">{{ stat.value }}</p>
        <div class="flex items-center gap-1 mt-1">
          <span class="text-xs font-medium" :class="stat.trendUp ? 'text-green-500' : 'text-red-500'">
            {{ stat.trendUp ? '↑' : '↓' }} {{ stat.trend }}
          </span>
          <span class="text-xs text-gray-400">{{ stat.trendLabel }}</span>
        </div>
        <!-- Mini sparkline background -->
        <div class="absolute bottom-0 right-0 w-20 h-8 opacity-20">
          <svg viewBox="0 0 80 32" class="w-full h-full" preserveAspectRatio="none">
            <polyline :points="stat.sparkline" fill="none" :stroke="stat.trendUp ? '#10b981' : '#ef4444'" stroke-width="1.5" />
          </svg>
        </div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- QPS Trend -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('monitor.qpsTrend') }}</h3>
          <div class="flex items-center gap-1 text-xs text-gray-400">
            <span class="w-1.5 h-1.5 rounded-full bg-indigo-500 animate-pulse"></span>
            Live
          </div>
        </div>
        <v-chart :option="qpsChartOption" autoresize style="height: 260px" />
      </div>

      <!-- Token Consumption Trend -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('monitor.tokenTrend') }}</h3>
          <div class="flex items-center gap-1 text-xs text-gray-400">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            Live
          </div>
        </div>
        <v-chart :option="tokenChartOption" autoresize style="height: 260px" />
      </div>
    </div>

    <!-- Latency + Success Rate Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Latency Distribution -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('monitor.latencyDist') }}</h3>
          <div class="flex items-center gap-3 text-xs">
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-sm bg-indigo-500"></span>P50</span>
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-sm bg-amber-500"></span>P99</span>
          </div>
        </div>
        <v-chart :option="latencyChartOption" autoresize style="height: 260px" />
      </div>

      <!-- Success Rate Trend -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('monitor.successRateTrend') }}</h3>
          <div class="flex items-center gap-1 text-xs text-gray-400">
            <span class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span>
            Live
          </div>
        </div>
        <v-chart :option="successRateChartOption" autoresize style="height: 260px" />
      </div>
    </div>

    <!-- Model Health Status -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('monitor.modelHealth') }}</h3>
        <div class="flex items-center gap-3 text-xs">
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-green-500"></span>{{ t('monitor.stateClosed') }}</span>
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-yellow-500"></span>{{ t('monitor.stateHalfOpen') }}</span>
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-red-500"></span>{{ t('monitor.stateOpen') }}</span>
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <th class="text-left py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">{{ t('monitor.modelName') }}</th>
              <th class="text-left py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">{{ t('monitor.provider') }}</th>
              <th class="text-center py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">QPS</th>
              <th class="text-center py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">{{ t('monitor.successRate') }}</th>
              <th class="text-center py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">{{ t('monitor.avgLatency') }}</th>
              <th class="text-center py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">Tokens</th>
              <th class="text-center py-3 px-3 text-gray-500 dark:text-gray-400 font-medium">{{ t('monitor.circuitState') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="model in modelHealthList" :key="model.model_id"
              class="border-b border-gray-100 dark:border-gray-700/50 hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors"
            >
              <td class="py-3 px-3">
                <div class="flex items-center gap-2">
                  <span class="relative flex h-2.5 w-2.5">
                    <span v-if="model.circuit_state === 'closed'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-2.5 w-2.5"
                      :class="{
                        'bg-green-500': model.circuit_state === 'closed',
                        'bg-yellow-500': model.circuit_state === 'half_open',
                        'bg-red-500': model.circuit_state === 'open'
                      }"
                    ></span>
                  </span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ model.model_name }}</span>
                </div>
              </td>
              <td class="py-3 px-3 text-gray-600 dark:text-gray-300">{{ model.provider }}</td>
              <td class="py-3 px-3 text-center">
                <span class="font-mono font-semibold text-gray-900 dark:text-white">{{ model.qps }}</span>
              </td>
              <td class="py-3 px-3 text-center">
                <span class="font-mono" :class="model.success_rate >= 99 ? 'text-green-600 dark:text-green-400' : model.success_rate >= 95 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400'">
                  {{ model.success_rate.toFixed(1) }}%
                </span>
              </td>
              <td class="py-3 px-3 text-center">
                <span class="font-mono" :class="model.avg_latency_ms <= 300 ? 'text-green-600 dark:text-green-400' : model.avg_latency_ms <= 500 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400'">
                  {{ model.avg_latency_ms }}ms
                </span>
              </td>
              <td class="py-3 px-3 text-center font-mono text-gray-600 dark:text-gray-300">
                {{ formatTokens(model.tokens) }}
              </td>
              <td class="py-3 px-3 text-center">
                <el-tag
                  :type="model.circuit_state === 'closed' ? 'success' : model.circuit_state === 'half_open' ? 'warning' : 'danger'"
                  size="small" effect="dark"
                >
                  <span class="flex items-center gap-1">
                    <span class="w-1.5 h-1.5 rounded-full"
                      :class="{
                        'bg-green-300': model.circuit_state === 'closed',
                        'bg-yellow-300': model.circuit_state === 'half_open',
                        'bg-red-300': model.circuit_state === 'open'
                      }"
                    ></span>
                    {{ getCircuitLabel(model.circuit_state) }}
                  </span>
                </el-tag>
              </td>
            </tr>
            <tr v-if="modelHealthList.length === 0">
              <td colspan="7" class="py-8 text-center text-gray-400">
                <el-icon class="text-3xl mb-2"><Monitor /></el-icon>
                <p>{{ t('monitor.noModelData') }}</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Alerts Section -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('monitor.realtimeAlerts') }}</h3>
        <div class="flex items-center gap-2 text-xs">
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-red-500"></span>{{ t('monitor.critical') }}</span>
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-orange-500"></span>{{ t('monitor.high') }}</span>
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-yellow-500"></span>{{ t('monitor.warning') }}</span>
          <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-blue-500"></span>{{ t('monitor.info') }}</span>
        </div>
      </div>
      <div class="space-y-2 max-h-64 overflow-y-auto">
        <div v-for="alert in alerts" :key="alert.id"
          class="flex items-start gap-3 p-3 rounded-lg border"
          :class="{
            'bg-red-50 dark:bg-red-900/10 border-red-200 dark:border-red-800': alert.severity === 'critical',
            'bg-orange-50 dark:bg-orange-900/10 border-orange-200 dark:border-orange-800': alert.severity === 'high',
            'bg-yellow-50 dark:bg-yellow-900/10 border-yellow-200 dark:border-yellow-800': alert.severity === 'warning',
            'bg-blue-50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-800': alert.severity === 'info',
          }"
        >
          <span class="relative flex h-3 w-3 mt-0.5 shrink-0">
            <span v-if="alert.severity === 'critical'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-3 w-3"
              :class="{
                'bg-red-500': alert.severity === 'critical',
                'bg-orange-500': alert.severity === 'high',
                'bg-yellow-500': alert.severity === 'warning',
                'bg-blue-500': alert.severity === 'info'
              }"
            ></span>
          </span>
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-gray-900 dark:text-white">{{ alert.title }}</span>
              <el-tag :type="alert.severity === 'critical' ? 'danger' : alert.severity === 'high' ? 'warning' : alert.severity === 'warning' ? 'info' : ''" size="small">
                {{ getSeverityLabel(alert.severity) }}
              </el-tag>
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{{ alert.message }}</p>
            <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ formatAlertTime(alert.created_at) }}</p>
          </div>
        </div>
        <div v-if="alerts.length === 0" class="py-8 text-center text-gray-400">
          <el-icon class="text-3xl mb-2"><CircleCheck /></el-icon>
          <p>{{ t('monitor.noAlerts') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Monitor, CircleCheck } from '@element-plus/icons-vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { useMonitorStore } from '@/stores/monitor'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const { t } = useI18n()
const monitorStore = useMonitorStore()
const wsConnected = computed(() => monitorStore.wsConnected)
const modelHealthList = computed(() => monitorStore.modelHealthList)
const alerts = computed(() => monitorStore.alerts)

// Format helpers
function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function getCircuitLabel(state: string): string {
  const map: Record<string, string> = {
    closed: t('monitor.stateClosed'),
    half_open: t('monitor.stateHalfOpen'),
    open: t('monitor.stateOpen'),
  }
  return map[state] || state
}

function getSeverityLabel(severity: string): string {
  const map: Record<string, string> = {
    critical: t('monitor.critical'),
    high: t('monitor.high'),
    warning: t('monitor.warning'),
    info: t('monitor.info'),
  }
  return map[severity] || severity
}

function formatAlertTime(time: string): string {
  return new Date(time).toLocaleString()
}

// Generate sparkline SVG points
function generateSparkline(arr: number[]): string {
  if (arr.length < 2) return '0,32 80,32'
  const max = Math.max(...arr)
  const min = Math.min(...arr)
  const range = max - min || 1
  return arr.map((v, i) => {
    const x = (i / (arr.length - 1)) * 80
    const y = 32 - ((v - min) / range) * 28
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

// Stat cards
const statCards = computed(() => {
  const qpsArr = monitorStore.qpsHistory.map(p => p.value)
  const connArr = monitorStore.connectionHistory.map(p => p.value)
  const tokenArr = monitorStore.tokenHistory.map(p => p.value)
  const srArr = monitorStore.successRateHistory.map(p => p.value)
  const p50Arr = monitorStore.latencyP50History.map(p => p.value)
  const p99Arr = monitorStore.latencyP99History.map(p => p.value)

  const prevQps = qpsArr.length >= 2 ? qpsArr[qpsArr.length - 2] : monitorStore.currentQps
  const qpsDiff = prevQps ? ((monitorStore.currentQps - prevQps) / prevQps * 100) : 0

  return [
    {
      label: t('monitor.qps'),
      value: monitorStore.currentQps,
      icon: '⚡',
      trend: Math.abs(qpsDiff).toFixed(1) + '%',
      trendUp: qpsDiff >= 0,
      trendLabel: t('monitor.vsPrev'),
      sparkline: generateSparkline(qpsArr.length > 0 ? qpsArr : [0]),
    },
    {
      label: t('monitor.activeConns'),
      value: monitorStore.activeConnections.toLocaleString(),
      icon: '🔗',
      trend: '0%',
      trendUp: true,
      trendLabel: '',
      sparkline: generateSparkline(connArr.length > 0 ? connArr : [0]),
    },
    {
      label: t('monitor.tokens') + '/s',
      value: formatTokens(monitorStore.tokensPerSecond),
      icon: '📊',
      trend: '0%',
      trendUp: true,
      trendLabel: '',
      sparkline: generateSparkline(tokenArr.length > 0 ? tokenArr : [0]),
    },
    {
      label: t('monitor.successRate'),
      value: monitorStore.successRate.toFixed(1) + '%',
      icon: '✅',
      trend: '0%',
      trendUp: true,
      trendLabel: '',
      sparkline: generateSparkline(srArr.length > 0 ? srArr : [100]),
    },
    {
      label: 'P50 ' + t('monitor.latency'),
      value: monitorStore.p50Latency + 'ms',
      icon: '⏱',
      trend: '0%',
      trendUp: false,
      trendLabel: '',
      sparkline: generateSparkline(p50Arr.length > 0 ? p50Arr : [0]),
    },
    {
      label: 'P99 ' + t('monitor.latency'),
      value: monitorStore.p99Latency + 'ms',
      icon: '⏳',
      trend: '0%',
      trendUp: false,
      trendLabel: '',
      sparkline: generateSparkline(p99Arr.length > 0 ? p99Arr : [0]),
    },
  ]
})

// Common chart theme
const chartTheme = {
  textStyle: { color: '#94a3b8' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: {
    type: 'category' as const,
    axisLine: { lineStyle: { color: '#e2e8f0' } },
    axisLabel: { color: '#94a3b8', fontSize: 10 },
  },
  yAxis: {
    type: 'value' as const,
    axisLine: { lineStyle: { color: '#e2e8f0' } },
    splitLine: { lineStyle: { color: '#f1f5f9' } },
    axisLabel: { color: '#94a3b8', fontSize: 10 },
  },
}

// QPS chart
const qpsChartOption = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  ...chartTheme,
  xAxis: {
    ...chartTheme.xAxis,
    data: monitorStore.qpsHistory.map(p => p.time),
  },
  yAxis: { ...chartTheme.yAxis, name: 'QPS' },
  series: [{
    type: 'line' as const,
    smooth: true,
    data: monitorStore.qpsHistory.map(p => p.value),
    areaStyle: {
      color: {
        type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(99,102,241,0.3)' },
          { offset: 1, color: 'rgba(99,102,241,0.01)' },
        ],
      },
    },
    lineStyle: { color: '#6366f1', width: 2 },
    itemStyle: { color: '#6366f1' },
    showSymbol: false,
  }],
}))

// Token chart
const tokenChartOption = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  ...chartTheme,
  xAxis: {
    ...chartTheme.xAxis,
    data: monitorStore.tokenHistory.map(p => p.time),
  },
  yAxis: { ...chartTheme.yAxis, name: t('monitor.tokens') },
  series: [{
    type: 'line' as const,
    smooth: true,
    data: monitorStore.tokenHistory.map(p => p.value),
    areaStyle: {
      color: {
        type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(16,185,129,0.3)' },
          { offset: 1, color: 'rgba(16,185,129,0.01)' },
        ],
      },
    },
    lineStyle: { color: '#10b981', width: 2 },
    itemStyle: { color: '#10b981' },
    showSymbol: false,
  }],
}))

// Latency chart
const latencyChartOption = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  legend: { data: ['P50', 'P99'], textStyle: { color: '#94a3b8' } },
  ...chartTheme,
  xAxis: {
    ...chartTheme.xAxis,
    data: monitorStore.latencyP50History.map(p => p.time),
  },
  yAxis: { ...chartTheme.yAxis, name: 'ms' },
  series: [
    {
      name: 'P50',
      type: 'line' as const,
      smooth: true,
      data: monitorStore.latencyP50History.map(p => p.value),
      lineStyle: { color: '#6366f1', width: 2 },
      itemStyle: { color: '#6366f1' },
      showSymbol: false,
    },
    {
      name: 'P99',
      type: 'line' as const,
      smooth: true,
      data: monitorStore.latencyP99History.map(p => p.value),
      lineStyle: { color: '#f59e0b', width: 2 },
      itemStyle: { color: '#f59e0b' },
      showSymbol: false,
    },
  ],
}))

// Success rate chart
const successRateChartOption = computed(() => ({
  tooltip: { trigger: 'axis' as const, formatter: (params: any) => {
    const p = params[0]
    return `${p.axisValue}<br/>${t('monitor.successRate')}: <b>${p.value.toFixed(2)}%</b>`
  }},
  ...chartTheme,
  xAxis: {
    ...chartTheme.xAxis,
    data: monitorStore.successRateHistory.map(p => p.time),
  },
  yAxis: {
    ...chartTheme.yAxis,
    name: '%',
    min: (value: { min: number }) => Math.max(0, Math.floor(value.min - 5)),
  },
  series: [{
    type: 'line' as const,
    smooth: true,
    data: monitorStore.successRateHistory.map(p => p.value),
    areaStyle: {
      color: {
        type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(34,197,94,0.3)' },
          { offset: 1, color: 'rgba(34,197,94,0.01)' },
        ],
      },
    },
    lineStyle: { color: '#22c55e', width: 2 },
    itemStyle: { color: '#22c55e' },
    showSymbol: false,
  }],
}))

onMounted(() => {
  monitorStore.connectWebSocket()
})

onUnmounted(() => {
  monitorStore.disconnect()
})
</script>
