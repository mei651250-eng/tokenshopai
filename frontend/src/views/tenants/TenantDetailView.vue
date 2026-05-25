<template>
  <div class="p-6">
    <!-- Back + Header -->
    <div class="flex items-center gap-3 mb-6">
      <el-button :icon="ArrowLeft" @click="router.push('/tenants')" text>{{ t('common.back') }}</el-button>
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ tenant?.name || t('tenants.detail') }}</h1>
      <el-tag v-if="tenant" :type="tenant.status === 'active' ? 'success' : tenant.status === 'suspended' ? 'warning' : 'info'" size="small">
        {{ t(`tenants.${tenant.status}`) }}
      </el-tag>
    </div>

    <div v-if="loading" class="space-y-4">
      <el-skeleton :rows="10" animated />
    </div>

    <template v-else>
      <!-- Stats -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('tenants.userCount') }}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ tenant?.user_count || 0 }} / {{ tenant?.max_users || 0 }}</p>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('tenants.apiKeys') }}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{{ tenant?.api_key_count || 0 }} / {{ tenant?.max_api_keys || 0 }}</p>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('tenants.monthlyUsage') }}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">¥ {{ tenant?.monthly_usage?.toLocaleString() || '0' }}</p>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
          <p class="text-sm text-gray-500">{{ t('tenants.plan') }}</p>
          <p class="text-2xl font-bold text-primary-600 mt-1">{{ tenant?.plan || '-' }}</p>
        </div>
      </div>

      <!-- Tabs -->
      <el-tabs v-model="activeTab">
        <!-- Info Tab -->
        <el-tab-pane :label="t('tenants.info')" name="info">
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
              <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('tenants.basicInfo') }}</h3>
              <el-form label-width="120px">
                <el-form-item :label="t('tenants.name')">
                  <el-input v-model="tenant!.name" />
                </el-form-item>
                <el-form-item :label="t('tenants.slug')">
                  <el-input v-model="tenant!.slug" disabled />
                </el-form-item>
                <el-form-item :label="t('tenants.region')">
                  <el-select v-model="tenant!.region">
                    <el-option label="CN" value="cn" />
                    <el-option label="US" value="us" />
                    <el-option label="EU" value="eu" />
                    <el-option label="AP" value="ap" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('tenants.language')">
                  <el-select v-model="tenant!.language">
                    <el-option label="中文" value="zh" />
                    <el-option label="English" value="en" />
                    <el-option label="日本語" value="ja" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('tenants.currency')">
                  <el-select v-model="tenant!.currency">
                    <el-option label="CNY" value="CNY" />
                    <el-option label="USD" value="USD" />
                    <el-option label="EUR" value="EUR" />
                    <el-option label="JPY" value="JPY" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary">{{ t('common.save') }}</el-button>
                </el-form-item>
              </el-form>
            </div>
            <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
              <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('tenants.quota') }}</h3>
              <el-form label-width="120px">
                <el-form-item :label="t('tenants.maxUsers')">
                  <el-input-number v-model="tenant!.max_users" :min="1" :max="10000" />
                </el-form-item>
                <el-form-item :label="t('tenants.maxApiKeys')">
                  <el-input-number v-model="tenant!.max_api_keys" :min="1" :max="100" />
                </el-form-item>
                <el-form-item :label="t('tenants.rateLimitRpm')">
                  <el-input-number v-model="tenant!.rpm_limit" :min="10" :max="100000" />
                </el-form-item>
                <el-form-item :label="t('tenants.rateLimitTpm')">
                  <el-input-number v-model="tenant!.tpm_limit" :min="1000" :max="10000000" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary">{{ t('common.save') }}</el-button>
                </el-form-item>
              </el-form>
            </div>
          </div>
        </el-tab-pane>

        <!-- Members Tab -->
        <el-tab-pane :label="t('tenants.members')" name="members">
          <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
            <div class="p-4 flex justify-between items-center border-b border-gray-200 dark:border-gray-700">
              <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('tenants.memberList') }}</h3>
              <el-button type="primary" size="small">{{ t('tenants.addMember') }}</el-button>
            </div>
            <el-table :data="members" stripe>
              <el-table-column prop="email" :label="t('users.email')" />
              <el-table-column prop="role" :label="t('users.role')" width="120">
                <template #default="{ row }">
                  <el-tag :type="row.role === 'admin' ? 'danger' : row.role === 'editor' ? 'warning' : 'info'" size="small">{{ row.role }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="status" :label="t('common.status')" width="100">
                <template #default="{ row }">
                  <span class="flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full" :class="row.status === 'active' ? 'bg-green-500' : 'bg-gray-400'" />
                    {{ row.status }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="joined_at" :label="t('tenants.joinedAt')" width="170" />
              <el-table-column :label="t('common.actions')" width="120" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link size="small">{{ t('common.edit') }}</el-button>
                  <el-button type="danger" link size="small">{{ t('common.remove') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Billing Tab -->
        <el-tab-pane :label="t('tenants.billing')" name="billing">
          <div class="bg-white dark:bg-gray-800 rounded-xl p-6 border border-gray-200 dark:border-gray-700">
            <h3 class="font-semibold text-gray-900 dark:text-white mb-4">{{ t('tenants.billingInfo') }}</h3>
            <el-descriptions :column="2" border>
              <el-descriptions-item :label="t('tenants.currentPlan')">{{ tenant?.plan }}</el-descriptions-item>
              <el-descriptions-item :label="t('tenants.billingCycle')">{{ t('tenants.monthly') }}</el-descriptions-item>
              <el-descriptions-item :label="t('tenants.monthlyFee')">¥ 999.00</el-descriptions-item>
              <el-descriptions-item :label="t('tenants.usageFee')">¥ {{ tenant?.monthly_usage?.toLocaleString() || '0' }}</el-descriptions-item>
              <el-descriptions-item :label="t('tenants.nextBilling')">{{ new Date(Date.now() + 86400000 * 7).toLocaleDateString() }}</el-descriptions-item>
            </el-descriptions>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const loading = ref(true)
const activeTab = ref('info')
const tenant = ref<any>(null)

const members = ref([
  { email: 'admin@example.com', role: 'admin', status: 'active', joined_at: '2024-01-15' },
  { email: 'user1@example.com', role: 'editor', status: 'active', joined_at: '2024-03-20' },
  { email: 'user2@example.com', role: 'viewer', status: 'inactive', joined_at: '2024-06-10' },
])

async function loadTenant() {
  loading.value = true
  const id = route.params.id
  // Mock data
  tenant.value = {
    id, name: `Tenant ${id}`, slug: `tenant-${id}`, status: 'active', plan: 'Enterprise',
    region: 'cn', language: 'zh', currency: 'CNY',
    user_count: 25, max_users: 100, api_key_count: 8, max_api_keys: 50,
    monthly_usage: 15800, rpm_limit: 5000, tpm_limit: 2000000,
  }
  loading.value = false
}

onMounted(loadTenant)
</script>
