<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div>
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">我的订阅</h1>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">管理您的订阅计划，选择最适合您的方案</p>
    </div>

    <!-- 当前订阅状态 -->
    <div v-if="currentSub" class="bg-gradient-to-r from-purple-600 to-indigo-600 rounded-2xl p-6 text-white">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm opacity-80">当前计划</p>
          <h2 class="text-3xl font-bold mt-1">{{ currentSub.plan_name }}</h2>
          <p class="text-sm opacity-80 mt-2">
            有效期至 {{ formatDate(currentSub.end_at) }}
            <span v-if="currentSub.auto_renew" class="ml-2 bg-white/20 px-2 py-0.5 rounded text-xs">自动续费</span>
          </p>
        </div>
        <div class="text-right">
          <p class="text-sm opacity-80">已用 Token</p>
          <p class="text-xl font-bold">{{ currentSub.token_used.toLocaleString() }}</p>
        </div>
      </div>
      <div class="mt-4 flex space-x-2">
        <el-button v-if="currentSub.plan_type!=='trial'&&currentSub.plan_type!=='payg'" @click="cancelSub" class="bg-white/20 border-white/30 hover:bg-white/30 text-white" size="small">取消续费</el-button>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="bg-white dark:bg-gray-800 rounded-xl p-8 text-center border border-gray-200 dark:border-gray-700">
      <p class="text-gray-500 dark:text-gray-400 text-lg mb-2">您还没有订阅任何计划</p>
      <p class="text-gray-400 dark:text-gray-500 text-sm mb-4">选择一个适合您的方案开始使用吧！</p>
    </div>

    <!-- 计划列表 -->
    <div>
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">可用计划</h2>

      <!-- 类型切换 -->
      <div class="flex gap-2 mb-4">
        <el-button v-for="t in planTypes" :key="t.value" :type="activeTab===t.value?'primary':'default'" @click="activeTab=t.value" size="small">{{ t.label }}</el-button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div v-for="plan in filteredPlans" :key="plan.id" class="bg-white dark:bg-gray-800 rounded-xl border-2 p-6 transition-all" :class="currentSub?.plan_id===plan.id ? 'border-purple-500 ring-1 ring-purple-500' : 'border-gray-200 dark:border-gray-700 hover:border-purple-300'">
          <div class="text-center mb-4">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ plan.name }}</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ plan.description }}</p>
          </div>

          <!-- 价格 -->
          <div class="text-center mb-4">
            <span class="text-3xl font-bold text-purple-600">¥{{ plan.price.toFixed(2) }}</span>
            <span v-if="plan.duration_days>0" class="text-sm text-gray-400">/{{ plan.duration_days }}天</span>
          </div>

          <!-- 权益 -->
          <div class="space-y-2 text-sm mb-4">
            <div class="flex items-center text-gray-600 dark:text-gray-300">
              <svg class="w-4 h-4 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
              {{ plan.token_limit > 0 ? plan.token_limit.toLocaleString() + ' Token' : '无限 Token' }}
            </div>
            <div class="flex items-center text-gray-600 dark:text-gray-300">
              <svg class="w-4 h-4 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
              {{ plan.daily_req_limit > 0 ? '每日 ' + plan.daily_req_limit + ' 次请求' : '无限请求' }}
            </div>
            <div v-if="plan.type==='payg'" class="flex items-center text-gray-600 dark:text-gray-300">
              <svg class="w-4 h-4 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
              输入 ¥{{ plan.input_token_price }}/千Token
            </div>
            <div v-if="plan.type==='payg'" class="flex items-center text-gray-600 dark:text-gray-300">
              <svg class="w-4 h-4 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
              输出 ¥{{ plan.output_token_price }}/千Token
            </div>
          </div>

          <!-- 按钮 -->
          <el-button v-if="currentSub?.plan_id===plan.id" type="default" class="w-full" disabled>当前方案</el-button>
          <el-button v-else-if="plan.type==='trial'" type="success" class="w-full" @click="subscribe(plan.id)">免费试用</el-button>
          <el-button v-else type="primary" class="w-full" @click="subscribe(plan.id)">立即订阅</el-button>
        </div>
      </div>
    </div>

    <!-- 订阅历史 -->
    <div v-if="history.length" class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="p-6 border-b border-gray-100 dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">订阅历史</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">计划</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">开始时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">结束时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">状态</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr v-for="h in history" :key="h.id" class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">{{ h.plan_name }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(h.start_at) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(h.end_at) }}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <el-tag :type="h.status==='active'?'success':h.status==='expired'?'warning':'info'" size="small">{{ statusLabels[h.status] || h.status }}</el-tag>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('all')
const currentSub = ref<any>(null)
const plans = ref<any[]>([])
const history = ref<any[]>([])

const planTypes = [
  { value: 'all', label: '全部' },
  { value: 'trial', label: '免费试用' },
  { value: 'monthly', label: '月度' },
  { value: 'quarterly', label: '季度' },
  { value: 'annual', label: '年度' },
  { value: 'payg', label: '按量付费' },
]
const statusLabels: Record<string, string> = { active: '使用中', expired: '已过期', cancelled: '已取消', suspended: '已暂停' }

const filteredPlans = computed(() => activeTab.value === 'all' ? plans.value : plans.value.filter(p => p.type === activeTab.value))

function formatDate(d: string) { return d ? new Date(d).toLocaleDateString('zh-CN') : '-' }

async function fetchData() {
  const token = localStorage.getItem('token')
  const auth = { Authorization: `Bearer ${token}` }
  try {
    const [subRes, plansRes, histRes] = await Promise.all([
      fetch('/user/subscription', { headers: auth }),
      fetch('/user/subscription/plans', { headers: auth }),
      fetch('/user/subscription/history', { headers: auth }),
    ])
    const subData = await subRes.json()
    if (subData.data?.subscription) currentSub.value = subData.data.subscription
    const pData = await plansRes.json()
    plans.value = pData.data || []
    const hData = await histRes.json()
    history.value = hData.data || []
  } catch (e) { console.error(e) }
}

async function subscribe(planId: string) {
  try {
    const plan = plans.value.find(p => p.id === planId)
    const msg = plan?.price > 0 ? `确认订阅 ¥${plan.price} 的「${plan.name}」？` : '确认开始免费试用？'
    await ElMessageBox.confirm(msg, '确认订阅', { type: 'info' })
    const res = await fetch('/user/subscription/subscribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('token')}` },
      body: JSON.stringify({ plan_id: planId }),
    })
    if (!res.ok) throw new Error((await res.json()).error)
    ElMessage.success('订阅成功！')
    fetchData()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e.message || e) }
}

async function cancelSub() {
  try {
    await ElMessageBox.confirm('确定取消自动续费？', '确认', { type: 'warning' })
    const res = await fetch('/user/subscription/cancel', {
      method: 'POST',
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
    })
    if (!res.ok) throw new Error((await res.json()).error)
    ElMessage.success('已取消')
    fetchData()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e.message || e) }
}

onMounted(fetchData)
</script>
