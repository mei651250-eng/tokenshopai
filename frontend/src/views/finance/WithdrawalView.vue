<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('withdrawal.title') }}</h1>
    </div>

    <!-- Tabs -->
    <el-tabs v-model="activeTab">
      <!-- 提现账户 Tab -->
      <el-tab-pane :label="t('withdrawal.accounts')" name="accounts">
        <div class="flex items-center justify-between mb-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('withdrawal.accountsDesc') }}</p>
          <el-button type="primary" size="small" @click="showAccountDialog">
            <el-icon class="mr-1"><Plus /></el-icon>
            {{ t('withdrawal.addAccount') }}
          </el-button>
        </div>

        <!-- Account Cards -->
        <div v-if="withdrawAccounts.length > 0" class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div v-for="account in withdrawAccounts" :key="account.id"
            class="bg-white dark:bg-gray-800 rounded-xl border p-4"
            :class="account.is_primary ? 'border-primary-300 dark:border-primary-600' : 'border-gray-200 dark:border-gray-700'"
          >
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <span class="text-lg">{{ getAccountIcon(account.account_type) }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ getAccountLabel(account.account_type) }}</span>
                <el-tag v-if="account.is_primary" type="success" size="small">{{ t('withdrawal.primary') }}</el-tag>
              </div>
              <el-button size="small" text type="danger" @click="deleteWithdrawAccount(account.id)">
                {{ t('common.delete') }}
              </el-button>
            </div>
            <div class="text-sm space-y-1">
              <div class="flex justify-between">
                <span class="text-gray-500">{{ t('withdrawal.holderName') }}</span>
                <span class="text-gray-900 dark:text-white">{{ account.account_name }}</span>
              </div>
              <div v-if="account.account_no" class="flex justify-between">
                <span class="text-gray-500">{{ t('withdrawal.accountNo') }}</span>
                <span class="font-mono text-gray-900 dark:text-white">{{ account.account_no }}</span>
              </div>
              <div v-if="account.bank_name" class="flex justify-between">
                <span class="text-gray-500">{{ t('withdrawal.bankName') }}</span>
                <span class="text-gray-900 dark:text-white">{{ account.bank_name }}</span>
              </div>
              <div v-if="account.wallet_address" class="flex justify-between">
                <span class="text-gray-500">{{ t('withdrawal.walletAddr') }}</span>
                <span class="font-mono text-xs text-gray-900 dark:text-white truncate max-w-[200px]">{{ account.wallet_address }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="text-center py-12">
          <el-icon class="text-5xl text-gray-300 dark:text-gray-600 mb-3"><CreditCard /></el-icon>
          <p class="text-gray-500">{{ t('withdrawal.noAccounts') }}</p>
          <el-button type="primary" size="small" class="mt-3" @click="showAccountDialog">
            {{ t('withdrawal.addFirst') }}
          </el-button>
        </div>
      </el-tab-pane>

      <!-- 提现申请 Tab -->
      <el-tab-pane :label="t('withdrawal.orders')" name="orders">
        <div class="flex items-center justify-between mb-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('withdrawal.ordersDesc') }}</p>
          <el-button type="primary" size="small" @click="showWithdrawDialog" :disabled="withdrawAccounts.length === 0">
            <el-icon class="mr-1"><Right /></el-icon>
            {{ t('withdrawal.apply') }}
          </el-button>
        </div>

        <!-- Orders Table -->
        <el-table :data="orders" stripe class="w-full">
          <el-table-column prop="order_no" :label="t('withdrawal.orderNo')" width="200">
            <template #default="{ row }">
              <span class="font-mono text-sm">{{ row.order_no }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="account_type" :label="t('withdrawal.toAccount')" width="140">
            <template #default="{ row }">
              {{ getAccountLabel(row.account_type) }}
            </template>
          </el-table-column>
          <el-table-column prop="amount" :label="t('withdrawal.amount')" width="140">
            <template #default="{ row }">
              <span class="font-semibold">{{ formatMoney(row.amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="fee_amount" :label="t('withdrawal.fee')" width="100">
            <template #default="{ row }">
              <span class="text-gray-500">{{ formatMoney(row.fee_amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="actual_amount" :label="t('withdrawal.actualAmount')" width="140">
            <template #default="{ row }">
              <span class="text-green-600 dark:text-green-400 font-semibold">{{ formatMoney(row.actual_amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" :label="t('common.status')" width="120">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">
                <span class="flex items-center gap-1">
                  <span class="w-1.5 h-1.5 rounded-full"
                    :class="{
                      'bg-yellow-400': row.status === 'pending',
                      'bg-blue-400': row.status === 'approved',
                      'bg-indigo-400': row.status === 'processing',
                      'bg-green-400': row.status === 'completed',
                      'bg-red-400': row.status === 'rejected',
                      'bg-gray-400': row.status === 'cancelled',
                    }"
                  ></span>
                  {{ getStatusLabel(row.status) }}
                </span>
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" :label="t('common.createdAt')" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </el-table-column>
        </el-table>

        <div v-if="orders.length === 0" class="text-center py-12">
          <el-icon class="text-5xl text-gray-300 dark:text-gray-600 mb-3"><Document /></el-icon>
          <p class="text-gray-500">{{ t('withdrawal.noOrders') }}</p>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- Add Withdraw Account Dialog -->
    <el-dialog v-model="accountDialogVisible" :title="t('withdrawal.addAccount')" width="500px">
      <el-form :model="accountForm" label-position="top">
        <el-form-item :label="t('withdrawal.accountType')">
          <el-select v-model="accountForm.account_type" class="w-full" @change="onAccountTypeChange">
            <el-option label="🏦 银行卡" value="bank_card" />
            <el-option label="💙 支付宝" value="alipay" />
            <el-option label="💚 微信" value="wechat" />
            <el-option label="💜 PayPal" value="paypal" />
            <el-option label="₿ Crypto" value="crypto" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('withdrawal.holderName')">
          <el-input v-model="accountForm.account_name" />
        </el-form-item>
        <template v-if="accountForm.account_type === 'bank_card'">
          <el-form-item :label="t('withdrawal.bankName')">
            <el-input v-model="accountForm.bank_name" />
          </el-form-item>
          <el-form-item :label="t('withdrawal.bankBranch')">
            <el-input v-model="accountForm.bank_branch" />
          </el-form-item>
          <el-form-item :label="t('withdrawal.cardNo')">
            <el-input v-model="accountForm.account_no" />
          </el-form-item>
          <el-form-item label="SWIFT Code">
            <el-input v-model="accountForm.swift_code" placeholder="Optional for international transfers" />
          </el-form-item>
        </template>
        <template v-if="accountForm.account_type === 'alipay' || accountForm.account_type === 'wechat'">
          <el-form-item :label="t('withdrawal.accountId')">
            <el-input v-model="accountForm.account_no" />
          </el-form-item>
        </template>
        <template v-if="accountForm.account_type === 'paypal'">
          <el-form-item :label="t('withdrawal.email')">
            <el-input v-model="accountForm.account_no" />
          </el-form-item>
        </template>
        <template v-if="accountForm.account_type === 'crypto'">
          <el-form-item :label="t('withdrawal.chainType')">
            <el-select v-model="accountForm.chain_type" class="w-full">
              <el-option label="Ethereum (ERC20)" value="ethereum" />
              <el-option label="BSC (BEP20)" value="bsc" />
              <el-option label="Tron (TRC20)" value="tron" />
              <el-option label="Solana" value="solana" />
            </el-select>
          </el-form-item>
          <el-form-item label="Wallet Address">
            <el-input v-model="accountForm.wallet_address" />
          </el-form-item>
        </template>
        <el-form-item :label="t('withdrawal.label')">
          <el-input v-model="accountForm.label" />
        </el-form-item>
        <el-form-item>
          <el-switch v-model="accountForm.is_primary" :active-text="t('withdrawal.setAsPrimary')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitAccountForm" :loading="submitting">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Withdraw Dialog -->
    <el-dialog v-model="withdrawDialogVisible" :title="t('withdrawal.apply')" width="480px">
      <el-form :model="withdrawForm" label-position="top">
        <el-form-item :label="t('withdrawal.selectAccount')">
          <el-select v-model="withdrawForm.account_id" class="w-full">
            <el-option v-for="acc in withdrawAccounts" :key="acc.id"
              :label="`${getAccountLabel(acc.account_type)} - ${acc.account_name} (${acc.account_no})`"
              :value="acc.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('withdrawal.amount')">
          <el-input-number v-model="withdrawForm.amount" :min="100" :step="1000" class="w-full" />
          <p class="text-xs text-gray-400 mt-1">{{ t('withdrawal.minWithdraw') }}</p>
        </el-form-item>
        <el-form-item :label="t('withdrawal.currency')">
          <el-select v-model="withdrawForm.currency" class="w-full">
            <el-option label="CNY" value="CNY" />
            <el-option label="USD" value="USD" />
            <el-option label="EUR" value="EUR" />
            <el-option label="USDT" value="USDT" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('withdrawal.remark')">
          <el-input v-model="withdrawForm.remark" type="textarea" :rows="2" />
        </el-form-item>
        <div class="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-3 text-sm">
          <div class="flex justify-between mb-1">
            <span class="text-gray-500">{{ t('withdrawal.amount') }}</span>
            <span>{{ formatMoney(withdrawForm.amount) }}</span>
          </div>
          <div class="flex justify-between mb-1">
            <span class="text-gray-500">{{ t('withdrawal.estFee') }}</span>
            <span class="text-gray-500">~{{ formatMoney(Math.ceil(withdrawForm.amount * 0.001 + 100)) }}</span>
          </div>
          <div class="flex justify-between font-semibold">
            <span>{{ t('withdrawal.estActual') }}</span>
            <span class="text-green-600 dark:text-green-400">{{ formatMoney(withdrawForm.amount - Math.ceil(withdrawForm.amount * 0.001 + 100)) }}</span>
          </div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="withdrawDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitWithdraw" :loading="submitting">{{ t('withdrawal.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, CreditCard, Right, Document } from '@element-plus/icons-vue'
import { financeApi } from '@/api'

const { t } = useI18n()
const activeTab = ref('accounts')
const submitting = ref(false)

interface WithdrawAccount {
  id: string
  account_type: string
  account_name: string
  account_no: string
  bank_name: string
  wallet_address: string
  is_primary: boolean
  label: string
}

interface WithdrawOrder {
  id: string
  order_no: string
  account_type: string
  amount: number
  fee_amount: number
  actual_amount: number
  status: string
  created_at: number
}

const withdrawAccounts = ref<WithdrawAccount[]>([])
const orders = ref<WithdrawOrder[]>([])
const accountDialogVisible = ref(false)
const withdrawDialogVisible = ref(false)

const accountForm = ref({
  account_type: 'bank_card',
  account_name: '',
  account_no: '',
  bank_name: '',
  bank_branch: '',
  swift_code: '',
  wallet_address: '',
  chain_type: 'ethereum',
  is_primary: false,
  label: '',
})

const withdrawForm = ref({
  account_id: '',
  amount: 1000,
  currency: 'CNY',
  remark: '',
})

function getAccountLabel(type: string): string {
  const map: Record<string, string> = {
    alipay: '支付宝', wechat: '微信', bank_card: '银行卡',
    paypal: 'PayPal', payoneer: 'Payoneer', wise: 'Wise',
    stripe: 'Stripe', crypto: 'Crypto',
  }
  return map[type] || type
}

function getAccountIcon(type: string): string {
  const map: Record<string, string> = {
    alipay: '💙', wechat: '💚', bank_card: '🏦',
    paypal: '💜', crypto: '₿',
  }
  return map[type] || '💰'
}

function getStatusType(status: string) {
  const map: Record<string, string> = {
    pending: 'warning', approved: '', processing: 'info',
    completed: 'success', rejected: 'danger', cancelled: 'info', failed: 'danger',
  }
  return map[status] || 'info'
}

function getStatusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: t('withdrawal.statusPending'),
    approved: t('withdrawal.statusApproved'),
    processing: t('withdrawal.statusProcessing'),
    completed: t('withdrawal.statusCompleted'),
    rejected: t('withdrawal.statusRejected'),
    cancelled: t('withdrawal.statusCancelled'),
    failed: t('withdrawal.statusFailed'),
  }
  return map[status] || status
}

function formatMoney(cents: number): string {
  return '¥' + (cents / 100).toFixed(2)
}

function formatDate(ts: number): string {
  return new Date(ts * 1000).toLocaleString()
}

function onAccountTypeChange() {
  accountForm.value.account_no = ''
  accountForm.value.wallet_address = ''
  accountForm.value.bank_name = ''
}

function showAccountDialog() {
  accountForm.value = {
    account_type: 'bank_card', account_name: '', account_no: '',
    bank_name: '', bank_branch: '', swift_code: '',
    wallet_address: '', chain_type: 'ethereum', is_primary: false, label: '',
  }
  accountDialogVisible.value = true
}

function showWithdrawDialog() {
  withdrawForm.value = { account_id: '', amount: 1000, currency: 'CNY', remark: '' }
  withdrawDialogVisible.value = true
}

async function loadAccounts() {
  try {
    const res: any = await financeApi.listWithdrawAccounts()
    withdrawAccounts.value = res.data || res || []
  } catch { withdrawAccounts.value = [] }
}

async function loadOrders() {
  try {
    const res: any = await financeApi.listWithdrawalOrders()
    orders.value = res.data || res || []
  } catch { orders.value = [] }
}

async function submitAccountForm() {
  submitting.value = true
  try {
    await financeApi.createWithdrawAccount(accountForm.value)
    ElMessage.success(t('common.success'))
    accountDialogVisible.value = false
    await loadAccounts()
  } catch { ElMessage.error(t('common.error')) }
  finally { submitting.value = false }
}

async function deleteWithdrawAccount(id: string) {
  try {
    await ElMessageBox.confirm(t('withdrawal.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await financeApi.deleteWithdrawAccount(id)
    ElMessage.success(t('common.success'))
    await loadAccounts()
  } catch { /* cancel */ }
}

async function submitWithdraw() {
  submitting.value = true
  try {
    await financeApi.createWithdrawalOrder(withdrawForm.value)
    ElMessage.success(t('withdrawal.submitSuccess'))
    withdrawDialogVisible.value = false
    await loadOrders()
  } catch { ElMessage.error(t('common.error')) }
  finally { submitting.value = false }
}

onMounted(() => {
  loadAccounts()
  loadOrders()
})
</script>
