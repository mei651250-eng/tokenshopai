<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { quotaApi, adminApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const loading = ref(false)
const quotas = ref<any[]>([])
const tenants = ref<any[]>([])

const showSetQuota = ref(false)
const saving = ref(false)
const quotaForm = reactive({
  tenant_id: '',
  quota_type: 'daily_tokens',
  limit: 1000000,
  period_days: 1,
  alert_at: 0.8,
  block_at: 1.0,
})

const quotaTypes = [
  { value: 'daily_tokens', label: '每日Token配额' },
  { value: 'monthly_tokens', label: '每月Token配额' },
  { value: 'daily_requests', label: '每日请求配额' },
  { value: 'monthly_amount', label: '每月消费上限(分)' },
]

async function loadQuotas() {
  loading.value = true
  try {
    const res = await quotaApi.list()
    quotas.value = res.quotas || res.data || []
  } catch {
    quotas.value = []
  } finally {
    loading.value = false
  }
}

async function loadTenants() {
  try {
    const res = await adminApi.listTenants()
    tenants.value = res.tenants || res.data || []
  } catch {
    tenants.value = []
  }
}

function openSetQuota(tenantId?: string) {
  quotaForm.tenant_id = tenantId || ''
  showSetQuota.value = true
}

async function saveQuota() {
  saving.value = true
  try {
    await quotaApi.set(quotaForm)
    ElMessage.success('配额设置成功')
    showSetQuota.value = false
    loadQuotas()
  } catch {
    ElMessage.error('设置失败')
  } finally {
    saving.value = false
  }
}

async function resetQuota(tenantId: string, quotaType: string) {
  await ElMessageBox.confirm('确认重置该配额的已用量？', '重置配额')
  try {
    await quotaApi.reset(tenantId, quotaType)
    ElMessage.success('配额已重置')
    loadQuotas()
  } catch {
    ElMessage.error('重置失败')
  }
}

function getQuotaColor(percentage: number) {
  if (percentage >= 1) return '#F56C6C'
  if (percentage >= 0.8) return '#E6A23C'
  return '#67C23A'
}

function formatQuotaType(type: string) {
  return quotaTypes.find(q => q.value === type)?.label || type
}

onMounted(() => {
  loadQuotas()
  loadTenants()
})
</script>

<template>
  <div class="quota-management">
    <div class="page-header">
      <h2>配额管理</h2>
      <el-button type="primary" @click="openSetQuota()">设置配额</el-button>
    </div>

    <el-alert type="info" :closable="false" style="margin-bottom: 16px">
      配额管理可以限制租户的 Token 使用量和 API 调用频率，防止恶意调用和超支。
    </el-alert>

    <el-table :data="quotas" v-loading="loading" stripe>
      <el-table-column prop="tenant_id" label="租户ID" width="200" />
      <el-table-column label="配额类型" width="160">
        <template #default="{ row }">{{ formatQuotaType(row.quota_type) }}</template>
      </el-table-column>
      <el-table-column label="限制" width="140">
        <template #default="{ row }">{{ row.limit?.toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="已用" width="140">
        <template #default="{ row }">{{ row.used?.toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="使用率" width="200">
        <template #default="{ row }">
          <el-progress
            :percentage="Math.min(100, Math.round((row.percentage || 0) * 100))"
            :color="getQuotaColor(row.percentage || 0)"
            :stroke-width="16"
          />
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.is_exceeded ? 'danger' : 'success'" size="small">
            {{ row.is_exceeded ? '已超限' : '正常' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button type="warning" size="small" link @click="resetQuota(row.tenant_id, row.quota_type)">
            重置
          </el-button>
          <el-button type="primary" size="small" link @click="openSetQuota(row.tenant_id)">
            编辑
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 设置配额弹窗 -->
    <el-dialog v-model="showSetQuota" title="设置配额" width="500px">
      <el-form :model="quotaForm" label-width="100px">
        <el-form-item label="租户" required>
          <el-select v-model="quotaForm.tenant_id" placeholder="选择租户" style="width: 100%">
            <el-option v-for="t in tenants" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="配额类型" required>
          <el-select v-model="quotaForm.quota_type" style="width: 100%">
            <el-option v-for="qt in quotaTypes" :key="qt.value" :label="qt.label" :value="qt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="限制值" required>
          <el-input-number v-model="quotaForm.limit" :min="1" :step="10000" style="width: 100%" />
        </el-form-item>
        <el-form-item label="周期(天)">
          <el-input-number v-model="quotaForm.period_days" :min="1" :max="365" />
        </el-form-item>
        <el-form-item label="告警阈值">
          <el-slider v-model="quotaForm.alert_at" :min="0.5" :max="1" :step="0.05" :format-tooltip="(v: number) => `${Math.round(v * 100)}%`" />
        </el-form-item>
        <el-form-item label="阻断阈值">
          <el-slider v-model="quotaForm.block_at" :min="0.8" :max="1.5" :step="0.05" :format-tooltip="(v: number) => `${Math.round(v * 100)}%`" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSetQuota = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveQuota">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.quota-management { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h2 { margin: 0; font-size: 20px; }
</style>
