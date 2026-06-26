<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">佣金记录</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">查看您的所有佣金收入记录</p>
      </div>
      <el-button @click="exportRecords">
        <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
        </svg>
        导出记录
      </el-button>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">累计佣金</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-2">¥{{ stats.total.toFixed(2) }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">已结算</p>
        <p class="text-2xl font-bold text-green-600 mt-2">¥{{ stats.settled.toFixed(2) }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">待结算</p>
        <p class="text-2xl font-bold text-yellow-600 mt-2">¥{{ stats.pending.toFixed(2) }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">本月收入</p>
        <p class="text-2xl font-bold text-blue-600 mt-2">¥{{ stats.thisMonth.toFixed(2) }}</p>
      </div>
    </div>

    <!-- 筛选 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="flex flex-wrap items-center gap-4">
        <el-select v-model="typeFilter" placeholder="佣金类型" class="w-40" clearable>
          <el-option label="全部" value="" />
          <el-option label="充值返佣" value="topup" />
          <el-option label="注册奖励" value="register" />
          <el-option label="消费返佣" value="consume" />
        </el-select>
        <el-select v-model="statusFilter" placeholder="结算状态" class="w-40" clearable>
          <el-option label="全部" value="" />
          <el-option label="已结算" value="settled" />
          <el-option label="待结算" value="pending" />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          class="w-72"
        />
      </div>
    </div>

    <!-- 佣金记录列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">来源用户</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">类型</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">订单金额</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">佣金比例</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">佣金金额</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">状态</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr v-for="record in filteredCommissions" :key="record.id" class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ record.time }}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div class="w-8 h-8 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center mr-2">
                    <span class="text-xs font-medium text-purple-600 dark:text-purple-400">{{ record.username.charAt(0).toUpperCase() }}</span>
                  </div>
                  <span class="text-sm text-gray-900 dark:text-white">{{ record.username }}</span>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span class="px-2 py-1 text-xs rounded-full" :class="getTypeClass(record.type)">
                  {{ getTypeText(record.type) }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">¥{{ record.orderAmount.toFixed(2) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ record.rate }}%</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-green-600">+¥{{ record.commission.toFixed(2) }}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span :class="record.status === 'settled' ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'" class="px-2 py-1 text-xs rounded-full">
                  {{ record.status === 'settled' ? '已结算' : '待结算' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="px-6 py-4 border-t border-gray-100 dark:border-gray-700 flex items-center justify-between">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          共 {{ totalCommissions }} 条记录
        </p>
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="totalCommissions"
          layout="prev, pager, next"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const typeFilter = ref('')
const statusFilter = ref('')
const dateRange = ref<[Date, Date] | null>(null)
const currentPage = ref(1)
const pageSize = ref(20)

const stats = ref({
  total: 1256.80,
  settled: 856.30,
  pending: 400.50,
  thisMonth: 328.50,
})

const commissions = ref([
  { id: 1, time: '2024-01-20 14:30', username: 'user***@gmail.com', type: 'topup', orderAmount: 125.00, rate: 10, commission: 12.50, status: 'settled' },
  { id: 2, time: '2024-01-20 10:15', username: 'test***@qq.com', type: 'register', orderAmount: 50.00, rate: 10, commission: 5.00, status: 'settled' },
  { id: 3, time: '2024-01-19 18:45', username: 'demo***@163.com', type: 'topup', orderAmount: 280.00, rate: 10, commission: 28.00, status: 'pending' },
  { id: 4, time: '2024-01-19 12:20', username: 'new***@outlook.com', type: 'consume', orderAmount: 158.00, rate: 10, commission: 15.80, status: 'settled' },
  { id: 5, time: '2024-01-18 16:30', username: 'api***@gmail.com', type: 'register', orderAmount: 50.00, rate: 10, commission: 5.00, status: 'settled' },
  { id: 6, time: '2024-01-18 09:10', username: 'user2***@qq.com', type: 'topup', orderAmount: 200.00, rate: 10, commission: 20.00, status: 'settled' },
])

const filteredCommissions = computed(() => {
  let result = commissions.value
  
  if (typeFilter.value) {
    result = result.filter(r => r.type === typeFilter.value)
  }
  
  if (statusFilter.value) {
    result = result.filter(r => r.status === statusFilter.value)
  }
  
  return result
})

const totalCommissions = computed(() => filteredCommissions.value.length)

function getTypeText(type: string): string {
  const map: Record<string, string> = {
    topup: '充值返佣',
    register: '注册奖励',
    consume: '消费返佣',
  }
  return map[type] || type
}

function getTypeClass(type: string): string {
  const map: Record<string, string> = {
    topup: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
    register: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
    consume: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  }
  return map[type] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
}

function handlePageChange(page: number) {
  currentPage.value = page
  // TODO: 从 API 获取分页数据
}

function exportRecords() {
  ElMessage.success('导出功能开发中')
}

onMounted(async () => {
  // TODO: 从 API 获取真实数据
})
</script>
