<template>
  <div class="skeleton-page">
    <!-- Header skeleton -->
    <div class="flex items-center justify-between mb-6">
      <div class="skeleton-line w-48 h-8 rounded" />
      <div class="skeleton-line w-28 h-9 rounded-lg" />
    </div>

    <!-- Stats cards skeleton -->
    <div v-if="showStats" class="grid gap-4 mb-6" :class="statsGridClass">
      <div v-for="i in statsCount" :key="i" class="p-4 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
        <div class="skeleton-line w-20 h-4 rounded mb-3" />
        <div class="skeleton-line w-16 h-7 rounded" />
      </div>
    </div>

    <!-- Table skeleton -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
      <!-- Table header -->
      <div class="flex gap-4 px-4 py-3 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
        <div v-for="i in columns" :key="'h'+i" class="skeleton-line rounded" :style="{ width: columnWidths[i-1] || '100px', height: '16px' }" />
      </div>
      <!-- Table rows -->
      <div v-for="r in rows" :key="'r'+r" class="flex gap-4 px-4 py-4 border-b border-gray-100 dark:border-gray-800 last:border-0">
        <div v-for="i in columns" :key="'c'+i" class="skeleton-line rounded" :style="{ width: columnWidths[i-1] || '100px', height: '16px' }" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  rows?: number
  columns?: number
  showStats?: boolean
  statsCount?: number
  columnWidths?: string[]
}>(), {
  rows: 5,
  columns: 5,
  showStats: true,
  statsCount: 4,
  columnWidths: () => ['180px', '120px', '100px', '80px', '150px'],
})

const statsGridClass = computed(() => {
  const cols = props.statsCount
  if (cols <= 2) return 'grid-cols-2'
  if (cols <= 3) return 'grid-cols-3'
  return 'grid-cols-4'
})
</script>

<style scoped>
.skeleton-line {
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
}

.dark .skeleton-line {
  background: linear-gradient(90deg, #374151 25%, #4b5563 50%, #374151 75%);
  background-size: 200% 100%;
}

@keyframes skeleton-loading {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
