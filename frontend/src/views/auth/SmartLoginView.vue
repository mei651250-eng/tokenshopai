<template>
  <div class="min-h-screen flex">
    <!-- 左侧装饰区 -->
    <div class="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-indigo-600 via-purple-600 to-pink-500 relative overflow-hidden">
      <div class="absolute inset-0 bg-black/10"></div>
      <div class="relative z-10 flex flex-col justify-center items-center w-full p-12 text-white">
        <div class="max-w-md text-center">
          <h1 class="text-4xl font-bold mb-6">TokenHub</h1>
          <p class="text-xl opacity-90 mb-8">AI API Gateway & Token Trading Platform</p>
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div class="bg-white/10 rounded-lg p-4 backdrop-blur">
              <div class="text-2xl font-bold">100+</div>
              <div class="opacity-80">AI 模型</div>
            </div>
            <div class="bg-white/10 rounded-lg p-4 backdrop-blur">
              <div class="text-2xl font-bold">50+</div>
              <div class="opacity-80">国家用户</div>
            </div>
            <div class="bg-white/10 rounded-lg p-4 backdrop-blur">
              <div class="text-2xl font-bold">99.9%</div>
              <div class="opacity-80">可用性</div>
            </div>
            <div class="bg-white/10 rounded-lg p-4 backdrop-blur">
              <div class="text-2xl font-bold">24/7</div>
              <div class="opacity-80">技术支持</div>
            </div>
          </div>
        </div>
      </div>
      <!-- 装饰元素 -->
      <div class="absolute -bottom-20 -left-20 w-64 h-64 bg-white/10 rounded-full blur-3xl"></div>
      <div class="absolute -top-20 -right-20 w-96 h-96 bg-pink-500/20 rounded-full blur-3xl"></div>
    </div>

    <!-- 右侧登录区 -->
    <div class="flex-1 flex flex-col justify-center items-center p-8 bg-gray-50 dark:bg-gray-900">
      <div class="w-full max-w-md">
        <!-- 语言切换 -->
        <div class="flex justify-end mb-4">
          <el-dropdown trigger="click" @command="handleLocaleChange">
            <button class="flex items-center space-x-1 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <circle cx="12" cy="12" r="10"/>
                <path d="M2 12h20"/>
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
              </svg>
              <span class="text-sm">{{ currentLangName }}</span>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="lang in languages" :key="lang.code" :command="lang.code">
                  {{ lang.name }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <!-- Logo -->
        <div class="text-center mb-8">
          <div class="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-2xl mb-4">
            <span class="text-2xl font-bold text-white">T</span>
          </div>
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ isChinaIP ? '欢迎登录' : 'Welcome Back' }}</h2>
          <p class="text-gray-500 dark:text-gray-400 mt-2">
            {{ isChinaIP ? '选择您喜欢的登录方式' : 'Choose your preferred login method' }}
          </p>
        </div>

        <!-- IP 检测状态 -->
        <div v-if="detecting" class="flex items-center justify-center py-4">
          <el-icon class="is-loading text-indigo-500 mr-2"><Loading /></el-icon>
          <span class="text-gray-500">{{ $t('login.detectingLocation') || '检测您的位置...' }}</span>
        </div>

        <!-- 登录方式选择 -->
        <div v-else class="space-y-6">
          <!-- 主要登录方式 -->
          <div class="space-y-3">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ isChinaIP ? '推荐登录方式' : 'Recommended' }}
            </p>
            <div class="grid grid-cols-2 gap-3">
              <button 
                v-for="method in primaryMethods" 
                :key="method"
                @click="selectLoginMethod(method)"
                class="flex items-center justify-center space-x-2 px-4 py-3 rounded-xl border-2 transition-all"
                :class="selectedMethod === method 
                  ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20' 
                  : 'border-gray-200 dark:border-gray-700 hover:border-indigo-300 dark:hover:border-indigo-600'"
              >
                <span class="text-xl">{{ getMethodInfo(method).icon }}</span>
                <span class="font-medium text-gray-700 dark:text-gray-200">{{ getMethodInfo(method).name }}</span>
              </button>
            </div>
          </div>

          <!-- 分割线 -->
          <div class="relative">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-gray-200 dark:border-gray-700"></div>
            </div>
            <div class="relative flex justify-center text-sm">
              <span class="px-4 bg-gray-50 dark:bg-gray-900 text-gray-500">{{ $t('login.or') || '或' }}</span>
            </div>
          </div>

          <!-- 其他登录方式 -->
          <div class="flex flex-wrap justify-center gap-2">
            <button 
              v-for="method in secondaryMethods" 
              :key="method"
              @click="selectLoginMethod(method)"
              class="flex items-center space-x-1 px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <span>{{ getMethodInfo(method).icon }}</span>
              <span class="text-sm text-gray-600 dark:text-gray-300">{{ getMethodInfo(method).name }}</span>
            </button>
          </div>

          <!-- 登录表单 -->
          <transition name="slide-fade">
            <div v-if="selectedMethod" class="mt-6">
              <!-- 手机号登录表单 -->
              <div v-if="selectedMethod === 'phone'" class="space-y-4">
                <div class="flex space-x-2">
                  <el-select v-model="phoneCountry" placeholder="区号" class="w-28" filterable>
                    <el-option 
                      v-for="(config, code) in countryPhoneCodes" 
                      :key="code"
                      :label="`${config.code} ${config.name}`"
                      :value="code"
                    />
                  </el-select>
                  <el-input 
                    v-model="phoneNumber" 
                    :placeholder="$t('login.enterPhone') || '请输入手机号'"
                    class="flex-1"
                    size="large"
                  />
                </div>
                <div class="flex space-x-2">
                  <el-input 
                    v-model="smsCode" 
                    :placeholder="$t('login.enterCode') || '请输入验证码'"
                    class="flex-1"
                    size="large"
                  />
                  <el-button 
                    @click="sendSMSCode" 
                    :disabled="smsCountdown > 0"
                    size="large"
                  >
                    {{ smsCountdown > 0 ? `${smsCountdown}s` : ($t('login.sendCode') || '发送验证码') }}
                  </el-button>
                </div>
                <el-button 
                  type="primary" 
                  size="large" 
                  class="w-full"
                  @click="handlePhoneLogin"
                  :loading="loading"
                >
                  {{ $t('login.login') || '登录' }}
                </el-button>
              </div>

              <!-- 邮箱登录表单 -->
              <div v-else-if="selectedMethod === 'email'" class="space-y-4">
                <el-input 
                  v-model="email" 
                  :placeholder="$t('login.enterEmail') || '请输入邮箱'"
                  size="large"
                />
                <el-input 
                  v-model="password" 
                  type="password"
                  :placeholder="$t('login.enterPassword') || '请输入密码'"
                  size="large"
                  show-password
                />
                <div class="flex justify-between text-sm">
                  <label class="flex items-center">
                    <input type="checkbox" v-model="rememberMe" class="mr-2 rounded">
                    <span class="text-gray-600 dark:text-gray-400">{{ $t('login.rememberMe') || '记住我' }}</span>
                  </label>
                  <a href="#" class="text-indigo-600 hover:text-indigo-500">{{ $t('login.forgotPassword') || '忘记密码？' }}</a>
                </div>
                <el-button 
                  type="primary" 
                  size="large" 
                  class="w-full"
                  @click="handleEmailLogin"
                  :loading="loading"
                >
                  {{ $t('login.login') || '登录' }}
                </el-button>
              </div>

              <!-- OAuth 登录 -->
              <div v-else-if="['google', 'github', 'alipay', 'wechat', 'apple', 'facebook', 'twitter', 'microsoft'].includes(selectedMethod)" class="text-center py-4">
                <p class="text-gray-500 dark:text-gray-400 mb-4">
                  {{ $t('login.redirectTo') || '即将跳转到' }}{{ getMethodInfo(selectedMethod).name }}{{ $t('login.toLogin') || '进行登录' }}
                </p>
                <el-button 
                  type="primary" 
                  size="large" 
                  class="w-full"
                  @click="handleOAuthLogin(selectedMethod)"
                  :style="{ backgroundColor: getMethodInfo(selectedMethod).color, borderColor: getMethodInfo(selectedMethod).color }"
                >
                  <span class="mr-2">{{ getMethodInfo(selectedMethod).icon }}</span>
                  {{ $t('login.loginWith') || '使用' }}{{ getMethodInfo(selectedMethod).name }}{{ $t('login.login') || '登录' }}
                </el-button>
              </div>

              <!-- Web3 钱包登录 -->
              <div v-else-if="selectedMethod === 'web3'" class="space-y-4">
                <p class="text-sm text-gray-500 dark:text-gray-400 text-center">
                  {{ $t('login.connectWallet') || '连接您的 Web3 钱包进行登录' }}
                </p>
                <div class="grid grid-cols-2 gap-3">
                  <el-button @click="handleWeb3Login('metamask')" size="large" class="flex items-center justify-center">
                    <span class="mr-2">🦊</span> MetaMask
                  </el-button>
                  <el-button @click="handleWeb3Login('walletconnect')" size="large" class="flex items-center justify-center">
                    <span class="mr-2">🔗</span> WalletConnect
                  </el-button>
                  <el-button @click="handleWeb3Login('coinbase')" size="large" class="flex items-center justify-center">
                    <span class="mr-2">💎</span> Coinbase
                  </el-button>
                  <el-button @click="handleWeb3Login('trust')" size="large" class="flex items-center justify-center">
                    <span class="mr-2">🛡️</span> Trust Wallet
                  </el-button>
                </div>
              </div>
            </div>
          </transition>
        </div>

        <!-- 注册链接 -->
        <div class="mt-6 text-center text-sm">
          <span class="text-gray-500 dark:text-gray-400">{{ $t('login.noAccount') || '还没有账号？' }}</span>
          <router-link to="/register" class="text-indigo-600 hover:text-indigo-500 font-medium ml-1">
            {{ $t('login.register') || '立即注册' }}
          </router-link>
        </div>

        <!-- 用户协议 -->
        <p class="mt-6 text-center text-xs text-gray-400 dark:text-gray-500">
          {{ $t('login.agreeTerms') || '登录即表示您同意我们的' }}
          <a href="/terms" class="underline hover:text-gray-600 dark:hover:text-gray-400">{{ $t('login.terms') || '服务条款' }}</a>
          {{ $t('login.and') || '和' }}
          <a href="/privacy" class="underline hover:text-gray-600 dark:hover:text-gray-400">{{ $t('login.privacy') || '隐私政策' }}</a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'
import { 
  detectIP, 
  getRecommendedLoginMethods, 
  getLoginMethodInfo, 
  countryPhoneCodes,
  getDefaultPhoneCode,
  validatePhone,
  type IPInfo,
  type LoginMethod
} from '@/utils/ip-detect'

const router = useRouter()
const { locale, t } = useI18n()
const userStore = useUserStore()

// 状态
const detecting = ref(true)
const ipInfo = ref<IPInfo | null>(null)
const selectedMethod = ref<LoginMethod | null>(null)
const loading = ref(false)

// 登录方式
const primaryMethods = ref<LoginMethod[]>(['email', 'google'])
const secondaryMethods = ref<LoginMethod[]>(['github', 'phone', 'web3'])

// 表单数据
const phoneNumber = ref('')
const phoneCountry = ref('CN')
const smsCode = ref('')
const email = ref('')
const password = ref('')
const rememberMe = ref(false)
const smsCountdown = ref(0)

// 计算属性
const isChinaIP = computed(() => ipInfo.value?.is_china ?? false)

const languages = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'en-US', name: 'English' },
  { code: 'ja-JP', name: '日本語' },
  { code: 'ko-KR', name: '한국어' },
  { code: 'zh-TW', name: '繁體中文' },
  { code: 'fr-FR', name: 'Français' },
  { code: 'de-DE', name: 'Deutsch' },
  { code: 'es-ES', name: 'Español' },
  { code: 'pt-BR', name: 'Português' },
  { code: 'ru-RU', name: 'Русский' },
  { code: 'ar-SA', name: 'العربية' },
  { code: 'vi-VN', name: 'Tiếng Việt' },
  { code: 'th-TH', name: 'ไทย' },
  { code: 'tr-TR', name: 'Türkçe' },
]

const currentLangName = computed(() => {
  const lang = languages.find(l => l.code === locale.value)
  return lang?.name || 'English'
})

// 方法
function getMethodInfo(method: LoginMethod) {
  return getLoginMethodInfo(method)
}

function handleLocaleChange(lang: string) {
  locale.value = lang
}

function selectLoginMethod(method: LoginMethod) {
  selectedMethod.value = method
}

async function sendSMSCode() {
  if (!phoneNumber.value) {
    ElMessage.warning(t('login.enterPhone') || '请输入手机号')
    return
  }
  
  if (!validatePhone(phoneNumber.value, phoneCountry.value)) {
    ElMessage.warning(t('login.invalidPhone') || '手机号格式不正确')
    return
  }

  try {
    const config = countryPhoneCodes[phoneCountry.value]
    await authApi.sendCode({
      type: 'sms',
      target: `${config?.code || '+86'}${phoneNumber.value}`,
      purpose: 'login',
      country_code: phoneCountry.value,
    })
    
    ElMessage.success(t('login.codeSent') || '验证码已发送')
    smsCountdown.value = 60
    
    const timer = setInterval(() => {
      smsCountdown.value--
      if (smsCountdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || t('login.sendFailed') || '发送失败')
  }
}

async function handlePhoneLogin() {
  if (!phoneNumber.value || !smsCode.value) {
    ElMessage.warning(t('login.fillAll') || '请填写完整信息')
    return
  }

  loading.value = true
  try {
    const config = countryPhoneCodes[phoneCountry.value]
    const res = await authApi.loginByCode({
      type: 'sms',
      target: `${config?.code || '+86'}${phoneNumber.value}`,
      code: smsCode.value,
      country_code: phoneCountry.value,
    })
    
    userStore.setToken(res.data.token)
    userStore.setUser(res.data.user)
    ElMessage.success(t('login.success') || '登录成功')
    router.push('/home')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || t('login.failed') || '登录失败')
  } finally {
    loading.value = false
  }
}

async function handleEmailLogin() {
  if (!email.value || !password.value) {
    ElMessage.warning(t('login.fillAll') || '请填写完整信息')
    return
  }

  loading.value = true
  try {
    const res = await authApi.login({ email: email.value, password: password.value })
    userStore.setToken(res.data.token)
    userStore.setUser(res.data.user)
    ElMessage.success(t('login.success') || '登录成功')
    router.push('/home')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || t('login.failed') || '登录失败')
  } finally {
    loading.value = false
  }
}

function handleOAuthLogin(provider: string) {
  // 跳转到 OAuth 授权页面
  window.location.href = `/auth/oauth/${provider}`
}

async function handleWeb3Login(walletType: string) {
  try {
    // 检查是否有对应的钱包扩展
    const provider = (window as any).ethereum
    
    if (!provider) {
      ElMessage.warning(t('login.installWallet') || '请先安装钱包扩展')
      window.open('https://metamask.io/download/', '_blank')
      return
    }

    loading.value = true
    
    // 请求连接钱包
    const accounts = await provider.request({ method: 'eth_requestAccounts' })
    const address = accounts[0]
    
    // 获取签名消息
    const message = `Login to TokenHub at ${new Date().toISOString()}`
    const signature = await provider.request({
      method: 'personal_sign',
      params: [message, address],
    })

    // 发送到后端验证
    const res = await authApi.walletLogin({
      address,
      signature,
      wallet_type: walletType,
    })

    userStore.setToken(res.data.token)
    userStore.setUser(res.data.user)
    ElMessage.success(t('login.success') || '登录成功')
    router.push('/home')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || t('login.failed') || '登录失败')
  } finally {
    loading.value = false
  }
}

// 生命周期
onMounted(async () => {
  try {
    // 检测 IP
    ipInfo.value = await detectIP()
    
    // 根据国家获取推荐登录方式
    const methods = getRecommendedLoginMethods(ipInfo.value.country_code)
    primaryMethods.value = methods.primary
    secondaryMethods.value = methods.secondary
    
    // 设置默认手机区号
    phoneCountry.value = ipInfo.value.country_code
    
    // 默认选择推荐的登录方式
    selectedMethod.value = methods.recommended
  } catch (error) {
    console.error('IP detection failed:', error)
    // 使用默认设置
    selectedMethod.value = 'email'
  } finally {
    detecting.value = false
  }
})
</script>

<style scoped>
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.2s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateY(-10px);
  opacity: 0;
}
</style>
