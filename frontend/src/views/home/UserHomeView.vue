<template>
  <div class="p-6 max-w-6xl mx-auto space-y-6">
    <!-- Welcome Banner -->
    <div class="bg-gradient-to-r from-primary-500 to-primary-700 rounded-2xl p-6 text-white">
      <h1 class="text-2xl font-bold">欢迎回来，{{ email }}</h1>
      <p class="mt-1 text-primary-100">管理您的 API 密钥，查看用量和余额</p>
      <div class="mt-4 flex gap-4">
        <div class="bg-white/20 backdrop-blur rounded-xl px-5 py-3">
          <p class="text-sm text-primary-100">账户余额</p>
          <p class="text-2xl font-bold">¥{{ (balance / 100).toFixed(2) }}</p>
        </div>
        <div class="bg-white/20 backdrop-blur rounded-xl px-5 py-3">
          <p class="text-sm text-primary-100">API 密钥</p>
          <p class="text-2xl font-bold">{{ apiKeyCount }}</p>
        </div>
        <div class="bg-white/20 backdrop-blur rounded-xl px-5 py-3">
          <p class="text-sm text-primary-100">本月用量</p>
          <p class="text-2xl font-bold" :title="`${requestCount} 次请求`">{{ monthlyTokens }} tokens</p>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="grid grid-cols-4 gap-4">
      <router-link to="/apikeys" class="group bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700 hover:shadow-md hover:border-primary-300 transition-all">
        <div class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center mb-3 group-hover:bg-blue-100 dark:group-hover:bg-blue-900/50 transition-colors">
          <el-icon :size="20" class="text-blue-600"><Key /></el-icon>
        </div>
        <h3 class="font-medium text-gray-900 dark:text-white">API 密钥</h3>
        <p class="text-xs text-gray-500 mt-1">创建和管理您的 API 密钥</p>
      </router-link>

      <router-link to="/topup" class="group bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700 hover:shadow-md hover:border-primary-300 transition-all">
        <div class="w-10 h-10 rounded-lg bg-green-50 dark:bg-green-900/30 flex items-center justify-center mb-3 group-hover:bg-green-100 dark:group-hover:bg-green-900/50 transition-colors">
          <el-icon :size="20" class="text-green-600"><CreditCard /></el-icon>
        </div>
        <h3 class="font-medium text-gray-900 dark:text-white">在线充值</h3>
        <p class="text-xs text-gray-500 mt-1">多种支付方式，即时到账</p>
      </router-link>

      <router-link to="/marketplace" class="group bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700 hover:shadow-md hover:border-primary-300 transition-all">
        <div class="w-10 h-10 rounded-lg bg-purple-50 dark:bg-purple-900/30 flex items-center justify-center mb-3 group-hover:bg-purple-100 dark:group-hover:bg-purple-900/50 transition-colors">
          <el-icon :size="20" class="text-purple-600"><ShoppingCart /></el-icon>
        </div>
        <h3 class="font-medium text-gray-900 dark:text-white">模型广场</h3>
        <p class="text-xs text-gray-500 mt-1">浏览可用模型和价格</p>
      </router-link>

      <router-link to="/docs" class="group bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700 hover:shadow-md hover:border-primary-300 transition-all">
        <div class="w-10 h-10 rounded-lg bg-orange-50 dark:bg-orange-900/30 flex items-center justify-center mb-3 group-hover:bg-orange-100 dark:group-hover:bg-orange-900/50 transition-colors">
          <el-icon :size="20" class="text-orange-600"><Reading /></el-icon>
        </div>
        <h3 class="font-medium text-gray-900 dark:text-white">API 文档</h3>
        <p class="text-xs text-gray-500 mt-1">快速接入指南</p>
      </router-link>
    </div>

    <!-- Recent Usage -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-5">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white mb-4">最近调用</h2>
      <div v-if="recentCalls.length === 0" class="py-12 text-center text-gray-400">
        <el-icon :size="40" class="mb-2"><Document /></el-icon>
        <p>暂无调用记录</p>
        <p class="text-sm mt-1">创建 API 密钥后开始使用</p>
      </div>
      <el-table v-else :data="recentCalls" stripe>
        <el-table-column prop="model" label="模型" />
        <el-table-column prop="tokens" label="Token 数" width="120" />
        <el-table-column prop="cost" label="费用" width="100" />
        <el-table-column prop="time" label="时间" width="180" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { userApi } from '@/api'
import { Key, CreditCard, ShoppingCart, Reading, Document } from '@element-plus/icons-vue'

const userStore = useUserStore()
const email = computed(() => userStore.email || '用户')
const balance = ref(0)
const apiKeyCount = ref(0)
const monthlyTokens = ref('0')
const monthlyAmount = ref('0')
const requestCount = ref(0)
const recentCalls = ref<any[]>([])

onMounted(async () => {
  // 加载余额
  try {
    const res: any = await userApi.getBalance()
    balance.value = res.balance || res.data?.balance || 0
  } catch { /* ignore */ }

  // 加载 API Key 列表获取数量
  try {
    const res: any = await userApi.getApiKeys()
    const keys = res.data || []
    apiKeyCount.value = keys.length
  } catch { /* ignore */ }

  // 加载月度统计
  try {
    const res: any = await userApi.getMonthlyStats()
    const stats = res.data || {}
    const tokens = stats.total_tokens || 0
    monthlyTokens.value = tokens >= 1000000 ? (tokens / 1000000).toFixed(1) + 'M' : tokens >= 1000 ? (tokens / 1000).toFixed(0) + 'K' : String(tokens)
    monthlyAmount.value = ((stats.total_amount || 0) / 100).toFixed(2)
    requestCount.value = stats.request_count || 0
  } catch { /* ignore */ }

  // 加载最近调用记录
  try {
    const res: any = await userApi.getUsageLogs({ limit: 10 })
    const logs = res.data || []
    recentCalls.value = logs.map((log: any) => ({
      model: log.model_name || log.model_id,
      tokens: log.total_tokens,
      cost: '¥' + ((log.amount || 0) / 100).toFixed(4),
      time: new Date(log.created_at * 1000).toLocaleString('zh-CN'),
    }))
  } catch { /* ignore */ }
})
</script>
