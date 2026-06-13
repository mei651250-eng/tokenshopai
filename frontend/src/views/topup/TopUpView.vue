<template>
  <div class="p-6 max-w-4xl mx-auto">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">在线充值</h1>

    <!-- 余额展示 -->
    <div class="bg-gradient-to-r from-indigo-500 to-purple-600 rounded-2xl p-6 text-white mb-8">
      <p class="text-sm opacity-80">当前余额</p>
      <p class="text-4xl font-bold mt-1">{{ balanceDisplay }} <span class="text-lg">元</span></p>
      <p class="text-sm opacity-70 mt-2">充值后即时到账，支持多种支付方式</p>
    </div>

    <!-- 套餐选择 -->
    <div class="mb-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">选择充值金额</h3>
      <div class="grid grid-cols-4 gap-3">
        <div v-for="pkg in packages" :key="pkg.amount"
          class="relative border-2 rounded-xl p-4 cursor-pointer transition-all text-center"
          :class="selectedAmount === pkg.amount ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20' : 'border-gray-200 dark:border-gray-700 hover:border-indigo-300'"
          @click="selectPackage(pkg)">
          <p class="text-2xl font-bold" :class="selectedAmount === pkg.amount ? 'text-indigo-600 dark:text-indigo-400' : 'text-gray-900 dark:text-white'">
            {{ pkg.amount }}
          </p>
          <p class="text-xs text-gray-500">元</p>
          <el-tag v-if="pkg.bonus" type="danger" size="small" class="absolute -top-2 -right-2">+{{ pkg.bonus }}赠金</el-tag>
        </div>
      </div>
    </div>

    <!-- 自定义金额 -->
    <div class="mb-6">
      <el-form-item label="自定义金额">
        <el-input-number v-model="customAmount" :min="1" :max="100000" :precision="2" :step="10" placeholder="输入自定义金额" style="width: 200px" />
        <el-button type="primary" text @click="selectedAmount = customAmount" class="ml-2">确定</el-button>
      </el-form-item>
    </div>

    <!-- 支付方式 -->
    <div class="mb-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">选择支付方式</h3>
      <div class="grid grid-cols-2 gap-3">
        <div v-for="ch in paymentChannels" :key="ch.channel"
          class="border-2 rounded-xl p-4 cursor-pointer transition-all flex items-center gap-3"
          :class="selectedChannel === ch.channel ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20' : 'border-gray-200 dark:border-gray-700 hover:border-indigo-300'"
          @click="selectedChannel = ch.channel">
          <span class="w-10 h-10 inline-flex items-center justify-center" v-html="ch.icon"></span>
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ ch.name }}</p>
            <p class="text-xs text-gray-500">{{ ch.desc }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 兑换码 -->
    <div class="mb-6 bg-gray-50 dark:bg-gray-800/50 rounded-xl p-4">
      <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">兑换码充值</h3>
      <div class="flex gap-2">
        <el-input v-model="redeemCode" placeholder="输入兑换码，如 TH-xxxxxxxxxxxx" class="flex-1" />
        <el-button type="success" @click="handleRedeem" :loading="redeeming">兑换</el-button>
      </div>
    </div>

    <!-- 支付按钮 -->
    <div class="text-center">
      <el-button type="primary" size="large" @click="handlePay" :loading="paying" :disabled="!selectedAmount || !selectedChannel"
        class="w-64 h-12 text-lg">
        立即充值 {{ selectedAmount ? selectedAmount + ' 元' : '' }}
      </el-button>
    </div>

    <!-- 支付二维码弹窗 -->
    <el-dialog v-model="showPayDialog" title="扫码支付" width="400px" :close-on-click-modal="false">
      <div class="text-center">
        <p class="text-lg font-bold mb-2">支付 {{ selectedAmount }} 元</p>
        <div class="bg-white dark:bg-gray-900 rounded-lg p-4 mb-4 inline-block border border-gray-200 dark:border-gray-700">
          <canvas ref="qrCanvas" class="mx-auto"></canvas>
          <p v-if="!qrLoaded" class="text-sm text-gray-500 mt-2">二维码生成中...</p>
        </div>
        <p class="text-sm text-gray-500">请使用{{ channelName }}扫码支付</p>
        <p class="text-xs text-gray-400 mt-1">支付完成后将自动到账，订单号：{{ orderNo }}</p>
      </div>
      <template #footer>
        <el-button @click="checkPayment" :loading="checking">我已完成支付</el-button>
        <el-button @click="showPayDialog = false">取消</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { Monitor } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { userApi, paymentApi } from '@/api'
import QRCode from 'qrcode'

const balance = ref(0)
const selectedAmount = ref(0)
const customAmount = ref(10)
const selectedChannel = ref('alipay')
const redeemCode = ref('')
const paying = ref(false)
const redeeming = ref(false)
const checking = ref(false)
const showPayDialog = ref(false)
const orderNo = ref('')
const payUrl = ref('')
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrLoaded = ref(false)
const pollTimer = ref<ReturnType<typeof setInterval> | null>(null)

const balanceDisplay = computed(() => (balance.value / 100).toFixed(2))

const channelName = computed(() => {
  const ch = paymentChannels.find(c => c.channel === selectedChannel.value)
  return ch?.name || ''
})

const packages = [
  { amount: 10, bonus: 0 },
  { amount: 50, bonus: 2 },
  { amount: 100, bonus: 8 },
  { amount: 500, bonus: 60 },
]

// 内联 SVG 图标（不依赖外部文件加载）
const channelIcons: Record<string, string> = {
  alipay: '<svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg"><rect width="48" height="48" rx="8" fill="#1677FF"/><path d="M14 16h20v4H14zm0 8h20v4H14zm0 8h14v4H14z" fill="#fff" opacity="0.9"/><path d="M36 28c-2 0-4 2-4 4s2 4 4 4 4-2 4-4-2-4-4-4z" fill="#fff"/></svg>',
  wechat_pay: '<svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg"><rect width="48" height="48" rx="8" fill="#07C160"/><path d="M14 18c0-3.3 4.5-6 10-6s10 2.7 10 6-4.5 6-10 6c-1.8 0-3.5-.3-5-.9l-3.5 1.5 1-2.8C15.5 20.5 14 19.3 14 18z" fill="#fff"/><circle cx="20" cy="18" r="1.5" fill="#07C160"/><circle cx="28" cy="18" r="1.5" fill="#07C160"/></svg>',
  stripe: '<svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg"><rect width="48" height="48" rx="8" fill="#635BFF"/><path d="M18 20h12v2H18zm0 4h8v2H18zm0 4h10v2H18z" fill="#fff"/><circle cx="34" cy="24" r="4" fill="#fff"/></svg>',
  crypto: '<svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg"><rect width="48" height="48" rx="8" fill="#F7931A"/><circle cx="24" cy="24" r="10" fill="#fff"/><path d="M24 14v20M18 20h12M18 28h12" stroke="#F7931A" stroke-width="2" stroke-linecap="round"/></svg>',
}

const paymentChannels = [
  { channel: 'alipay', name: '支付宝', desc: '即时到账', icon: channelIcons.alipay },
  { channel: 'wechat_pay', name: '微信支付', desc: '扫码支付', icon: channelIcons.wechat_pay },
  { channel: 'stripe', name: 'Stripe', desc: '国际信用卡', icon: channelIcons.stripe },
  { channel: 'crypto', name: 'USDT', desc: '加密货币', icon: channelIcons.crypto },
]

function selectPackage(pkg: { amount: number }) {
  selectedAmount.value = pkg.amount
  customAmount.value = pkg.amount
}

async function loadBalance() {
  try {
    const res: any = await userApi.getBalance()
    balance.value = res.balance || 0
  } catch (e) { /* ignore */ }
}

async function handlePay() {
  if (!selectedAmount.value || selectedAmount.value <= 0) {
    ElMessage.warning('请选择充值金额')
    return
  }
  paying.value = true
  try {
    const res: any = await paymentApi.create({
      channel: selectedChannel.value,
      amount: selectedAmount.value,
      currency: 'CNY',
    })
    const order = res.order || res.data || res
    orderNo.value = order.order_no || order.id || ''
    payUrl.value = order.pay_url || order.qr_url || `https://pay.tokenhub.cc/${orderNo.value}`
    showPayDialog.value = true
    qrLoaded.value = false
    await nextTick()
    // 生成二维码
    if (qrCanvas.value && payUrl.value) {
      try {
        await QRCode.toCanvas(qrCanvas.value, payUrl.value, { width: 200, margin: 2 })
        qrLoaded.value = true
      } catch { qrLoaded.value = false }
    }
    // 自动轮询支付状态
    if (pollTimer.value) clearInterval(pollTimer.value)
    pollTimer.value = setInterval(async () => {
      try {
        const checkRes: any = await paymentApi.getOrder(orderNo.value)
        const checkOrder = checkRes.order || checkRes.data || checkRes
        if (checkOrder.status === 'paid' || checkOrder.status === 'completed') {
          if (pollTimer.value) clearInterval(pollTimer.value)
          ElMessage.success('充值成功！')
          showPayDialog.value = false
          await loadBalance()
        }
      } catch { /* ignore */ }
    }, 5000)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '创建支付订单失败')
  } finally {
    paying.value = false
  }
}

async function checkPayment() {
  if (!orderNo.value) return
  checking.value = true
  try {
    const res: any = await paymentApi.getOrder(orderNo.value)
    const order = res.order || res.data || res
    if (order.status === 'paid' || order.status === 'completed') {
      ElMessage.success('充值成功！')
      showPayDialog.value = false
      await loadBalance()
    } else {
      ElMessage.info('尚未收到支付，请稍后再试')
    }
  } catch (e) {
    ElMessage.error('查询订单失败')
  } finally {
    checking.value = false
  }
}

async function handleRedeem() {
  if (!redeemCode.value.trim()) {
    ElMessage.warning('请输入兑换码')
    return
  }
  redeeming.value = true
  try {
    await userApi.redeemCode(redeemCode.value.trim())
    ElMessage.success('兑换成功！')
    redeemCode.value = ''
    await loadBalance()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '兑换失败')
  } finally {
    redeeming.value = false
  }
}

onMounted(loadBalance)

// 关闭弹窗时清除轮询
import { onBeforeUnmount } from 'vue'
onBeforeUnmount(() => {
  if (pollTimer.value) clearInterval(pollTimer.value)
})
</script>
