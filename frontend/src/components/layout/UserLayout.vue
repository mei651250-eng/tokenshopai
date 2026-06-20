<template>
  <div class="flex h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Sidebar -->
    <aside
      class="bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col w-60"
    >
      <!-- Logo -->
      <div class="h-16 flex items-center px-6 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-brand flex items-center justify-center">
            <span class="text-white font-bold text-sm">T</span>
          </div>
          <span class="font-bold text-lg text-gray-900 dark:text-white">TokenHub</span>
        </div>
      </div>

      <!-- Nav -->
      <nav class="flex-1 py-4 overflow-y-auto">
        <div v-for="item in navItems" :key="item.path">
          <router-link
            :to="item.path"
            class="flex items-center gap-3 px-6 py-2.5 text-sm transition-colors"
            :class="isActive(item.path)
              ? 'bg-primary-50 dark:bg-primary-900/20 text-primary-600 dark:text-primary-400 font-medium border-r-2 border-primary-600'
              : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
          >
            <el-icon :size="18"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </nav>

      <!-- Admin Entry (only for admin roles) -->
      <div v-if="isAdmin" class="border-t border-gray-200 dark:border-gray-700 p-3">
        <router-link
          to="/admin/dashboard"
          class="flex items-center gap-3 px-3 py-2 text-sm text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 rounded-lg transition-colors"
        >
          <el-icon :size="16"><Setting /></el-icon>
          <span>管理后台</span>
        </router-link>
      </div>
    </aside>

    <!-- Main Area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Top Header Bar -->
      <header class="h-14 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center px-4 gap-3 flex-shrink-0">
        <!-- Page Title -->
        <h1 class="text-base font-semibold text-gray-900 dark:text-white flex-1">{{ currentPageTitle }}</h1>

        <!-- Balance Badge -->
        <div class="flex items-center gap-1.5 px-3 py-1 bg-green-50 dark:bg-green-900/20 rounded-full">
          <el-icon :size="14" class="text-green-600"><Wallet /></el-icon>
          <span class="text-sm font-medium text-green-700 dark:text-green-400">¥{{ (balance / 100).toFixed(2) }}</span>
        </div>

        <!-- Language Switcher -->
        <el-dropdown trigger="click" @command="handleLocaleChange">
          <button class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="#4b5563" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="zh-CN">简体中文</el-dropdown-item>
              <el-dropdown-item command="zh-TW">繁體中文</el-dropdown-item>
              <el-dropdown-item command="en-US">English</el-dropdown-item>
              <el-dropdown-item command="ja-JP">日本語</el-dropdown-item>
              <el-dropdown-item command="ko-KR">한국어</el-dropdown-item>
              <el-dropdown-item command="fr-FR">Français</el-dropdown-item>
              <el-dropdown-item command="de-DE">Deutsch</el-dropdown-item>
              <el-dropdown-item command="es-ES">Español</el-dropdown-item>
              <el-dropdown-item command="pt-BR">Português</el-dropdown-item>
              <el-dropdown-item command="it-IT">Italiano</el-dropdown-item>
              <el-dropdown-item command="ru-RU">Русский</el-dropdown-item>
              <el-dropdown-item command="ar-SA">العربية</el-dropdown-item>
              <el-dropdown-item command="hi-IN">हिन्दी</el-dropdown-item>
              <el-dropdown-item command="id-ID">Bahasa Indonesia</el-dropdown-item>
              <el-dropdown-item command="vi-VN">Tiếng Việt</el-dropdown-item>
              <el-dropdown-item command="th-TH">ไทย</el-dropdown-item>
              <el-dropdown-item command="tr-TR">Türkçe</el-dropdown-item>
              <el-dropdown-item command="nl-NL">Nederlands</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <!-- Dark Mode Toggle -->
        <el-tooltip :content="appStore.darkMode ? '浅色模式' : '深色模式'" placement="bottom">
          <button
            class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            @click="appStore.toggleDarkMode()"
          >
            <el-icon :size="18">
              <Sunny v-if="appStore.darkMode" />
              <Moon v-else />
            </el-icon>
          </button>
        </el-tooltip>

        <!-- Notification Bell -->
        <el-popover placement="bottom-end" :width="340" trigger="click">
          <template #reference>
            <button class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors relative">
              <el-icon :size="18"><Bell /></el-icon>
              <span v-if="notifStore.unreadCount > 0" class="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] bg-red-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center px-1">
                {{ notifStore.unreadCount > 99 ? '99+' : notifStore.unreadCount }}
              </span>
            </button>
          </template>
          <div class="max-h-80 overflow-y-auto">
            <div v-if="notifStore.notifications.length === 0" class="py-8 text-center text-gray-400 text-sm">
              暂无通知
            </div>
            <div
              v-for="n in notifStore.notifications.slice(0, 8)"
              :key="n.id"
              class="px-3 py-2 border-b border-gray-50 dark:border-gray-700/50 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer transition-colors"
              :class="{ 'bg-blue-50/50 dark:bg-blue-900/10': !n.read }"
              @click="notifStore.markAsRead(n.id)"
            >
              <p class="text-sm text-gray-900 dark:text-white truncate">{{ n.title }}</p>
              <p class="text-xs text-gray-500 mt-0.5 line-clamp-1">{{ n.message }}</p>
            </div>
          </div>
        </el-popover>

        <!-- User Dropdown -->
        <el-dropdown trigger="click" @command="handleUserCommand">
          <div class="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors">
            <div class="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
              <span class="text-primary-600 text-sm font-medium">{{ userStore.email?.charAt(0).toUpperCase() }}</span>
            </div>
            <div class="hidden sm:block min-w-0">
              <p class="text-sm font-medium text-gray-900 dark:text-white truncate max-w-[100px]">{{ userStore.email }}</p>
            </div>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon class="mr-2"><User /></el-icon>个人中心
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">
                <el-icon class="mr-2"><SwitchButton /></el-icon>退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </header>

      <!-- Main Content -->
      <main class="flex-1 overflow-auto">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { useNotificationStore } from '@/stores/notification'
import { adminApi, userApi } from '@/api'
import {
  DataBoard, Key, ShoppingCart, Reading, CreditCard,
  Wallet, Money, Bell, User, SwitchButton,
  Sunny, Moon, Share, Setting, TrendCharts,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const { locale: i18nLocale } = useI18n()
const userStore = useUserStore()
const appStore = useAppStore()
const notifStore = useNotificationStore()

const balance = ref(0)

const navItems = computed(() => [
  { path: '/home', label: '控制台', icon: DataBoard },
  { path: '/apikeys', label: 'API 密钥', icon: Key },
  { path: '/usage', label: '用量明细', icon: TrendCharts },
  { path: '/marketplace', label: '模型广场', icon: ShoppingCart },
  { path: '/docs', label: 'API 文档', icon: Reading },
  { path: '/topup', label: '在线充值', icon: CreditCard },
  { path: '/billing', label: '账单', icon: Money },
  { path: '/wallet', label: '钱包', icon: Wallet },
  { path: '/referrals', label: '邀请奖励', icon: Share },
])

const isAdmin = computed(() => {
  const role = userStore.role
  return role === 'super_admin' || role === 'tenant_admin'
})

const currentPageTitle = computed(() => {
  const item = navItems.value.find(n => route.path.startsWith(n.path) && n.path !== '/')
  return item?.label || 'TokenHub'
})

function isActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

function handleUserCommand(cmd: string) {
  if (cmd === 'profile') router.push('/profile')
  else if (cmd === 'logout') {
    userStore.logout()
    router.push('/login')
  }
}

function handleLocaleChange(locale: string) {
  i18nLocale.value = locale
  userStore.setLocale(locale)
}

async function loadBalance() {
  try {
    const res: any = await userApi.getBalance()
    balance.value = res.balance || res.data?.balance || 0
  } catch { /* ignore */ }
}

onMounted(() => {
  loadBalance()
})
</script>
