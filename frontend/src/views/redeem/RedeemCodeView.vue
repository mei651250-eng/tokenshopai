<template>
  <div class="redeem-page">
    <div class="page-header">
      <div>
        <h2>兑换码管理</h2>
        <p class="text-sm text-gray-500 mt-1">批量生成充值兑换码，用于推广活动</p>
      </div>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon> 批量生成
      </el-button>
    </div>

    <!-- 用户兑换入口 -->
    <div class="redeem-card mb-6">
      <h3 class="text-base font-semibold mb-3">🎫 兑换充值码</h3>
      <div class="flex gap-3">
        <el-input v-model="redeemInput" placeholder="输入兑换码，如 TH-xxxxxxxxxxxx" class="flex-1" />
        <el-button type="success" :loading="redeeming" @click="handleRedeem">兑换</el-button>
      </div>
    </div>

    <!-- 统计 -->
    <div class="stat-cards mb-6">
      <div class="stat-card"><div class="stat-icon" style="background:#d1fae5">🎫</div><div><div class="stat-value">{{ codes.length }}</div><div class="stat-label">总计</div></div></div>
      <div class="stat-card"><div class="stat-icon" style="background:#dbeafe">✅</div><div><div class="stat-value">{{ codes.filter(c => c.status === 'active').length }}</div><div class="stat-label">可用</div></div></div>
      <div class="stat-card"><div class="stat-icon" style="background:#fef3c7">✔️</div><div><div class="stat-value">{{ codes.filter(c => c.status === 'used').length }}</div><div class="stat-label">已使用</div></div></div>
    </div>

    <!-- 列表 -->
    <el-table :data="codes" v-loading="loading" stripe border>
      <el-table-column prop="code" label="兑换码" width="220">
        <template #default="{ row }">
          <code class="code-text">{{ row.code }}</code>
        </template>
      </el-table-column>
      <el-table-column prop="amount" label="面额 (元)" width="120" align="center" />
      <el-table-column prop="status" label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : row.status === 'used' ? 'warning' : 'danger'" size="small">
            {{ row.status === 'active' ? '可用' : row.status === 'used' ? '已使用' : '已过期' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="used_by" label="使用者" width="160">
        <template #default="{ row }">{{ row.used_by || '-' }}</template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN') }}</template>
      </el-table-column>
      <el-table-column label="操作" width="80" align="center">
        <template #default="{ row }">
          <el-button type="danger" size="small" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 批量生成对话框 -->
    <el-dialog v-model="showCreateDialog" title="批量生成兑换码" width="480px">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="生成数量" required>
          <el-input-number v-model="createForm.count" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item label="面额 (元)" required>
          <el-input-number v-model="createForm.amount" :min="0.01" :step="1" :precision="2" />
        </el-form-item>
        <el-form-item label="前缀">
          <el-input v-model="createForm.prefix" placeholder="TH" maxlength="6" />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="createForm.expires_at" type="datetime" placeholder="留空永不过期" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">生成</el-button>
      </template>
    </el-dialog>

    <!-- 生成结果对话框 -->
    <el-dialog v-model="showResultDialog" title="兑换码已生成" width="520px">
      <el-alert type="success" :closable="false" class="mb-4" show-icon>
        成功生成 {{ generatedCodes.length }} 个兑换码
      </el-alert>
      <div class="result-codes">
        <div v-for="code in generatedCodes" :key="code" class="result-code-row">
          <code>{{ code }}</code>
        </div>
      </div>
      <template #footer>
        <el-button @click="copyAllCodes">复制全部</el-button>
        <el-button type="primary" @click="showResultDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api'

interface RedeemCodeItem { id: string; code: string; amount: number; status: string; used_by: string; used_at: string; expires_at: string; created_at: string }

const codes = ref<RedeemCodeItem[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showResultDialog = ref(false)
const creating = ref(false)
const redeeming = ref(false)
const redeemInput = ref('')
const generatedCodes = ref<string[]>([])

const createForm = reactive({ count: 10, amount: 1, prefix: 'TH', expires_at: null as string | null })

async function loadCodes() {
  loading.value = true
  try {
    const res: any = await adminApi.getRedeemCodes()
    codes.value = res.data || []
  } catch (e: any) { ElMessage.error(e.message || '加载失败') }
  finally { loading.value = false }
}

async function handleCreate() {
  if (createForm.count < 1 || createForm.amount < 0.01) { ElMessage.warning('请填写数量和面额'); return }
  creating.value = true
  try {
    const res: any = await adminApi.batchCreateRedeemCodes({
      count: createForm.count,
      amount: createForm.amount,
      prefix: createForm.prefix || 'TH',
      expires_at: createForm.expires_at || undefined,
    })
    generatedCodes.value = res.codes || []
    showCreateDialog.value = false
    showResultDialog.value = true
    loadCodes()
  } catch (e: any) { ElMessage.error(e.message || '生成失败') }
  finally { creating.value = false }
}

async function handleDelete(row: RedeemCodeItem) {
  try {
    await ElMessageBox.confirm(`确定删除兑换码 ${row.code}？`, '确认', { type: 'warning' })
    await adminApi.deleteRedeemCode(row.id)
    ElMessage.success('已删除')
    loadCodes()
  } catch {}
}

async function handleRedeem() {
  if (!redeemInput.value.trim()) { ElMessage.warning('请输入兑换码'); return }
  redeeming.value = true
  try {
    const res: any = await adminApi.redeemCode(redeemInput.value.trim())
    ElMessage.success(`兑换成功！充值 ${res.amount} 元`)
    redeemInput.value = ''
  } catch (e: any) { ElMessage.error(e.message || '兑换失败') }
  finally { redeeming.value = false }
}

function copyAllCodes() {
  navigator.clipboard.writeText(generatedCodes.value.join('\n'))
  ElMessage.success('已复制到剪贴板')
}

onMounted(loadCodes)
</script>

<style scoped>
.redeem-card { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { font-size: 20px; font-weight: 700; margin: 0; }
.stat-cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.stat-card { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 16px; display: flex; align-items: center; gap: 14px; }
.stat-icon { font-size: 24px; width: 40px; height: 40px; border-radius: 10px; display: flex; align-items: center; justify-content: center; }
.stat-value { font-size: 22px; font-weight: 800; }
.stat-label { font-size: 12px; color: #94a3b8; }
.code-text { font-family: monospace; font-size: 13px; background: #f1f5f9; padding: 2px 8px; border-radius: 4px; }
.result-codes { max-height: 300px; overflow-y: auto; background: #f8fafc; border-radius: 8px; padding: 12px; }
.result-code-row { padding: 4px 0; font-family: monospace; font-size: 13px; }
.mb-6 { margin-bottom: 24px; }
.mb-4 { margin-bottom: 16px; }
</style>
