<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">订阅计划管理</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">配置试用、月度、季度、年度和按量付费计划</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">
        <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
        创建计划
      </el-button>
    </div>

    <!-- 类型筛选 -->
    <div class="flex gap-2">
      <el-button v-for="t in types" :key="t.value" :type="filterType===t.value?'primary':'default'" @click="filterType=t.value" size="small">{{ t.label }}</el-button>
    </div>

    <!-- 计划卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div v-for="plan in filteredPlans" :key="plan.id" class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border" :class="plan.status==='active'?'border-gray-200 dark:border-gray-700':'border-red-300 dark:border-red-800 opacity-60'">
        <div class="p-6">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ plan.name }}</h3>
            <el-tag :type="plan.status==='active'?'success':'danger'" size="small">{{ plan.status === 'active' ? '启用' : '禁用' }}</el-tag>
          </div>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{{ plan.description }}</p>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between"><span class="text-gray-500">类型</span><span class="font-medium">{{ typeLabels[plan.type] }}</span></div>
            <div class="flex justify-between"><span class="text-gray-500">价格</span><span class="font-medium text-purple-600">¥{{ plan.price.toFixed(2) }}</span></div>
            <div class="flex justify-between"><span class="text-gray-500">有效期</span><span class="font-medium">{{ plan.duration_days > 0 ? plan.duration_days+'天' : '长期有效' }}</span></div>
            <div class="flex justify-between"><span class="text-gray-500">Token限额</span><span class="font-medium">{{ plan.token_limit > 0 ? plan.token_limit.toLocaleString() : '无限' }}</span></div>
            <div class="flex justify-between"><span class="text-gray-500">请求限额</span><span class="font-medium">{{ plan.request_limit > 0 ? plan.request_limit.toLocaleString() : '无限' }}</span></div>
            <div class="flex justify-between"><span class="text-gray-500">日请求</span><span class="font-medium">{{ plan.daily_req_limit > 0 ? plan.daily_req_limit : '无限' }}</span></div>
            <div v-if="plan.type==='payg'" class="flex justify-between"><span class="text-gray-500">输入价格</span><span class="font-medium">¥{{ plan.input_token_price }}/千Token</span></div>
            <div v-if="plan.type==='payg'" class="flex justify-between"><span class="text-gray-500">输出价格</span><span class="font-medium">¥{{ plan.output_token_price }}/千Token</span></div>
          </div>
        </div>
        <div class="border-t border-gray-100 dark:border-gray-700 px-6 py-3 flex justify-end space-x-2">
          <el-button size="small" @click="openEditDialog(plan)">编辑</el-button>
          <el-button size="small" type="danger" @click="deletePlan(plan.id)">删除</el-button>
        </div>
      </div>
    </div>

    <!-- 编辑/创建对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingPlan.id ? '编辑计划' : '创建计划'" width="600px">
      <el-form :model="form" label-width="100px" size="default">
        <el-form-item label="计划名称"><el-input v-model="form.name" placeholder="如：月度订阅" /></el-form-item>
        <el-form-item label="计划类型">
          <el-select v-model="form.type" class="w-full">
            <el-option v-for="t in types.filter(t=>t.value!=='all')" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort_order" :min="0" /></el-form-item>
        <el-form-item label="价格(元)"><el-input-number v-model="form.price" :min="0" :precision="2" class="w-full" /></el-form-item>
        <el-form-item label="有效期(天)"><el-input-number v-model="form.duration_days" :min="0" class="w-full" /></el-form-item>
        <el-form-item label="Token限额"><el-input-number v-model="form.token_limit" :min="0" :step="100000" class="w-full" /></el-form-item>
        <el-form-item label="请求限额"><el-input-number v-model="form.request_limit" :min="0" :step="100" class="w-full" /></el-form-item>
        <el-form-item label="日请求限制"><el-input-number v-model="form.daily_req_limit" :min="0" :step="50" class="w-full" /></el-form-item>
        <el-form-item v-if="form.type==='payg'" label="输入价格"><el-input-number v-model="form.input_token_price" :min="0" :precision="4" :step="0.001" class="w-full" /><span class="text-xs text-gray-400">元/千Token</span></el-form-item>
        <el-form-item v-if="form.type==='payg'" label="输出价格"><el-input-number v-model="form.output_token_price" :min="0" :precision="4" :step="0.001" class="w-full" /><span class="text-xs text-gray-400">元/千Token</span></el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" active-value="active" inactive-value="inactive" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible=false">取消</el-button>
        <el-button type="primary" @click="savePlan" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const apiBase = '' // 使用相对路径

const filterType = ref('all')
const dialogVisible = ref(false)
const saving = ref(false)
const plans = ref<any[]>([])

const editingPlan = ref<any>({})
const form = ref<any>({
  name: '', type: 'monthly', description: '', sort_order: 0,
  price: 0, duration_days: 30, token_limit: 0, request_limit: 0, daily_req_limit: 0,
  input_token_price: 0.01, output_token_price: 0.03, status: 'active'
})

const types = [
  { value: 'all', label: '全部' },
  { value: 'trial', label: '试用' },
  { value: 'monthly', label: '月订阅' },
  { value: 'quarterly', label: '季订阅' },
  { value: 'annual', label: '年订阅' },
  { value: 'payg', label: '按量付费' },
]
const typeLabels: Record<string, string> = { trial: '3天试用', monthly: '月度订阅', quarterly: '季度订阅', annual: '年度订阅', payg: '按量付费' }

const filteredPlans = computed(() => filterType.value === 'all' ? plans.value : plans.value.filter(p => p.type === filterType.value))

async function fetchPlans() {
  try {
    const res = await fetch('/admin/subscription/plans?type=all', { headers: { Authorization: `Bearer ${localStorage.getItem('token')}` } })
    const data = await res.json()
    plans.value = data.data || []
  } catch (e) { console.error(e) }
}

function openCreateDialog() {
  editingPlan.value = {}
  form.value = { name: '', type: 'monthly', description: '', sort_order: plans.value.length + 1, price: 0, duration_days: 30, token_limit: 0, request_limit: 0, daily_req_limit: 0, input_token_price: 0.01, output_token_price: 0.03, status: 'active' }
  dialogVisible.value = true
}

function openEditDialog(plan: any) {
  editingPlan.value = plan
  form.value = { ...plan }
  form.value.status = plan.status
  dialogVisible.value = true
}

async function savePlan() {
  saving.value = true
  try {
    const token = localStorage.getItem('token')
    const isEdit = !!editingPlan.value.id
    const url = isEdit ? `/admin/subscription/plans/${editingPlan.value.id}` : '/admin/subscription/plans'
    const method = isEdit ? 'PUT' : 'POST'
    const body = isEdit ? form.value : { ...form.value }
    const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify(body) })
    if (!res.ok) throw new Error((await res.json()).error)
    ElMessage.success(isEdit ? '更新成功' : '创建成功')
    dialogVisible.value = false
    fetchPlans()
  } catch (e: any) { ElMessage.error(e.message) }
  finally { saving.value = false }
}

async function deletePlan(id: string) {
  try {
    await ElMessageBox.confirm('确定删除此计划？有活跃订阅时无法删除', '确认', { type: 'warning' })
    const res = await fetch(`/admin/subscription/plans/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${localStorage.getItem('token')}` } })
    if (!res.ok) throw new Error((await res.json()).error)
    ElMessage.success('已删除')
    fetchPlans()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e.message || e) }
}

onMounted(fetchPlans)
</script>
