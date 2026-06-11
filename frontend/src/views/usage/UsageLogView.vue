<template>
  <div class="usage-log-page">
    <div class="page-header">
      <div>
        <h2>用量明细</h2>
        <p class="text-gray-500 text-sm mt-1">查看您的 API 调用记录和 Token 消耗详情</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stat-cards">
      <div class="stat-card">
        <div class="stat-icon blue">📊</div>
        <div>
          <div class="stat-value">{{ formatNumber(monthlyStats.total_tokens) }}</div>
          <div class="stat-label">本月 Token 消耗</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon green">💰</div>
        <div>
          <div class="stat-value">¥{{ (monthlyStats.total_amount / 100).toFixed(2) }}</div>
          <div class="stat-label">本月费用</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon purple">🔢</div>
        <div>
          <div class="stat-value">{{ formatNumber(monthlyStats.request_count) }}</div>
          <div class="stat-label">本月请求数</div>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-select v-model="filterModel" placeholder="筛选模型" clearable style="width: 220px" @change="loadLogs">
        <el-option v-for="m in modelOptions" :key="m" :label="m" :value="m" />
      </el-select>
    </div>

    <!-- 调用日志表格 -->
    <div class="log-table" v-loading="loading">
      <el-empty v-if="!loading && logs.length === 0" description="暂无调用记录" />
      <el-table v-else :data="logs" stripe>
        <el-table-column prop="model_name" label="模型" width="200">
          <template #default="{ row }">
            <span class="font-medium">{{ row.model_name || row.model_id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="输入/输出 Tokens" width="180">
          <template #default="{ row }">
            <span class="text-blue-600">{{ row.input_tokens?.toLocaleString() }}</span>
            <span class="text-gray-400 mx-1">/</span>
            <span class="text-green-600">{{ row.output_tokens?.toLocaleString() }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="total_tokens" label="总 Tokens" width="120">
          <template #default="{ row }">
            {{ row.total_tokens?.toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="费用" width="100">
          <template #default="{ row }">
            <span class="text-amber-600 font-medium">¥{{ (row.amount / 100).toFixed(4) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="余额变化" width="160">
          <template #default="{ row }">
            <span class="text-gray-500 text-xs">{{ (row.balance_before / 100).toFixed(2) }}</span>
            <span class="text-gray-400 mx-1">→</span>
            <span class="text-gray-500 text-xs">{{ (row.balance_after / 100).toFixed(2) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            <span class="text-sm text-gray-500">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination" v-if="total > pageSize">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { userApi } from '@/api'

interface LogEntry {
  id: string
  model_name: string
  model_id: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  amount: number
  balance_before: number
  balance_after: number
  created_at: number
}

const logs = ref<LogEntry[]>([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = 20
const filterModel = ref('')
const modelOptions = ref<string[]>([])
const monthlyStats = ref({ total_tokens: 0, total_amount: 0, request_count: 0 })

function formatNumber(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(0) + 'K'
  return String(n)
}

function formatTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

async function loadLogs() {
  loading.value = true
  try {
    const offset = (currentPage.value - 1) * pageSize
    const params: any = { limit: pageSize, offset }
    if (filterModel.value) params.model = filterModel.value
    const res: any = await userApi.getUsageLogs(params)
    logs.value = res.data || []
    total.value = res.total || 0

    // 提取去重模型名
    const models = new Set<string>()
    logs.value.forEach(l => { if (l.model_name) models.add(l.model_name) })
    modelOptions.value = Array.from(models)
  } catch { /* ignore */ }
  finally { loading.value = false }
}

async function loadStats() {
  try {
    const res: any = await userApi.getMonthlyStats()
    monthlyStats.value = res.data || { total_tokens: 0, total_amount: 0, request_count: 0 }
  } catch { /* ignore */ }
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadLogs()
}

onMounted(() => {
  loadLogs()
  loadStats()
})
</script>

<style scoped>
.usage-log-page { padding: 0; }
.page-header { margin-bottom: 24px; }
.page-header h2 { font-size: 20px; font-weight: 700; margin: 0; }
.stat-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.stat-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
}
.stat-icon { font-size: 28px; width: 44px; height: 44px; display: flex; align-items: center; justify-content: center; border-radius: 10px; }
.stat-icon.blue { background: #dbeafe; }
.stat-icon.green { background: #d1fae5; }
.stat-icon.purple { background: #ede9fe; }
.stat-value { font-size: 22px; font-weight: 800; }
.stat-label { font-size: 13px; color: #94a3b8; }
.filter-bar { margin-bottom: 16px; }
.pagination { margin-top: 16px; display: flex; justify-content: center; }
</style>
