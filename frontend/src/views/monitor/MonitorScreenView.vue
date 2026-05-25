<template>
  <div class="monitor-screen p-6 min-h-screen bg-gradient-to-br from-gray-950 via-gray-900 to-slate-900">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-4">
        <div class="w-10 h-10 rounded-lg bg-brand flex items-center justify-center">
          <span class="text-white font-bold">T</span>
        </div>
        <div>
          <h1 class="text-xl font-bold text-white">Token中站站 实时监控大屏</h1>
          <p class="text-sm text-blue-300">{{ currentTime }}</p>
        </div>
      </div>
      <div class="flex items-center gap-4">
        <!-- WebSocket Status -->
        <div class="flex items-center gap-2 text-sm px-3 py-1.5 rounded-full"
          :class="wsConnected ? 'bg-green-900/30' : 'bg-red-900/30'"
        >
          <span class="relative flex h-2.5 w-2.5">
            <span v-if="wsConnected" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-2.5 w-2.5"
              :class="wsConnected ? 'bg-green-500' : 'bg-red-500'"
            ></span>
          </span>
          <span :class="wsConnected ? 'text-green-400' : 'text-red-400'">
            {{ wsConnected ? '实时连接' : '连接断开' }}
          </span>
        </div>
        <el-button @click="toggleFullscreen" type="primary" size="small">
          全屏
        </el-button>
      </div>
    </div>

    <!-- Top Stats -->
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-6">
      <div v-for="stat in topStats" :key="stat.label" class="glow-border rounded-xl p-4 bg-gray-900/50 relative overflow-hidden">
        <p class="text-xs text-blue-300 mb-1">{{ stat.label }}</p>
        <p class="text-2xl font-bold text-white">{{ stat.value }}</p>
        <div class="flex items-center gap-1 mt-1">
          <span class="text-xs" :class="stat.trendUp ? 'text-green-400' : 'text-red-400'">
            {{ stat.trendUp ? '↑' : '↓' }} {{ stat.trend }}
          </span>
        </div>
        <!-- Mini sparkline -->
        <div class="absolute bottom-0 right-0 w-16 h-6 opacity-30">
          <svg viewBox="0 0 64 24" class="w-full h-full" preserveAspectRatio="none">
            <polyline :points="stat.sparkline" fill="none" :stroke="stat.trendUp ? '#34d399' : '#f87171'" stroke-width="1.5" />
          </svg>
        </div>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
      <!-- QPS Trend -->
      <div class="glow-border rounded-xl p-4 bg-gray-900/50">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm text-blue-300">QPS 实时趋势</h3>
          <div class="flex items-center gap-1 text-xs text-blue-400">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-indigo-500"></span>
            </span>
            Live
          </div>
        </div>
        <v-chart :option="realtimeQPSOption" autoresize style="height: 250px" />
      </div>

      <!-- Latency Distribution -->
      <div class="glow-border rounded-xl p-4 bg-gray-900/50">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm text-blue-300">延迟分布</h3>
          <div class="flex items-center gap-1 text-xs text-blue-400">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
            </span>
            Live
          </div>
        </div>
        <v-chart :option="latencyBarOption" autoresize style="height: 250px" />
      </div>
    </div>

    <!-- Token + Success Rate Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
      <div class="glow-border rounded-xl p-4 bg-gray-900/50">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm text-blue-300">Token 消耗趋势</h3>
          <div class="flex items-center gap-1 text-xs text-blue-400">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
            </span>
            Live
          </div>
        </div>
        <v-chart :option="tokenTrendOption" autoresize style="height: 220px" />
      </div>

      <div class="glow-border rounded-xl p-4 bg-gray-900/50">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm text-blue-300">成功率趋势</h3>
          <div class="flex items-center gap-1 text-xs text-blue-400">
            <span class="relative flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
            </span>
            Live
          </div>
        </div>
        <v-chart :option="successRateOption" autoresize style="height: 220px" />
      </div>
    </div>

    <!-- Model Status -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <!-- Model Health -->
      <div class="lg:col-span-2 glow-border rounded-xl p-4 bg-gray-900/50">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm text-blue-300">模型健康状态</h3>
          <div class="flex items-center gap-3 text-xs text-gray-400">
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-green-400"></span>正常</span>
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-yellow-400"></span>半开</span>
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-red-400"></span>熔断</span>
          </div>
        </div>
        <div class="space-y-2">
          <div v-for="model in modelHealthList" :key="model.model_id"
            class="flex items-center justify-between bg-gray-800/50 rounded-lg p-3"
          >
            <div class="flex items-center gap-3">
              <span class="relative flex h-2.5 w-2.5">
                <span v-if="model.circuit_state === 'closed'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-2.5 w-2.5"
                  :class="{
                    'bg-green-400': model.circuit_state === 'closed',
                    'bg-yellow-400': model.circuit_state === 'half_open',
                    'bg-red-400': model.circuit_state === 'open'
                  }"
                ></span>
              </span>
              <span class="text-sm text-white">{{ model.model_name }}</span>
              <span class="text-xs text-gray-400">{{ model.provider }}</span>
            </div>
            <div class="flex items-center gap-6 text-xs">
              <span class="text-blue-300">{{ model.qps }} QPS</span>
              <span :class="model.success_rate >= 99 ? 'text-green-300' : model.success_rate >= 95 ? 'text-yellow-300' : 'text-red-300'">
                {{ model.success_rate.toFixed(1) }}%
              </span>
              <span :class="model.avg_latency_ms <= 300 ? 'text-green-300' : model.avg_latency_ms <= 500 ? 'text-yellow-300' : 'text-red-300'">
                {{ model.avg_latency_ms }}ms
              </span>
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
                  {{ model.circuit_state === 'closed' ? '正常' : model.circuit_state === 'half_open' ? '半开' : '熔断' }}
                </span>
              </el-tag>
            </div>
          </div>
          <div v-if="modelHealthList.length === 0" class="py-6 text-center text-gray-500">
            暂无模型数据
          </div>
        </div>
      </div>

      <!-- Alerts -->
      <div class="glow-border rounded-xl p-4 bg-gray-900/50">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm text-blue-300">实时告警</h3>
          <div class="flex items-center gap-2 text-xs">
            <span class="flex items-center gap-1 text-gray-400"><span class="w-1.5 h-1.5 rounded-full bg-red-500"></span></span>
            <span class="flex items-center gap-1 text-gray-400"><span class="w-1.5 h-1.5 rounded-full bg-orange-500"></span></span>
            <span class="flex items-center gap-1 text-gray-400"><span class="w-1.5 h-1.5 rounded-full bg-yellow-500"></span></span>
          </div>
        </div>
        <div class="space-y-2 max-h-64 overflow-y-auto">
          <div v-for="alert in alerts" :key="alert.id" class="bg-gray-800/50 rounded-lg p-2.5">
            <div class="flex items-center gap-2 mb-1">
              <span class="relative flex h-2 w-2">
                <span v-if="alert.severity === 'critical'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-2 w-2"
                  :class="{
                    'bg-red-400': alert.severity === 'critical',
                    'bg-orange-400': alert.severity === 'high',
                    'bg-yellow-400': alert.severity === 'warning',
                    'bg-blue-400': alert.severity === 'info'
                  }"
                ></span>
              </span>
              <span class="text-xs text-white">{{ alert.title }}</span>
              <el-tag
                :type="alert.severity === 'critical' ? 'danger' : alert.severity === 'high' ? 'warning' : 'info'"
                size="small"
              >
                {{ alert.severity }}
              </el-tag>
            </div>
            <p class="text-xs text-gray-400">{{ alert.message }}</p>
          </div>
          <div v-if="alerts.length === 0" class="py-6 text-center text-gray-500">
            <span class="text-2xl mb-1 block">✅</span>
            暂无告警
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { useMonitorStore } from '@/stores/monitor'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent])

const monitorStore = useMonitorStore()
const wsConnected = computed(() => monitorStore.wsConnected)
const modelHealthList = computed(() => monitorStore.modelHealthList)
const alerts = computed(() => monitorStore.alerts)

const currentTime = ref(new Date().toLocaleString('zh-CN'))
let timer: ReturnType<typeof setInterval>

// Sparkline helper
function generateSparkline(arr: number[]): string {
  if (arr.length < 2) return '0,24 64,24'
  const max = Math.max(...arr)
  const min = Math.min(...arr)
  const range = max - min || 1
  return arr.map((v, i) => {
    const x = (i / (arr.length - 1)) * 64
    const y = 24 - ((v - min) / range) * 20
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

// Top stats
const topStats = computed(() => {
  const qpsArr = monitorStore.qpsHistory.map(p => p.value)
  const connArr = monitorStore.connectionHistory.map(p => p.value)
  const tokenArr = monitorStore.tokenHistory.map(p => p.value)
  const srArr = monitorStore.successRateHistory.map(p => p.value)
  const p50Arr = monitorStore.latencyP50History.map(p => p.value)
  const p99Arr = monitorStore.latencyP99History.map(p => p.value)

  return [
    { label: '当前 QPS', value: monitorStore.currentQps, trend: '0%', trendUp: true, sparkline: generateSparkline(qpsArr.length > 0 ? qpsArr : [0]) },
    { label: '活跃连接', value: monitorStore.activeConnections.toLocaleString(), trend: '0%', trendUp: true, sparkline: generateSparkline(connArr.length > 0 ? connArr : [0]) },
    { label: 'Token 消耗/s', value: formatTokens(monitorStore.tokensPerSecond), trend: '0%', trendUp: true, sparkline: generateSparkline(tokenArr.length > 0 ? tokenArr : [0]) },
    { label: '成功率', value: monitorStore.successRate.toFixed(1) + '%', trend: '0%', trendUp: true, sparkline: generateSparkline(srArr.length > 0 ? srArr : [100]) },
    { label: 'P50 延迟', value: monitorStore.p50Latency + 'ms', trend: '0%', trendUp: false, sparkline: generateSparkline(p50Arr.length > 0 ? p50Arr : [0]) },
    { label: 'P99 延迟', value: monitorStore.p99Latency + 'ms', trend: '0%', trendUp: false, sparkline: generateSparkline(p99Arr.length > 0 ? p99Arr : [0]) },
  ]
})

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

// Dark chart theme
const darkAxis = { axisLine: { lineStyle: { color: '#334155' } }, axisLabel: { color: '#94a3b8' } }
const darkSplit = { splitLine: { lineStyle: { color: '#1e293b' } } }

// QPS chart
const realtimeQPSOption = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category' as const, data: monitorStore.qpsHistory.map(p => p.time), ...darkAxis },
  yAxis: { type: 'value' as const, ...darkAxis, ...darkSplit },
  series: [{
    type: 'line' as const,
    smooth: true,
    data: monitorStore.qpsHistory.map(p => p.value),
    areaStyle: {
      color: { type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: 'rgba(99,102,241,0.3)' },
        { offset: 1, color: 'rgba(99,102,241,0.01)' },
      ]},
    },
    lineStyle: { color: '#6366f1', width: 2 },
    itemStyle: { color: '#6366f1' },
    showSymbol: false,
  }],
}))

// Latency bar chart - per model
const latencyBarOption = computed(() => {
  const models = modelHealthList.value
  return {
    tooltip: { trigger: 'axis' as const },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category' as const,
      data: models.length > 0 ? models.map(m => m.model_name) : ['gpt-4o', 'claude-3.5', 'qwen-max', 'deepseek', 'glm-4'],
      ...darkAxis,
    },
    yAxis: { type: 'value' as const, name: 'ms', ...darkAxis, ...darkSplit },
    series: [{
      type: 'bar' as const,
      data: models.length > 0
        ? models.map(m => ({
            value: m.avg_latency_ms,
            itemStyle: {
              color: m.avg_latency_ms <= 300 ? '#10b981' : m.avg_latency_ms <= 500 ? '#f59e0b' : '#ef4444',
              borderRadius: [4, 4, 0, 0],
            },
          }))
        : [0, 0, 0, 0, 0],
      barWidth: '40%',
    }],
  }
})

// Token trend
const tokenTrendOption = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category' as const, data: monitorStore.tokenHistory.map(p => p.time), ...darkAxis },
  yAxis: { type: 'value' as const, ...darkAxis, ...darkSplit },
  series: [{
    type: 'line' as const,
    smooth: true,
    data: monitorStore.tokenHistory.map(p => p.value),
    areaStyle: {
      color: { type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: 'rgba(16,185,129,0.3)' },
        { offset: 1, color: 'rgba(16,185,129,0.01)' },
      ]},
    },
    lineStyle: { color: '#10b981', width: 2 },
    itemStyle: { color: '#10b981' },
    showSymbol: false,
  }],
}))

// Success rate trend
const successRateOption = computed(() => ({
  tooltip: { trigger: 'axis' as const },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category' as const, data: monitorStore.successRateHistory.map(p => p.time), ...darkAxis },
  yAxis: { type: 'value' as const, name: '%', ...darkAxis, ...darkSplit, min: (value: { min: number }) => Math.max(0, Math.floor(value.min - 5)) },
  series: [{
    type: 'line' as const,
    smooth: true,
    data: monitorStore.successRateHistory.map(p => p.value),
    areaStyle: {
      color: { type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: 'rgba(34,197,94,0.3)' },
        { offset: 1, color: 'rgba(34,197,94,0.01)' },
      ]},
    },
    lineStyle: { color: '#22c55e', width: 2 },
    itemStyle: { color: '#22c55e' },
    showSymbol: false,
  }],
}))

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
  } else {
    document.exitFullscreen()
  }
}

onMounted(() => {
  monitorStore.connectWebSocket()
  timer = setInterval(() => {
    currentTime.value = new Date().toLocaleString('zh-CN')
  }, 1000)
})

onUnmounted(() => {
  monitorStore.disconnect()
  clearInterval(timer)
})
</script>

<style scoped>
.glow-border {
  border: 1px solid rgba(99, 102, 241, 0.2);
  box-shadow: 0 0 15px rgba(99, 102, 241, 0.05);
}

.monitor-screen :deep(.el-tag) {
  border: none;
}
</style>
