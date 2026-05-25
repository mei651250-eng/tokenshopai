<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-600 via-purple-600 to-pink-500">
    <div class="w-full max-w-md">
      <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl p-8 animate-fade-in-up">
        <!-- Header -->
        <div class="text-center mb-8">
          <div class="w-20 h-20 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 mx-auto flex items-center justify-center mb-4 shadow-lg">
            <svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">人脸识别登录</h1>
          <p class="text-gray-500 mt-1">使用 Windows Hello / Touch ID / Face ID 快速登录</p>
        </div>

        <!-- Step 1: 输入邮箱 -->
        <div v-if="step === 'email'">
          <el-form @submit.prevent="requestAuth">
            <el-form-item label="邮箱">
              <el-input
                v-model="email"
                placeholder="请输入注册邮箱"
                prefix-icon="Message"
                size="large"
              />
            </el-form-item>
            <el-button
              type="primary"
              size="large"
              class="w-full mt-2"
              :loading="loading"
              :disabled="!email"
              native-type="submit"
            >
              下一步：人脸识别
            </el-button>
          </el-form>
        </div>

        <!-- Step 2: 人脸识别进行中 -->
        <div v-else-if="step === 'authenticating'" class="text-center py-8">
          <div class="face-scanner mx-auto mb-6">
            <div class="face-ring outer"></div>
            <div class="face-ring inner"></div>
            <svg class="w-16 h-16 text-white absolute" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="top: 50%; left: 50%; transform: translate(-50%, -50%);">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
          </div>
          <p class="text-gray-600 dark:text-gray-300 text-lg mb-2">正在等待人脸识别...</p>
          <p class="text-gray-400 text-sm">请看向摄像头或使用指纹传感器</p>
        </div>

        <!-- Step 3: 成功 -->
        <div v-else-if="step === 'success'" class="text-center py-8">
          <div class="w-20 h-20 rounded-full bg-green-100 mx-auto flex items-center justify-center mb-4">
            <svg class="w-10 h-10 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <p class="text-green-600 text-lg font-medium">认证成功，正在跳转...</p>
        </div>

        <!-- Step 4: 失败 -->
        <div v-else-if="step === 'error'" class="text-center py-8">
          <div class="w-20 h-20 rounded-full bg-red-100 mx-auto flex items-center justify-center mb-4">
            <svg class="w-10 h-10 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <p class="text-red-600 text-lg font-medium mb-2">认证失败</p>
          <p class="text-gray-500 text-sm mb-4">{{ errorMsg }}</p>
          <el-button type="primary" @click="reset">重试</el-button>
        </div>

        <!-- 不支持提示 -->
        <div v-else-if="step === 'unsupported'" class="text-center py-8">
          <div class="w-20 h-20 rounded-full bg-yellow-100 mx-auto flex items-center justify-center mb-4">
            <svg class="w-10 h-10 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
          </div>
          <p class="text-yellow-600 text-lg font-medium mb-2">不支持人脸识别</p>
          <p class="text-gray-500 text-sm mb-4">您的浏览器或设备不支持 WebAuthn，请使用其他登录方式</p>
          <router-link to="/login">
            <el-button type="primary">返回密码登录</el-button>
          </router-link>
        </div>

        <!-- Footer -->
        <div class="mt-6">
          <el-divider>其他登录方式</el-divider>
          <div class="flex items-center justify-center gap-4">
            <router-link to="/login" class="text-indigo-600 hover:underline text-sm flex items-center gap-1">
              <el-icon><Lock /></el-icon> 密码登录
            </router-link>
            <span class="text-gray-300">|</span>
            <router-link to="/login/code" class="text-indigo-600 hover:underline text-sm flex items-center gap-1">
              <el-icon><Message /></el-icon> 验证码登录
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const step = ref<'email' | 'authenticating' | 'success' | 'error' | 'unsupported'>('email')
const email = ref('')
const loading = ref(false)
const errorMsg = ref('')
const sessionKey = ref('')
const authOptions = ref<any>(null)

onMounted(() => {
  // 检查浏览器是否支持 WebAuthn
  if (!window.PublicKeyCredential) {
    step.value = 'unsupported'
    return
  }
  // 可选：检测平台认证器是否可用
  if (PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable) {
    PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable().then(available => {
      if (!available) {
        // 平台认证器不可用，但仍可尝试（可能有外部认证器）
        console.warn('Platform authenticator not available')
      }
    })
  }
})

// 将 ArrayBuffer 转为 Base64URL
function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

// 将 Base64URL 转为 ArrayBuffer
function base64urlToBuffer(base64url: string): ArrayBuffer {
  let base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  while (base64.length % 4) {
    base64 += '='
  }
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

async function requestAuth() {
  if (!email.value) {
    ElMessage.warning('请输入邮箱')
    return
  }

  loading.value = true
  try {
    // 1. 请求服务端生成认证选项
    const options: any = await authApi.faceAuthOptions({ email: email.value })

    if (!options.publicKey) {
      ElMessage.error('获取认证选项失败')
      return
    }

    sessionKey.value = options.sessionKey
    authOptions.value = options

    // 2. 转换 challenge 和 allowCredentials 为 ArrayBuffer
    const publicKey = options.publicKey
    publicKey.challenge = base64urlToBuffer(publicKey.challenge)

    if (publicKey.allowCredentials) {
      publicKey.allowCredentials = publicKey.allowCredentials.map((c: any) => ({
        ...c,
        id: base64urlToBuffer(c.id),
      }))
    }

    step.value = 'authenticating'

    // 3. 调用浏览器 WebAuthn API
    const credential = await navigator.credentials.get({ publicKey }) as PublicKeyCredential

    if (!credential) {
      step.value = 'error'
      errorMsg.value = '认证被取消'
      return
    }

    // 4. 将认证结果转为可序列化格式
    const response = credential.response as AuthenticatorAssertionResponse
    const credentialData = {
      id: credential.id,
      type: credential.type,
      rawId: bufferToBase64url(credential.rawId),
      response: {
        authenticatorData: bufferToBase64url(response.authenticatorData),
        clientDataJSON: bufferToBase64url(response.clientDataJSON),
        signature: bufferToBase64url(response.signature),
        signCount: response.authenticatorData ? new DataView(response.authenticatorData).getUint32(32) : 0,
      },
    }

    // 5. 验证认证结果
    const result: any = await authApi.faceAuthVerify({
      session_key: sessionKey.value,
      credential: credentialData,
    })

    // 6. 登录成功
    step.value = 'success'

    // 保存登录状态
    const tokenData = result.token || result
    const userData = result.user || {}
    localStorage.setItem('token', tokenData.access_token)
    localStorage.setItem('refresh_token', tokenData.refresh_token)
    localStorage.setItem('user_id', userData.id || '')
    localStorage.setItem('tenant_id', userData.tenant_id || '')
    localStorage.setItem('email', userData.email || '')
    localStorage.setItem('role', userData.role || '')

    // 更新 Pinia store 状态
    userStore.token = tokenData.access_token
    userStore.userId = userData.id || ''
    userStore.tenantId = userData.tenant_id || ''
    userStore.email = userData.email || ''
    userStore.role = userData.role || ''

    ElMessage.success('登录成功')

    setTimeout(() => {
      const redirect = (route.query.redirect as string) || '/'
      router.push(redirect)
    }, 800)

  } catch (e: any) {
    console.error('Face auth error:', e)
    step.value = 'error'

    if (e.name === 'NotAllowedError') {
      errorMsg.value = '认证操作被取消或超时'
    } else if (e.name === 'SecurityError') {
      errorMsg.value = '安全验证失败，请检查域名配置'
    } else if (e.message?.includes('no face credentials')) {
      errorMsg.value = '该账号尚未注册人脸识别，请先使用密码登录后在设置中开启'
    } else if (e.message?.includes('user not found')) {
      errorMsg.value = '该邮箱未注册'
    } else {
      errorMsg.value = e.message || '人脸识别失败，请重试'
    }
  } finally {
    loading.value = false
  }
}

function reset() {
  step.value = 'email'
  errorMsg.value = ''
  sessionKey.value = ''
  authOptions.value = null
}
</script>

<style scoped>
.face-scanner {
  width: 120px;
  height: 120px;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.face-ring {
  position: absolute;
  border-radius: 50%;
  border: 3px solid;
}

.face-ring.outer {
  width: 120px;
  height: 120px;
  border-color: rgba(99, 102, 241, 0.3);
  animation: pulse-ring 2s ease-in-out infinite;
}

.face-ring.inner {
  width: 100px;
  height: 100px;
  border-color: rgba(99, 102, 241, 0.6);
  animation: pulse-ring 2s ease-in-out infinite 0.5s;
}

@keyframes pulse-ring {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.05);
    opacity: 0.7;
  }
}
</style>
