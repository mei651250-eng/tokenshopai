<template>
  <div class="flex h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Sidebar -->
    <aside
      class="bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col transition-all duration-300"
      :class="appStore.sidebarCollapsed ? 'w-16' : 'w-60'"
    >
      <!-- Logo -->
      <div class="h-16 flex items-center border-b border-gray-200 dark:border-gray-700" :class="appStore.sidebarCollapsed ? 'justify-center px-2' : 'px-6'">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-brand flex items-center justify-center flex-shrink-0">
            <span class="text-white font-bold text-sm">T</span>
          </div>
          <span v-if="!appStore.sidebarCollapsed" class="font-bold text-lg text-gray-900 dark:text-white">TokenHub</span>
        </div>
      </div>

      <!-- Nav -->
      <nav class="flex-1 py-4 overflow-y-auto">
        <div v-for="item in navItems" :key="item.path">
          <el-tooltip v-if="appStore.sidebarCollapsed" :content="item.label" placement="right" :show-after="200">
            <router-link
              :to="item.path"
              class="flex items-center justify-center py-2.5 text-sm transition-colors"
              :class="isActive(item.path)
                ? 'bg-primary-50 dark:bg-primary-900/20 text-primary-600 dark:text-primary-400 font-medium border-r-2 border-primary-600'
                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
            >
              <el-icon :size="20"><component :is="item.icon" /></el-icon>
            </router-link>
          </el-tooltip>
          <router-link
            v-else
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

      <!-- Collapse Toggle -->
      <div class="border-t border-gray-200 dark:border-gray-700 p-2">
        <button
          class="w-full flex items-center justify-center gap-2 py-2 text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          @click="appStore.toggleSidebar()"
        >
          <el-icon :size="18">
            <Fold v-if="!appStore.sidebarCollapsed" />
            <Expand v-else />
          </el-icon>
          <span v-if="!appStore.sidebarCollapsed">{{ t('nav.collapse') }}</span>
        </button>
      </div>
    </aside>

    <!-- Main Area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Top Header Bar -->
      <header class="h-14 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center px-4 gap-3 flex-shrink-0">
        <!-- Breadcrumb -->
        <el-breadcrumb separator="/" class="flex-1">
          <el-breadcrumb-item :to="{ path: '/' }">
            <span class="text-gray-500 dark:text-gray-400">{{ t('nav.home') }}</span>
          </el-breadcrumb-item>
          <el-breadcrumb-item v-for="(crumb, i) in breadcrumbs" :key="i">
            <router-link v-if="crumb.path && i < breadcrumbs.length - 1" :to="crumb.path" class="text-gray-600 dark:text-gray-300 hover:text-primary-600">
              {{ crumb.label }}
            </router-link>
            <span v-else class="text-gray-900 dark:text-white font-medium">{{ crumb.label }}</span>
          </el-breadcrumb-item>
        </el-breadcrumb>

        <!-- Search Trigger -->
        <el-tooltip :content="t('nav.searchTip')" placement="bottom">
          <button
            class="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-500 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            @click="showSearch = true"
          >
            <el-icon :size="14"><Search /></el-icon>
            <span class="hidden sm:inline">{{ t('nav.search') }}</span>
            <kbd class="hidden sm:inline text-xs bg-gray-200 dark:bg-gray-600 px-1.5 py-0.5 rounded">⌘K</kbd>
          </button>
        </el-tooltip>

        <!-- Language Switcher -->
        <el-dropdown trigger="click" @command="handleLocaleChange">
          <button class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
            <el-icon :size="18"><Globe /></el-icon>
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
        <el-tooltip :content="appStore.darkMode ? t('nav.lightMode') : t('nav.darkMode')" placement="bottom">
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

        <!-- Fullscreen Toggle -->
        <el-tooltip :content="t('nav.fullscreen')" placement="bottom">
          <button
            class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors hidden sm:block"
            @click="toggleFullscreen"
          >
            <el-icon :size="18"><FullScreen /></el-icon>
          </button>
        </el-tooltip>

        <!-- Notification Bell -->
        <el-popover placement="bottom-end" :width="380" trigger="click">
          <template #reference>
            <button class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors relative">
              <el-icon :size="18"><Bell /></el-icon>
              <span v-if="notifStore.unreadCount > 0" class="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] bg-red-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center px-1">
                {{ notifStore.unreadCount > 99 ? '99+' : notifStore.unreadCount }}
              </span>
            </button>
          </template>
          <div class="max-h-96 overflow-y-auto">
            <div class="flex items-center justify-between px-3 py-2 border-b border-gray-100 dark:border-gray-700">
              <span class="font-semibold text-sm text-gray-900 dark:text-white">{{ t('notification.title') }}</span>
              <button class="text-xs text-primary-600 hover:underline" @click="notifStore.markAllAsRead()">{{ t('notification.markAllRead') }}</button>
            </div>
            <div v-if="notifStore.notifications.length === 0" class="py-8 text-center text-gray-400 text-sm">
              {{ t('notification.empty') }}
            </div>
            <div
              v-for="n in notifStore.notifications.slice(0, 10)"
              :key="n.id"
              class="px-3 py-2.5 border-b border-gray-50 dark:border-gray-700/50 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer transition-colors"
              :class="{ 'bg-blue-50/50 dark:bg-blue-900/10': !n.read }"
              @click="notifStore.markAsRead(n.id)"
            >
              <div class="flex items-start gap-2">
                <span
                  class="w-2 h-2 rounded-full mt-1.5 flex-shrink-0"
                  :class="{
                    'bg-red-500': n.level === 'error',
                    'bg-yellow-500': n.level === 'warning',
                    'bg-blue-500': n.level === 'info',
                    'bg-green-500': n.level === 'success',
                  }"
                />
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ n.title }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-2">{{ n.message }}</p>
                  <p class="text-[10px] text-gray-400 mt-1">{{ formatTime(n.createdAt) }}</p>
                </div>
              </div>
            </div>
            <div v-if="notifStore.notifications.length > 0" class="px-3 py-2 text-center">
              <router-link to="/notifications" class="text-xs text-primary-600 hover:underline">{{ t('notification.viewAll') }}</router-link>
            </div>
          </div>
        </el-popover>

        <!-- User Dropdown -->
        <el-dropdown trigger="click" @command="handleUserCommand">
          <div class="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors">
            <div class="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
              <span class="text-primary-600 text-sm font-medium">{{ userStore.email.charAt(0).toUpperCase() }}</span>
            </div>
            <div class="hidden sm:block min-w-0">
              <p class="text-sm font-medium text-gray-900 dark:text-white truncate max-w-[120px]">{{ userStore.email }}</p>
              <p class="text-[10px] text-gray-500">{{ userStore.role }}</p>
            </div>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon class="mr-2"><User /></el-icon>{{ t('nav.profile') }}
              </el-dropdown-item>
              <el-dropdown-item command="userportal">
                <el-icon class="mr-2"><ShoppingCart /></el-icon>返回用户端
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">
                <el-icon class="mr-2"><SwitchButton /></el-icon>{{ t('nav.logout') }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </header>

      <!-- Multi-tab Navigation -->
      <div v-if="appStore.openedTabs.length > 0" class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center px-2 py-1 gap-1 overflow-x-auto flex-shrink-0">
        <div
          v-for="tab in appStore.openedTabs"
          :key="tab.path"
          class="flex items-center gap-1 px-3 py-1 text-xs rounded-md cursor-pointer whitespace-nowrap transition-colors"
          :class="route.path === tab.path
            ? 'bg-primary-50 dark:bg-primary-900/20 text-primary-600 dark:text-primary-400 font-medium'
            : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'"
          @click="router.push(tab.path)"
        >
          <span>{{ tab.label }}</span>
          <el-icon
            v-if="tab.path !== '/'"
            :size="12"
            class="hover:text-red-500 ml-0.5"
            @click.stop="closeTab(tab.path)"
          >
            <Close />
          </el-icon>
        </div>
        <button
          v-if="appStore.openedTabs.length > 1"
          class="ml-auto p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          @click="appStore.clearTabs()"
        >
          <el-icon :size="14"><Close /></el-icon>
        </button>
      </div>

      <!-- Main Content -->
      <main class="flex-1 overflow-auto">
        <router-view />
      </main>
    </div>

    <!-- Global Search Dialog -->
    <el-dialog v-model="showSearch" :show-close="false" width="520" top="15vh" class="search-dialog">
      <div class="relative">
        <el-input
          v-model="searchQuery"
          :placeholder="t('nav.searchPlaceholder')"
          size="large"
          prefix-icon="Search"
          autofocus
          clearable
        />
      </div>
      <div class="mt-3 max-h-80 overflow-y-auto">
        <div v-for="group in filteredSearchResults" :key="group.label" class="mb-3">
          <p class="text-xs text-gray-400 px-2 mb-1">{{ group.label }}</p>
          <div
            v-for="item in group.items"
            :key="item.path"
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            @click="navigateTo(item.path)"
          >
            <el-icon :size="16" class="text-gray-400"><component :is="item.icon" /></el-icon>
            <div>
              <p class="text-sm text-gray-900 dark:text-white">{{ item.label }}</p>
              <p v-if="item.desc" class="text-xs text-gray-400">{{ item.desc }}</p>
            </div>
          </div>
        </div>
        <div v-if="searchQuery && filteredSearchResults.length === 0" class="py-8 text-center text-gray-400 text-sm">
          {{ t('nav.noResults') }}
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { useNotificationStore } from '@/stores/notification'
import {
  DataBoard, SetUp, Money, OfficeBuilding,
  Lock, Monitor, Document, Setting, MoreFilled,
  Wallet, CreditCard, Sell, Stamp,
  Fold, Expand, Search, Sunny, Moon, FullScreen,
  Bell, User, SwitchButton, Close,
  User as UserIcon, Key, Tickets, Promotion,
  TrendCharts, Histogram, Connection, CircleCheck,
  Reading, ShoppingCart, Present, Notification, Operation, UserFilled, Share,
  Globe,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const { t, locale: i18nLocale } = useI18n()
const userStore = useUserStore()
const appStore = useAppStore()
const notifStore = useNotificationStore()

const showSearch = ref(false)
const searchQuery = ref('')

const allNavItems = [
  { path: '/admin/dashboard', label: t('nav.dashboard'), icon: DataBoard, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/models', label: t('nav.models'), icon: SetUp, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/channels', label: '渠道管理', icon: Connection, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/tokens', label: '令牌管理', icon: Key, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/tenants', label: t('nav.tenants'), icon: OfficeBuilding, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/users', label: t('nav.users'), icon: UserIcon, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/quota', label: '配额管理', icon: Histogram, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/reconciliation', label: '对账中心', icon: TrendCharts, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/distribution', label: '分销管理', icon: Connection, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/redeem-codes', label: '兑换码', icon: Present, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/announcements', label: '公告管理', icon: Notification, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/model-mappings', label: '模型映射', icon: Operation, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/user-groups', label: '用户组', icon: UserFilled, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/refund', label: '退款管理', icon: CircleCheck, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/audit', label: t('nav.audit'), icon: Tickets, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/settings', label: t('nav.settings'), icon: Setting, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/security', label: t('nav.security'), icon: Lock, roles: ['super_admin'] },
  { path: '/admin/monitor', label: t('nav.monitor'), icon: Monitor, roles: ['super_admin', 'tenant_admin'] },
  { path: '/admin/reports', label: t('nav.reports'), icon: Document, roles: ['super_admin', 'tenant_admin'] },
]

const navItems = computed(() => {
  const role = userStore.role || ''
  return allNavItems.filter(item => item.roles.includes(role))
})

const breadcrumbMap: Record<string, { label: string; path?: string }> = {}
// Will be populated from navItems dynamically

const breadcrumbs = computed(() => {
  const map: Record<string, { label: string; path?: string }> = {}
  navItems.value.forEach(item => {
    map[item.path] = { label: item.label, path: item.path }
  })
  // Special routes
  map['/models'] = { label: t('nav.models'), path: '/models' }
  map['/billing/transactions'] = { label: t('nav.transactions'), path: '/billing/transactions' }

  const parts = route.path.split('/').filter(Boolean)
  const result: Array<{ label: string; path?: string }> = []
  let currentPath = ''
  for (const part of parts) {
    currentPath += '/' + part
    if (map[currentPath]) {
      result.push(map[currentPath])
    } else {
      result.push({ label: part, path: currentPath })
    }
  }
  // Handle root
  if (route.path === '/') {
    return [{ label: t('nav.dashboard') }]
  }
  return result
})

// Tab management: auto-add current route
watch(() => route.path, (path) => {
  if (path === '/') return
  const item = navItems.value.find(n => path.startsWith(n.path) && n.path !== '/')
  if (item) {
    appStore.addTab({ path, name: item.label, label: item.label })
  } else if (path.includes('/models/')) {
    appStore.addTab({ path, name: 'ModelDetail', label: t('nav.modelDetail') })
  } else if (path.includes('/tenants/')) {
    appStore.addTab({ path, name: 'TenantDetail', label: t('nav.tenantDetail') })
  } else if (path.includes('/billing/transactions')) {
    appStore.addTab({ path, name: 'Transactions', label: t('nav.transactions') })
  } else {
    const label = path.split('/').pop() || path
    appStore.addTab({ path, name: label, label: label.charAt(0).toUpperCase() + label.slice(1) })
  }
}, { immediate: true })

function closeTab(path: string) {
  appStore.removeTab(path)
  if (route.path === path) {
    const remaining = appStore.openedTabs
    router.push(remaining.length > 0 ? remaining[remaining.length - 1].path : '/dashboard')
  }
}

function isActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

function handleLogout() {
  userStore.logout()
  router.push('/login')
}

function handleUserCommand(cmd: string) {
  if (cmd === 'profile') router.push('/profile')
  else if (cmd === 'userportal') router.push('/home')
  else if (cmd === 'logout') handleLogout()
}

function handleLocaleChange(locale: string) {
  i18nLocale.value = locale
  userStore.setLocale(locale)
}

function toggleFullscreen() {
  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    document.documentElement.requestFullscreen()
  }
}

function formatTime(ts: number) {
  const d = new Date(ts)
  const now = Date.now()
  const diff = now - ts
  if (diff < 60000) return t('notification.justNow')
  if (diff < 3600000) return `${Math.floor(diff / 60000)} ${t('notification.minutesAgo')}`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} ${t('notification.hoursAgo')}`
  return d.toLocaleDateString()
}

// Global search
const searchItems = computed(() => {
  const items: Array<{ label: string; desc: string; path: string; icon: any; group: string }> = []
  navItems.value.forEach(n => {
    if (n.path !== '/') {
      items.push({ label: n.label, desc: n.path, path: n.path, icon: n.icon, group: t('nav.searchPages') })
    }
  })
  items.push({ label: t('nav.profile'), desc: t('nav.profileDesc'), path: '/profile', icon: User, group: t('nav.searchActions') })
  items.push({ label: t('nav.darkMode'), desc: t('nav.darkModeDesc'), path: '__dark__', icon: Moon, group: t('nav.searchActions') })
  items.push({ label: t('nav.fullscreen'), desc: t('nav.fullscreenDesc'), path: '__fullscreen__', icon: FullScreen, group: t('nav.searchActions') })
  return items
})

const filteredSearchResults = computed(() => {
  if (!searchQuery.value) {
    // Show recent/suggested
    const pages = searchItems.value.filter(i => i.group === t('nav.searchPages')).slice(0, 6)
    return [{ label: t('nav.searchPages'), items: pages }]
  }
  const q = searchQuery.value.toLowerCase()
  const filtered = searchItems.value.filter(i => i.label.toLowerCase().includes(q) || i.desc.toLowerCase().includes(q))
  const groups: Record<string, Array<typeof filtered[0]>> = {}
  filtered.forEach(i => {
    if (!groups[i.group]) groups[i.group] = []
    groups[i.group].push(i)
  })
  return Object.entries(groups).map(([label, items]) => ({ label, items }))
})

function navigateTo(path: string) {
  showSearch.value = false
  searchQuery.value = ''
  if (path === '__dark__') { appStore.toggleDarkMode(); return }
  if (path === '__fullscreen__') { toggleFullscreen(); return }
  router.push(path)
}

// Keyboard shortcut
function handleKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    showSearch.value = !showSearch.value
  }
  if (e.key === 'Escape' && showSearch.value) {
    showSearch.value = false
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
  // Connect notification WebSocket
  const baseUrl = import.meta.env.VITE_API_BASE_URL || ''
  if (baseUrl) notifStore.connectWs(baseUrl)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  notifStore.disconnectWs()
})
</script>

<style scoped>
.search-dialog :deep(.el-dialog__header) {
  display: none;
}
.search-dialog :deep(.el-dialog__body) {
  padding: 16px 20px;
}
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
