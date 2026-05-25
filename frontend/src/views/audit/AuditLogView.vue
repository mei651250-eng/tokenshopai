<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{{ t('audit.title') }}</h1>

    <!-- Filters -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-4 border border-gray-200 dark:border-gray-700 mb-6">
      <div class="flex flex-wrap items-center gap-3">
        <el-input v-model="filters.user" :placeholder="t('audit.userPlaceholder')" prefix-icon="User" clearable style="width: 200px" />
        <el-select v-model="filters.action" :placeholder="t('audit.allActions')" clearable style="width: 150px">
          <el-option label="CREATE" value="create" />
          <el-option label="UPDATE" value="update" />
          <el-option label="DELETE" value="delete" />
          <el-option label="LOGIN" value="login" />
          <el-option label="LOGOUT" value="logout" />
          <el-option label="EXPORT" value="export" />
        </el-select>
        <el-select v-model="filters.resource" :placeholder="t('audit.allResources')" clearable style="width: 150px">
          <el-option label="Model" value="model" />
          <el-option label="User" value="user" />
          <el-option label="Tenant" value="tenant" />
          <el-option label="Payment" value="payment" />
          <el-option label="Security" value="security" />
          <el-option label="Setting" value="setting" />
        </el-select>
        <el-select v-model="filters.level" :placeholder="t('audit.allLevels')" clearable style="width: 120px">
          <el-option label="Info" value="info" />
          <el-option label="Warning" value="warning" />
          <el-option label="Error" value="error" />
        </el-select>
        <el-date-picker v-model="filters.dateRange" type="daterange" :start-placeholder="t('common.startDate')" :end-placeholder="t('common.endDate')" style="width: 260px" />
        <el-button type="primary" @click="loadData">{{ t('common.search') }}</el-button>
        <el-button @click="exportLogs">{{ t('common.export') }}</el-button>
      </div>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('audit.todayOps') }}</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">256</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('audit.createOps') }}</p>
        <p class="text-2xl font-bold text-green-600 mt-1">89</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('audit.deleteOps') }}</p>
        <p class="text-2xl font-bold text-red-600 mt-1">12</p>
      </div>
      <div class="bg-white dark:bg-gray-800 rounded-xl p-5 border border-gray-200 dark:border-gray-700">
        <p class="text-sm text-gray-500">{{ t('audit.errorOps') }}</p>
        <p class="text-2xl font-bold text-orange-600 mt-1">3</p>
      </div>
    </div>

    <!-- Table -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
      <el-table :data="logs" stripe>
        <el-table-column prop="timestamp" :label="t('audit.timestamp')" width="170" />
        <el-table-column prop="user" :label="t('audit.user')" width="180" />
        <el-table-column prop="action" :label="t('audit.action')" width="100">
          <template #default="{ row }">
            <el-tag :type="actionTagType[row.action] || 'info'" size="small">{{ row.action.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" :label="t('audit.resource')" width="120" />
        <el-table-column prop="resource_id" :label="t('audit.resourceId')" width="140" />
        <el-table-column prop="detail" :label="t('audit.detail')" min-width="250" show-overflow-tooltip />
        <el-table-column prop="ip" :label="t('audit.ip')" width="130" />
        <el-table-column prop="level" :label="t('audit.level')" width="90">
          <template #default="{ row }">
            <span class="flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full" :class="row.level === 'error' ? 'bg-red-500' : row.level === 'warning' ? 'bg-yellow-500' : 'bg-blue-400'" />
              {{ row.level }}
            </span>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="80" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="viewDetail(row)">{{ t('common.detail') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="p-4 flex justify-end">
        <el-pagination v-model:current-page="page" :page-size="20" :total="256" layout="total, sizes, prev, pager, next" :page-sizes="[10, 20, 50, 100]" />
      </div>
    </div>

    <!-- Detail Drawer -->
    <el-drawer v-model="showDetail" :title="t('audit.logDetail')" size="500px">
      <el-descriptions v-if="selectedLog" :column="1" border>
        <el-descriptions-item :label="t('audit.timestamp')">{{ selectedLog.timestamp }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.user')">{{ selectedLog.user }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.action')">{{ selectedLog.action }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.resource')">{{ selectedLog.resource }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.resourceId')">{{ selectedLog.resource_id }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.ip')">{{ selectedLog.ip }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.userAgent')">{{ selectedLog.user_agent }}</el-descriptions-item>
        <el-descriptions-item :label="t('audit.detail')">
          <pre class="whitespace-pre-wrap text-sm bg-gray-50 dark:bg-gray-900 p-3 rounded-lg">{{ selectedLog.detail }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'

const { t } = useI18n()

const filters = reactive({ user: '', action: '', resource: '', level: '', dateRange: null as any })
const page = ref(1)
const logs = ref<any[]>([])
const showDetail = ref(false)
const selectedLog = ref<any>(null)
const actionTagType: Record<string, string> = { create: 'success', update: 'warning', delete: 'danger', login: '', logout: 'info', export: 'info' }

async function loadData() {
  const now = Date.now()
  const actions = ['create', 'update', 'delete', 'login', 'logout', 'export']
  const resources = ['model', 'user', 'tenant', 'payment', 'security', 'setting']
  const levels = ['info', 'info', 'info', 'warning', 'error']
  logs.value = Array.from({ length: 20 }, (_, i) => ({
    timestamp: new Date(now - i * 180000).toLocaleString(),
    user: ['admin@example.com', 'user1@example.com', 'user2@example.com'][i % 3],
    action: actions[i % actions.length],
    resource: resources[i % resources.length],
    resource_id: `res_${100 + i}`,
    detail: `${actions[i % actions.length].toUpperCase()} ${resources[i % resources.length]} res_${100 + i} by ${['admin@example.com', 'user1@example.com', 'user2@example.com'][i % 3]}`,
    ip: `192.168.${i % 10}.${(i * 37) % 256}`,
    level: levels[i % levels.length],
    user_agent: ['Chrome/120', 'Firefox/121', 'Safari/17'][i % 3],
  }))
}

function viewDetail(row: any) {
  selectedLog.value = row
  showDetail.value = true
}

function exportLogs() {
  ElMessage.success(t('common.exporting'))
}

onMounted(loadData)
</script>
