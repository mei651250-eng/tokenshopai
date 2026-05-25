<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { reconciliationApi } from '@/api'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent])

const { t } = useI18n()

const loading = ref(false)
const summaries = ref<any[]>([])
const dateRange = ref<string[]>([])

const searchForm = reactive({
  startDate: '',
  endDate: '',
  tenantId: '',
})

// 默认最近7天
function initDateRange() {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 7)
  dateRange.value = [
    start.toISOString().slice(0, 10),
    end.toISOString().slice(0, 10),
  ]
  searchForm.startDate = dateRange.value[0]
  searchForm.endDate = dateRange.value[1]
}

async function loadSummary() {
  loading.value = true
  try {
    const res = await reconciliationApi.getRange({
      start_date: searchForm.startDate,
      end_date: searchForm.endDate,
      tenant_id: searchForm.tenantId,
    })
    summaries.value = res.summaries || res.data || []
  } catch {
    summaries.value = []
  } finally {
    loading.value = false
  }
}

// 汇总统计
const aggregated = computed(() => {
  const agg = { paymentAmount: 0, apiCost: 0, grossProfit: 0, netProfit: 0, tokenConsumed: 0, apiCallCount: 0 }
  for (const s of summaries.value) {
    agg.paymentAmount += s.payment_amount || 0
    agg.apiCost += s.api_cost || 0
    agg.grossProfit += s.gross_profit || 0
    agg.netProfit += s.net_profit || 0
    agg.tokenConsumed += s.token_consumed || 0
    agg.apiCallCount += s.api_call_count || 0
  }
  return agg
})

// 收支趋势图
const revenueChartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['充值收入', 'API成本', '净利润'] },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: summaries.value.map((s: any) => s.date) },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => `¥${(v / 100).toFixed(0)}` } },
  series: [
    { name: '充值收入', type: 'bar', data: summaries.value.map((s: any) => s.payment_amount), itemStyle: { color: '#409EFF' } },
    { name: 'API成本', type: 'bar', data: summaries.value.map((s: any) => s.api_cost), itemStyle: { color: '#F56C6C' } },
    { name: '净利润', type: 'line', data: summaries.value.map((s: any) => s.net_profit), itemStyle: { color: '#67C23A' } },
  ],
}))

function onDateChange(val: string[]) {
  if (val) {
    searchForm.startDate = val[0]
    searchForm.endDate = val[1]
  }
}

function formatAmount(amount: number) {
  return `¥${(amount / 100).toFixed(2)}`
}

function getProfitColor(val: number) {
  return val >= 0 ? '#67C23A' : '#F56C6C'
}

onMounted(() => {
  initDateRange()
  loadSummary()
})
</script>

<template>
  <div class="reconciliation-view">
    <div class="page-header">
      <h2>对账中心</h2>
      <div>
        <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" value-format="YYYY-MM-DD" @change="onDateChange" style="margin-right: 12px" />
        <el-button type="primary" @click="loadSummary">查询</el-button>
      </div>
    </div>

    <!-- 汇总卡片 -->
    <el-row :gutter="16" style="margin-bottom: 20px">
      <el-col :span="4">
        <el-statistic title="充值总收入" :value="aggregated.paymentAmount / 100" :precision="2" prefix="¥" />
      </el-col>
      <el-col :span="4">
        <el-statistic title="API总成本" :value="aggregated.apiCost / 100" :precision="2" prefix="¥" />
      </el-col>
      <el-col :span="4">
        <el-statistic title="毛利" :value="aggregated.grossProfit / 100" :precision="2" prefix="¥" />
      </el-col>
      <el-col :span="4">
        <el-statistic title="净利润" :value="aggregated.netProfit / 100" :precision="2" prefix="¥" />
      </el-col>
      <el-col :span="4">
        <el-statistic title="Token消耗" :value="aggregated.tokenConsumed" />
      </el-col>
      <el-col :span="4">
        <el-statistic title="API调用量" :value="aggregated.apiCallCount" />
      </el-col>
    </el-row>

    <!-- 收支趋势图 -->
    <el-card style="margin-bottom: 20px">
      <v-chart :option="revenueChartOption" style="height: 300px" autoresize />
    </el-card>

    <!-- 日明细表 -->
    <el-table :data="summaries" v-loading="loading" stripe>
      <el-table-column prop="date" label="日期" width="120" />
      <el-table-column label="充值金额" width="120">
        <template #default="{ row }">{{ formatAmount(row.payment_amount) }}</template>
      </el-table-column>
      <el-table-column label="手续费" width="100">
        <template #default="{ row }">{{ formatAmount(row.payment_fee) }}</template>
      </el-table-column>
      <el-table-column label="实际到账" width="120">
        <template #default="{ row }">{{ formatAmount(row.actual_income) }}</template>
      </el-table-column>
      <el-table-column label="API成本" width="120">
        <template #default="{ row }">{{ formatAmount(row.api_cost) }}</template>
      </el-table-column>
      <el-table-column label="毛利" width="120">
        <template #default="{ row }">
          <span :style="{ color: getProfitColor(row.gross_profit) }">{{ formatAmount(row.gross_profit) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="退款" width="100">
        <template #default="{ row }">{{ formatAmount(row.refund_amount) }}</template>
      </el-table-column>
      <el-table-column label="净利润" width="120">
        <template #default="{ row }">
          <span :style="{ color: getProfitColor(row.net_profit), fontWeight: 'bold' }">{{ formatAmount(row.net_profit) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="API调用" width="100">
        <template #default="{ row }">{{ row.api_call_count?.toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="Token" width="100">
        <template #default="{ row }">{{ row.token_consumed?.toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="对账状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.recon_status === 'matched' ? 'success' : 'warning'" size="small">
            {{ row.recon_status === 'matched' ? '已对齐' : '有差异' }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.reconciliation-view { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h2 { margin: 0; font-size: 20px; }
</style>
