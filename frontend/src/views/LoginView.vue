<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-600 via-brand to-primary-800">
    <div class="w-full max-w-md">
      <!-- Card -->
      <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl p-8 animate-fade-in-up">
        <!-- Language Switcher -->
        <div class="flex justify-end mb-2">
          <el-dropdown trigger="click" @command="handleLocaleChange">
            <button class="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 rounded-lg transition-colors">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="#4b5563" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="zh-CN">简体中文</el-dropdown-item>
                <el-dropdown-item command="en-US">English</el-dropdown-item>
                <el-dropdown-item command="ja-JP">日本語</el-dropdown-item>
                <el-dropdown-item command="ko-KR">한국어</el-dropdown-item>
                <el-dropdown-item command="zh-TW">繁體中文</el-dropdown-item>
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
        </div>
        <!-- Header -->
        <div class="text-center mb-8">
          <router-link to="/" class="inline-block">
            <div class="w-16 h-16 rounded-2xl bg-brand mx-auto flex items-center justify-center mb-4">
              <span class="text-white text-2xl font-bold">T</span>
            </div>
          </router-link>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">TokenHub</h1>
          <p class="text-gray-500 mt-1">企业级 AI API 网关</p>
        </div>

        <!-- Form -->
        <el-form :model="form" @submit.prevent="handleLogin" label-position="top">
          <el-form-item label="邮箱">
            <el-input
              v-model="form.email"
              placeholder="请输入邮箱"
              prefix-icon="Message"
              size="large"
            />
          </el-form-item>

          <el-form-item label="密码">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              prefix-icon="Lock"
              size="large"
              show-password
            />
          </el-form-item>

          <el-button
            type="primary"
            size="large"
            class="w-full mt-2"
            :loading="loading"
            native-type="submit"
          >
            登录
          </el-button>
        </el-form>

        <!-- OAuth -->
        <el-divider>其他登录方式</el-divider>
        <div class="flex gap-3 mb-4">
          <button
            class="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            @click="oauthLogin('github')"
          >
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>
            GitHub
          </button>
          <button
            class="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            @click="oauthLogin('google')"
          >
            <svg class="w-5 h-5" viewBox="0 0 24 24"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>
            Google
          </button>
        </div>

        <!-- 更多登录方式 -->
        <div class="flex items-center justify-center gap-4 text-xs">
          <router-link to="/login/face" class="text-indigo-600 hover:underline flex items-center gap-1">
            人脸识别
          </router-link>
          <span class="text-gray-300">|</span>
          <router-link to="/login/code" class="text-indigo-600 hover:underline flex items-center gap-1">
            验证码登录
          </router-link>
          <span class="text-gray-300">|</span>
          <a href="#" class="text-indigo-600 hover:underline flex items-center gap-1" @click.prevent="$router.push('/login/code')">
            Web3钱包
          </a>
        </div>

        <!-- Footer -->
        <div class="mt-4 text-center text-sm text-gray-500">
          <router-link to="/" class="text-primary-600 hover:text-primary-700 font-medium">返回首页</router-link>
          <span class="mx-2">|</span>
          <span>还没有账号？</span>
          <router-link to="/register" class="text-primary-600 hover:text-primary-700 font-medium">注册</router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const { locale } = useI18n()
const userStore = useUserStore()
const loading = ref(false)

function handleLocaleChange(localeCode: string) {
  locale.value = localeCode
}

const form = reactive({
  email: '',
  password: '',
})

function oauthLogin(provider: string) {
  window.location.href = `/auth/oauth/${provider}`
}

async function handleLogin() {
  if (!form.email || !form.password) {
    ElMessage.warning('请填写邮箱和密码')
    return
  }
  loading.value = true
  try {
    await userStore.login(form.email, form.password)
    ElMessage.success('登录成功')
    // 根据角色分流：管理员进管理端，普通用户进用户端
    const role = localStorage.getItem('role') || ''
    const isAdmin = role === 'super_admin' || role === 'tenant_admin'
    const defaultPath = isAdmin ? '/admin/dashboard' : '/home'
    const redirect = (route.query.redirect as string) || defaultPath
    router.push(redirect)
  } catch (e: any) {
    ElMessage.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
