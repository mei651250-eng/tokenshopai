<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('receiving.title') }}</h1>
      <el-button type="primary" @click="showAddDialog">
        <el-icon class="mr-1"><Plus /></el-icon>
        {{ t('receiving.addAccount') }}
      </el-button>
    </div>

    <!-- Account Cards -->
    <div v-if="accounts.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="account in accounts" :key="account.id"
        class="bg-white dark:bg-gray-800 rounded-xl border-2 p-5 transition-all hover:shadow-md"
        :class="account.is_primary ? 'border-primary-500 dark:border-primary-400' : 'border-gray-200 dark:border-gray-700'"
      >
        <!-- Card Header -->
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg flex items-center justify-center text-white text-lg"
              :class="getTypeColor(account.account_type)"
            >
              {{ getTypeIcon(account.account_type) }}
            </div>
            <div>
              <p class="font-medium text-gray-900 dark:text-white">{{ getTypeName(account.account_type) }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ account.label || account.account_name }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <el-tag v-if="account.is_primary" type="success" size="small" effect="dark">{{ t('receiving.primary') }}</el-tag>
            <el-tag :type="account.verified ? 'success' : 'warning'" size="small">
              {{ account.verified ? t('receiving.verified') : t('receiving.unverified') }}
            </el-tag>
          </div>
        </div>

        <!-- Account Info -->
        <div class="space-y-2 text-sm">
          <div v-if="account.account_no" class="flex justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('receiving.accountNo') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">{{ account.account_no }}</span>
          </div>
          <div v-if="account.wallet_address" class="flex justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('receiving.walletAddr') }}</span>
            <span class="font-mono text-gray-900 dark:text-white text-xs truncate max-w-[180px]">{{ account.wallet_address }}</span>
          </div>
          <div v-if="account.bank_name" class="flex justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('receiving.bankName') }}</span>
            <span class="text-gray-900 dark:text-white">{{ account.bank_name }}</span>
          </div>
          <div v-if="account.qrcode_url" class="flex justify-between">
            <span class="text-gray-500 dark:text-gray-400">{{ t('receiving.qrcode') }}</span>
            <el-tag size="small" type="info">{{ t('receiving.uploaded') }}</el-tag>
          </div>
        </div>

        <!-- Card Actions -->
        <div class="flex items-center justify-between mt-4 pt-3 border-t border-gray-100 dark:border-gray-700">
          <div class="flex items-center gap-1">
            <span class="relative flex h-2 w-2">
              <span v-if="account.enabled" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2"
                :class="account.enabled ? 'bg-green-500' : 'bg-gray-400'"
              ></span>
            </span>
            <span class="text-xs" :class="account.enabled ? 'text-green-600 dark:text-green-400' : 'text-gray-400'">
              {{ account.enabled ? t('receiving.active') : t('receiving.inactive') }}
            </span>
          </div>
          <div class="flex gap-1">
            <el-button v-if="!account.is_primary" size="small" text @click="setPrimary(account.id)">
              {{ t('receiving.setPrimary') }}
            </el-button>
            <el-button size="small" text type="danger" @click="deleteAccount(account.id)">
              {{ t('common.delete') }}
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="text-center py-16">
      <el-icon class="text-6xl text-gray-300 dark:text-gray-600 mb-4"><Wallet /></el-icon>
      <p class="text-lg text-gray-500 dark:text-gray-400 mb-2">{{ t('receiving.emptyTitle') }}</p>
      <p class="text-sm text-gray-400 dark:text-gray-500 mb-4">{{ t('receiving.emptyDesc') }}</p>
      <el-button type="primary" @click="showAddDialog">
        <el-icon class="mr-1"><Plus /></el-icon>
        {{ t('receiving.addFirst') }}
      </el-button>
    </div>

    <!-- Add/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('receiving.editAccount') : t('receiving.addAccount')" width="520px">
      <el-form :model="form" label-width="120px" label-position="top">
        <el-form-item :label="t('receiving.accountType')">
          <el-select v-model="form.account_type" class="w-full" @change="onTypeChange">
            <el-option v-for="at in accountTypes" :key="at.value" :label="at.label" :value="at.value">
              <div class="flex items-center gap-2">
                <span>{{ at.icon }}</span>
                <span>{{ at.label }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item :label="t('receiving.accountName')">
          <el-input v-model="form.account_name" :placeholder="t('receiving.accountNamePlaceholder')" />
        </el-form-item>

        <!-- 银行卡字段 -->
        <template v-if="form.account_type === 'bank_card'">
          <el-form-item :label="t('receiving.bankName')">
            <el-input v-model="form.bank_name" :placeholder="t('receiving.bankNamePlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('receiving.bankBranch')">
            <el-input v-model="form.bank_branch" :placeholder="t('receiving.bankBranchPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('receiving.cardNo')">
            <el-input v-model="form.account_no" :placeholder="t('receiving.cardNoPlaceholder')" />
          </el-form-item>
        </template>

        <!-- 支付宝/微信 -->
        <template v-if="form.account_type === 'alipay' || form.account_type === 'wechat'">
          <el-form-item :label="t('receiving.accountId')">
            <el-input v-model="form.account_no" :placeholder="t('receiving.accountIdPlaceholder')" />
          </el-form-item>
        </template>

        <!-- 加密货币 -->
        <template v-if="form.account_type === 'crypto'">
          <el-form-item :label="t('receiving.chainType')">
            <el-select v-model="form.chain_type" class="w-full">
              <el-option label="Ethereum" value="ethereum" />
              <el-option label="BSC" value="bsc" />
              <el-option label="Polygon" value="polygon" />
              <el-option label="Arbitrum" value="arbitrum" />
              <el-option label="Tron (TRC20)" value="tron" />
              <el-option label="Solana" value="solana" />
            </el-select>
          </el-form-item>
          <el-form-item label="Wallet Address">
            <el-input v-model="form.wallet_address" placeholder="0x..." />
          </el-form-item>
        </template>

        <!-- PayPal/Payoneer/Wise/Stripe -->
        <template v-if="['paypal', 'payoneer', 'wise', 'stripe'].includes(form.account_type)">
          <el-form-item :label="t('receiving.email')">
            <el-input v-model="form.account_no" :placeholder="t('receiving.emailPlaceholder')" />
          </el-form-item>
        </template>

        <el-form-item :label="t('receiving.label')">
          <el-input v-model="form.label" :placeholder="t('receiving.labelPlaceholder')" />
        </el-form-item>

        <el-form-item>
          <el-switch v-model="form.is_primary" :active-text="t('receiving.setAsPrimary')" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Wallet } from '@element-plus/icons-vue'
import { financeApi } from '@/api'

const { t } = useI18n()

interface ReceivingAccount {
  id: string
  account_type: string
  account_name: string
  account_no: string
  bank_name: string
  bank_branch: string
  qrcode_url: string
  wallet_address: string
  chain_type: string
  is_primary: boolean
  verified: boolean
  enabled: boolean
  label: string
}

const accounts = ref<ReceivingAccount[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)

const form = ref({
  id: '',
  account_type: 'bank_card',
  account_name: '',
  account_no: '',
  bank_name: '',
  bank_branch: '',
  qrcode_url: '',
  wallet_address: '',
  chain_type: 'ethereum',
  is_primary: false,
  label: '',
})

const accountTypes = [
  { value: 'alipay', label: '支付宝', icon: '💙' },
  { value: 'wechat', label: '微信', icon: '💚' },
  { value: 'bank_card', label: '银行卡', icon: '🏦' },
  { value: 'paypal', label: 'PayPal', icon: '💜' },
  { value: 'payoneer', label: 'Payoneer', icon: '🟠' },
  { value: 'wise', label: 'Wise', icon: '🔵' },
  { value: 'stripe', label: 'Stripe', icon: '🟣' },
  { value: 'crypto', label: 'Crypto', icon: '₿' },
]

function getTypeName(type: string): string {
  const found = accountTypes.find(a => a.value === type)
  return found ? found.label : type
}

function getTypeIcon(type: string): string {
  const found = accountTypes.find(a => a.value === type)
  return found ? found.icon : '💰'
}

function getTypeColor(type: string): string {
  const map: Record<string, string> = {
    alipay: 'bg-blue-500',
    wechat: 'bg-green-500',
    bank_card: 'bg-amber-500',
    paypal: 'bg-indigo-500',
    payoneer: 'bg-orange-500',
    wise: 'bg-cyan-500',
    stripe: 'bg-purple-500',
    crypto: 'bg-yellow-500',
  }
  return map[type] || 'bg-gray-500'
}

function onTypeChange() {
  form.value.account_no = ''
  form.value.wallet_address = ''
  form.value.bank_name = ''
  form.value.bank_branch = ''
}

function showAddDialog() {
  isEdit.value = false
  form.value = {
    id: '', account_type: 'bank_card', account_name: '', account_no: '',
    bank_name: '', bank_branch: '', qrcode_url: '', wallet_address: '',
    chain_type: 'ethereum', is_primary: false, label: '',
  }
  dialogVisible.value = true
}

async function loadAccounts() {
  try {
    const res: any = await financeApi.listReceiving()
    accounts.value = res.data || res || []
  } catch {
    accounts.value = []
  }
}

async function submitForm() {
  submitting.value = true
  try {
    await financeApi.createReceiving(form.value)
    ElMessage.success(t('common.success'))
    dialogVisible.value = false
    await loadAccounts()
  } catch {
    ElMessage.error(t('common.error'))
  } finally {
    submitting.value = false
  }
}

async function setPrimary(id: string) {
  try {
    await financeApi.setPrimaryReceiving(id)
    await loadAccounts()
  } catch { /* ignore */ }
}

async function deleteAccount(id: string) {
  try {
    await ElMessageBox.confirm(t('receiving.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await financeApi.deleteReceiving(id)
    ElMessage.success(t('common.success'))
    await loadAccounts()
  } catch { /* cancel */ }
}

onMounted(loadAccounts)
</script>
