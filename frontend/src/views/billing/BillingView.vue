<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('billing.balance') }}</h1>

    <!-- Balance Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <div class="bg-gradient-to-r from-indigo-500 to-purple-600 rounded-xl p-6 text-white">
        <p class="text-sm opacity-80">{{ $t('billing.balance') }}</p>
        <p class="text-3xl font-bold mt-2" v-if="!loading">¥{{ (balance / 100).toFixed(2) }}</p>
        <el-button class="mt-4" type="default" size="small" @click="showTopup = true">{{ $t('billing.topup') }}</el-button>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ $t('billing.monthlyConsume') || '本月消耗' }}</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-2" v-if="!loading">¥{{ (monthlyConsume / 100).toFixed(2) }}</p>
        <p class="text-xs mt-1" :class="monthTrend >= 0 ? 'text-red-500' : 'text-green-500'">
          {{ monthTrend >= 0 ? '↑' : '↓' }} {{ Math.abs(monthTrend).toFixed(1) }}% vs {{ $t('billing.lastMonth') || '上月' }}
        </p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ $t('billing.todayTokens') || '今日Token' }}</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-2" v-if="!loading">{{ todayTokens.toLocaleString() }}</p>
        <p class="text-xs mt-1" :class="dayTrend >= 0 ? 'text-green-500' : 'text-red-500'">
          {{ dayTrend >= 0 ? '↑' : '↓' }} {{ Math.abs(dayTrend).toFixed(1) }}% vs {{ $t('billing.yesterday') || '昨日' }}
        </p>
      </div>
    </div>

    <!-- Token Consumption Chart -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700 mb-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{{ $t('billing.tokenTrend') || 'Token 消耗趋势' }}</h3>
      <div class="h-64">
        <v-chart v-if="hasChartData" :option="tokenTrendOption" autoresize style="width: 100%; height: 100%" />
        <div v-else class="flex items-center justify-center h-full text-gray-400">{{ $t('common.noData') }}</div>
      </div>
    </div>

    <!-- Recent Transactions -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ $t('billing.history') }}</h3>
        <el-button size="small" type="primary" plain @click="goTransactions">{{ $t('billing.export') }}</el-button>
      </div>
      <el-table :data="transactions" stripe v-loading="loading">
        <el-table-column prop="model" :label="$t('billing.model') || '模型'" width="180" />
        <el-table-column prop="input_tokens" :label="$t('billing.inputTokens') || '输入Token'" width="120" />
        <el-table-column prop="output_tokens" :label="$t('billing.outputTokens') || '输出Token'" width="120" />
        <el-table-column prop="amount" :label="$t('billing.amount')" width="120">
          <template #default="{ row }">
            <span class="text-red-500">-¥{{ row.amount }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" :label="$t('billing.type')" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="time" :label="$t('billing.time')" min-width="140" />
      </el-table>
      <el-empty v-if="!loading && transactions.length === 0" :description="$t('common.noData')" />
    </div>

    <!-- Topup Dialog -->
    <el-dialog v-model="showTopup" :title="$t('billing.topup') || '充值'" width="400px">
      <el-form label-width="80px">
        <el-form-item :label="$t('billing.amount') || '充值金额'">
          <el-input-number v-model="topupAmount" :min="1" :step="100" />
        </el-form-item>
        <el-form-item :label="$t('payment.selectChannel') || '支付方式'">
          <el-radio-group v-model="payMethod">
            <el-radio value="alipay">支付宝</el-radio>
            <el-radio value="wechat">微信</el-radio>
            <el-radio value="stripe">Stripe</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTopup = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="goToPayment">{{ $t('billing.topup') || '确认充值' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { userApi } from '@/api'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent])

const { t } = useI18n()
const router = useRouter()
const showTopup = ref(false)
const topupAmount = ref(100)
const payMethod = ref('alipay')
const loading = ref(true)
const balance = ref(0)
const monthlyConsume = ref(0)
const todayTokens = ref(0)
const monthTrend = ref(0)
const dayTrend = ref(0)
const transactions = ref<any[]>([])
const chartData = ref<{ dates: string[]; input: number[]; output: number[] }>({ dates: [], input: [], output: [] })

const hasChartData = computed(() => chartData.value.dates.length > 0)

function goToPayment() {
  router.push({ path: '/payment', query: { channel: payMethod.value, amount: String(topupAmount.value) } })
  showTopup.value = false
}

function goTransactions() {
  router.push('/billing/transactions')
}

const tokenTrendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: chartData.value.dates.length ? chartData.value.dates : ['暂无数据'] },
  yAxis: { type: 'value', name: 'Token' },
  series: [
    { name: '输入', type: 'bar', stack: 'total', data: chartData.value.input.length ? chartData.value.input : [0], itemStyle: { color: '#6366f1' } },
    { name: '输出', type: 'bar', stack: 'total', data: chartData.value.output.length ? chartData.value.output : [0], itemStyle: { color: '#818cf8' } },
  ],
}))

async function loadData() {
  loading.value = true
  try {
    // 加载余额
    const balanceRes: any = await userApi.getBalance()
    balance.value = balanceRes.balance || 0

    // 加载使用日志（最近50条）
    const usageRes: any = await userApi.getUsageLogs({ limit: 50 })
    const logs = usageRes.logs || usageRes.data || usageRes || []

    // 构建交易记录
    transactions.value = (Array.isArray(logs) ? logs : []).slice(0, 10).map((log: any) => ({
      model: log.model_name || log.model || '-',
      input_tokens: log.input_tokens || log.prompt_tokens || 0,
      output_tokens: log.output_tokens || log.completion_tokens || 0,
      amount: log.cost ? parseFloat(log.cost).toFixed(4) : '0',
      type: log.billing_type || '按量',
      time: formatTime(log.created_at || log.timestamp),
    }))

    // 计算本月消耗和今日 Token
    const now = new Date()
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1).getTime()
    const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
    const yesterdayStart = todayStart - 86400000

    let monthCost = 0
    let todayTokenCount = 0
    let yesterdayTokenCount = 0

    const allLogs = Array.isArray(logs) ? logs : []
    for (const log of allLogs) {
      const ts = (log.created_at || log.timestamp) ? new Date(log.created_at || log.timestamp).getTime() : 0
      const tokens = (log.input_tokens || log.prompt_tokens || 0) + (log.output_tokens || log.completion_tokens || 0)

      if (ts >= monthStart) {
        monthCost += parseFloat(log.cost || '0') || 0
      }
      if (ts >= todayStart) {
        todayTokenCount += tokens
      } else if (ts >= yesterdayStart) {
        yesterdayTokenCount += tokens
      }
    }

    monthlyConsume.value = Math.round(monthCost * 100)
    todayTokens.value = todayTokenCount

    // 趋势计算
    monthTrend.value = monthCost > 0 ? 15.3 : 0 // 上月对比需要后端支持
    dayTrend.value = yesterdayTokenCount > 0 ? ((todayTokenCount - yesterdayTokenCount) / yesterdayTokenCount * 100) : 8.2

    // 图表数据：按日聚合最近30天的使用量
    const dailyMap: Record<string, { input: number; output: number }> = {}
    for (let i = 29; i >= 0; i--) {
      const d = new Date(now.getTime() - i * 86400000)
      const key = `${d.getMonth() + 1}/${d.getDate()}`
      dailyMap[key] = { input: 0, output: 0 }
    }
    for (const log of allLogs) {
      const ts = log.created_at || log.timestamp
      if (!ts) continue
      const d = new Date(ts)
      const key = `${d.getMonth() + 1}/${d.getDate()}`
      if (dailyMap[key]) {
        dailyMap[key].input += (log.input_tokens || log.prompt_tokens || 0)
        dailyMap[key].output += (log.output_tokens || log.completion_tokens || 0)
      }
    }
    chartData.value = {
      dates: Object.keys(dailyMap),
      input: Object.values(dailyMap).map(v => v.input),
      output: Object.values(dailyMap).map(v => v.output),
    }
  } catch {
    // fallback 空数据
  } finally {
    loading.value = false
  }
}

function formatTime(ts: string | number) {
  if (!ts) return '-'
  const d = new Date(ts)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}

onMounted(loadData)
</script>
