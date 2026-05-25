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
    path: '/',
    component: () => import('@/components/layout/AdminLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/DashboardView.vue'),
      },
      {
        path: 'models',
        name: 'Models',
        component: () => import('@/views/models/ModelListView.vue'),
        meta: { permission: 'model:list' },
      },
      {
        path: 'models/:id',
        name: 'ModelDetail',
        component: () => import('@/views/models/ModelDetailView.vue'),
        meta: { permission: 'model:list' },
      },
      {
        path: 'billing',
        name: 'Billing',
        component: () => import('@/views/billing/BillingView.vue'),
        meta: { permission: 'billing:view' },
      },
      {
        path: 'billing/transactions',
        name: 'Transactions',
        component: () => import('@/views/billing/TransactionView.vue'),
        meta: { permission: 'billing:view' },
      },
      {
        path: 'wallet',
        name: 'Wallet',
        component: () => import('@/views/wallet/WalletView.vue'),
      },
      {
        path: 'payment',
        name: 'Payment',
        component: () => import('@/views/payment/PaymentView.vue'),
      },
      {
        path: 'receiving',
        name: 'Receiving',
        component: () => import('@/views/finance/ReceivingAccountView.vue'),
      },
      {
        path: 'withdrawal',
        name: 'Withdrawal',
        component: () => import('@/views/finance/WithdrawalView.vue'),
      },
      {
        path: 'tenants',
        name: 'Tenants',
        component: () => import('@/views/tenants/TenantListView.vue'),
        meta: { roles: ['super_admin', 'tenant_admin'] },
      },
      {
        path: 'tenants/:id',
        name: 'TenantDetail',
        component: () => import('@/views/tenants/TenantDetailView.vue'),
        meta: { roles: ['super_admin', 'tenant_admin'] },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/users/UserManagementView.vue'),
        meta: { roles: ['super_admin', 'tenant_admin'], permission: 'user:create' },
      },
      {
        path: 'audit',
        name: 'AuditLog',
        component: () => import('@/views/audit/AuditLogView.vue'),
        meta: { roles: ['super_admin', 'tenant_admin'] },
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
        meta: { roles: ['super_admin', 'tenant_admin'] },
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/profile/ProfileView.vue'),
      },
      {
        path: 'notifications',
        name: 'Notifications',
        component: () => import('@/views/notifications/NotificationCenterView.vue'),
      },
      {
        path: 'quota',
        name: 'QuotaManagement',
        component: () => import('@/views/quota/QuotaManagementView.vue'),
        meta: { roles: ['super_admin', 'tenant_admin'] },
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
        meta: { roles: ['super_admin', 'tenant_admin'] },
      },
      {
        path: 'distribution',
        name: 'Distribution',
        component: () => import('@/views/distribution/DistributionView.vue'),
        meta: { roles: ['super_admin', 'tenant_admin'] },
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
}

function hasPermission(role: string, permission?: string): boolean {
  if (!permission) return true
  const perms = rolePermissions[role] || []
  if (perms.includes('*')) return true
  return perms.includes(permission)
}

router.beforeEach((to, _from, next) => {
  NProgress.start()

  const token = localStorage.getItem('token')
  const role = localStorage.getItem('role') || ''

  // 认证检查
  if (to.meta.requiresAuth && !token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
    return
  }

  // 已登录访问登录页，重定向到首页
  if (to.name === 'Login' && token) {
    next({ name: 'Dashboard' })
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
