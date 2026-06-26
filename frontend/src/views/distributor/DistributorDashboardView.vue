<template>
  <div class="space-y-6">
    <!-- 欢迎横幅 -->
    <div class="bg-gradient-to-r from-purple-600 to-pink-500 rounded-2xl p-6 text-white">
      <h1 class="text-2xl font-bold mb-2">欢迎回来，{{ username }}！</h1>
      <p class="opacity-90">今天是推广赚钱的好日子，让我们一起努力吧！</p>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">累计佣金</p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">¥{{ stats.totalCommission.toFixed(2) }}</p>
          </div>
          <div class="w-12 h-12 bg-green-100 dark:bg-green-900/30 rounded-xl flex items-center justify-center">
            <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
          </div>
        </div>
        <div class="mt-4 flex items-center text-sm">
          <span class="text-green-500">↑ {{ stats.commissionGrowth }}%</span>
          <span class="text-gray-400 ml-2">较上月</span>
        </div>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">可提现余额</p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">¥{{ stats.availableBalance.toFixed(2) }}</p>
          </div>
          <div class="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-xl flex items-center justify-center">
            <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"/>
            </svg>
          </div>
        </div>
        <button @click="router.push('/distributor/withdraw')" class="mt-4 text-sm text-blue-600 hover:text-blue-700 font-medium">
          立即提现 →
        </button>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">下级用户</p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ stats.totalReferrals }}</p>
          </div>
          <div class="w-12 h-12 bg-purple-100 dark:bg-purple-900/30 rounded-xl flex items-center justify-center">
            <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
            </svg>
          </div>
        </div>
        <div class="mt-4 flex items-center text-sm">
          <span class="text-purple-500">+{{ stats.newReferralsThisMonth }}</span>
          <span class="text-gray-400 ml-2">本月新增</span>
        </div>
      </div>

      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">推广链接点击</p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ stats.totalClicks }}</p>
          </div>
          <div class="w-12 h-12 bg-orange-100 dark:bg-orange-900/30 rounded-xl flex items-center justify-center">
            <svg class="w-6 h-6 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"/>
            </svg>
          </div>
        </div>
        <div class="mt-4 flex items-center text-sm">
          <span class="text-orange-500">转化率 {{ stats.conversionRate }}%</span>
        </div>
      </div>
    </div>

    <!-- 推广链接 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">我的推广链接</h2>
      <div class="flex items-center space-x-4">
        <div class="flex-1 bg-gray-50 dark:bg-gray-700 rounded-lg px-4 py-3">
          <code class="text-sm text-gray-700 dark:text-gray-300 select-all">{{ referralLink }}</code>
        </div>
        <el-button type="primary" @click="copyLink">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
          </svg>
          复制链接
        </el-button>
        <el-button @click="shareLink">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"/>
          </svg>
          分享
        </el-button>
      </div>
    </div>

    <!-- 最近佣金记录 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="p-6 border-b border-gray-100 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">最近佣金记录</h2>
          <router-link to="/distributor/commissions" class="text-sm text-purple-600 hover:text-purple-700">
            查看全部 →
          </router-link>
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">用户</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">类型</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">金额</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">状态</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr v-for="record in recentCommissions" :key="record.id" class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">{{ record.username }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ record.type }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-green-600">+¥{{ record.amount.toFixed(2) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ record.time }}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span :class="record.status === 'settled' ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'" class="px-2 py-1 text-xs rounded-full">
                  {{ record.status === 'settled' ? '已结算' : '待结算' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 下级用户列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="p-6 border-b border-gray-100 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">活跃下级用户</h2>
          <router-link to="/distributor/referrals" class="text-sm text-purple-600 hover:text-purple-700">
            查看全部 →
          </router-link>
        </div>
      </div>
      <div class="p-6">
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div v-for="user in activeReferrals" :key="user.id" class="flex items-center space-x-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div class="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center">
              <span class="text-sm font-medium text-purple-600 dark:text-purple-400">{{ user.username.charAt(0).toUpperCase() }}</span>
            </div>
            <div class="flex-1">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ user.username }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">贡献佣金 ¥{{ user.contribution.toFixed(2) }}</p>
            </div>
            <div class="text-right">
              <p class="text-xs text-gray-400">{{ user.lastActive }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()

const username = computed(() => userStore.user?.username || '分销商')
const referralCode = computed(() => userStore.user?.referral_code || 'ABC123')
const referralLink = computed(() => `https://tokenshopai.com/register?ref=${referralCode.value}`)

// 统计数据
const stats = ref({
  totalCommission: 1256.80,
  commissionGrowth: 23.5,
  availableBalance: 856.30,
  totalReferrals: 47,
  newReferralsThisMonth: 12,
  totalClicks: 1523,
  conversionRate: 3.1,
})

// 最近佣金记录
const recentCommissions = ref([
  { id: 1, username: 'user***@gmail.com', type: '充值返佣', amount: 12.50, time: '2小时前', status: 'settled' },
  { id: 2, username: 'test***@qq.com', type: '注册奖励', amount: 5.00, time: '5小时前', status: 'settled' },
  { id: 3, username: 'demo***@163.com', type: '充值返佣', amount: 28.00, time: '1天前', status: 'pending' },
  { id: 4, username: 'new***@outlook.com', type: '充值返佣', amount: 15.80, time: '2天前', status: 'settled' },
  { id: 5, username: 'api***@gmail.com', type: '注册奖励', amount: 5.00, time: '3天前', status: 'settled' },
])

// 活跃下级用户
const activeReferrals = ref([
  { id: 1, username: 'active_user1', contribution: 156.80, lastActive: '刚刚' },
  { id: 2, username: 'active_user2', contribution: 89.50, lastActive: '1小时前' },
  { id: 3, username: 'active_user3', contribution: 67.30, lastActive: '2小时前' },
  { id: 4, username: 'active_user4', contribution: 45.00, lastActive: '5小时前' },
  { id: 5, username: 'active_user5', contribution: 38.20, lastActive: '1天前' },
  { id: 6, username: 'active_user6', contribution: 25.00, lastActive: '2天前' },
])

function copyLink() {
  navigator.clipboard.writeText(referralLink.value)
  ElMessage.success('链接已复制到剪贴板')
}

function shareLink() {
  if (navigator.share) {
    navigator.share({
      title: 'TokenHub - AI API Gateway',
      text: '注册 TokenHub，享受 AI API 服务！',
      url: referralLink.value,
    })
  } else {
    copyLink()
  }
}

onMounted(async () => {
  // TODO: 从 API 获取真实数据
})
</script>
