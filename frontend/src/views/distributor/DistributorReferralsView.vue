<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">推荐用户管理</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">查看和管理您的下级用户</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">总用户数</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-2">{{ stats.total }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">活跃用户</p>
        <p class="text-2xl font-bold text-green-600 mt-2">{{ stats.active }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">本月新增</p>
        <p class="text-2xl font-bold text-blue-600 mt-2">{{ stats.newThisMonth }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <p class="text-sm text-gray-500 dark:text-gray-400">总贡献佣金</p>
        <p class="text-2xl font-bold text-purple-600 mt-2">¥{{ stats.totalContribution.toFixed(2) }}</p>
      </div>
    </div>

    <!-- 筛选和搜索 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="flex flex-wrap items-center gap-4">
        <el-input
          v-model="searchQuery"
          placeholder="搜索用户名或邮箱"
          class="w-64"
          clearable
        >
          <template #prefix>
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
          </template>
        </el-input>
        <el-select v-model="statusFilter" placeholder="用户状态" class="w-40" clearable>
          <el-option label="全部" value="" />
          <el-option label="活跃" value="active" />
          <el-option label="不活跃" value="inactive" />
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

    <!-- 用户列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">用户</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">注册时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">最后活跃</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">消费金额</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">贡献佣金</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">状态</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr v-for="user in filteredReferrals" :key="user.id" class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div class="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center mr-3">
                    <span class="text-sm font-medium text-purple-600 dark:text-purple-400">{{ user.username.charAt(0).toUpperCase() }}</span>
                  </div>
                  <div>
                    <p class="text-sm font-medium text-gray-900 dark:text-white">{{ user.username }}</p>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ user.email }}</p>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ user.registeredAt }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ user.lastActive }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">¥{{ user.totalSpent.toFixed(2) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-green-600">¥{{ user.contribution.toFixed(2) }}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span :class="user.status === 'active' ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'" class="px-2 py-1 text-xs rounded-full">
                  {{ user.status === 'active' ? '活跃' : '不活跃' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <el-button text type="primary" size="small" @click="viewDetail(user)">详情</el-button>
                <el-button text type="primary" size="small" @click="sendMessage(user)">发消息</el-button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="px-6 py-4 border-t border-gray-100 dark:border-gray-700 flex items-center justify-between">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          共 {{ totalReferrals }} 个用户
        </p>
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="totalReferrals"
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

const searchQuery = ref('')
const statusFilter = ref('')
const dateRange = ref<[Date, Date] | null>(null)
const currentPage = ref(1)
const pageSize = ref(10)

const stats = ref({
  total: 47,
  active: 32,
  newThisMonth: 12,
  totalContribution: 1256.80,
})

const referrals = ref([
  { id: 1, username: 'user001', email: 'user***@gmail.com', registeredAt: '2024-01-15', lastActive: '刚刚', totalSpent: 256.80, contribution: 25.68, status: 'active' },
  { id: 2, username: 'user002', email: 'test***@qq.com', registeredAt: '2024-01-16', lastActive: '1小时前', totalSpent: 189.50, contribution: 18.95, status: 'active' },
  { id: 3, username: 'user003', email: 'demo***@163.com', registeredAt: '2024-01-17', lastActive: '2小时前', totalSpent: 67.30, contribution: 6.73, status: 'active' },
  { id: 4, username: 'user004', email: 'new***@outlook.com', registeredAt: '2024-01-18', lastActive: '5小时前', totalSpent: 45.00, contribution: 4.50, status: 'active' },
  { id: 5, username: 'user005', email: 'api***@gmail.com', registeredAt: '2024-01-19', lastActive: '1天前', totalSpent: 38.20, contribution: 3.82, status: 'inactive' },
  { id: 6, username: 'user006', email: 'test2***@qq.com', registeredAt: '2024-01-20', lastActive: '2天前', totalSpent: 25.00, contribution: 2.50, status: 'inactive' },
])

const filteredReferrals = computed(() => {
  let result = referrals.value
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(u => 
      u.username.toLowerCase().includes(query) || 
      u.email.toLowerCase().includes(query)
    )
  }
  
  if (statusFilter.value) {
    result = result.filter(u => u.status === statusFilter.value)
  }
  
  return result
})

const totalReferrals = computed(() => filteredReferrals.value.length)

function handlePageChange(page: number) {
  currentPage.value = page
  // TODO: 从 API 获取分页数据
}

function viewDetail(user: any) {
  ElMessage.info(`查看用户 ${user.username} 的详情`)
}

function sendMessage(user: any) {
  ElMessage.info(`发送消息给 ${user.username}`)
}

onMounted(async () => {
  // TODO: 从 API 获取真实数据
})
</script>
