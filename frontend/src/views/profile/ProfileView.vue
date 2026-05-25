<template>
  <div class="p-6 max-w-4xl mx-auto">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('profile.title') }}</h1>

    <div class="space-y-6">
      <!-- Avatar & Basic Info -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center gap-6 mb-6">
          <div class="relative group">
            <div class="w-20 h-20 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center text-3xl font-bold text-primary-600">
              {{ userStore.email.charAt(0).toUpperCase() }}
            </div>
            <div class="absolute inset-0 bg-black/30 rounded-full opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity cursor-pointer" @click="uploadAvatar">
              <el-icon :size="24" class="text-white"><Camera /></el-icon>
            </div>
            <input ref="avatarInput" type="file" accept="image/*" class="hidden" @change="handleAvatarChange" />
          </div>
          <div>
            <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ profileForm.displayName || userStore.email }}</h2>
            <p class="text-gray-500">{{ userStore.email }}</p>
            <el-tag size="small" type="primary" class="mt-1">{{ userStore.role }}</el-tag>
          </div>
        </div>

        <el-form :model="profileForm" label-width="120px">
          <el-form-item :label="t('profile.displayName')">
            <el-input v-model="profileForm.displayName" :placeholder="t('profile.displayNamePlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('profile.email')">
            <el-input :model-value="userStore.email" disabled />
          </el-form-item>
          <el-form-item :label="t('profile.phone')">
            <el-input v-model="profileForm.phone" :placeholder="t('profile.phonePlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('profile.company')">
            <el-input v-model="profileForm.company" :placeholder="t('profile.companyPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('profile.bio')">
            <el-input v-model="profileForm.bio" type="textarea" :rows="3" :placeholder="t('profile.bioPlaceholder')" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="saveProfile" :loading="saving">{{ t('common.save') }}</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- Change Password -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('profile.changePassword') }}</h3>
        <el-form :model="passwordForm" label-width="120px">
          <el-form-item :label="t('profile.currentPassword')">
            <el-input v-model="passwordForm.current" type="password" show-password />
          </el-form-item>
          <el-form-item :label="t('profile.newPassword')">
            <el-input v-model="passwordForm.newPwd" type="password" show-password />
          </el-form-item>
          <el-form-item :label="t('profile.confirmPassword')">
            <el-input v-model="passwordForm.confirm" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="changePassword" :loading="changingPwd">{{ t('profile.updatePassword') }}</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 2FA -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('profile.twoFactor') }}</h3>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('profile.twoFactorDesc') }}</p>
            <el-tag v-if="twoFactorEnabled" type="success" size="small" class="mt-2">{{ t('profile.twoFactorOn') }}</el-tag>
            <el-tag v-else type="warning" size="small" class="mt-2">{{ t('profile.twoFactorOff') }}</el-tag>
          </div>
          <el-switch v-model="twoFactorEnabled" @change="toggle2FA" />
        </div>
      </div>

      <!-- Login Devices -->
      <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
        <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('profile.loginDevices') }}</h3>
        <div class="space-y-3">
          <div v-for="device in devices" :key="device.id" class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div class="flex items-center gap-3">
              <el-icon :size="24" class="text-gray-400"><Monitor /></el-icon>
              <div>
                <p class="text-sm font-medium text-gray-900 dark:text-white">{{ device.name }}</p>
                <p class="text-xs text-gray-500">{{ device.location }} · {{ device.lastActive }}</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <el-tag v-if="device.current" type="success" size="small">{{ t('profile.currentDevice') }}</el-tag>
              <el-button v-else type="danger" link size="small" @click="revokeDevice(device.id)">{{ t('profile.revoke') }}</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { profileApi } from '@/api'
import { ElMessage } from 'element-plus'
import { Camera, Monitor } from '@element-plus/icons-vue'

const { t } = useI18n()
const userStore = useUserStore()

const avatarInput = ref<HTMLInputElement>()
const saving = ref(false)
const changingPwd = ref(false)
const twoFactorEnabled = ref(false)
const avatarUrl = ref('')

const profileForm = reactive({
  displayName: '',
  phone: '',
  company: '',
  bio: '',
})

const passwordForm = reactive({ current: '', newPwd: '', confirm: '' })

const devices = ref<any[]>([])

async function loadProfile() {
  try {
    const res = await profileApi.getProfile()
    const data = res.data || res
    profileForm.displayName = data.display_name || data.displayName || ''
    profileForm.phone = data.phone || ''
    profileForm.company = data.company || ''
    profileForm.bio = data.bio || ''
    twoFactorEnabled.value = data.two_factor_enabled || false
    avatarUrl.value = data.avatar_url || ''
  } catch {
    // 使用 localStorage 作为 fallback
    profileForm.displayName = localStorage.getItem('display_name') || ''
    profileForm.phone = localStorage.getItem('phone') || ''
    profileForm.company = localStorage.getItem('company') || ''
    profileForm.bio = localStorage.getItem('bio') || ''
  }
}

async function loadDevices() {
  try {
    const res = await profileApi.getDevices()
    devices.value = (res.data || res) || []
  } catch {
    // fallback
    devices.value = [
      { id: '1', name: 'Chrome on Windows', location: 'Beijing, China', last_active: t('profile.justNow'), current: true },
    ]
  }
}

function uploadAvatar() {
  avatarInput.value?.click()
}

async function handleAvatarChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  try {
    const formData = new FormData()
    formData.append('avatar', file)
    const res = await profileApi.uploadAvatar(formData)
    avatarUrl.value = res.avatar_url || res.data?.avatar_url || ''
    ElMessage.success(t('profile.avatarUpdated'))
  } catch {
    ElMessage.error('头像上传失败')
  }
}

async function saveProfile() {
  saving.value = true
  try {
    await profileApi.updateProfile({
      display_name: profileForm.displayName,
      phone: profileForm.phone,
      company: profileForm.company,
      bio: profileForm.bio,
    })
    localStorage.setItem('display_name', profileForm.displayName)
    ElMessage.success(t('profile.saved'))
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  if (!passwordForm.current || !passwordForm.newPwd || !passwordForm.confirm) {
    ElMessage.warning(t('profile.fillAllFields'))
    return
  }
  if (passwordForm.newPwd !== passwordForm.confirm) {
    ElMessage.error(t('profile.passwordMismatch'))
    return
  }
  changingPwd.value = true
  try {
    await profileApi.changePassword({
      current_password: passwordForm.current,
      new_password: passwordForm.newPwd,
    })
    ElMessage.success(t('profile.passwordChanged'))
    passwordForm.current = ''
    passwordForm.newPwd = ''
    passwordForm.confirm = ''
  } catch {
    ElMessage.error('密码修改失败，请检查当前密码是否正确')
  } finally {
    changingPwd.value = false
  }
}

async function toggle2FA() {
  try {
    await profileApi.toggle2FA({ enabled: twoFactorEnabled.value })
    ElMessage.success(twoFactorEnabled.value ? t('profile.twoFactorEnabled') : t('profile.twoFactorDisabled'))
  } catch {
    twoFactorEnabled.value = !twoFactorEnabled.value
    ElMessage.error('操作失败')
  }
}

async function revokeDevice(id: string) {
  try {
    await profileApi.revokeDevice(id)
    devices.value = devices.value.filter((d: any) => d.id !== id)
    ElMessage.success(t('profile.deviceRevoked'))
  } catch {
    ElMessage.error('操作失败')
  }
}

onMounted(() => {
  loadProfile()
  loadDevices()
})
</script>
