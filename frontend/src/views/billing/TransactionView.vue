<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('billing.transactionTitle') }}</h1>

    <!-- Filters -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700 mb-6">
      <div class="flex flex-wrap items-center gap-3">
        <el-select v-model="filters.type" :placeholder="t('billing.allTypes')" clearable style="width: 150px">
          <el-option :label="t('billing.charge')" value="charge" />
          <el-option :label="t('billing.consume')" value="consume" />
          <el-option :label="t('billing.refund')" value="refund" />
          <el-option :label="t('billing.withdraw')" value="withdraw" />
        </el-select>
        <el-select v-model="filters.status" :placeholder="t('billing.allStatus')" clearable style="width: 150px">
          <el-option :label="t('billing.success')" value="success" />
          <el-option :label="t('billing.pending')" value="pending" />
          <el-option :label="t('billing.failed')" value="failed" />
        </el-select>
        <el-date-picker v-model="filters.dateRange" type="daterange" :start-placeholder="t('common.startDate')" :end-placeholder="t('common.endDate')" style="width: 260px" />
        <el-button type="primary" @click="loadData">{{ t('common.search') }}</el-button>
        <el-button @click="exportData">{{ t('common.export') }}</el-button>
      </div>
    </div>

    <!-- Summary Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('billing.totalCharge') }}</p>
        <p class="text-2xl font-bold text-green-600 mt-1">¥ 12,580.00</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('billing.totalConsume') }}</p>
        <p class="text-2xl font-bold text-orange-600 mt-1">¥ 8,320.50</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('billing.totalRefund') }}</p>
        <p class="text-2xl font-bold text-blue-600 mt-1">¥ 450.00</p>
      </div>
    </div>

    <!-- Table -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="transactions" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="type" :label="t('billing.type')" width="100">
          <template #default="{ row }">
            <el-tag :type="typeTagMap[row.type] || 'info'" size="small">
              {{ t(`billing.${row.type}`) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="amount" :label="t('billing.amount')" width="120">
          <template #default="{ row }">
            <span :class="row.amount > 0 ? 'text-green-600' : 'text-red-600'">
              {{ row.amount > 0 ? '+' : '' }}{{ row.amount.toFixed(2) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="currency" :label="t('billing.currency')" width="80" />
        <el-table-column prop="model_name" :label="t('billing.modelName')" min-width="140" />
        <el-table-column prop="tokens" :label="t('billing.tokens')" width="100">
          <template #default="{ row }">
            {{ row.tokens ? row.tokens.toLocaleString() : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('billing.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : row.status === 'pending' ? 'warning' : 'danger'" size="small">
              {{ t(`billing.${row.status}`) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('common.createdAt')" width="170">
          <template #default="{ row }">
            {{ new Date(row.created_at * 1000).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="viewDetail(row)">{{ t('common.detail') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="p-4 flex justify-end">
        <el-pagination v-model:current-page="page" :page-size="20" :total="total" layout="total, prev, pager, next" />
      </div>
    </div>

    <!-- Detail Dialog -->
    <el-dialog v-model="showDetail" :title="t('billing.transactionDetail')" width="500">
      <el-descriptions v-if="selectedTx" :column="1" border>
        <el-descriptions-item :label="t('billing.orderNo')">{{ selectedTx.id }}</el-descriptions-item>
        <el-descriptions-item :label="t('billing.type')">{{ t(`billing.${selectedTx.type}`) }}</el-descriptions-item>
        <el-descriptions-item :label="t('billing.amount')">
          <span :class="selectedTx.amount > 0 ? 'text-green-600' : 'text-red-600'">{{ selectedTx.amount }}</span>
        </el-descriptions-item>
        <el-descriptions-item :label="t('billing.currency')">{{ selectedTx.currency }}</el-descriptions-item>
        <el-descriptions-item :label="t('billing.modelName')">{{ selectedTx.model_name }}</el-descriptions-item>
        <el-descriptions-item :label="t('billing.tokens')">{{ selectedTx.tokens?.toLocaleString() || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('billing.status')">{{ t(`billing.${selectedTx.status}`) }}</el-descriptions-item>
        <el-descriptions-item :label="t('common.createdAt')">{{ new Date(selectedTx.created_at * 1000).toLocaleString() }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { adminApi, default as api } from '@/api'

const { t } = useI18n()

const filters = reactive({ type: '', status: '', dateRange: null as any })
const page = ref(1)
const total = ref(0)
const transactions = ref<any[]>([])
const showDetail = ref(false)
const selectedTx = ref<any>(null)
const typeTagMap: Record<string, string> = { charge: 'success', consume: 'warning', refund: 'info', withdraw: 'danger' }

async function loadData() {
  loading.value = true
  try {
    const res: any = await adminApi.getBalance()
    const billingRes: any = await api.get('/admin/billing/transactions', { params: { ...filters.value, page: currentPage.value, page_size: pageSize.value } })
    transactions.value = billingRes.transactions || billingRes.data || []
    total.value = billingRes.total || transactions.value.length
  } catch (e) {
    console.error('Failed to load transactions', e)
  } finally {
    loading.value = false
  }
}

function viewDetail(row: any) {
  selectedTx.value = row
  showDetail.value = true
}

function exportData() {
  ElMessage.success(t('common.exporting'))
}

onMounted(loadData)
</script>
