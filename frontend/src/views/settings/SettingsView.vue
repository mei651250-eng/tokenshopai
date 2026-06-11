<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('settings.title') || '系统设置' }}</h1>

    <div class="space-y-6">
      <!-- Language -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('common.config') || '语言与区域' }}</h3>
        <el-form label-width="120px">
          <el-form-item :label="t('settings.language') || '界面语言'">
            <el-select v-model="locale" @change="handleLocaleChange" style="width: 260px">
              <el-option label="简体中文" value="zh-CN" />
              <el-option label="繁體中文 (台灣)" value="zh-TW" />
              <el-option label="English" value="en-US" />
              <el-option label="日本語" value="ja-JP" />
              <el-option label="한국어" value="ko-KR" />
              <el-option label="Français" value="fr-FR" />
              <el-option label="Deutsch" value="de-DE" />
              <el-option label="Español" value="es-ES" />
              <el-option label="Português (Brasil)" value="pt-BR" />
              <el-option label="Italiano" value="it-IT" />
              <el-option label="Русский" value="ru-RU" />
              <el-option label="العربية" value="ar-SA" />
              <el-option label="हिन्दी" value="hi-IN" />
              <el-option label="Bahasa Indonesia" value="id-ID" />
              <el-option label="Tiếng Việt" value="vi-VN" />
              <el-option label="ไทย" value="th-TH" />
              <el-option label="Türkçe" value="tr-TR" />
              <el-option label="Nederlands" value="nl-NL" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('settings.timezone') || '时区'">
            <el-select v-model="timezone" style="width: 260px">
              <el-option label="Asia/Shanghai (UTC+8)" value="Asia/Shanghai" />
              <el-option label="Asia/Tokyo (UTC+9)" value="Asia/Tokyo" />
              <el-option label="Asia/Seoul (UTC+9)" value="Asia/Seoul" />
              <el-option label="Asia/Singapore (UTC+8)" value="Asia/Singapore" />
              <el-option label="Asia/Dubai (UTC+4)" value="Asia/Dubai" />
              <el-option label="Asia/Kolkata (UTC+5:30)" value="Asia/Kolkata" />
              <el-option label="Europe/London (UTC+0)" value="Europe/London" />
              <el-option label="Europe/Paris (UTC+1)" value="Europe/Paris" />
              <el-option label="Europe/Berlin (UTC+1)" value="Europe/Berlin" />
              <el-option label="Europe/Moscow (UTC+3)" value="Europe/Moscow" />
              <el-option label="America/New_York (UTC-5)" value="America/New_York" />
              <el-option label="America/Los_Angeles (UTC-8)" value="America/Los_Angeles" />
              <el-option label="America/Sao_Paulo (UTC-3)" value="America/Sao_Paulo" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>

      <!-- API Settings -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('models.config') || 'API 设置' }}</h3>
        <el-form label-width="120px">
          <el-form-item :label="t('models.name') || '默认模型'">
            <el-select v-model="defaultModel">
              <el-option label="gpt-4o" value="gpt-4o" />
              <el-option label="claude-3.5-sonnet" value="claude-3.5-sonnet" />
              <el-option label="qwen-max" value="qwen-max" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('settings.routeStrategy') || '路由策略'">
            <el-select v-model="routeStrategy">
              <el-option :label="t('settings.weightedRandom') || '加权随机'" value="weighted_random" />
              <el-option :label="t('settings.roundRobin') || '轮询'" value="round_robin" />
              <el-option :label="t('settings.leastLatency') || '最低延迟'" value="least_latency" />
              <el-option :label="t('settings.leastCost') || '最低成本'" value="least_cost" />
              <el-option :label="t('settings.compositeScore') || '综合评分'" value="composite_score" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>

      <!-- Face ID -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('settings.biometrics') || '人脸识别 / 生物识别' }}</h3>
            <p class="text-sm text-gray-500 mt-1">{{ t('settings.biometricsDesc') || '注册 Windows Hello、Touch ID 或 Face ID 以实现快速安全登录' }}</p>
          </div>
          <el-button type="primary" :loading="registering" :disabled="!faceSupported" @click="registerFace">
            {{ faceCredentials.length > 0 ? (t('settings.addBiometrics') || '添加新凭据') : (t('settings.registerBiometrics') || '注册人脸识别') }}
          </el-button>
        </div>

        <div v-if="!faceSupported" class="text-yellow-600 text-sm bg-yellow-50 dark:bg-yellow-900/20 rounded-lg p-3">
          {{ t('settings.biometricsNotSupported') || '您的浏览器不支持 WebAuthn，无法使用人脸识别功能' }}
        </div>

        <div v-else-if="faceCredentials.length === 0" class="text-gray-500 text-sm">
          {{ t('settings.noBiometrics') || '尚未注册人脸识别凭据' }}
        </div>

        <div v-else class="space-y-3">
          <div v-for="cred in faceCredentials" :key="cred.id"
               class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-full bg-indigo-100 dark:bg-indigo-900/30 flex items-center justify-center">
                <svg class="w-5 h-5 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              </div>
              <div>
                <p class="font-medium text-gray-900 dark:text-white">{{ cred.name || (t('settings.biometricCredential') || '生物识别凭据') }}</p>
                <p class="text-xs text-gray-500">
                  {{ t('profile.createdAt') || '注册于' }} {{ new Date(cred.created_at).toLocaleDateString() }}
                  <span v-if="cred.last_used_at"> · {{ t('profile.lastLogin') || '最近使用' }} {{ new Date(cred.last_used_at).toLocaleDateString() }}</span>
                </p>
              </div>
            </div>
            <el-button type="danger" text size="small" @click="removeFaceCredential(cred.id)">
              {{ t('common.delete') || '删除' }}
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t, locale: i18nLocale } = useI18n()
const userStore = useUserStore()

const locale = ref(localStorage.getItem('locale') || 'zh-CN')
const timezone = ref('Asia/Shanghai')
const defaultModel = ref('gpt-4o')
const routeStrategy = ref('weighted_random')

const faceSupported = ref(false)
const registering = ref(false)
const faceCredentials = ref<any[]>([])

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

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

onMounted(() => {
  faceSupported.value = !!window.PublicKeyCredential
  loadFaceCredentials()
})

async function loadFaceCredentials() {
  try {
    const res: any = await authApi.faceCredentials()
    faceCredentials.value = res.credentials || []
  } catch {
    // 未认证或其他错误
  }
}

async function registerFace() {
  registering.value = true
  try {
    const options: any = await authApi.faceRegisterOptions()
    if (!options.publicKey) {
      ElMessage.error(t('common.error') || '获取注册选项失败')
      return
    }

    const sessionKey = options.sessionKey
    const publicKey = options.publicKey

    publicKey.challenge = base64urlToBuffer(publicKey.challenge)

    if (publicKey.excludeCredentials) {
      publicKey.excludeCredentials = publicKey.excludeCredentials.map((c: any) => ({
        ...c,
        id: base64urlToBuffer(c.id),
      }))
    }

    if (publicKey.user?.id) {
      publicKey.user.id = base64urlToBuffer(publicKey.user.id)
    }

    const credential = await navigator.credentials.create({ publicKey }) as PublicKeyCredential

    if (!credential) {
      ElMessage.warning(t('settings.registrationCancelled') || '注册被取消')
      return
    }

    const response = credential.response as AuthenticatorAttestationResponse
    const credentialData = {
      id: credential.id,
      type: credential.type,
      rawId: bufferToBase64url(credential.rawId),
      response: {
        attestationObject: bufferToBase64url(response.attestationObject),
        clientDataJSON: bufferToBase64url(response.clientDataJSON),
      },
    }

    await authApi.faceRegisterVerify({
      session_key: sessionKey,
      credential: credentialData,
    })

    ElMessage.success(t('settings.biometricsRegistered') || '人脸识别注册成功！')
    loadFaceCredentials()
  } catch (e: any) {
    console.error('Face registration error:', e)
    if (e.name === 'NotAllowedError') {
      ElMessage.warning(t('settings.registrationCancelled') || '注册操作被取消')
    } else if (e.name === 'InvalidStateError') {
      ElMessage.warning(t('settings.alreadyRegistered') || '该设备已注册过人脸识别')
    } else {
      ElMessage.error(e.message || (t('common.error') || '注册失败'))
    }
  } finally {
    registering.value = false
  }
}

async function removeFaceCredential(credId: string) {
  try {
    await ElMessageBox.confirm(
      t('settings.deleteBiometricsConfirm') || '确定要删除此人脸识别凭据吗？',
      t('common.confirm') || '删除确认',
      {
        confirmButtonText: t('common.delete') || '删除',
        cancelButtonText: t('common.cancel') || '取消',
        type: 'warning',
      }
    )
    await authApi.removeFaceCredential(credId)
    ElMessage.success(t('common.success') || '已删除')
    loadFaceCredentials()
  } catch {
    // 取消
  }
}

function handleLocaleChange(val: string) {
  i18nLocale.value = val
  userStore.setLocale(val)
}
</script>
