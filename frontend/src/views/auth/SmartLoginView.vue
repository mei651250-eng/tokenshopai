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
                <span class="w-5 h-5 inline-flex items-center justify-center" v-html="renderMethodIcon(method, 20, getMethodInfo(method).color)"></span>
                <span class="font-medium text-gray-700 dark:text-gray-200">{{ getMethodInfo(method).name }}</span>
              </button>
            </div>
          </div>

          <!-- 分割线（仅当有其他登录方式时显示） -->
          <div v-if="secondaryMethods.length > 0" class="relative">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-gray-200 dark:border-gray-700"></div>
            </div>
            <div class="relative flex justify-center text-sm">
              <span class="px-4 bg-gray-50 dark:bg-gray-900 text-gray-500">{{ $t('login.or') || '或' }}</span>
            </div>
          </div>

          <!-- 其他登录方式 -->
          <div v-if="secondaryMethods.length > 0" class="flex flex-wrap justify-center gap-2">
            <button 
              v-for="method in secondaryMethods" 
              :key="method"
              @click="selectLoginMethod(method)"
              class="flex items-center space-x-1.5 px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <span class="w-4 h-4 inline-flex items-center justify-center" v-html="renderMethodIcon(method, 16, '#6b7280')"></span>
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

              <!-- 支付宝/微信 = 扫码登录 -->
              <div v-else-if="['alipay', 'wechat'].includes(selectedMethod)" class="text-center py-4">
                <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
                  {{ $t('login.scanToLogin') || '请使用' }}{{ getMethodInfo(selectedMethod).name }}{{ $t('login.scanToLogin2') || '扫描二维码登录' }}
                </p>
                <div class="inline-block bg-white dark:bg-gray-700 p-4 rounded-xl border-2 border-gray-200 dark:border-gray-600 mb-4">
                  <canvas ref="qrCanvas" width="200" height="200" class="w-50 h-50"></canvas>
                </div>
                <p class="text-xs text-gray-400">{{ qrHint }}</p>
                <el-button type="primary" size="small" class="mt-3" @click="refreshQRCode" :loading="qrLoading">
                  {{ $t('login.refreshQR') || '刷新二维码' }}
                </el-button>
              </div>

              <!-- 其他 OAuth 登录 -->
              <div v-else-if="['google', 'github', 'apple', 'facebook', 'twitter', 'microsoft'].includes(selectedMethod)" class="text-center py-4">
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
                  <span class="mr-2 inline-flex items-center" v-html="renderMethodIcon(selectedMethod, 18, '#ffffff')"></span>
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
                    <span class="mr-2 w-5 h-5 inline-flex" v-html="walletIcon('metamask')"></span> MetaMask
                  </el-button>
                  <el-button @click="handleWeb3Login('walletconnect')" size="large" class="flex items-center justify-center">
                    <span class="mr-2 w-5 h-5 inline-flex" v-html="walletIcon('walletconnect')"></span> WalletConnect
                  </el-button>
                  <el-button @click="handleWeb3Login('coinbase')" size="large" class="flex items-center justify-center">
                    <span class="mr-2 w-5 h-5 inline-flex" v-html="walletIcon('coinbase')"></span> Coinbase
                  </el-button>
                  <el-button @click="handleWeb3Login('trust')" size="large" class="flex items-center justify-center">
                    <span class="mr-2 w-5 h-5 inline-flex" v-html="walletIcon('trust')"></span> Trust Wallet
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
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'
import QRCode from 'qrcode'
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

// QR 码相关
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrLoading = ref(false)
const qrHint = ref('')
let qrPollTimer: any = null

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

// 渲染登录方式 SVG 图标
function renderMethodIcon(method: LoginMethod, size: number, color: string): string {
  const s = size
  const icons: Record<string, string> = {
    phone: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="2" width="14" height="20" rx="2" ry="2"/><line x1="12" y1="18" x2="12.01" y2="18"/></svg>`,
    email: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>`,
    alipay: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2Z"/><path d="M8.56 2.75c4.37 6.03 6.02 9.42 8.03 17.72m2.54-15.38h-8.1m0 0C6.4 5.84 4.68 7.9 3 9.94"/></svg>`,
    wechat: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 10.9c3.9-2.1 8.3-1.3 10.5 1.7M9.5 16.5c-1.2-.8-2.5-1.3-4-1.3-3.6 0-6.5 2.3-6.5 5 0 1 .5 2 1.3 2.7"/><circle cx="10" cy="13" r="1"/><circle cx="15" cy="13" r="1"/><circle cx="13" cy="9" r="1"/><circle cx="7" cy="9" r="1"/></svg>`,
    google: `<svg width="${s}" height="${s}" viewBox="0 0 24 24"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>`,
    github: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="${color}"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>`,
    apple: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="${color}"><path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/></svg>`,
    facebook: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="${color}"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>`,
    twitter: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="${color}"><path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/></svg>`,
    microsoft: `<svg width="${s}" height="${s}" viewBox="0 0 24 24"><rect x="1" y="1" width="10" height="10" fill="#F25022"/><rect x="13" y="1" width="10" height="10" fill="#7FBA00"/><rect x="1" y="13" width="10" height="10" fill="#00A4EF"/><rect x="13" y="13" width="10" height="10" fill="#FFB900"/></svg>`,
    wallet: `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><circle cx="16" cy="12" r="2"/><line x1="2" y1="9" x2="6" y2="9"/></svg>`,
  }
  return icons[method] || icons.email
}

// 渲染 Web3 钱包图标
function walletIcon(type: string): string {
  const icons: Record<string, string> = {
    metamask: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M20.8 2H3.2C2.54 2 2 2.54 2 3.2v17.6c0 .66.54 1.2 1.2 1.2h17.6c.66 0 1.2-.54 1.2-1.2V3.2c0-.66-.54-1.2-1.2-1.2z" fill="#E2761B"/><path d="M19.4 5.6L12 11.2l2.8-6.4 4.6.8zM4.6 5.6L12 11.2 9.2 4.8l-4.6.8zM18 16.4l-2.4 3.6 4.8 1.2 1.4-4.8-3.8 0zM6 16.4l-3.8 0 1.4 4.8 4.8-1.2L6 16.4z" fill="#E4761B"/><path d="M12 14l-2.4 3.6h4.8L12 14z" fill="#F6851B"/><circle cx="12" cy="11" r="2" fill="#F6851B"/></svg>`,
    walletconnect: `<svg width="20" height="20" viewBox="0 0 24 24"><rect width="24" height="24" rx="12" fill="#3B99FC"/><path d="M6.5 9.5c3-2.9 7.9-2.9 10.9 0l.4.4c.1.1.2.3 0 .4l-1.2 1.2c-.1.1-.2.1-.3 0-.1 0-.1 0-.2-.1-2.4-2.3-6.2-2.3-8.6 0-.1.1-.2.1-.3 0L6.1 10.3c-.1-.1-.1-.3 0-.4l.4-.4zM7.9 11.5l1.1 1c.1.1.1.2 0 .3-1.6 1.5-1.6 4 0 5.5.1.1.1.2 0 .3l-1.1 1c-.1.1-.2.1-.3 0-2.3-2.2-2.3-5.8 0-8.1.1-.1.2-.1.3 0zm8.2 0c.1-.1.2-.1.3 0 2.3 2.2 2.3 5.8 0 8.1-.1.1-.2.1-.3 0l-1.1-1c-.1-.1-.1-.2 0-.3 1.6-1.5 1.6-4 0-5.5-.1-.1-.1-.2 0-.3l1.1-1zM10.2 12.9c1.1-1.1 2.9-1.1 4 0l.2.2c.1.1.1.3 0 .4l-.2.2-.2.2c-.1-.1-.3-.1-.4 0-.7.7-1.8.7-2.5 0-.1-.1-.2-.1-.3 0-.1 0-.1.1 0 .2-.1.1-.3.1-.4 0l-.2-.2c-.1-.1-.1-.3 0-.4l.2-.2.2-.2z" fill="#fff"/></svg>`,
    coinbase: `<svg width="20" height="20" viewBox="0 0 24 24"><rect width="24" height="24" rx="12" fill="#0052FF"/><path d="M12 4a8 8 0 1 0 0 16 8 8 0 0 0 0-16zm-1.65 6.65h3.3c.55 0 1 .45 1 1v.7c0 .55-.45 1-1 1h-3.3c-.55 0-1-.45-1-1v-.7c0-.55.45-1 1-1z" fill="#fff"/></svg>`,
    trust: `<svg width="20" height="20" viewBox="0 0 24 24" fill="#3375BB"><path d="M12 1L3 5v6c0 5.5 3.8 10.7 9 12 5.2-1.3 9-6.5 9-12V5l-9-4zm-2 16l-4-4 1.4-1.4L10 14.2l6.6-6.6L18 9l-8 8z"/></svg>`,
  }
  return icons[type] || ''
}

function handleLocaleChange(lang: string) {
  locale.value = lang
  localStorage.setItem('locale', lang)
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

// 生成二维码登录
async function generateQRCode(method: LoginMethod) {
  qrLoading.value = true
  try {
    // 生成临时登录 token
    const token = 'qr_' + Math.random().toString(36).slice(2) + Date.now().toString(36)
    // 将 token 存到后端 Redis（简化版：用 URL 参数传递）
    const baseUrl = window.location.origin
    const qrUrl = `${baseUrl}/auth/oauth/${method}/callback?code=qrcode_${token}&state=qrcode_${token}`
    
    await nextTick()
    if (qrCanvas.value) {
      await QRCode.toCanvas(qrCanvas.value, qrUrl, {
        width: 200,
        margin: 2,
        color: { dark: '#000000', light: '#ffffff' }
      })
    }
    qrHint.value = '请打开' + (method === 'alipay' ? '支付宝' : '微信') + '扫一扫'
    
    // 轮询检查是否扫码登录成功（mock: 10秒后自动成功）
    startQRPolling(method, token)
  } catch (e) {
    ElMessage.error('生成二维码失败')
  } finally {
    qrLoading.value = false
  }
}

function startQRPolling(method: string, token: string) {
  if (qrPollTimer) clearInterval(qrPollTimer)
  let countdown = 30
  qrPollTimer = setInterval(async () => {
    countdown--
    if (countdown <= 0) {
      clearInterval(qrPollTimer)
      qrHint.value = '二维码已过期，请刷新'
      return
    }
    try {
      // 检查后端是否已确认扫码
      const res = await fetch(`/auth/qrcode/check?token=${token}`)
      if (res.ok) {
        const data = await res.json()
        if (data.status === 'scanned') {
          clearInterval(qrPollTimer)
          qrHint.value = '已扫码，正在登录...'
          // 等待登录结果
          setTimeout(async () => {
            const loginRes = await fetch(`/auth/qrcode/login?token=${token}`)
            if (loginRes.ok) {
              const loginData = await loginRes.json()
              userStore.setToken(loginData.token)
              userStore.setUser(loginData.user)
              ElMessage.success(t('login.success') || '登录成功')
              router.push('/home')
            }
          }, 1000)
        }
      }
    } catch {}
  }, 2000)
}

function refreshQRCode() {
  if (selectedMethod.value && ['alipay', 'wechat'].includes(selectedMethod.value)) {
    generateQRCode(selectedMethod.value)
  }
}

// 监听登录方式切换，自动生成二维码
watch(selectedMethod, (newMethod) => {
  if (qrPollTimer) { clearInterval(qrPollTimer); qrPollTimer = null }
  if (newMethod && ['alipay', 'wechat'].includes(newMethod)) {
    generateQRCode(newMethod)
  }
})

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
    
    // 根据 IP 设置默认语言（中国→中文，国外→英文）
    if (ipInfo.value.is_china && !localStorage.getItem('locale')) {
      locale.value = 'zh-CN'
      localStorage.setItem('locale', 'zh-CN')
    } else if (!ipInfo.value.is_china && !localStorage.getItem('locale')) {
      locale.value = 'en-US'
      localStorage.setItem('locale', 'en-US')
    }
    
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
