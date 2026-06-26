import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'

NProgress.configure({ showSpinner: false })

// 路由元信息类型
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    fullscreen?: boolean
    roles?: string[]  // 允许访问的角色列表，为空表示所有已认证用户可访问
    permission?: string  // 需要的权限标识
  }
}

const routes: RouteRecordRaw[] = [
  // ========== 公开页面 ==========
  {
    path: '/',
    name: 'Landing',
    component: () => import('@/views/landing/LandingView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/login/code',
    name: 'CodeLogin',
    component: () => import('@/views/auth/CodeLoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/login/face',
    name: 'FaceLogin',
    component: () => import('@/views/auth/FaceLoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/auth/RegisterView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/terms',
    name: 'Terms',
    component: () => import('@/views/legal/TermsView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/privacy',
    name: 'Privacy',
    component: () => import('@/views/legal/PrivacyView.vue'),
    meta: { requiresAuth: false },
  },

  // ========== 用户端（UserLayout）==========
  {
    path: '/home',
    component: () => import('@/components/layout/UserLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'UserHome',
        component: () => import('@/views/home/UserHomeView.vue'),
      },
      {
        path: '/apikeys',
        name: 'ApiKeys',
        component: () => import('@/views/apikeys/ApiKeyView.vue'),
      },
      {
        path: '/marketplace',
        name: 'Marketplace',
        component: () => import('@/views/marketplace/MarketplaceView.vue'),
      },
      {
        path: '/docs',
        name: 'Docs',
        component: () => import('@/views/docs/DocsView.vue'),
      },
      {
        path: '/topup',
        name: 'TopUp',
        component: () => import('@/views/topup/TopUpView.vue'),
      },
      {
        path: '/payment',
        name: 'Payment',
        component: () => import('@/views/payment/PaymentView.vue'),
      },
      {
        path: '/billing',
        name: 'Billing',
        component: () => import('@/views/billing/BillingView.vue'),
        meta: { permission: 'billing:view' },
      },
      {
        path: '/billing/transactions',
        name: 'Transactions',
        component: () => import('@/views/billing/TransactionView.vue'),
        meta: { permission: 'billing:view' },
      },
      {
        path: '/wallet',
        name: 'Wallet',
        component: () => import('@/views/wallet/WalletView.vue'),
      },
      {
        path: '/usage',
        name: 'UsageLogs',
        component: () => import('@/views/usage/UsageLogView.vue'),
      },
      {
        path: '/referrals',
        name: 'Referrals',
        component: () => import('@/views/referral/ReferralView.vue'),
      },
      {
        path: '/profile',
        name: 'Profile',
        component: () => import('@/views/profile/ProfileView.vue'),
      },
      {
        path: '/subscription',
        name: 'Subscription',
        component: () => import('@/views/subscription/SubscriptionView.vue'),
      },
      {
        path: '/notifications',
        name: 'Notifications',
        component: () => import('@/views/notifications/NotificationCenterView.vue'),
      },
      {
        path: '/models',
        name: 'Models',
        component: () => import('@/views/models/ModelListView.vue'),
        meta: { permission: 'model:list' },
      },
      {
        path: '/models/:id',
        name: 'ModelDetail',
        component: () => import('@/views/models/ModelDetailView.vue'),
        meta: { permission: 'model:list' },
      },
    ],
  },

  // ========== 管理端（AdminLayout）==========
  {
    path: '/admin',
    component: () => import('@/components/layout/AdminLayout.vue'),
    meta: { requiresAuth: true, roles: ['super_admin', 'tenant_admin'] },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/DashboardView.vue'),
      },
      {
        path: 'models',
        name: 'AdminModels',
        component: () => import('@/views/models/ModelListView.vue'),
        meta: { permission: 'model:list' },
      },
      {
        path: 'models/:id',
        name: 'AdminModelDetail',
        component: () => import('@/views/models/ModelDetailView.vue'),
        meta: { permission: 'model:list' },
      },
      {
        path: 'channels',
        name: 'Channels',
        component: () => import('@/views/channels/ChannelView.vue'),
        meta: { permission: 'model:list' },
      },
      {
        path: 'tokens',
        name: 'Tokens',
        component: () => import('@/views/tokens/TokenView.vue'),
      },
      {
        path: 'tenants',
        name: 'Tenants',
        component: () => import('@/views/tenants/TenantListView.vue'),
      },
      {
        path: 'tenants/:id',
        name: 'TenantDetail',
        component: () => import('@/views/tenants/TenantDetailView.vue'),
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/users/UserManagementView.vue'),
        meta: { permission: 'user:create' },
      },
      {
        path: 'audit',
        name: 'AuditLog',
        component: () => import('@/views/audit/AuditLogView.vue'),
      },
      {
        path: 'security',
        name: 'Security',
        component: () => import('@/views/security/SecurityView.vue'),
        meta: { roles: ['super_admin'], permission: 'security:view' },
      },
      {
        path: 'monitor',
        name: 'Monitor',
        component: () => import('@/views/monitor/MonitorView.vue'),
        meta: { permission: 'monitor:view' },
      },
      {
        path: 'reports',
        name: 'Reports',
        component: () => import('@/views/reports/ReportView.vue'),
        meta: { permission: 'report:view' },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/settings/SettingsView.vue'),
      },
      {
        path: 'quota',
        name: 'QuotaManagement',
        component: () => import('@/views/quota/QuotaManagementView.vue'),
      },
      {
        path: 'refund',
        name: 'RefundManagement',
        component: () => import('@/views/refund/RefundManagementView.vue'),
      },
      {
        path: 'reconciliation',
        name: 'Reconciliation',
        component: () => import('@/views/reconciliation/ReconciliationView.vue'),
      },
      {
        path: 'distribution',
        name: 'Distribution',
        component: () => import('@/views/distribution/DistributionView.vue'),
      },
      {
        path: 'redeem-codes',
        name: 'RedeemCodes',
        component: () => import('@/views/redeem/RedeemCodeView.vue'),
      },
      {
        path: 'announcements',
        name: 'Announcements',
        component: () => import('@/views/announcements/AnnouncementView.vue'),
      },
      {
        path: 'model-mappings',
        name: 'ModelMappings',
        component: () => import('@/views/mapping/ModelMappingView.vue'),
      },
      {
        path: 'user-groups',
        name: 'UserGroups',
        component: () => import('@/views/groups/UserGroupView.vue'),
      },
      {
        path: 'subscription-plans',
        name: 'SubscriptionPlans',
        component: () => import('@/views/admin/SubscriptionPlansView.vue'),
      },
    ],
  },

  // ========== 分销商端（DistributorLayout）==========
  {
    path: '/distributor',
    component: () => import('@/components/layout/DistributorLayout.vue'),
    meta: { requiresAuth: true, roles: ['agent', 'referrer', 'reseller', 'affiliate'] },
    children: [
      {
        path: '',
        name: 'DistributorDashboard',
        component: () => import('@/views/distributor/DistributorDashboardView.vue'),
      },
      {
        path: 'links',
        name: 'DistributorLinks',
        component: () => import('@/views/distributor/DistributorLinksView.vue'),
      },
      {
        path: 'referrals',
        name: 'DistributorReferrals',
        component: () => import('@/views/distributor/DistributorReferralsView.vue'),
      },
      {
        path: 'commissions',
        name: 'DistributorCommissions',
        component: () => import('@/views/distributor/DistributorCommissionsView.vue'),
      },
      {
        path: 'withdraw',
        name: 'DistributorWithdraw',
        component: () => import('@/views/distributor/DistributorWithdrawView.vue'),
      },
      {
        path: 'materials',
        name: 'DistributorMaterials',
        component: () => import('@/views/distributor/DistributorMaterialsView.vue'),
      },
      {
        path: 'profile',
        name: 'DistributorProfile',
        component: () => import('@/views/profile/ProfileView.vue'),
      },
    ],
  },
  {
    path: '/monitor/screen',
    name: 'MonitorScreen',
    component: () => import('@/views/monitor/MonitorScreenView.vue'),
    meta: { requiresAuth: true, fullscreen: true, permission: 'monitor:view' },
  },
  {
    path: '/error/:code',
    name: 'Error',
    component: () => import('@/views/errors/ErrorPageView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/errors/ErrorPageView.vue'),
    meta: { requiresAuth: false },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 角色权限映射（与后端 auth.RBAC 保持一致）
const rolePermissions: Record<string, string[]> = {
  super_admin: ['*'],
  tenant_admin: ['model:list', 'model:create', 'model:update', 'model:route', 'apikey:list', 'apikey:create', 'apikey:revoke', 'billing:view', 'billing:topup', 'billing:export', 'user:create', 'user:update', 'user:delete', 'monitor:view', 'monitor:export', 'security:view', 'report:view', 'report:export'],
  dept_admin: ['model:list', 'model:route', 'apikey:list', 'apikey:create', 'billing:view', 'user:create', 'user:update', 'monitor:view', 'report:view'],
  developer: ['model:list', 'model:route', 'apikey:list', 'apikey:create', 'billing:view', 'monitor:view'],
  viewer: ['model:list', 'billing:view', 'monitor:view', 'report:view'],
  api_consumer: ['model:route'],
  // 分销商角色权限
  agent: ['distributor:dashboard', 'distributor:links', 'distributor:referrals', 'distributor:commissions', 'distributor:withdraw', 'distributor:materials'],
  referrer: ['distributor:dashboard', 'distributor:links', 'distributor:referrals', 'distributor:commissions'],
  reseller: ['distributor:dashboard', 'distributor:links', 'distributor:referrals', 'distributor:commissions', 'distributor:withdraw'],
  affiliate: ['distributor:dashboard', 'distributor:links', 'distributor:commissions'],
}

function hasPermission(role: string, permission?: string): boolean {
  if (!permission) return true
  const perms = rolePermissions[role] || []
  if (perms.includes('*')) return true
  return perms.includes(permission)
}

router.beforeEach((to, _from, next) => {
  NProgress.start()

  // 处理 OAuth 回调带来的 token
  if (window.location.hash.includes('token=')) {
    const params = new URLSearchParams(window.location.hash.split('?')[1] || '')
    const token = params.get('token')
    const refreshToken = params.get('refresh_token')
    if (token) {
      localStorage.setItem('token', token)
      if (refreshToken) localStorage.setItem('refresh_token', refreshToken)
      // 解析 JWT 获取用户信息
      try {
        const payload = JSON.parse(atob(token.split('.')[1]))
        if (payload.user_id) localStorage.setItem('user_id', payload.user_id)
        if (payload.tenant_id) localStorage.setItem('tenant_id', payload.tenant_id)
        if (payload.email) localStorage.setItem('email', payload.email)
        if (payload.role) localStorage.setItem('role', payload.role)
      } catch { /* ignore */ }
      // 清除 URL 参数，根据角色跳转
      const payload = JSON.parse(atob(token.split('.')[1]))
      const userRole = payload.role || ''
      const isAdmin = userRole === 'super_admin' || userRole === 'tenant_admin'
      const isDistributor = ['agent', 'referrer', 'reseller', 'affiliate'].includes(userRole)
      if (isAdmin) {
        window.location.hash = '#/admin/dashboard'
      } else if (isDistributor) {
        window.location.hash = '#/distributor'
      } else {
        window.location.hash = '#/home'
      }
      return
    }
  }

  const token = localStorage.getItem('token')
  const role = localStorage.getItem('role') || ''

  // 认证检查
  if (to.meta.requiresAuth && !token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
    return
  }

  // 已登录访问着陆页或登录页，根据角色重定向
  if ((to.name === 'Landing' || to.name === 'Login') && token) {
    const isAdmin = role === 'super_admin' || role === 'tenant_admin'
    const isDistributor = ['agent', 'referrer', 'reseller', 'affiliate'].includes(role)
    if (isAdmin) {
      next({ name: 'Dashboard' })
    } else if (isDistributor) {
      next({ name: 'DistributorDashboard' })
    } else {
      next({ name: 'UserHome' })
    }
    return
  }

  // 角色检查
  if (to.meta.roles && to.meta.roles.length > 0 && !to.meta.roles.includes(role)) {
    next({ name: 'Error', params: { code: '403' } })
    return
  }

  // 权限检查
  if (to.meta.permission && !hasPermission(role, to.meta.permission)) {
    next({ name: 'Error', params: { code: '403' } })
    return
  }

  next()
})

router.afterEach(() => {
  NProgress.done()
})

export default router
