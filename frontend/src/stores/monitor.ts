import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { MetricsData } from '@/api'

export interface MetricPoint {
  time: string
  value: number
}

export interface ModelHealthState {
  model_id: string
  model_name: string
  provider: string
  qps: number
  success_rate: number
  avg_latency_ms: number
  tokens: number
  errors: number
  circuit_state: 'closed' | 'half_open' | 'open'
}

export interface AlertItem {
  id: string
  type: string
  severity: 'critical' | 'high' | 'warning' | 'info'
  title: string
  message: string
  created_at: string
}

const MAX_HISTORY_POINTS = 60

export const useMonitorStore = defineStore('monitor', () => {
  const metrics = ref<MetricsData | null>(null)
  const wsConnected = ref(false)
  const alerts = ref<AlertItem[]>([])

  // 时间序列数据
  const qpsHistory = ref<MetricPoint[]>([])
  const tokenHistory = ref<MetricPoint[]>([])
  const latencyP50History = ref<MetricPoint[]>([])
  const latencyP99History = ref<MetricPoint[]>([])
  const successRateHistory = ref<MetricPoint[]>([])
  const connectionHistory = ref<MetricPoint[]>([])

  // 模型健康状态
  const modelHealthList = ref<ModelHealthState[]>([])

  // 计算属性
  const currentQps = computed(() => metrics.value?.qps ?? 0)
  const activeConnections = computed(() => metrics.value?.active_connections ?? 0)
  const successRate = computed(() => metrics.value?.success_rate ?? 0)
  const p50Latency = computed(() => metrics.value?.p50_latency_ms ?? 0)
  const p99Latency = computed(() => metrics.value?.p99_latency_ms ?? 0)
  const tokensPerSecond = computed(() => {
    if (!metrics.value) return 0
    return metrics.value.input_tokens + metrics.value.output_tokens
  })
  const totalRequests = computed(() => metrics.value?.total_requests ?? 0)

  // 格式化时间标签
  function formatTime(ts: number): string {
    const d = new Date(ts)
    return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
  }

  // 推送时间序列数据点
  function pushHistory(history: typeof qpsHistory, time: string, value: number) {
    history.value.push({ time, value })
    if (history.value.length > MAX_HISTORY_POINTS) {
      history.value.shift()
    }
  }

  // 更新模型健康状态
  function updateModelHealth(data: MetricsData) {
    if (data.models && data.models.length > 0) {
      modelHealthList.value = data.models.map(m => ({
        model_id: m.model_id,
        model_name: m.model_name,
        provider: m.provider,
        qps: m.requests,
        success_rate: m.success_rate,
        avg_latency_ms: m.avg_latency_ms,
        tokens: m.tokens,
        errors: m.errors,
        circuit_state: (m.circuit_state || 'closed') as ModelHealthState['circuit_state'],
      }))
    }
  }

  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectAttempts = 0

  function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/ws/monitor`

    ws = new WebSocket(url)

    ws.onopen = () => {
      wsConnected.value = true
      reconnectAttempts = 0
      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
    }

    ws.onmessage = (event) => {
      try {
        const data: MetricsData = JSON.parse(event.data)
        metrics.value = data

        // 推送时间序列
        const time = formatTime(data.timestamp)
        pushHistory(qpsHistory, time, data.qps)
        pushHistory(tokenHistory, time, data.input_tokens + data.output_tokens)
        pushHistory(latencyP50History, time, data.p50_latency_ms)
        pushHistory(latencyP99History, time, data.p99_latency_ms)
        pushHistory(successRateHistory, time, data.success_rate)
        pushHistory(connectionHistory, time, data.active_connections)

        // 更新模型健康
        updateModelHealth(data)
      } catch (e) {
        console.error('Failed to parse monitor data', e)
      }
    }

    ws.onclose = () => {
      wsConnected.value = false
      // 指数退避重连
      const delay = Math.min(5000 * Math.pow(1.5, reconnectAttempts), 30000)
      reconnectAttempts++
      reconnectTimer = setTimeout(() => {
        connectWebSocket()
      }, delay)
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    reconnectAttempts = 0
    ws?.close()
    ws = null
  }

  function addAlert(alert: AlertItem) {
    alerts.value.unshift(alert)
    if (alerts.value.length > 50) {
      alerts.value.pop()
    }
  }

  function removeAlert(id: string) {
    alerts.value = alerts.value.filter(a => a.id !== id)
  }

  return {
    // State
    metrics, wsConnected, alerts,
    qpsHistory, tokenHistory, latencyP50History, latencyP99History,
    successRateHistory, connectionHistory,
    modelHealthList,
    // Computed
    currentQps, activeConnections, successRate, p50Latency, p99Latency,
    tokensPerSecond, totalRequests,
    // Actions
    connectWebSocket, disconnect, addAlert, removeAlert,
  }
})
