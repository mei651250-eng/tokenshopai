<template>
  <div class="code-login-view min-h-screen flex items-center justify-center bg-gray-50">
    <div class="login-card w-full max-w-md p-8 bg-white rounded-2xl shadow-xl">
      <!-- Logo -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-indigo-600">Token中站站</h1>
        <p class="text-gray-500 mt-2">{{ $t('auth.codeLoginSubtitle') }}</p>
      </div>

      <!-- 登录方式切换 -->
      <el-tabs v-model="loginType" class="login-tabs">
        <!-- 手机号验证码登录 -->
        <el-tab-pane :label="$t('auth.smsLogin')" name="sms">
          <el-form :model="smsForm" :rules="smsRules" ref="smsFormRef" @submit.prevent="handleSmsLogin">
            <el-form-item prop="country_code">
              <el-select v-model="smsForm.country_code" style="width: 110px" placeholder="区号">
                <el-option label="+86 中国" value="+86" />
                <el-option label="+1 美国/加拿大" value="+1" />
                <el-option label="+44 英国" value="+44" />
                <el-option label="+81 日本" value="+81" />
                <el-option label="+82 韩国" value="+82" />
                <el-option label="+65 新加坡" value="+65" />
                <el-option label="+852 香港" value="+852" />
                <el-option label="+853 澳门" value="+853" />
                <el-option label="+886 台湾" value="+886" />
                <el-option label="+91 印度" value="+91" />
                <el-option label="+49 德国" value="+49" />
                <el-option label="+33 法国" value="+33" />
                <el-option label="+61 澳大利亚" value="+61" />
                <el-option label="+55 巴西" value="+55" />
                <el-option label="+7 俄罗斯" value="+7" />
              </el-select>
              <el-input v-model="smsForm.phone" :placeholder="$t('auth.phonePlaceholder')" style="flex:1" />
            </el-form-item>
            <el-form-item prop="code">
              <div class="flex gap-2 w-full">
                <el-input v-model="smsForm.code" :placeholder="$t('auth.codePlaceholder')" />
                <el-button
                  @click="sendSmsCode"
                  :disabled="smsCooldown > 0"
                  :loading="sendingCode"
                  style="width: 130px"
                >
                  {{ smsCooldown > 0 ? `${smsCooldown}s` : $t('auth.sendCode') }}
                </el-button>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSmsLogin" :loading="logging" class="w-full" size="large">
                {{ $t('auth.login') }}
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 邮箱验证码登录 -->
        <el-tab-pane :label="$t('auth.emailLogin')" name="email">
          <el-form :model="emailForm" :rules="emailRules" ref="emailFormRef" @submit.prevent="handleEmailLogin">
            <el-form-item prop="email">
              <el-input v-model="emailForm.email" :placeholder="$t('auth.emailPlaceholder')" prefix-icon="Message" />
            </el-form-item>
            <el-form-item prop="code">
              <div class="flex gap-2 w-full">
                <el-input v-model="emailForm.code" :placeholder="$t('auth.codePlaceholder')" />
                <el-button
                  @click="sendEmailCode"
                  :disabled="emailCooldown > 0"
                  :loading="sendingCode"
                  style="width: 130px"
                >
                  {{ emailCooldown > 0 ? `${emailCooldown}s` : $t('auth.sendCode') }}
                </el-button>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleEmailLogin" :loading="logging" class="w-full" size="large">
                {{ $t('auth.login') }}
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- Web3 钱包登录 -->
        <el-tab-pane :label="$t('auth.walletLogin')" name="wallet">
          <div class="wallet-login-section">
            <p class="text-gray-500 text-sm mb-4">{{ $t('auth.walletLoginDesc') }}</p>
            <div class="wallet-buttons">
              <button class="wallet-btn" @click="connectWallet('metamask')">
                <span class="text-xl mr-2">🦊</span> MetaMask
              </button>
              <button class="wallet-btn" @click="connectWallet('walletconnect')">
                <span class="text-xl mr-2">🔗</span> WalletConnect
              </button>
              <button class="wallet-btn" @click="connectWallet('phantom')">
                <span class="text-xl mr-2">👻</span> Phantom
              </button>
              <button class="wallet-btn" @click="connectWallet('okx_wallet')">
                <span class="text-xl mr-2">⭕</span> OKX Wallet
              </button>
              <button class="wallet-btn" @click="connectWallet('coinbase')">
                <span class="text-xl mr-2">🔵</span> Coinbase
              </button>
              <button class="wallet-btn" @click="connectWallet('bitget')">
                <span class="text-xl mr-2">🟢</span> Bitget
              </button>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- 分隔线 -->
      <el-divider>{{ $t('auth.or') }}</el-divider>

      <!-- 其他登录方式 -->
      <div class="text-center">
        <router-link to="/login" class="text-indigo-600 hover:underline">
          {{ $t('auth.passwordLogin') }}
        </router-link>
        <span class="text-gray-300 mx-2">|</span>
        <router-link to="/login/face" class="text-indigo-600 hover:underline">
          人脸识别登录
        </router-link>
      </div>

      <!-- 注册提示 -->
      <p class="text-center text-sm text-gray-500 mt-4">
        {{ $t('auth.noAccount') }}
        <a href="#" class="text-indigo-600 hover:underline" @click.prevent="showRegister = true">
          {{ $t('auth.register') }}
        </a>
      </p>
    </div>

    <!-- 注册对话框 -->
    <el-dialog v-model="showRegister" :title="$t('auth.register')" width="420px">
      <el-tabs v-model="registerType">
        <el-tab-pane :label="$t('auth.phoneRegister')" name="sms">
          <el-form :model="regForm" label-width="80px">
            <el-form-item :label="$t('auth.country')">
              <el-select v-model="regForm.country_code">
                <el-option label="+86 中国" value="+86" />
                <el-option label="+1 美国" value="+1" />
                <el-option label="+81 日本" value="+81" />
                <el-option label="+82 韩国" value="+82" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('auth.phone')">
              <el-input v-model="regForm.phone" />
            </el-form-item>
            <el-form-item :label="$t('auth.verifyCode')">
              <div class="flex gap-2">
                <el-input v-model="regForm.code" />
                <el-button @click="sendRegCode('sms')" :disabled="regCooldown > 0">
                  {{ regCooldown > 0 ? `${regCooldown}s` : $t('auth.sendCode') }}
                </el-button>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleRegister" class="w-full">{{ $t('auth.register') }}</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane :label="$t('auth.emailRegister')" name="email">
          <el-form :model="regForm" label-width="80px">
            <el-form-item :label="$t('auth.email')">
              <el-input v-model="regForm.email" />
            </el-form-item>
            <el-form-item :label="$t('auth.verifyCode')">
              <div class="flex gap-2">
                <el-input v-model="regForm.code" />
                <el-button @click="sendRegCode('email')" :disabled="regCooldown > 0">
                  {{ regCooldown > 0 ? `${regCooldown}s` : $t('auth.sendCode') }}
                </el-button>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleRegister" class="w-full">{{ $t('auth.register') }}</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authApi, walletApi } from '@/api'

const router = useRouter()
const loginType = ref('sms')
const logging = ref(false)
const sendingCode = ref(false)
const showRegister = ref(false)
const registerType = ref('sms')

// SMS form
const smsForm = reactive({ country_code: '+86', phone: '', code: '' })
const smsRules = {
  phone: [{ required: true, message: 'Please enter phone number', trigger: 'blur' }],
  code: [{ required: true, message: 'Please enter verification code', trigger: 'blur' }],
}
const smsCooldown = ref(0)
let smsTimer: ReturnType<typeof setInterval> | null = null

// Email form
const emailForm = reactive({ email: '', code: '' })
const emailRules = {
  email: [
    { required: true, message: 'Please enter email', trigger: 'blur' },
    { type: 'email', message: 'Invalid email format', trigger: 'blur' },
  ],
  code: [{ required: true, message: 'Please enter verification code', trigger: 'blur' }],
}
const emailCooldown = ref(0)
let emailTimer: ReturnType<typeof setInterval> | null = null

// Register form
const regForm = reactive({ country_code: '+86', phone: '', email: '', code: '' })
const regCooldown = ref(0)
let regTimer: ReturnType<typeof setInterval> | null = null

function startCooldown(type: 'sms' | 'email' | 'reg') {
  const refMap = { sms: smsCooldown, email: emailCooldown, reg: regCooldown }
  const timerMap = { sms: smsTimer, email: emailTimer, reg: regTimer }
  const cooldownRef = refMap[type]
  cooldownRef.value = 60
  const timer = setInterval(() => {
    cooldownRef.value--
    if (cooldownRef.value <= 0) {
      clearInterval(timer)
    }
  }, 1000)
  if (type === 'sms') smsTimer = timer
  else if (type === 'email') emailTimer = timer
  else regTimer = timer
}

async function sendSmsCode() {
  if (!smsForm.phone) { ElMessage.warning('Please enter phone number'); return }
  sendingCode.value = true
  try {
    await authApi.sendCode({ type: 'sms', target: smsForm.phone, country_code: smsForm.country_code, purpose: 'login' })
    ElMessage.success('Code sent')
    startCooldown('sms')
  } catch { ElMessage.error('Failed to send code') }
  finally { sendingCode.value = false }
}

async function sendEmailCode() {
  if (!emailForm.email) { ElMessage.warning('Please enter email'); return }
  sendingCode.value = true
  try {
    await authApi.sendCode({ type: 'email', target: emailForm.email, purpose: 'login' })
    ElMessage.success('Code sent')
    startCooldown('email')
  } catch { ElMessage.error('Failed to send code') }
  finally { sendingCode.value = false }
}

async function sendRegCode(type: string) {
  const target = type === 'sms' ? regForm.phone : regForm.email
  if (!target) { ElMessage.warning('Please fill the field'); return }
  try {
    await authApi.sendCode({ type: type as any, target, country_code: regForm.country_code, purpose: 'register' })
    ElMessage.success('Code sent')
    startCooldown('reg')
  } catch { ElMessage.error('Failed to send code') }
}

async function handleSmsLogin() {
  if (!smsForm.phone || !smsForm.code) return
  logging.value = true
  try {
    const res = await authApi.loginByCode({ type: 'sms', target: smsForm.phone, code: smsForm.code, country_code: smsForm.country_code })
    onLoginSuccess(res)
  } catch { ElMessage.error('Login failed') }
  finally { logging.value = false }
}

async function handleEmailLogin() {
  if (!emailForm.email || !emailForm.code) return
  logging.value = true
  try {
    const res = await authApi.loginByCode({ type: 'email', target: emailForm.email, code: emailForm.code })
    onLoginSuccess(res)
  } catch { ElMessage.error('Login failed') }
  finally { logging.value = false }
}

async function connectWallet(type: string) {
  // 实际实现中：
  // 1. 检测 window.ethereum / window.solana 等
  // 2. 请求连接钱包
  // 3. 获取地址
  // 4. 获取挑战消息
  // 5. 请求签名
  // 6. 验证签名登录
  ElMessage.info(`Connecting ${type}...`)
  try {
    // 伪代码: const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' })
    // const address = accounts[0]
    // const challenge = await walletApi.getChallenge(address)
    // const signature = await window.ethereum.request({ method: 'personal_sign', params: [challenge, address] })
    // const res = await authApi.walletLogin({ address, signature, wallet_type: type })
    // onLoginSuccess(res)
  } catch {
    ElMessage.error('Wallet connection failed')
  }
}

async function handleRegister() {
  try {
    const res = await authApi.registerByCode({
      type: registerType.value as any,
      target: registerType.value === 'sms' ? regForm.phone : regForm.email,
      code: regForm.code,
      country_code: registerType.value === 'sms' ? regForm.country_code : undefined,
    })
    onLoginSuccess(res)
    showRegister.value = false
  } catch {
    ElMessage.error('Registration failed')
  }
}

function onLoginSuccess(res: any) {
  localStorage.setItem('token', res.token?.access_token)
  localStorage.setItem('user_id', res.user?.id)
  localStorage.setItem('tenant_id', res.user?.tenant_id)
  ElMessage.success('Login successful')
  router.push('/')
}

onUnmounted(() => {
  if (smsTimer) clearInterval(smsTimer)
  if (emailTimer) clearInterval(emailTimer)
  if (regTimer) clearInterval(regTimer)
})
</script>

<style scoped>
.wallet-login-section { padding: 12px 0; }
.wallet-buttons {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}
.wallet-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  border: 2px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  background: white;
  transition: all 0.2s;
}
.wallet-btn:hover {
  border-color: #6366f1;
  background: #eef2ff;
}
</style>
