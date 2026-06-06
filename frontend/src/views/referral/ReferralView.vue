<template>
  <div class="p-6 space-y-6">
    <!-- 页面头部 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">邀请奖励</h1>
        <p class="text-sm text-gray-500 mt-1">分享邀请链接，双方均可获得奖励</p>
      </div>
    </div>

    <!-- 邀请码卡片 -->
    <div class="bg-gradient-to-r from-primary-500 to-purple-600 rounded-2xl p-6 text-white">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm opacity-80">我的邀请码</p>
          <p class="text-2xl font-bold mt-1 font-mono">{{ myCode || '加载中...' }}</p>
          <p class="text-xs opacity-70 mt-2">每成功邀请一位好友，双方各得 ¥5 奖励</p>
        </div>
        <div class="flex gap-2">
          <el-button size="small" class="bg-white/20 border-white/40 text-white hover:bg-white/30" @click="copyCode">
            复制邀请码
          </el-button>
          <el-button size="small" class="bg-white/20 border-white/40 text-white hover:bg-white/30" @click="copyLink">
            复制邀请链接
          </el-button>
        </div>
      </div>
    </div>

    <!-- 统计 -->
    <div class="grid grid-cols-3 gap-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
        <p class="text-xs text-gray-500">累计邀请</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ stats.total }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
        <p class="text-xs text-gray-500">待发放</p>
        <p class="text-2xl font-bold text-amber-500 mt-1">{{ stats.pending }}</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700">
        <p class="text-xs text-gray-500">已获得奖励</p>
        <p class="text-2xl font-bold text-green-500 mt-1">¥{{ stats.rewarded }}</p>
      </div>
    </div>

    <!-- 邀请记录 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <div class="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <h3 class="font-medium text-gray-900 dark:text-white">邀请记录</h3>
      </div>
      <el-table :data="invites" stripe class="w-full">
        <el-table-column prop="invite_code" label="邀请码" width="160">
          <template #default="{ row }">
            <code class="text-xs bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded">{{ row.invite_code }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="invitee_id" label="被邀请人" width="200">
          <template #default="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-400 font-mono">{{ row.invitee_id?.slice(0, 8) }}...</span>
          </template>
        </el-table-column>
        <el-table-column prop="reward_amount" label="奖励金额" width="120">
          <template #default="{ row }">
            <span class="text-green-600 font-medium">¥{{ row.reward_amount }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'rewarded' ? 'success' : row.status === 'pending' ? 'warning' : 'info'" size="small">
              {{ row.status === 'rewarded' ? '已发放' : row.status === 'pending' ? '待发放' : '已过期' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="邀请时间">
          <template #default="{ row }">
            <span class="text-sm text-gray-500">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { adminApi } from '@/api'
import { ElMessage } from 'element-plus'

const myCode = ref('')
const invites = ref<any[]>([])

const stats = computed(() => {
  const total = invites.value.length
  const pending = invites.value.filter(i => i.status === 'pending').length
  const rewarded = invites.value.filter(i => i.status === 'rewarded').reduce((s, i) => s + (i.reward_amount || 0), 0)
  return { total, pending, rewarded }
})

async function loadData() {
  try {
    const [codeRes, refsRes]: any[] = await Promise.all([
      adminApi.getReferralCode(),
      adminApi.getReferrals(),
    ])
    myCode.value = codeRes.code || ''
    invites.value = refsRes.data || []
  } catch { /* ignore */ }
}

function copyCode() {
  navigator.clipboard.writeText(myCode.value)
  ElMessage.success('邀请码已复制')
}

function copyLink() {
  const link = `${window.location.origin}/#/login?invite=${myCode.value}`
  navigator.clipboard.writeText(link)
  ElMessage.success('邀请链接已复制')
}

function formatDate(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('zh-CN')
}

onMounted(loadData)
</script>
