import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userId = ref(localStorage.getItem('user_id') || '')
  const tenantId = ref(localStorage.getItem('tenant_id') || '')
  const email = ref(localStorage.getItem('email') || '')
  const role = ref(localStorage.getItem('role') || '')
  const locale = ref(localStorage.getItem('locale') || 'zh-CN')
  const timezone = ref(localStorage.getItem('timezone') || 'Asia/Shanghai')

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => ['super_admin', 'tenant_admin'].includes(role.value))

  async function login(emailVal: string, password: string) {
    const res: any = await authApi.login({ email: emailVal, password })
    // 兼容新旧API格式：新格式 { token: { access_token, ... }, user: { ... } }
    const tokenData = res.token || res
    const userData = res.user || {}
    token.value = tokenData.access_token
    localStorage.setItem('token', tokenData.access_token)
    localStorage.setItem('refresh_token', tokenData.refresh_token)
    // 优先使用user对象，其次从JWT解码
    if (userData.id) {
      userId.value = userData.id || ''
      tenantId.value = userData.tenant_id || ''
      email.value = userData.email || ''
      role.value = userData.role || ''
    } else {
      const payload = JSON.parse(atob(tokenData.access_token.split('.')[1]))
      userId.value = payload.user_id || ''
      tenantId.value = payload.tenant_id || ''
      email.value = payload.email || ''
      role.value = payload.role || ''
    }
    localStorage.setItem('user_id', userId.value)
    localStorage.setItem('tenant_id', tenantId.value)
    localStorage.setItem('email', email.value)
    localStorage.setItem('role', role.value)
  }

  function logout() {
    token.value = ''
    userId.value = ''
    tenantId.value = ''
    email.value = ''
    role.value = ''
    localStorage.clear()
  }

  function setLocale(l: string) {
    locale.value = l
    localStorage.setItem('locale', l)
  }

  return {
    token, userId, tenantId, email, role, locale, timezone,
    isLoggedIn, isAdmin,
    login, logout, setLocale,
  }
})
