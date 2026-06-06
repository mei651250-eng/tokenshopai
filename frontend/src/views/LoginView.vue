<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-600 via-brand to-primary-800">
    <div class="w-full max-w-md">
      <!-- Card -->
      <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl p-8 animate-fade-in-up">
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

        <!-- Footer -->
        <div class="mt-6 text-center text-sm text-gray-500">
          <router-link to="/" class="text-primary-600 hover:text-primary-700 font-medium">← 返回首页</router-link>
          <span class="mx-2">|</span>
          <span>还没有账号？</span>
          <a href="#" class="text-primary-600 hover:text-primary-700 font-medium">注册</a>
        </div>

        <!-- 其他登录方式 -->
        <el-divider>其他登录方式</el-divider>
        <div class="flex items-center justify-center gap-4">
          <router-link to="/login/face" class="text-indigo-600 hover:underline text-sm flex items-center gap-1">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg> 人脸识别登录
          </router-link>
          <span class="text-gray-300">|</span>
          <router-link to="/login/code" class="text-indigo-600 hover:underline text-sm flex items-center gap-1">
            <el-icon><Message /></el-icon> 验证码登录
          </router-link>
          <span class="text-gray-300">|</span>
          <a href="#" class="text-indigo-600 hover:underline text-sm flex items-center gap-1" @click.prevent="$router.push('/login/code')">
            🦊 Web3钱包登录
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const loading = ref(false)

const form = reactive({
  email: '',
  password: '',
})

async function handleLogin() {
  if (!form.email || !form.password) {
    ElMessage.warning('请填写邮箱和密码')
    return
  }
  loading.value = true
  try {
    await userStore.login(form.email, form.password)
    ElMessage.success('登录成功')
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e: any) {
    ElMessage.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
