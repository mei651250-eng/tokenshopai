<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('notification.title') }}</h1>
      <div class="flex gap-2">
        <el-button @click="markAllRead">{{ t('notification.markAllRead') }}</el-button>
        <el-button type="danger" plain @click="clearAll">{{ t('notification.clearAll') }}</el-button>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-3 mb-6">
      <el-radio-group v-model="filterType" size="small">
        <el-radio-button label="all">{{ t('notification.all') }}</el-radio-button>
        <el-radio-button label="unread">{{ t('notification.unread') }}</el-radio-button>
        <el-radio-button label="system">{{ t('notification.system') }}</el-radio-button>
        <el-radio-button label="alert">{{ t('notification.alert') }}</el-radio-button>
        <el-radio-button label="billing">{{ t('notification.billing') }}</el-radio-button>
        <el-radio-button label="security">{{ t('notification.security') }}</el-radio-button>
      </el-radio-group>
    </div>

    <!-- Notifications List -->
    <div class="space-y-3">
      <div
        v-for="n in filteredNotifications"
        :key="n.id"
        class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700 hover:shadow-md transition-shadow cursor-pointer"
        :class="{ 'border-l-4 border-l-primary-500': !n.read }"
        @click="handleClick(n)"
      >
        <div class="flex items-start gap-3">
          <!-- Level Indicator -->
          <div class="mt-1">
            <span
              class="w-3 h-3 rounded-full flex items-center justify-center"
              :class="{
                'bg-red-100': n.level === 'error',
                'bg-yellow-100': n.level === 'warning',
                'bg-blue-100': n.level === 'info',
                'bg-green-100': n.level === 'success',
              }"
            >
              <span
                class="w-2 h-2 rounded-full"
                :class="{
                  'bg-red-500': n.level === 'error',
                  'bg-yellow-500': n.level === 'warning',
                  'bg-blue-500': n.level === 'info',
                  'bg-green-500': n.level === 'success',
                }"
              />
            </span>
          </div>

          <!-- Content -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ n.title }}</h3>
              <el-tag :type="typeTagMap[n.type] || 'info'" size="small">{{ n.type }}</el-tag>
              <span v-if="!n.read" class="w-2 h-2 rounded-full bg-primary-500" />
            </div>
            <p class="text-sm text-gray-600 dark:text-gray-400">{{ n.message }}</p>
            <p class="text-xs text-gray-400 mt-2">{{ formatTime(n.createdAt) }}</p>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-1">
            <el-button v-if="!n.read" type="primary" link size="small" @click.stop="markRead(n.id)">{{ t('notification.markRead') }}</el-button>
            <el-button type="danger" link size="small" @click.stop="removeNotif(n.id)">
              <el-icon :size="14"><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="filteredNotifications.length === 0" class="text-center py-20">
        <el-icon :size="64" class="text-gray-300 dark:text-gray-600"><Bell /></el-icon>
        <p class="text-gray-400 mt-4">{{ t('notification.empty') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotificationStore } from '@/stores/notification'
import { useRouter } from 'vue-router'
import { Delete, Bell } from '@element-plus/icons-vue'
import type { Notification } from '@/stores/notification'

const { t } = useI18n()
const notifStore = useNotificationStore()
const router = useRouter()
const filterType = ref('all')

const typeTagMap: Record<string, string> = { system: '', alert: 'warning', billing: 'success', security: 'danger', model: 'info' }

const filteredNotifications = computed(() => {
  let list = notifStore.notifications
  if (filterType.value === 'unread') {
    list = list.filter(n => !n.read)
  } else if (filterType.value !== 'all') {
    list = list.filter(n => n.type === filterType.value)
  }
  return list
})

function handleClick(n: Notification) {
  notifStore.markAsRead(n.id)
  if (n.link) router.push(n.link)
}

function markRead(id: string) {
  notifStore.markAsRead(id)
}

function markAllRead() {
  notifStore.markAllAsRead()
}

function removeNotif(id: string) {
  notifStore.removeNotification(id)
}

function clearAll() {
  notifStore.clearAll()
}

function formatTime(ts: number) {
  const d = new Date(ts)
  const now = Date.now()
  const diff = now - ts
  if (diff < 60000) return t('notification.justNow')
  if (diff < 3600000) return `${Math.floor(diff / 60000)} ${t('notification.minutesAgo')}`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} ${t('notification.hoursAgo')}`
  return d.toLocaleString()
}

// Seed some demo notifications on mount
onMounted(() => {
  if (notifStore.notifications.length === 0) {
    const now = Date.now()
    ;[
      { type: 'system' as const, level: 'info' as const, title: t('notification.demoSystem'), message: t('notification.demoSystemMsg') },
      { type: 'alert' as const, level: 'warning' as const, title: t('notification.demoAlert'), message: t('notification.demoAlertMsg') },
      { type: 'billing' as const, level: 'success' as const, title: t('notification.demoBilling'), message: t('notification.demoBillingMsg') },
      { type: 'security' as const, level: 'error' as const, title: t('notification.demoSecurity'), message: t('notification.demoSecurityMsg') },
      { type: 'model' as const, level: 'info' as const, title: t('notification.demoModel'), message: t('notification.demoModelMsg') },
    ].forEach((n, i) => {
      notifStore.addNotification({ ...n, id: `demo-${i}`, read: i > 2, createdAt: now - i * 3600000 })
    })
  }
})
</script>
