<template>
  <div class="payment-view">
    <div class="page-header">
      <h2>{{ $t('payment.title') }}</h2>
    </div>

    <!-- 支付渠道选择 -->
    <el-card class="mb-6">
      <template #header>
        <span>{{ $t('payment.selectChannel') }}</span>
      </template>

      <div v-if="loading" class="loading-state">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <p>{{ $t('common.loading') || '加载中...' }}</p>
      </div>
      <div v-else-if="channels.length === 0" class="empty-state">
        <el-empty :description="$t('payment.noChannels') || '暂无可用支付渠道'" />
      </div>
      <div v-else class="channel-grid">
        <div
          v-for="ch in channels"
          :key="ch.channel"
          class="channel-card"
          :class="{ selected: selectedChannel === ch.channel }"
          @click="selectChannel(ch)"
        >
          <div class="channel-icon">
            <img v-if="getChannelIcon(ch)" :src="getChannelIcon(ch)" :alt="ch.name" class="channel-logo" />
            <span v-else class="text-2xl">{{ getChannelEmoji(ch.channel) }}</span>
          </div>
          <div class="channel-info">
            <div class="channel-name">{{ ch.name }}</div>
            <div v-if="ch.name_cn" class="channel-name-cn">{{ ch.name_cn }}</div>
            <div class="channel-currencies">{{ ch.supported_currencies?.join(', ') }}</div>
            <div class="channel-fee">
              {{ $t('payment.feeRate') }}: {{ (ch.fee_rate * 100).toFixed(1) }}%
              <span v-if="ch.fee_fixed">+ {{ Object.values(ch.fee_fixed)[0] }}分</span>
            </div>
            <el-tag v-if="ch.is_global" type="success" size="small">{{ $t('payment.global') }}</el-tag>
            <el-tag v-else type="info" size="small">{{ $t('payment.regional') }}</el-tag>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 支付表单 -->
    <el-card v-if="selectedChannel" class="mb-6">
      <template #header>
        <span>{{ $t('payment.fillInfo') }} - {{ selectedChannelInfo?.name }}</span>
      </template>

      <el-form :model="paymentForm" label-width="140px" style="max-width: 500px;">
        <el-form-item :label="$t('payment.currency')">
          <el-select v-model="paymentForm.currency">
            <el-option
              v-for="c in selectedChannelInfo?.supported_currencies || []"
              :key="c"
              :label="c"
              :value="c"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('payment.amount')">
          <el-input-number
            v-model="paymentForm.amount"
            :min="minAmount"
            :max="maxAmount"
            :step="100"
            controls-position="right"
          />
        </el-form-item>
        <el-form-item :label="$t('payment.targetCurrency')">
          <el-select v-model="paymentForm.to_currency">
            <el-option label="CNY" value="CNY" />
            <el-option label="USD" value="USD" />
            <el-option label="EUR" value="EUR" />
            <el-option label="JPY" value="JPY" />
          </el-select>
        </el-form-item>

        <!-- 费用明细 -->
        <el-form-item :label="$t('payment.feeDetail')">
          <div class="fee-detail">
            <div>{{ $t('payment.paymentAmount') }}: {{ (paymentForm.amount / 100).toFixed(2) }} {{ paymentForm.currency }}</div>
            <div>{{ $t('payment.fee') }}: {{ (feeAmount / 100).toFixed(2) }} {{ paymentForm.currency }}</div>
            <div class="font-bold">
              {{ $t('payment.actualCredit') }}: {{ (actualCredit / 100).toFixed(2) }} {{ paymentForm.to_currency }}
            </div>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="createPayment" :loading="paying" size="large">
            {{ $t('payment.confirmPay') }}
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 支付二维码/跳转 -->
      <div v-if="currentOrder" class="payment-redirect mt-4">
        <el-alert v-if="currentOrder.redirect_url" type="info" :closable="false">
          <template #title>
            <a :href="currentOrder.redirect_url" target="_blank" class="text-blue-600 underline">
              {{ $t('payment.goToPay') }}
            </a>
          </template>
        </el-alert>
        <div v-if="currentOrder.qr_code" class="text-center mt-4">
          <p>{{ $t('payment.scanQR') }}</p>
          <div class="qr-placeholder">{{ $t('payment.qrCode') }}</div>
          <p class="text-sm text-gray-500 mt-2">{{ currentOrder.qr_code }}</p>
        </div>
      </div>
    </el-card>

    <!-- 支付订单列表 -->
    <el-card>
      <template #header>
        <span>{{ $t('payment.orderHistory') }}</span>
      </template>
      <el-table :data="orders" style="width: 100%">
        <el-table-column prop="order_no" :label="$t('payment.orderNo')" width="220" />
        <el-table-column prop="channel" :label="$t('payment.channel')" width="140">
          <template #default="{ row }">
            <el-tag>{{ row.channel }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="amount" :label="$t('payment.amount')" width="140">
          <template #default="{ row }">
            {{ (row.amount / 100).toFixed(2) }} {{ row.currency }}
          </template>
        </el-table-column>
        <el-table-column prop="fee_amount" :label="$t('payment.fee')" width="120">
          <template #default="{ row }">
            {{ (row.fee_amount / 100).toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="actual_amount" :label="$t('payment.credit')" width="140">
          <template #default="{ row }">
            {{ (row.actual_amount / 100).toFixed(2) }} {{ row.to_currency }}
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="$t('common.status')" width="120">
          <template #default="{ row }">
            <el-tag :type="getPaymentStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="$t('common.createdAt')" width="180">
          <template #default="{ row }">
            {{ new Date(row.created_at * 1000).toLocaleString() }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { paymentApi } from '@/api'

const route = useRoute()

const channels = ref<any[]>([])
const selectedChannel = ref('')
const selectedChannelInfo = ref<any>(null)
const orders = ref<any[]>([])
const paying = ref(false)
const currentOrder = ref<any>(null)
const loading = ref(false)

const paymentForm = reactive({
  currency: 'CNY',
  amount: 10000,
  to_currency: 'CNY',
})

const minAmount = computed(() => {
  if (!selectedChannelInfo.value) return 100
  const map = selectedChannelInfo.value.min_amount || {}
  return map[paymentForm.currency] || 100
})

const maxAmount = computed(() => {
  if (!selectedChannelInfo.value) return 999999999
  const map = selectedChannelInfo.value.max_amount || {}
  return map[paymentForm.currency] || 999999999
})

const feeAmount = computed(() => {
  if (!selectedChannelInfo.value) return 0
  const rate = selectedChannelInfo.value.fee_rate || 0
  const fixed = selectedChannelInfo.value.fee_fixed?.[paymentForm.currency] || 0
  return Math.floor(paymentForm.amount * rate) + fixed
})

const actualCredit = computed(() => {
  return paymentForm.amount - feeAmount.value
})

function getChannelEmoji(channel: string) {
  const map: Record<string, string> = {
    alipay: '💳', wechat_pay: '💚', paypal: '🅿️',
    worldfirst: '🌍', payoneer: '💵', wise: '🟢',
    stripe: '💜', crypto: '₿', airwallex: '🌐',
  }
  return map[channel] || '💰'
}

// 获取渠道 Logo 图标路径
function getChannelIcon(ch: any): string {
  // 优先使用后端返回的 icon 字段（可配置）
  if (ch.icon) {
    return ch.icon
  }
  // 备选：使用本地默认图标
  const defaultIcons: Record<string, string> = {
    alipay: '/icons/alipay.svg',
    wechat_pay: '/icons/wechat_pay.svg',
    paypal: '/icons/paypal.svg',
    worldfirst: '/icons/worldfirst.svg',
    payoneer: '/icons/payoneer.svg',
    wise: '/icons/wise.svg',
    stripe: '/icons/stripe.svg',
    airwallex: '/icons/airwallex.svg',
    crypto: '/icons/crypto.svg',
  }
  return defaultIcons[ch.channel] || ''
}

function selectChannel(ch: any) {
  selectedChannel.value = ch.channel
  selectedChannelInfo.value = ch
  paymentForm.currency = ch.supported_currencies?.[0] || 'CNY'
}

function getPaymentStatusType(status: string) {
  const map: Record<string, string> = {
    completed: 'success', pending: 'warning', paid: 'info',
    created: 'info', failed: 'danger', cancelled: 'info',
  }
  return map[status] || 'info'
}

async function fetchChannels() {
  loading.value = true
  try {
    const res = await paymentApi.channels()
    channels.value = res.channels || []
  } catch {
    // 后端不可用时使用本地默认渠道数据（带 Logo）
    channels.value = getDefaultChannels()
  } finally {
    loading.value = false
  }
}

// 本地默认渠道数据，后端不可用时的 fallback
function getDefaultChannels() {
  return [
    { channel: 'alipay', name: 'Alipay', name_cn: '支付宝', icon: '/icons/alipay.svg', supported_currencies: ['CNY'], fee_rate: 0.006, fee_fixed: { CNY: 0 }, is_global: false, min_amount: { CNY: 100 }, max_amount: { CNY: 5000000 } },
    { channel: 'wechat_pay', name: 'WeChat Pay', name_cn: '微信支付', icon: '/icons/wechat_pay.svg', supported_currencies: ['CNY'], fee_rate: 0.006, fee_fixed: { CNY: 0 }, is_global: false, min_amount: { CNY: 100 }, max_amount: { CNY: 5000000 } },
    { channel: 'paypal', name: 'PayPal', name_cn: 'PayPal', icon: '/icons/paypal.svg', supported_currencies: ['USD', 'EUR'], fee_rate: 0.039, fee_fixed: { USD: 30 }, is_global: true, min_amount: { USD: 100 }, max_amount: { USD: 10000000 } },
    { channel: 'worldfirst', name: 'WorldFirst', name_cn: '万里汇', icon: '/icons/worldfirst.svg', supported_currencies: ['USD', 'EUR', 'GBP', 'CNY'], fee_rate: 0.01, fee_fixed: {}, is_global: true, min_amount: { USD: 100 }, max_amount: { USD: 10000000 } },
    { channel: 'payoneer', name: 'Payoneer', name_cn: 'Payoneer', icon: '/icons/payoneer.svg', supported_currencies: ['USD', 'EUR', 'GBP'], fee_rate: 0.015, fee_fixed: {}, is_global: true, min_amount: { USD: 100 }, max_amount: { USD: 5000000 } },
    { channel: 'wise', name: 'Wise', name_cn: 'Wise', icon: '/icons/wise.svg', supported_currencies: ['USD', 'EUR', 'GBP', 'CNY'], fee_rate: 0.007, fee_fixed: {}, is_global: true, min_amount: { USD: 100 }, max_amount: { USD: 1000000 } },
    { channel: 'stripe', name: 'Stripe', name_cn: 'Stripe', icon: '/icons/stripe.svg', supported_currencies: ['USD', 'EUR', 'GBP', 'JPY'], fee_rate: 0.029, fee_fixed: { USD: 30 }, is_global: true, min_amount: { USD: 50 }, max_amount: { USD: 99999999 } },
    { channel: 'airwallex', name: 'Airwallex', name_cn: '空中云汇', icon: '/icons/airwallex.svg', supported_currencies: ['USD', 'EUR', 'GBP', 'CNY', 'AUD'], fee_rate: 0.012, fee_fixed: {}, is_global: true, min_amount: { USD: 100 }, max_amount: { USD: 10000000 } },
  ]
}

async function fetchOrders() {
  try {
    const res = await paymentApi.orders()
    orders.value = res.orders || []
  } catch {
    orders.value = []
  }
}

async function createPayment() {
  paying.value = true
  try {
    const res = await paymentApi.create({
      channel: selectedChannel.value,
      amount: paymentForm.amount,
      currency: paymentForm.currency,
      to_currency: paymentForm.to_currency,
    })
    currentOrder.value = res
    ElMessage.success('Payment order created')
    fetchOrders()
  } catch {
    ElMessage.error('Failed to create payment')
  } finally {
    paying.value = false
  }
}

onMounted(async () => {
  await fetchChannels()
  fetchOrders()

  // 从 query 参数自动选中渠道和金额（从钱包/计费页跳转过来时）
  const qChannel = route.query.channel as string
  const qAmount = route.query.amount as string
  if (qChannel) {
    const ch = channels.value.find((c: any) => c.channel === qChannel)
    if (ch) selectChannel(ch)
  }
  if (qAmount) {
    const parsed = Number(qAmount)
    if (parsed > 0) paymentForm.amount = parsed
  }
})
</script>

<style scoped>
.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.channel-card {
  display: flex;
  gap: 12px;
  padding: 16px;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.loading-state, .empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 0;
  color: #9ca3af;
}
.loading-state p, .empty-state p {
  margin-top: 12px;
  font-size: 14px;
}
.channel-card:hover { border-color: #6366f1; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.channel-card.selected { border-color: #6366f1; background: #eef2ff; }
.channel-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f3f4f6;
  border-radius: 10px;
  flex-shrink: 0;
  overflow: hidden;
}
/* 深色模式 */
:global(html.dark) .channel-icon { background: #374151; }
:global(html.dark) .channel-card { border-color: #4b5563; background: #1f2937; }
:global(html.dark) .channel-card:hover { border-color: #818cf8; box-shadow: 0 2px 8px rgba(0,0,0,0.3); }
:global(html.dark) .channel-card.selected { border-color: #818cf8; background: #312e81; }
.channel-logo {
  width: 36px;
  height: 36px;
  object-fit: contain;
  border-radius: 6px;
}
.channel-name { font-weight: 600; font-size: 14px; }
.channel-name-cn { font-size: 12px; color: #6b7280; }
.channel-currencies { font-size: 11px; color: #9ca3af; margin: 2px 0; }
.channel-fee { font-size: 11px; color: #6b7280; margin: 2px 0; }
.fee-detail { line-height: 1.8; font-size: 14px; }
.qr-placeholder {
  width: 200px; height: 200px; margin: 12px auto;
  background: #f3f4f6; border: 1px dashed #d1d5db;
  display: flex; align-items: center; justify-content: center;
  border-radius: 8px;
}
</style>
