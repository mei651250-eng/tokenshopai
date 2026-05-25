<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center px-4">
    <div class="text-center">
      <!-- 404 -->
      <template v-if="errorCode === 404">
        <h1 class="text-8xl font-bold text-gray-200 dark:text-gray-700">404</h1>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white mt-4">{{ t('errors.notFound') }}</h2>
        <p class="text-gray-500 mt-2">{{ t('errors.notFoundDesc') }}</p>
      </template>
      <!-- 403 -->
      <template v-else-if="errorCode === 403">
        <h1 class="text-8xl font-bold text-gray-200 dark:text-gray-700">403</h1>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white mt-4">{{ t('errors.forbidden') }}</h2>
        <p class="text-gray-500 mt-2">{{ t('errors.forbiddenDesc') }}</p>
      </template>
      <!-- 500 -->
      <template v-else>
        <h1 class="text-8xl font-bold text-gray-200 dark:text-gray-700">500</h1>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white mt-4">{{ t('errors.serverError') }}</h2>
        <p class="text-gray-500 mt-2">{{ t('errors.serverErrorDesc') }}</p>
      </template>
      <div class="mt-8 flex items-center justify-center gap-3">
        <el-button type="primary" @click="router.push('/')">{{ t('errors.goHome') }}</el-button>
        <el-button @click="router.back()">{{ t('errors.goBack') }}</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const errorCode = computed(() => Number(route.params.code) || 404)
</script>
