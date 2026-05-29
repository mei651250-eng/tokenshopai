import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'

const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || (window.Capacitor ? 'http://localhost:8080' : ''),
  timeout: 5000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    const tenantId = localStorage.getItem('tenant_id')
    if (tenantId) {
      config.headers['X-Tenant-ID'] = tenantId
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
api.interceptors.response.use(
  (response: AxiosResponse) => response.data,
  (error) => {
    const { response } = error
    if (response) {
      switch (response.status) {
        case 401:
          ElMessage.error('登录已过期，请重新登录')
          localStorage.removeItem('token')
          window.location.href = '/login'
          break
        case 403:
          ElMessage.error('权限不足')
          break
        case 429:
          ElMessage.error('请求频率超限，请稍后重试')
          break
        default:
          ElMessage.error(response.data?.error?.message || '请求失败')
      }
    } else {
      ElMessage.error('网络错误，请检查网络连接')
    }
    return Promise.reject(error)
  }
)

// 类型定义
export interface ChatCompletionRequest {
  model: string
  messages: Array<{ role: string; content: string }>
  stream?: boolean
  max_tokens?: number
  temperature?: number
}

export interface ModelConfig {
  id: string
  name: string
  provider: string
  model_id: string
  endpoint: string
  api_key?: string
  max_tokens: number
  input_price: number
  output_price: number
  currency: string
  enabled: boolean
  streamable: boolean
  latency_ms: number
  success_rate: number
  weight: number
  priority: number
  tenant_id: string
  tags: string[]
  created_at: number
  updated_at: number
}

export interface MetricsData {
  timestamp: number
  qps: number
  active_connections: number
  total_requests: number
  success_rate: number
  p50_latency_ms: number
  p99_latency_ms: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  total_amount: number
  models: Array<{
    model_id: string
    model_name: string
    provider: string
    requests: number
    success_rate: number
    avg_latency_ms: number
    tokens: number
    errors: number
    circuit_state: string
  }>
}

export interface BillingRecord {
  id: string
  tenant_id: string
  model_name: string
  provider: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  amount: number
  currency: string
  created_at: number
}

export interface Tenant {
  id: string
  name: string
  slug: string
  status: string
  plan: string
  region: string
  language: string
  currency: string
  timezone: string
  max_users: number
  max_api_keys: number
  created_at: string
}

// API 方法
export const authApi = {
  login: (data: { email: string; password: string }) =>
    api.post('/auth/login', data),
  register: (data: { username: string; email: string; password: string; tenant_id?: string }) =>
    api.post('/auth/register', data),
  sendCode: (data: { type: string; target: string; purpose: string; country_code?: string }) =>
    api.post('/auth/verification/send', data),
  loginByCode: (data: { type: string; target: string; code: string; country_code?: string }) =>
    api.post('/auth/login/code', data),
  registerByCode: (data: { type: string; target: string; code: string; country_code?: string }) =>
    api.post('/auth/register/code', data),
  walletLogin: (data: { address: string; signature: string; wallet_type: string }) =>
    api.post('/auth/login/wallet', data),

  // 人脸识别 (WebAuthn)
  faceRegisterOptions: () =>
    api.post('/auth/face/register-options'),
  faceRegisterVerify: (data: { session_key: string; credential: any }) =>
    api.post('/auth/face/register-verify', data),
  faceAuthOptions: (data: { email: string }) =>
    api.post('/auth/face/auth-options', data),
  faceAuthVerify: (data: { session_key: string; credential: any }) =>
    api.post('/auth/face/auth-verify', data),
}

export const modelApi = {
  list: () => api.get('/v1/models'),
  chat: (data: ChatCompletionRequest) => api.post('/v1/chat/completions', data),
}

export const adminApi = {
  // 模型管理
  getModels: () => api.get('/admin/models'),
  getModel: (id: string) => api.get(`/admin/models/${id}`),
  createModel: (data: Partial<ModelConfig>) => api.post('/admin/models', data),
  updateModel: (id: string, data: Partial<ModelConfig>) => api.put(`/admin/models/${id}`, data),
  deleteModel: (id: string) => api.delete(`/admin/models/${id}`),
  toggleModel: (id: string) => api.put(`/admin/models/${id}/toggle`),
  
  // 计费
  getBalance: () => api.get('/admin/billing/balance'),
  topUp: (amount: number) => api.post('/admin/billing/topup', { amount }),
  
  // 监控
  getMetrics: () => api.get('/admin/monitor/metrics'),
  
  // 安全
  getBlockedIPs: () => api.get('/admin/security/waf/blocked-ips'),
  blockIP: (ip: string) => api.post('/admin/security/waf/block-ip', { ip }),

  // 人脸识别凭据管理
  faceCredentials: () => api.get('/admin/face/credentials'),
  removeFaceCredential: (id: string) => api.delete(`/admin/face/credentials/${id}`),
}

// 钱包 API
export const walletApi = {
  list: () => api.get('/admin/wallet/list'),
  bind: (data: { wallet_type: string; address: string; chain_type: string; label?: string }) =>
    api.post('/admin/wallet/bind', data),
  unbind: (address: string) =>
    api.delete('/admin/wallet/unbind', { data: { address } }),
  getChallenge: (address: string) =>
    api.get('/admin/wallet/challenge', { params: { address } }),
  verify: (data: { address: string; signature: string }) =>
    api.post('/admin/wallet/verify', data),
  createDeposit: (data: { currency: string; chain_type: string; amount: string; fiat_currency?: string; from_address?: string; wallet_type?: string }) =>
    api.post('/admin/wallet/deposit', data),
  getDepositOrder: (orderNo: string) =>
    api.get(`/admin/wallet/deposit/${orderNo}`),
  depositOrders: () =>
    api.get('/admin/wallet/deposit'),
  exchangeRate: (crypto: string, fiat: string) =>
    api.get('/admin/wallet/exchange-rate', { params: { crypto, fiat } }),
  supportedTypes: () =>
    api.get('/admin/wallet/supported-types'),
  /** 连接钱包后同步绑定到后端 */
  connectAndBind: (data: { wallet_type: string; address: string; chain_type: string; signature?: string }) =>
    api.post('/admin/wallet/connect-bind', data),
  /** 获取连接钱包的余额 */
  getWalletBalance: (address: string, chain_type: string) =>
    api.get('/admin/wallet/balance', { params: { address, chain_type } }),
}

// 支付 API
export const paymentApi = {
  channels: (currency?: string) =>
    api.get('/admin/payment/channels', { params: { currency } }),
  create: (data: { channel: string; amount: number; currency: string; to_currency?: string }) =>
    api.post('/admin/payment/create', data),
  getOrder: (orderNo: string) =>
    api.get(`/admin/payment/order/${orderNo}`),
  orders: () =>
    api.get('/admin/payment/orders'),
}

// 财务 API（收款账号 + 提现）
export const financeApi = {
  // 收款账号
  listReceiving: () => api.get('/admin/finance/receiving'),
  createReceiving: (data: any) => api.post('/admin/finance/receiving', data),
  deleteReceiving: (id: string) => api.delete(`/admin/finance/receiving/${id}`),
  setPrimaryReceiving: (id: string) => api.put(`/admin/finance/receiving/${id}/primary`),

  // 提现账户
  listWithdrawAccounts: () => api.get('/admin/finance/withdraw-accounts'),
  createWithdrawAccount: (data: any) => api.post('/admin/finance/withdraw-accounts', data),
  deleteWithdrawAccount: (id: string) => api.delete(`/admin/finance/withdraw-accounts/${id}`),

  // 提现订单
  createWithdrawalOrder: (data: { account_id: string; amount: number; currency: string; remark?: string }) =>
    api.post('/admin/finance/withdrawal', data),
  listWithdrawalOrders: () => api.get('/admin/finance/withdrawal'),
  getWithdrawalOrder: (orderNo: string) => api.get(`/admin/finance/withdrawal/${orderNo}`),
}

// 报表 API
export const reportApi = {
  /** 获取概览统计数据 */
  summary: (params: { time_range: string }) =>
    api.get('/admin/report/summary', { params }),
  /** 获取请求趋势 */
  requestTrend: (params: { time_range: string; type?: string }) =>
    api.get('/admin/report/request-trend', { params }),
  /** 获取 Token 消耗趋势 */
  tokenTrend: (params: { time_range: string }) =>
    api.get('/admin/report/token-trend', { params }),
  /** 获取模型调用分布 */
  modelDistribution: (params: { time_range: string }) =>
    api.get('/admin/report/model-distribution', { params }),
  /** 获取费用分布 */
  costDistribution: (params: { time_range: string }) =>
    api.get('/admin/report/cost-distribution', { params }),
  /** 获取延迟分布 */
  latencyDistribution: (params: { time_range: string }) =>
    api.get('/admin/report/latency-distribution', { params }),
  /** 获取错误分析 */
  errorAnalysis: (params: { time_range: string }) =>
    api.get('/admin/report/error-analysis', { params }),
  /** 获取模型排行 */
  modelRanking: (params: { time_range: string }) =>
    api.get('/admin/report/model-ranking', { params }),
  /** 获取租户用量排行 */
  tenantRanking: (params: { time_range: string }) =>
    api.get('/admin/report/tenant-ranking', { params }),
  /** 导出报表 */
  exportReport: (params: { time_range: string; format: string }) =>
    api.get('/admin/report/export', { params, responseType: 'blob' }),
}

// 用户管理 API
export const userApi = {
  list: (params?: { search?: string; role?: string; status?: string; page?: number }) =>
    api.get('/admin/users', { params }),
  create: (data: { email: string; password: string; role: string; display_name?: string; tenant_id?: string }) =>
    api.post('/admin/users', data),
  update: (id: string, data: { role?: string; display_name?: string; status?: string }) =>
    api.put(`/admin/users/${id}`, data),
  delete: (id: string) =>
    api.delete(`/admin/users/${id}`),
  resetPassword: (id: string) =>
    api.post(`/admin/users/${id}/reset-password`),
  getRoles: () =>
    api.get('/admin/roles'),
  createRole: (data: { name: string; description: string; permissions: string[] }) =>
    api.post('/admin/roles', data),
  updateRole: (id: string, data: { name?: string; description?: string; permissions?: string[] }) =>
    api.put(`/admin/roles/${id}`, data),
  deleteRole: (id: string) =>
    api.delete(`/admin/roles/${id}`),
}

// 审计日志 API
export const auditApi = {
  list: (params?: { user?: string; action?: string; resource?: string; level?: string; start_date?: string; end_date?: string; page?: number; page_size?: number }) =>
    api.get('/admin/audit-logs', { params }),
  getDetail: (id: string) =>
    api.get(`/admin/audit-logs/${id}`),
  export: (params?: { user?: string; action?: string; resource?: string; start_date?: string; end_date?: string; format?: string }) =>
    api.get('/admin/audit-logs/export', { params, responseType: 'blob' }),
}

// 通知 API
export const notificationApi = {
  list: (params?: { type?: string; read?: boolean; page?: number }) =>
    api.get('/admin/notifications', { params }),
  markAsRead: (id: string) =>
    api.put(`/admin/notifications/${id}/read`),
  markAllAsRead: () =>
    api.put('/admin/notifications/read-all'),
  delete: (id: string) =>
    api.delete(`/admin/notifications/${id}`),
  clearAll: () =>
    api.delete('/admin/notifications'),
}

// 个人中心 API
export const profileApi = {
  getProfile: () =>
    api.get('/admin/profile'),
  updateProfile: (data: { display_name?: string; phone?: string; company?: string; bio?: string }) =>
    api.put('/admin/profile', data),
  uploadAvatar: (data: FormData) =>
    api.post('/admin/profile/avatar', data, { headers: { 'Content-Type': 'multipart/form-data' } }),
  changePassword: (data: { current_password: string; new_password: string }) =>
    api.put('/admin/profile/password', data),
  toggle2FA: (data: { enabled: boolean }) =>
    api.put('/admin/profile/2fa', data),
  getDevices: () =>
    api.get('/admin/profile/devices'),
  revokeDevice: (id: string) =>
    api.delete(`/admin/profile/devices/${id}`),
}

// 配额管理 API
export const quotaApi = {
  list: (params?: { tenant_id?: string }) =>
    api.get('/admin/quotas', { params }),
  get: (tenantId: string, quotaType: string) =>
    api.get(`/admin/quotas/${tenantId}/${quotaType}`),
  set: (data: { tenant_id: string; quota_type: string; limit: number; period_days?: number; alert_at?: number; block_at?: number }) =>
    api.post('/admin/quotas', data),
  reset: (tenantId: string, quotaType: string) =>
    api.post(`/admin/quotas/${tenantId}/${quotaType}/reset`),
}

// 退款管理 API
export const refundApi = {
  create: (data: { payment_order_no: string; amount: number; reason: string }) =>
    api.post('/admin/refunds', data),
  list: (params?: { page?: number }) =>
    api.get('/admin/refunds', { params }),
  listPending: (params?: { page?: number }) =>
    api.get('/admin/refunds/pending', { params }),
  get: (orderNo: string) =>
    api.get(`/admin/refunds/${orderNo}`),
  review: (data: { order_no: string; approved: boolean; reason?: string }) =>
    api.post(`/admin/refunds/${data.order_no}/review`, data),
}

// 对账 API
export const reconciliationApi = {
  getDaily: (params: { date: string; tenant_id?: string }) =>
    api.get('/admin/reconciliation/daily', { params }),
  getRange: (params: { start_date: string; end_date: string; tenant_id?: string }) =>
    api.get('/admin/reconciliation/range', { params }),
  getAggregated: (params: { start_date: string; end_date: string; tenant_id?: string }) =>
    api.get('/admin/reconciliation/aggregated', { params }),
}

// 分销管理 API
export const distributionApi = {
  register: (data: { role: string; commission_type: string; commission_rate: number }) =>
    api.post('/admin/distribution/register', data),
  listDistributors: (params?: { tenant_id?: string }) =>
    api.get('/admin/distribution/distributors', { params }),
  getDistributor: (id: string) =>
    api.get(`/admin/distribution/distributors/${id}`),
  listCommissions: (params?: { distributor_id?: string; page?: number }) =>
    api.get('/admin/distribution/commissions', { params }),
  settle: (data: { period: string }) =>
    api.post('/admin/distribution/settle', data),
}

export default api
