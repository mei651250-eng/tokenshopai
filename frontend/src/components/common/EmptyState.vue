<template>
  <div class="flex flex-col items-center justify-center py-16 px-4">
    <!-- Illustration -->
    <div class="mb-6">
      <svg v-if="type === 'no-data'" width="160" height="160" viewBox="0 0 160 160" fill="none">
        <circle cx="80" cy="80" r="60" fill="#f3f4f6" />
        <rect x="55" y="55" width="50" height="35" rx="4" fill="#d1d5db" />
        <line x1="60" y1="65" x2="100" y2="65" stroke="#9ca3af" stroke-width="2" />
        <line x1="60" y1="73" x2="90" y2="73" stroke="#9ca3af" stroke-width="2" />
        <line x1="60" y1="81" x2="85" y2="81" stroke="#9ca3af" stroke-width="2" />
        <circle cx="112" cy="100" r="20" fill="#e5e7eb" />
        <line x1="105" y1="100" x2="119" y2="100" stroke="#9ca3af" stroke-width="2" stroke-linecap="round" />
        <line x1="112" y1="93" x2="112" y2="107" stroke="#9ca3af" stroke-width="2" stroke-linecap="round" />
      </svg>
      <svg v-else-if="type === 'no-result'" width="160" height="160" viewBox="0 0 160 160" fill="none">
        <circle cx="80" cy="80" r="60" fill="#f3f4f6" />
        <circle cx="65" cy="70" r="12" fill="#d1d5db" />
        <circle cx="95" cy="70" r="12" fill="#d1d5db" />
        <path d="M62 95 Q80 108 98 95" stroke="#9ca3af" stroke-width="3" fill="none" stroke-linecap="round" />
        <line x1="50" y1="45" x2="70" y2="55" stroke="#9ca3af" stroke-width="2" stroke-linecap="round" />
        <line x1="110" y1="45" x2="90" y2="55" stroke="#9ca3af" stroke-width="2" stroke-linecap="round" />
      </svg>
      <svg v-else-if="type === 'error'" width="160" height="160" viewBox="0 0 160 160" fill="none">
        <circle cx="80" cy="80" r="60" fill="#fef2f2" />
        <circle cx="80" cy="80" r="30" fill="#fecaca" />
        <text x="80" y="90" text-anchor="middle" fill="#ef4444" font-size="32" font-weight="bold">!</text>
      </svg>
      <svg v-else-if="type === 'notification'" width="160" height="160" viewBox="0 0 160 160" fill="none">
        <circle cx="80" cy="80" r="60" fill="#f3f4f6" />
        <path d="M65 60 Q65 85 55 95 h50 Q95 85 95 60 Z" fill="#d1d5db" />
        <circle cx="80" cy="50" r="5" fill="#9ca3af" />
        <rect x="75" y="100" width="10" height="10" rx="5" fill="#9ca3af" />
      </svg>
      <svg v-else width="160" height="160" viewBox="0 0 160 160" fill="none">
        <circle cx="80" cy="80" r="60" fill="#f3f4f6" />
        <rect x="55" y="55" width="50" height="50" rx="8" fill="#d1d5db" />
        <path d="M72 80 L78 86 L88 74" stroke="#9ca3af" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </div>

    <!-- Text -->
    <h3 class="text-lg font-medium text-gray-700 dark:text-gray-300 mb-2">{{ title || defaultTitle }}</h3>
    <p v-if="description" class="text-sm text-gray-500 dark:text-gray-400 mb-6 max-w-md text-center">{{ description }}</p>

    <!-- Action -->
    <slot name="action">
      <el-button v-if="actionText" type="primary" @click="$emit('action')">{{ actionText }}</el-button>
    </slot>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  type?: 'no-data' | 'no-result' | 'error' | 'notification' | 'success'
  title?: string
  description?: string
  actionText?: string
}>()

defineEmits<{
  action: []
}>()

const { t } = useI18n()

const defaultTitle = computed(() => {
  switch (props.type) {
    case 'no-data': return t('common.noData')
    case 'no-result': return t('common.noResult')
    case 'error': return t('common.loadError')
    case 'notification': return t('common.noNotifications')
    case 'success': return t('common.success')
    default: return t('common.noData')
  }
})
</script>
