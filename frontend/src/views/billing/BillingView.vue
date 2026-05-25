<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('billing.balance') }}</h1>

    <!-- Balance Card -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <div class="bg-gradient-to-r from-brand to-primary-600 rounded-xl p-6 text-white">
        <p class="text-sm opacity-80">当前余额</p>
        <p class="text-3xl font-bold mt-2">¥ 12,580.50</p>
        <el-button class="mt-4" type="default" size="small" @click="showTopup = true">充值</el-button>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">本月消耗</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-2">¥ 3,240.80</p>
        <p class="text-xs text-red-500 mt-1">↑ 15.3% vs 上月</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">今日Token</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-2">456,789</p>
        <p class="text-xs text-green-500 mt-1">↑ 8.2% vs 昨日</p>
      </div>
    </div>

    <!-- Token Consumption Chart -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700 mb-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Token 消耗趋势</h3>
      <div class="h-64 flex items-center justify-center text-gray-400">
        <v-chart :option="tokenTrendOption" autoresize style="width: 100%; height: 100%" />
      </div>
    </div>

    <!-- Recent Transactions -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('billing.history') }}</h3>
        <el-button size="small" type="primary" plain>{{ t('billing.export') }}</el-button>
      </div>
      <el-table :data="transactions" stripe>
        <el-table-column prop="model" :label="t('billing.model')" width="180" />
        <el-table-column prop="input_tokens" label="输入Token" width="120" />
        <el-table-column prop="output_tokens" label="输出Token" width="120" />
        <el-table-column prop="amount" :label="t('billing.amount')" width="100">
          <template #default="{ row }">
            <span class="text-red-500">-¥{{ row.amount }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" :label="t('billing.type')" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="time" :label="t('billing.time')" />
      </el-table>
    </div>

    <!-- Topup Dialog -->
    <el-dialog v-model="showTopup" title="充值" width="400px">
      <el-form label-width="80px">
        <el-form-item label="充值金额">
          <el-input-number v-model="topupAmount" :min="1" :step="100" />
        </el-form-item>
        <el-form-item label="支付方式">
          <el-radio-group v-model="payMethod">
            <el-radio value="alipay">支付宝</el-radio>
            <el-radio value="wechat">微信</el-radio>
            <el-radio value="stripe">Stripe</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTopup = false">取消</el-button>
        <el-button type="primary" @click="goToPayment">确认充值</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent])

const { t } = useI18n()
const router = useRouter()
const showTopup = ref(false)
const topupAmount = ref(100)
const payMethod = ref('alipay')

/** 跳转到充值支付页面 */
function goToPayment() {
  router.push({
    path: '/payment',
    query: {
      channel: payMethod.value,
      amount: String(topupAmount.value),
    },
  })
  showTopup.value = false
}

const transactions = [
  { model: 'gpt-4o', input_tokens: 2048, output_tokens: 1536, amount: '0.21', type: '按量', time: '2分钟前' },
  { model: 'claude-3.5-sonnet', input_tokens: 1024, output_tokens: 2048, amount: '0.18', type: '按量', time: '5分钟前' },
  { model: 'qwen-max', input_tokens: 512, output_tokens: 1024, amount: '0.08', type: '按量', time: '8分钟前' },
  { model: 'deepseek-v2', input_tokens: 4096, output_tokens: 2048, amount: '0.04', type: '按量', time: '12分钟前' },
]

const tokenTrendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: {
    type: 'category',
    data: ['1日', '5日', '10日', '15日', '20日', '25日', '30日'],
  },
  yAxis: { type: 'value', name: 'Token' },
  series: [
    {
      name: '输入',
      type: 'bar',
      stack: 'total',
      data: [320, 450, 380, 520, 480, 560, 620],
      itemStyle: { color: '#6366f1' },
    },
    {
      name: '输出',
      type: 'bar',
      stack: 'total',
      data: [180, 240, 210, 280, 260, 320, 350],
      itemStyle: { color: '#818cf8' },
    },
  ],
}))
</script>
