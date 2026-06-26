<template>
  <div class="space-y-6">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">提现管理</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">管理您的佣金提现</p>
      </div>
    </div>

    <!-- 余额卡片 -->
    <div class="bg-gradient-to-r from-purple-600 to-pink-500 rounded-2xl p-6 text-white">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm opacity-80">可提现余额</p>
          <p class="text-4xl font-bold mt-2">¥{{ balance.available.toFixed(2) }}</p>
          <p class="text-sm opacity-80 mt-2">冻结金额：¥{{ balance.frozen.toFixed(2) }}</p>
        </div>
        <el-button @click="showWithdrawDialog = true" class="bg-white/20 border-white/30 hover:bg-white/30 text-white" size="large">
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"/>
          </svg>
          申请提现
        </el-button>
      </div>
    </div>

    <!-- 提现方式 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">提现方式</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div
          v-for="method in withdrawMethods"
          :key="method.id"
          class="border-2 rounded-xl p-4 cursor-pointer transition-all"
          :class="selectedMethod === method.id ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20' : 'border-gray-200 dark:border-gray-700 hover:border-purple-300'"
          @click="selectedMethod = method.id"
        >
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center">
              <span class="text-lg">{{ method.icon }}</span>
            </div>
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ method.name }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ method.description }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 提现记录 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700">
      <div class="p-6 border-b border-gray-100 dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">提现记录</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">申请时间</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">提现金额</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">提现方式</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">收款账号</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">状态</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
            <tr v-for="record in withdrawRecords" :key="record.id" class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ record.time }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">¥{{ record.amount.toFixed(2) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">{{ record.method }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{{ record.account }}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span :class="getStatusClass(record.status)" class="px-2 py-1 text-xs rounded-full">
                  {{ getStatusText(record.status) }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <el-button v-if="record.status === 'pending'" text type="danger" size="small" @click="cancelWithdraw(record.id)">取消</el-button>
                <el-button v-if="record.status === 'completed'" text type="primary" size="small" @click="viewDetail(record)">详情</el-button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 提现申请对话框 -->
    <el-dialog v-model="showWithdrawDialog" title="申请提现" width="500px">
      <el-form :model="withdrawForm" label-width="100px">
        <el-form-item label="提现金额">
          <el-input-number v-model="withdrawForm.amount" :min="10" :max="balance.available" :precision="2" class="w-full" />
          <p class="text-xs text-gray-500 mt-1">最低提现金额：¥10.00</p>
        </el-form-item>
        <el-form-item label="提现方式">
          <el-select v-model="withdrawForm.method" placeholder="选择提现方式" class="w-full">
            <el-option label="支付宝" value="alipay" />
            <el-option label="微信" value="wechat" />
            <el-option label="银行卡" value="bank" />
          </el-select>
        </el-form-item>
        <el-form-item label="收款账号">
          <el-input v-model="withdrawForm.account" placeholder="请输入收款账号" />
        </el-form-item>
        <el-form-item label="真实姓名">
          <el-input v-model="withdrawForm.realName" placeholder="请输入真实姓名" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="withdrawForm.note" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showWithdrawDialog = false">取消</el-button>
        <el-button type="primary" @click="submitWithdraw">提交申请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const showWithdrawDialog = ref(false)
const selectedMethod = ref('alipay')

const balance = ref({
  available: 856.30,
  frozen: 100.00,
})

const withdrawMethods = ref([
  { id: 'alipay', name: '支付宝', description: '1-3个工作日到账', icon: '💳' },
  { id: 'wechat', name: '微信', description: '1-3个工作日到账', icon: '💬' },
  { id: 'bank', name: '银行卡', description: '3-5个工作日到账', icon: '🏦' },
])

const withdrawForm = ref({
  amount: 100,
  method: 'alipay',
  account: '',
  realName: '',
  note: '',
})

const withdrawRecords = ref([
  { id: 1, time: '2024-01-20 14:30', amount: 200.00, method: '支付宝', account: 'test***@gmail.com', status: 'completed' },
  { id: 2, time: '2024-01-19 10:15', amount: 150.00, method: '微信', account: 'user***@qq.com', status: 'processing' },
  { id: 3, time: '2024-01-18 16:45', amount: 300.00, method: '银行卡', account: '**** **** **** 1234', status: 'pending' },
  { id: 4, time: '2024-01-17 12:20', amount: 100.00, method: '支付宝', account: 'demo***@163.com', status: 'completed' },
])

function getStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '待处理',
    processing: '处理中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
  }
  return map[status] || status
}

function getStatusClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
    processing: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
    completed: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    failed: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
    cancelled: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400',
  }
  return map[status] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
}

function cancelWithdraw(id: number) {
  ElMessageBox.confirm('确定要取消这笔提现申请吗？', '确认取消', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(() => {
    const record = withdrawRecords.value.find(r => r.id === id)
    if (record) {
      record.status = 'cancelled'
    }
    ElMessage.success('已取消提现申请')
  }).catch(() => {})
}

function viewDetail(record: any) {
  ElMessage.info(`查看提现详情：${record.id}`)
}

function submitWithdraw() {
  if (!withdrawForm.value.account) {
    ElMessage.warning('请输入收款账号')
    return
  }
  if (!withdrawForm.value.realName) {
    ElMessage.warning('请输入真实姓名')
    return
  }
  if (withdrawForm.value.amount < 10) {
    ElMessage.warning('最低提现金额为 ¥10.00')
    return
  }
  if (withdrawForm.value.amount > balance.value.available) {
    ElMessage.warning('提现金额不能超过可提现余额')
    return
  }
  
  // 添加新记录
  withdrawRecords.value.unshift({
    id: Date.now(),
    time: new Date().toLocaleString('zh-CN'),
    amount: withdrawForm.value.amount,
    method: withdrawForm.value.method === 'alipay' ? '支付宝' : withdrawForm.value.method === 'wechat' ? '微信' : '银行卡',
    account: withdrawForm.value.account.replace(/(.{3}).*(.{3})/, '$1***$2'),
    status: 'pending',
  })
  
  balance.value.available -= withdrawForm.value.amount
  showWithdrawDialog.value = false
  ElMessage.success('提现申请已提交')
}

onMounted(async () => {
  // TODO: 从 API 获取真实数据
})
</script>
