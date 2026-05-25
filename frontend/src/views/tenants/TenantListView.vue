<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('tenant.list') }}</h1>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon class="mr-1"><Plus /></el-icon> {{ t('tenant.create') }}
      </el-button>
    </div>

    <!-- 搜索过滤 -->
    <div class="flex gap-3 mb-4">
      <el-input v-model="searchQuery" :placeholder="t('common.search')" clearable class="w-60" prefix-icon="Search" />
      <el-select v-model="filterPlan" :placeholder="t('tenant.plan')" clearable class="w-36">
        <el-option label="Enterprise" value="enterprise" />
        <el-option label="Pro" value="pro" />
        <el-option label="Starter" value="starter" />
        <el-option label="Free" value="free" />
      </el-select>
      <el-select v-model="filterStatus" :placeholder="t('tenant.status')" clearable class="w-32">
        <el-option label="Active" value="active" />
        <el-option label="Suspended" value="suspended" />
        <el-option label="Trial" value="trial" />
      </el-select>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-4 gap-4 mb-6">
      <el-card shadow="never" class="!border-l-4 !border-l-blue-500">
        <div class="text-sm text-gray-500">{{ t('tenant.totalTenants') }}</div>
        <div class="text-2xl font-bold mt-1">{{ filteredTenants.length }}</div>
      </el-card>
      <el-card shadow="never" class="!border-l-4 !border-l-green-500">
        <div class="text-sm text-gray-500">{{ t('tenant.activeTenants') }}</div>
        <div class="text-2xl font-bold mt-1">{{ filteredTenants.filter(t => t.status === 'active').length }}</div>
      </el-card>
      <el-card shadow="never" class="!border-l-4 !border-l-orange-500">
        <div class="text-sm text-gray-500">{{ t('tenant.totalUsers') }}</div>
        <div class="text-2xl font-bold mt-1">{{ filteredTenants.reduce((sum, t) => sum + t.users, 0) }}</div>
      </el-card>
      <el-card shadow="never" class="!border-l-4 !border-l-purple-500">
        <div class="text-sm text-gray-500">{{ t('tenant.enterpriseTenants') }}</div>
        <div class="text-2xl font-bold mt-1">{{ filteredTenants.filter(t => t.plan === 'enterprise').length }}</div>
      </el-card>
    </div>

    <!-- 数据表格 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="paginatedTenants" stripe empty-text="暂无租户数据">
        <el-table-column prop="name" :label="t('tenant.name')" min-width="180">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-avatar :size="32" class="!bg-blue-100 !text-blue-600">
                {{ row.name.charAt(0) }}
              </el-avatar>
              <div>
                <div class="font-medium">{{ row.name }}</div>
                <div class="text-xs text-gray-400">{{ row.slug || '-' }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="plan" :label="t('tenant.plan')" width="120">
          <template #default="{ row }">
            <el-tag :type="planTagType(row.plan)" size="small">{{ planLabel(row.plan) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="region" :label="t('tenant.region')" width="80">
          <template #default="{ row }">
            <span class="text-sm">{{ regionLabel(row.region) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="users" :label="t('tenant.users')" width="80" />
        <el-table-column prop="max_api_keys" label="API Keys" width="100" />
        <el-table-column prop="status" :label="t('tenant.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : row.status === 'suspended' ? 'danger' : 'info'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('common.createdAt')" width="120" />
        <el-table-column :label="t('common.actions')" width="160" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="$router.push(`/tenants/${row.id}`)">
              {{ t('common.detail') }}
            </el-button>
            <el-button size="small" text type="warning">{{ t('common.config') }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="flex justify-end p-4">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="filteredTenants.length"
          layout="total, prev, pager, next"
          small
        />
      </div>
    </div>

    <!-- 创建租户对话框 -->
    <el-dialog v-model="showCreateDialog" :title="t('tenant.create')" width="500">
      <el-form :model="createForm" label-width="100px">
        <el-form-item :label="t('tenant.name')">
          <el-input v-model="createForm.name" />
        </el-form-item>
        <el-form-item :label="t('tenant.plan')">
          <el-select v-model="createForm.plan" class="w-full">
            <el-option label="Free" value="free" />
            <el-option label="Starter" value="starter" />
            <el-option label="Pro" value="pro" />
            <el-option label="Enterprise" value="enterprise" />
          </el-select>
        </el-form-item>
        <el-form-item label="Region">
          <el-select v-model="createForm.region" class="w-full">
            <el-option label="CN" value="cn" />
            <el-option label="US" value="us" />
            <el-option label="JP" value="jp" />
            <el-option label="KR" value="kr" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'

const { t } = useI18n()

const searchQuery = ref('')
const filterPlan = ref('')
const filterStatus = ref('')
const currentPage = ref(1)
const pageSize = 10
const showCreateDialog = ref(false)
const createForm = ref({ name: '', plan: 'free', region: 'cn' })

const tenants = ref([
  { id: '1', name: '示例科技有限公司', slug: 'example-tech', plan: 'enterprise', region: 'CN', users: 50, max_api_keys: 100, status: 'active', created_at: '2025-01-15' },
  { id: '2', name: 'TechCorp Inc.', slug: 'techcorp', plan: 'pro', region: 'US', users: 20, max_api_keys: 50, status: 'active', created_at: '2025-02-20' },
  { id: '3', name: 'テック株式会社', slug: 'tech-jp', plan: 'starter', region: 'JP', users: 5, max_api_keys: 10, status: 'trial', created_at: '2025-03-10' },
  { id: '4', name: '테크주식회사', slug: 'tech-kr', plan: 'free', region: 'KR', users: 2, max_api_keys: 3, status: 'active', created_at: '2025-04-05' },
  { id: '5', name: 'Acme Corp', slug: 'acme', plan: 'enterprise', region: 'US', users: 120, max_api_keys: 200, status: 'active', created_at: '2024-11-01' },
  { id: '6', name: '数据智能科技', slug: 'data-ai', plan: 'pro', region: 'CN', users: 35, max_api_keys: 50, status: 'active', created_at: '2025-01-20' },
  { id: '7', name: 'AI Solutions Ltd', slug: 'ai-sol', plan: 'starter', region: 'US', users: 8, max_api_keys: 10, status: 'suspended', created_at: '2025-03-15' },
])

const filteredTenants = computed(() => {
  return tenants.value.filter(t => {
    if (searchQuery.value && !t.name.toLowerCase().includes(searchQuery.value.toLowerCase())) return false
    if (filterPlan.value && t.plan !== filterPlan.value) return false
    if (filterStatus.value && t.status !== filterStatus.value) return false
    return true
  })
})

const paginatedTenants = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredTenants.value.slice(start, start + pageSize)
})

function planTagType(plan: string) {
  switch (plan) {
    case 'enterprise': return 'danger'
    case 'pro': return 'warning'
    case 'starter': return 'primary'
    default: return 'info'
  }
}

function planLabel(plan: string) {
  const map: Record<string, string> = { enterprise: 'Enterprise', pro: 'Pro', starter: 'Starter', free: 'Free' }
  return map[plan] || plan
}

function regionLabel(region: string) {
  const map: Record<string, string> = { CN: '🇨🇳', US: '🇺🇸', JP: '🇯🇵', KR: '🇰🇷' }
  return map[region] || region
}

function handleCreate() {
  if (!createForm.value.name) {
    ElMessage.warning('请输入租户名称')
    return
  }
  ElMessage.success('租户创建成功')
  showCreateDialog.value = false
  createForm.value = { name: '', plan: 'free', region: 'cn' }
}
</script>
