<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 顶部导航栏 -->
    <header class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between items-center h-16">
          <!-- Logo -->
          <div class="flex items-center space-x-3">
            <router-link to="/distributor" class="flex items-center space-x-2">
              <div class="w-8 h-8 bg-gradient-to-br from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
                <span class="text-white font-bold text-sm">DH</span>
              </div>
              <span class="text-xl font-bold text-gray-900 dark:text-white">分销中心</span>
            </router-link>
          </div>

          <!-- 中间导航 -->
          <nav class="hidden md:flex items-center space-x-6">
            <router-link 
              v-for="item in navItems" 
              :key="item.path"
              :to="item.path"
              class="text-sm font-medium transition-colors"
              :class="isActive(item.path) 
                ? 'text-purple-600 dark:text-purple-400' 
                : 'text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white'"
            >
              {{ item.name }}
            </router-link>
          </nav>

          <!-- 右侧工具栏 -->
          <div class="flex items-center space-x-4">
            <!-- 余额显示 -->
            <div class="hidden sm:flex items-center space-x-2 px-3 py-1.5 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <svg class="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <span class="text-sm font-medium text-green-700 dark:text-green-400">¥{{ (balance / 100).toFixed(2) }}</span>
            </div>

            <!-- 语言切换 -->
            <el-dropdown trigger="click" @command="handleLocaleChange">
              <button class="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg transition-colors">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="#4b5563" stroke-width="2" stroke-linecap="round">
                  <circle cx="12" cy="12" r="10"/>
                  <path d="M2 12h20"/>
                  <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                </svg>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="zh-CN">简体中文</el-dropdown-item>
                  <el-dropdown-item command="en-US">English</el-dropdown-item>
                  <el-dropdown-item command="ja-JP">日本語</el-dropdown-item>
                  <el-dropdown-item command="ko-KR">한국어</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>

            <!-- 暗色模式 -->
            <button @click="toggleDark" class="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg transition-colors">
              <svg v-if="isDark" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/>
              </svg>
              <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
              </svg>
            </button>

            <!-- 用户菜单 -->
            <el-dropdown trigger="click">
              <div class="flex items-center space-x-2 cursor-pointer">
                <div class="w-8 h-8 bg-purple-100 dark:bg-purple-900 rounded-full flex items-center justify-center">
                  <span class="text-sm font-medium text-purple-600 dark:text-purple-300">{{ userInitial }}</span>
                </div>
                <span class="hidden sm:block text-sm font-medium text-gray-700 dark:text-gray-200">{{ username }}</span>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="router.push('/distributor/profile')">
                    <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                    </svg>
                    个人资料
                  </el-dropdown-item>
                  <el-dropdown-item divided @click="handleLogout">
                    <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                    </svg>
                    退出登录
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
    </header>

    <!-- 主内容区 -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <router-view />
    </main>

    <!-- 底部 -->
    <footer class="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 mt-auto">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex flex-col sm:flex-row justify-between items-center text-sm text-gray-500 dark:text-gray-400">
          <span>© 2026 TokenHub 分销中心. All rights reserved.</span>
          <div class="flex space-x-4 mt-2 sm:mt-0">
            <a href="/terms" class="hover:text-gray-700 dark:hover:text-gray-200">服务条款</a>
            <a href="/privacy" class="hover:text-gray-700 dark:hover:text-gray-200">隐私政策</a>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useDark, useToggle } from '@vueuse/core'

const router = useRouter()
const route = useRoute()
const { locale } = useI18n()
const userStore = useUserStore()

const isDark = useDark()
const toggleDark = useToggle(isDark)

const balance = ref(0)
const username = computed(() => userStore.user?.username || '分销商')
const userInitial = computed(() => username.value.charAt(0).toUpperCase())

const navItems = [
  { name: '仪表盘', path: '/distributor' },
  { name: '推广链接', path: '/distributor/links' },
  { name: '下级用户', path: '/distributor/referrals' },
  { name: '佣金记录', path: '/distributor/commissions' },
  { name: '提现', path: '/distributor/withdraw' },
  { name: '推广素材', path: '/distributor/materials' },
]

function isActive(path: string) {
  if (path === '/distributor') {
    return route.path === '/distributor'
  }
  return route.path.startsWith(path)
}

function handleLocaleChange(lang: string) {
  locale.value = lang
}

async function handleLogout() {
  await userStore.logout()
  router.push('/login')
}

onMounted(async () => {
  // 获取余额
  // const res = await distributorApi.getBalance()
  // balance.value = res.data.balance
})
</script>
