<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { refundApi } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const loading = ref(false)
const refunds = ref<any[]>([])
const pendingRefunds = ref<any[]>([])
const activeTab = ref('my')

async function loadMyRefunds() {
  loading.value = true
  try {
    const res = await refundApi.list()
    refunds.value = res.refunds || res.data || []
  } catch {
    refunds.value = []
  } finally {
    loading.value = false
  }
}

async function loadPendingRefunds() {
  try {
    const res = await refundApi.listPending()
    pendingRefunds.value = res.refunds || res.data || []
  } catch {
    pendingRefunds.value = []
  }
}

const showApply = ref(false)
const applyForm = ref({ payment_order_no: '', amount: 0, reason: '' })
const applying = ref(false)

async function applyRefund() {
  applying.value = true
  try {
    await refundApi.create(applyForm.value)
    ElMessage.success('退款申请已提交')
    showApply.value = false
    loadMyRefunds()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '申请失败')
  } finally {
    applying.value = false
  }
}

async function reviewRefund(orderNo: string, approved: boolean) {
  const action = approved ? '通过' : '拒绝'
  await ElMessageBox.confirm(`确认${action}此退款申请？`, `审核退款`)
  try {
    await refundApi.review({ order_no: orderNo, approved, reason: approved ? '' : '管理员拒绝' })
    ElMessage.success(`退款已${action}`)
    loadPendingRefunds()
  } catch {
    ElMessage.error('操作失败')
  }
}

function getStatusType(status: string) {
  const map: Record<string, string> = {
    pending: 'warning', approved: 'primary', processing: 'info',
    completed: 'success', rejected: 'danger', failed: 'danger',
  }
  return map[status] || 'info'
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = {
    pending: '待审核', approved: '已通过', processing: '处理中',
    completed: '已完成', rejected: '已拒绝', failed: '失败',
  }
  return map[status] || status
}

onMounted(() => {
  loadMyRefunds()
  loadPendingRefunds()
})
</script>

<template>
  <div class="refund-management">
    <div class="page-header">
      <h2>退款管理</h2>
      <el-button type="primary" @click="showApply = true">申请退款</el-button>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="我的退款" name="my">
        <el-table :data="refunds" v-loading="loading" stripe>
          <el-table-column prop="order_no" label="退款单号" width="200" />
          <el-table-column prop="payment_order_no" label="原支付单号" width="200" />
          <el-table-column label="退款金额" width="120">
            <template #default="{ row }">¥{{ (row.amount / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="150" show-overflow-tooltip />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reject_reason" label="拒绝原因" width="150" show-overflow-tooltip />
          <el-table-column label="创建时间" width="170">
            <template #default="{ row }">{{ new Date(row.created_at * 1000).toLocaleString() }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="待审核" name="pending">
        <el-table :data="pendingRefunds" stripe>
          <el-table-column prop="order_no" label="退款单号" width="200" />
          <el-table-column prop="payment_order_no" label="原支付单号" width="200" />
          <el-table-column label="退款金额" width="120">
            <template #default="{ row }">¥{{ (row.amount / 100).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="150" show-overflow-tooltip />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button type="success" size="small" @click="reviewRefund(row.order_no, true)">通过</el-button>
              <el-button type="danger" size="small" @click="reviewRefund(row.order_no, false)">拒绝</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 申请退款弹窗 -->
    <el-dialog v-model="showApply" title="申请退款" width="500px">
      <el-form :model="applyForm" label-width="100px">
        <el-form-item label="支付单号" required>
          <el-input v-model="applyForm.payment_order_no" placeholder="输入原支付订单号" />
        </el-form-item>
        <el-form-item label="退款金额(元)" required>
          <el-input-number v-model="applyForm.amount" :min="0.01" :precision="2" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="退款原因" required>
          <el-input v-model="applyForm.reason" type="textarea" :rows="3" placeholder="请说明退款原因" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showApply = false">取消</el-button>
        <el-button type="primary" :loading="applying" @click="applyRefund">提交申请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.refund-management { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h2 { margin: 0; font-size: 20px; }
</style>
