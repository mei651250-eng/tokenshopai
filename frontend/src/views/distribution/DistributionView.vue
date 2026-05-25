<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { distributionApi } from '@/api'
import { ElMessage } from 'element-plus'
import { CopyDocument, Share } from '@element-plus/icons-vue'

const { t } = useI18n()

const loading = ref(false)
const distributors = ref<any[]>([])
const commissionRecords = ref<any[]>([])
const activeTab = ref('overview')

const showRegister = ref(false)
const registering = ref(false)
const registerForm = reactive({
  role: 'agent',
  commission_type: 'percent',
  commission_rate: 15,
})

async function loadDistributors() {
  loading.value = true
  try {
    const res = await distributionApi.listDistributors()
    distributors.value = res.distributors || res.data || []
  } catch {
    distributors.value = []
  } finally {
    loading.value = false
  }
}

async function loadCommissions() {
  try {
    const res = await distributionApi.listCommissions()
    commissionRecords.value = res.commissions || res.data || []
  } catch {
    commissionRecords.value = []
  }
}

async function registerDistributor() {
  registering.value = true
  try {
    await distributionApi.register(registerForm)
    ElMessage.success('分销商注册成功')
    showRegister.value = false
    loadDistributors()
  } catch {
    ElMessage.error('注册失败')
  } finally {
    registering.value = false
  }
}

async function settleCommissions() {
  try {
    const period = new Date().toISOString().slice(0, 7)
    const res = await distributionApi.settle({ period })
    ElMessage.success(`结算完成，总金额: ¥${((res.total || 0) / 100).toFixed(2)}`)
    loadCommissions()
  } catch {
    ElMessage.error('结算失败')
  }
}

function copyReferralCode(code: string) {
  navigator.clipboard.writeText(code)
  ElMessage.success('推广码已复制')
}

function getRoleLabel(role: string) {
  const map: Record<string, string> = { agent: '代理商', referrer: '推荐人', reseller: '经销商', affiliate: '联盟推广' }
  return map[role] || role
}

function getCommissionTypeLabel(type: string) {
  const map: Record<string, string> = { percent: '百分比', fixed: '固定金额', tiered: '阶梯式' }
  return map[type] || type
}

function getStatusType(status: string) {
  return status === 'active' ? 'success' : status === 'suspended' ? 'warning' : 'danger'
}

onMounted(() => {
  loadDistributors()
  loadCommissions()
})
</script>

<template>
  <div class="distribution-view">
    <div class="page-header">
      <h2>分销管理</h2>
      <div>
        <el-button type="success" @click="settleCommissions">结算佣金</el-button>
        <el-button type="primary" @click="showRegister = true">注册分销商</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <!-- 分销商列表 -->
      <el-tab-pane label="分销商" name="overview">
        <el-table :data="distributors" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="200" show-overflow-tooltip />
          <el-table-column label="角色" width="100">
            <template #default="{ row }">
              <el-tag size="small">{{ getRoleLabel(row.role) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="推广码" width="180">
            <template #default="{ row }">
              <div style="display: flex; align-items: center; gap: 8px">
                <code>{{ row.referral_code }}</code>
                <el-button :icon="CopyDocument" size="small" circle @click="copyReferralCode(row.referral_code)" />
              </div>
            </template>
          </el-table-column>
          <el-table-column label="层级" width="80">
            <template #default="{ row }">{{ row.level }}级</template>
          </el-table-column>
          <el-table-column label="佣金" width="120">
            <template #default="{ row }">
              {{ row.commission_type === 'percent' ? `${row.commission_rate}%` : `¥${row.commission_rate}` }}
              <el-tag size="small" type="info">{{ getCommissionTypeLabel(row.commission_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="推荐人数" width="100">
            <template #default="{ row }">{{ row.total_referred }}</template>
          </el-table-column>
          <el-table-column label="带来营收" width="120">
            <template #default="{ row }">¥{{ (row.total_revenue / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="累计佣金" width="120">
            <template #default="{ row }">¥{{ (row.total_commission / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="待结算" width="120">
            <template #default="{ row }">
              <span style="color: #E6A23C; font-weight: bold">¥{{ (row.pending_commission / 100).toFixed(2) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 佣金记录 -->
      <el-tab-pane label="佣金记录" name="commissions">
        <el-table :data="commissionRecords" stripe>
          <el-table-column prop="order_no" label="关联订单" width="200" show-overflow-tooltip />
          <el-table-column label="订单金额" width="120">
            <template #default="{ row }">¥{{ (row.order_amount / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="佣金比例" width="100">
            <template #default="{ row }">{{ (row.commission_rate * 100).toFixed(1) }}%</template>
          </el-table-column>
          <el-table-column label="佣金金额" width="120">
            <template #default="{ row }">
              <span style="color: #67C23A; font-weight: bold">¥{{ (row.commission_amt / 100).toFixed(2) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'settled' ? 'success' : 'warning'" size="small">
                {{ row.status === 'settled' ? '已结算' : '待结算' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="period" label="结算周期" width="100" />
          <el-table-column label="时间" width="170">
            <template #default="{ row }">{{ new Date(row.created_at * 1000).toLocaleString() }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 注册分销商弹窗 -->
    <el-dialog v-model="showRegister" title="注册分销商" width="500px">
      <el-form :model="registerForm" label-width="100px">
        <el-form-item label="角色" required>
          <el-select v-model="registerForm.role" style="width: 100%">
            <el-option label="代理商" value="agent" />
            <el-option label="推荐人" value="referrer" />
            <el-option label="经销商" value="reseller" />
            <el-option label="联盟推广" value="affiliate" />
          </el-select>
        </el-form-item>
        <el-form-item label="佣金类型">
          <el-select v-model="registerForm.commission_type" style="width: 100%">
            <el-option label="百分比" value="percent" />
            <el-option label="固定金额" value="fixed" />
          </el-select>
        </el-form-item>
        <el-form-item label="佣金率">
          <el-input-number v-model="registerForm.commission_rate" :min="1" :max="50" style="width: 100%" />
          <span style="margin-left: 8px; color: #999">{{ registerForm.commission_type === 'percent' ? '%' : '元' }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRegister = false">取消</el-button>
        <el-button type="primary" :loading="registering" @click="registerDistributor">确认注册</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.distribution-view { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h2 { margin: 0; font-size: 20px; }
</style>
