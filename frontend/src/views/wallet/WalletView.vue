<template>
  <div class="wallet-view">
    <div class="page-header">
      <h2>{{ $t('wallet.title') }}</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showConnectDialog = true">
          <el-icon><Link /></el-icon>
          {{ $t('wallet.connectWallet') }}
        </el-button>
        <el-button @click="showBindDialog = true">
          <el-icon><Plus /></el-icon>
          {{ $t('wallet.bind') }}
        </el-button>
      </div>
    </div>

    <!-- 快捷操作区 -->
    <el-card class="mb-6 quick-actions-card">
      <template #header>
        <span>{{ $t('wallet.quickActions') }}</span>
      </template>
      <div class="quick-actions">
        <div class="action-item" @click="goToPayment('crypto')">
          <div class="action-icon crypto-icon">
            <img src="/icons/crypto.svg" alt="Crypto" />
          </div>
          <div class="action-text">
            <div class="action-title">{{ $t('wallet.cryptoRecharge') }}</div>
            <div class="action-desc">{{ $t('wallet.cryptoRechargeDesc') }}</div>
          </div>
          <el-icon class="action-arrow"><ArrowRight /></el-icon>
        </div>
        <div class="action-item" @click="goToPayment('alipay')">
          <div class="action-icon alipay-icon">
            <img src="/icons/alipay.svg" alt="Alipay" />
          </div>
          <div class="action-text">
            <div class="action-title">{{ $t('wallet.fiatRecharge') }}</div>
            <div class="action-desc">{{ $t('wallet.fiatRechargeDesc') }}</div>
          </div>
          <el-icon class="action-arrow"><ArrowRight /></el-icon>
        </div>
        <div class="action-item" @click="showConnectDialog = true">
          <div class="action-icon connect-icon">
            <el-icon :size="28"><Link /></el-icon>
          </div>
          <div class="action-text">
            <div class="action-title">{{ $t('wallet.connectAndDeposit') }}</div>
            <div class="action-desc">{{ $t('wallet.connectAndDepositDesc') }}</div>
          </div>
          <el-icon class="action-arrow"><ArrowRight /></el-icon>
        </div>
        <div class="action-item" @click="goToPayment()">
          <div class="action-icon payment-icon">
            <el-icon :size="28"><CreditCard /></el-icon>
          </div>
          <div class="action-text">
            <div class="action-title">{{ $t('wallet.allPaymentMethods') }}</div>
            <div class="action-desc">{{ $t('wallet.allPaymentMethodsDesc') }}</div>
          </div>
          <el-icon class="action-arrow"><ArrowRight /></el-icon>
        </div>
      </div>
    </el-card>

    <!-- 已连接的钱包 -->
    <el-card class="mb-6">
      <template #header>
        <div class="card-header-flex">
          <span>{{ $t('wallet.connectedWallets') }}</span>
          <el-tag v-if="connectedWallet" type="success" size="small">
            <el-icon><CircleCheck /></el-icon>
            {{ $t('wallet.walletConnected') }}
          </el-tag>
        </div>
      </template>

      <!-- 实时连接的钱包 -->
      <div v-if="connectedWallet" class="connected-wallet-card">
        <div class="wallet-avatar" :style="{ background: getWalletGradient(connectedWallet.type) }">
          <span class="wallet-emoji">{{ getWalletEmoji(connectedWallet.type) }}</span>
        </div>
        <div class="wallet-detail">
          <div class="wallet-name-row">
            <span class="wallet-name">{{ getWalletLabel(connectedWallet.type) }}</span>
            <el-tag size="small" type="success">{{ $t('wallet.live') }}</el-tag>
            <span class="chain-badge">{{ connectedWallet.chain }}</span>
          </div>
          <div class="wallet-address-row">
            <span class="font-mono">{{ formatAddress(connectedWallet.address) }}</span>
            <el-button link size="small" @click="copyAddress(connectedWallet.address)">
              <el-icon><CopyDocument /></el-icon>
            </el-button>
          </div>
          <div v-if="connectedWallet.balance" class="wallet-balance-row">
            <span>{{ $t('wallet.balance') }}: <strong>{{ connectedWallet.balance }} ETH</strong></span>
            <span v-if="connectedWallet.balanceUsd" class="text-gray-400 ml-2">≈ ${{ connectedWallet.balanceUsd }}</span>
          </div>
        </div>
        <div class="wallet-actions-col">
          <el-button type="primary" size="small" @click="depositFromConnected">
            <el-icon><TopRight /></el-icon>
            {{ $t('wallet.depositNow') }}
          </el-button>
          <el-button size="small" @click="switchChain">
            <el-icon><Switch /></el-icon>
            {{ $t('wallet.switchChain') }}
          </el-button>
          <el-button size="small" type="danger" plain @click="disconnectWallet">
            {{ $t('wallet.disconnect') }}
          </el-button>
        </div>
      </div>

      <!-- 已绑定的钱包列表 -->
      <el-table v-if="wallets.length > 0" :data="wallets" style="width: 100%" class="mt-4">
        <el-table-column prop="wallet_type" :label="$t('wallet.type')" width="160">
          <template #default="{ row }">
            <div class="wallet-type-cell">
              <span class="wallet-emoji-sm">{{ getWalletEmoji(row.wallet_type) }}</span>
              <el-tag :type="getWalletTagType(row.wallet_type)" size="small">
                {{ getWalletLabel(row.wallet_type) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="address" :label="$t('wallet.address')" min-width="280">
          <template #default="{ row }">
            <span class="font-mono text-sm">{{ formatAddress(row.address) }}</span>
            <el-button link size="small" @click="copyAddress(row.address)">
              <el-icon><CopyDocument /></el-icon>
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="chain_type" :label="$t('wallet.chain')" width="120">
          <template #default="{ row }">
            <span class="uppercase">{{ row.chain_type }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="label" :label="$t('wallet.label')" width="140" />
        <el-table-column prop="is_primary" :label="$t('wallet.primary')" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_primary" type="success" size="small">{{ $t('common.yes') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.actions')" width="200">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="depositFromBound(row)">
              <el-icon><TopRight /></el-icon>
              {{ $t('wallet.recharge') }}
            </el-button>
            <el-popconfirm :title="$t('wallet.unbindConfirm')" @confirm="unbindWallet(row.address)">
              <template #reference>
                <el-button type="danger" size="small" link>{{ $t('wallet.unbind') }}</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!connectedWallet && wallets.length === 0" :description="$t('wallet.noWallets')">
        <el-button type="primary" @click="showConnectDialog = true">
          <el-icon><Link /></el-icon>
          {{ $t('wallet.connectFirst') }}
        </el-button>
      </el-empty>
    </el-card>

    <!-- 加密货币充值 -->
    <el-card class="mb-6">
      <template #header>
        <div class="card-header-flex">
          <span>{{ $t('wallet.cryptoDeposit') }}</span>
          <el-button v-if="connectedWallet" type="primary" size="small" plain @click="depositFromConnected">
            {{ $t('wallet.useConnectedWallet') }}
          </el-button>
        </div>
      </template>

      <el-form :model="depositForm" label-width="120px">
        <el-form-item :label="$t('wallet.fromWallet')">
          <div v-if="connectedWallet" class="from-wallet-selector selected">
            <span class="wallet-emoji-sm">{{ getWalletEmoji(connectedWallet.type) }}</span>
            <span>{{ getWalletLabel(connectedWallet.type) }}</span>
            <span class="font-mono text-xs text-gray-400 ml-2">{{ formatAddress(connectedWallet.address) }}</span>
          </div>
          <div v-else class="from-wallet-selector empty" @click="showConnectDialog = true">
            <el-icon><Plus /></el-icon>
            <span>{{ $t('wallet.connectToDeposit') }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('wallet.currency')">
          <el-select v-model="depositForm.currency" @change="onCurrencyChange">
            <el-option label="USDT" value="USDT" />
            <el-option label="USDC" value="USDC" />
            <el-option label="ETH" value="ETH" />
            <el-option label="BTC" value="BTC" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('wallet.chain')">
          <el-select v-model="depositForm.chainType">
            <el-option
              v-for="chain in availableChains"
              :key="chain"
              :label="chain.toUpperCase()"
              :value="chain"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('wallet.amount')">
          <el-input v-model="depositForm.amount" :placeholder="$t('wallet.amountPlaceholder')">
            <template #append>{{ depositForm.currency }}</template>
          </el-input>
        </el-form-item>
        <el-form-item :label="$t('wallet.exchangeRate')">
          <span v-if="exchangeRate">1 {{ depositForm.currency }} ≈ {{ exchangeRate }} {{ depositForm.fiatCurrency }}</span>
          <el-button link @click="fetchExchangeRate">{{ $t('wallet.refreshRate') }}</el-button>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="createDeposit" :loading="depositing">
            {{ $t('wallet.createDeposit') }}
          </el-button>
          <el-button @click="goToPayment('crypto')">
            {{ $t('wallet.goToPaymentPage') }}
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 充值订单列表 -->
      <el-divider />
      <h4>{{ $t('wallet.depositOrders') }}</h4>
      <el-table :data="depositOrders" style="width: 100%">
        <el-table-column prop="order_no" :label="$t('wallet.orderNo')" width="200" />
        <el-table-column prop="currency" :label="$t('wallet.currency')" width="100" />
        <el-table-column prop="amount" :label="$t('wallet.amount')" width="120" />
        <el-table-column prop="status" :label="$t('common.status')" width="120">
          <template #default="{ row }">
            <el-tag :type="getDepositStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="to_address" :label="$t('wallet.depositAddress')" min-width="200">
          <template #default="{ row }">
            <span class="font-mono text-xs">{{ row.to_address }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="tx_hash" :label="$t('wallet.txHash')" min-width="200">
          <template #default="{ row }">
            <span v-if="row.tx_hash" class="font-mono text-xs">{{ row.tx_hash }}</span>
            <span v-else class="text-gray-400">--</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 连接钱包对话框 -->
    <el-dialog v-model="showConnectDialog" :title="$t('wallet.connectWallet')" width="600px" top="5vh">
      <div class="connect-wallet-content">
        <el-alert type="info" :closable="false" class="mb-4">
          <template #title>
            {{ $t('wallet.connectTip') }}
          </template>
        </el-alert>

        <div class="wallet-connect-grid">
          <div
            v-for="w in connectableWallets"
            :key="w.value"
            class="wallet-connect-card"
            :class="{ 'is-connecting': connectingType === w.value, 'is-installed': w.installed }"
            @click="connectWeb3Wallet(w)"
          >
            <div class="wc-icon" :style="{ background: w.gradient }">
              <span class="wc-emoji">{{ w.icon }}</span>
            </div>
            <div class="wc-info">
              <div class="wc-name">{{ w.label }}</div>
              <div class="wc-status">
                <span v-if="w.installed" class="installed-badge">
                  <el-icon><CircleCheck /></el-icon> {{ $t('wallet.installed') }}
                </span>
                <span v-else class="not-installed-badge">
                  {{ $t('wallet.notInstalled') }}
                </span>
              </div>
            </div>
            <el-icon v-if="connectingType === w.value" class="is-loading" :size="20"><Loading /></el-icon>
            <el-icon v-else-if="w.installed" class="wc-arrow"><ArrowRight /></el-icon>
          </div>
        </div>

        <!-- 二维码连接区域（WalletConnect 等） -->
        <div class="qr-connect-section" v-if="showQrCode">
          <el-divider>{{ $t('wallet.orScanQR') }}</el-divider>
          <div class="qr-area">
            <div v-if="wcConnecting" class="qr-loading">
              <el-icon class="is-loading" :size="32"><Loading /></el-icon>
              <p>{{ $t('common.loading') || '连接中...' }}</p>
            </div>
            <div v-else-if="wcURI" class="qr-content">
              <canvas ref="wcQrCanvas" class="mx-auto"></canvas>
              <p class="qr-tip">{{ $t('wallet.scanWithMobile') }}</p>
            </div>
            <div v-else @click="startWalletConnect" class="qr-placeholder">
              <el-icon :size="48" class="text-gray-300"><Cellphone /></el-icon>
              <p>{{ $t('wallet.scanWithMobile') }}</p>
            </div>
          </div>
          <p class="qr-hint">{{ $t('wallet.qrHint') }}</p>
        </div>
      </div>
    </el-dialog>

    <!-- 绑定钱包对话框（手动输入） -->
    <el-dialog v-model="showBindDialog" :title="$t('wallet.bindNewWallet')" width="500px">
      <el-steps :active="bindStep" finish-status="success" simple>
        <el-step :title="$t('wallet.selectWallet')" />
        <el-step :title="$t('wallet.verify')" />
        <el-step :title="$t('wallet.complete')" />
      </el-steps>

      <div v-if="bindStep === 0" class="mt-6">
        <div class="wallet-grid">
          <div
            v-for="w in walletTypes"
            :key="w.value"
            class="wallet-option"
            :class="{ selected: bindForm.wallet_type === w.value }"
            @click="selectWalletType(w.value)"
          >
            <div class="wallet-icon">{{ w.icon }}</div>
            <div class="wallet-name">{{ w.label }}</div>
          </div>
        </div>
        <el-form class="mt-4">
          <el-form-item :label="$t('wallet.chain')">
            <el-select v-model="bindForm.chain_type">
              <el-option label="Ethereum" value="ethereum" />
              <el-option label="BSC (Binance)" value="bsc" />
              <el-option label="Polygon" value="polygon" />
              <el-option label="Arbitrum" value="arbitrum" />
              <el-option label="Optimism" value="optimism" />
              <el-option label="Tron" value="tron" />
              <el-option label="Solana" value="solana" />
              <el-option label="Avalanche" value="avalanche" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('wallet.address')">
            <el-input v-model="bindForm.address" :placeholder="$t('wallet.addressPlaceholder')" />
          </el-form-item>
          <el-form-item :label="$t('wallet.label')">
            <el-input v-model="bindForm.label" :placeholder="$t('wallet.labelPlaceholder')" />
          </el-form-item>
        </el-form>
        <el-button type="primary" @click="bindStep = 1" :disabled="!bindForm.address">
          {{ $t('wallet.next') }}
        </el-button>
      </div>

      <div v-if="bindStep === 1" class="mt-6">
        <el-alert type="info" :closable="false" class="mb-4">
          {{ $t('wallet.verifyInstruction') }}
        </el-alert>
        <div class="challenge-box">
          <pre class="text-xs whitespace-pre-wrap break-all">{{ challengeText }}</pre>
        </div>
        <el-form class="mt-4">
          <el-form-item :label="$t('wallet.signature')">
            <el-input v-model="bindForm.signature" type="textarea" :rows="3" :placeholder="$t('wallet.signaturePlaceholder')" />
          </el-form-item>
        </el-form>
        <div class="flex gap-2">
          <el-button @click="bindStep = 0">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" @click="verifyAndBind" :loading="binding">
            {{ $t('wallet.verifyAndBind') }}
          </el-button>
        </div>
      </div>

      <div v-if="bindStep === 2" class="mt-6 text-center">
        <el-result icon="success" :title="$t('wallet.bindSuccess')" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  CopyDocument, Link, Plus, ArrowRight, TopRight, Switch,
  CircleCheck, Loading, CreditCard, Cellphone,
} from '@element-plus/icons-vue'
import { walletApi } from '@/api'
import QRCode from 'qrcode'

const { t } = useI18n()
const router = useRouter()

// ===== 状态 =====
const wallets = ref<any[]>([])
const depositOrders = ref<any[]>([])
const showBindDialog = ref(false)
const showConnectDialog = ref(false)
const bindStep = ref(0)
const binding = ref(false)
const depositing = ref(false)
const exchangeRate = ref<number | null>(null)
const challengeText = ref('')
const connectingType = ref('')
const wcQrCanvas = ref<HTMLCanvasElement | null>(null)
const wcURI = ref('')
const showQrCode = ref(false)
const wcConnecting = ref(false)

// 实时连接的钱包
const connectedWallet = ref<{
  type: string
  address: string
  chain: string
  chainId: number
  balance: string
  balanceUsd: string
} | null>(null)

const depositForm = reactive({
  currency: 'USDT',
  chainType: 'ethereum',
  amount: '',
  fiatCurrency: 'CNY',
})

const bindForm = reactive({
  wallet_type: '',
  address: '',
  chain_type: 'ethereum',
  label: '',
  signature: '',
})

// ===== 钱包类型定义 =====
const walletTypes = [
  { value: 'metamask', label: 'MetaMask', icon: '🦊' },
  { value: 'trustwallet', label: 'Trust Wallet', icon: '🛡️' },
  { value: 'walletconnect', label: 'WalletConnect', icon: '🔗' },
  { value: 'phantom', label: 'Phantom', icon: '👻' },
  { value: 'coinbase', label: 'Coinbase Wallet', icon: '🔵' },
  { value: 'okx_wallet', label: 'OKX Wallet', icon: '⭕' },
  { value: 'bitget', label: 'Bitget Wallet', icon: '🟢' },
  { value: 'keplr', label: 'Keplr', icon: '⚛️' },
  { value: 'rabby', label: 'Rabby', icon: '🐰' },
]

const connectableWallets = computed(() => [
  {
    value: 'metamask', label: 'MetaMask', icon: '🦊',
    gradient: 'linear-gradient(135deg, #f6851b, #e2761b)',
    installed: !!window.ethereum?.isMetaMask,
    provider: 'ethereum' as const,
  },
  {
    value: 'coinbase', label: 'Coinbase Wallet', icon: '🔵',
    gradient: 'linear-gradient(135deg, #0052ff, #0040dd)',
    installed: !!(window.ethereum?.isCoinbaseWallet || window.coinbaseWalletExtension),
    provider: 'ethereum' as const,
  },
  {
    value: 'okx_wallet', label: 'OKX Wallet', icon: '⭕',
    gradient: 'linear-gradient(135deg, #000000, #333333)',
    installed: !!window.okxwallet,
    provider: 'ethereum' as const,
  },
  {
    value: 'bitget', label: 'Bitget Wallet', icon: '🟢',
    gradient: 'linear-gradient(135deg, #00f0ff, #00b4d8)',
    installed: !!window.bitkeep?.ethereum,
    provider: 'ethereum' as const,
  },
  {
    value: 'trustwallet', label: 'Trust Wallet', icon: '🛡️',
    gradient: 'linear-gradient(135deg, #3375bb, #0500d7)',
    installed: !!window.ethereum?.isTrust,
    provider: 'ethereum' as const,
  },
  {
    value: 'phantom', label: 'Phantom', icon: '👻',
    gradient: 'linear-gradient(135deg, #ab9ff2, #6e4ec2)',
    installed: !!window.solana?.isPhantom || !!window.phantom?.ethereum,
    provider: 'solana' as const,
  },
  {
    value: 'rabby', label: 'Rabby', icon: '🐰',
    gradient: 'linear-gradient(135deg, #7c7aff, #5b5bef)',
    installed: !!window.ethereum?.isRabby,
    provider: 'ethereum' as const,
  },
  {
    value: 'walletconnect', label: 'WalletConnect', icon: '🔗',
    gradient: 'linear-gradient(135deg, #3b99fc, #2b7ddb)',
    installed: false, // WalletConnect 通过扫码连接
    provider: 'walletconnect' as const,
  },
])

const chainMap: Record<string, string[]> = {
  USDT: ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'tron', 'avalanche', 'solana'],
  USDC: ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'avalanche', 'solana'],
  ETH: ['ethereum', 'arbitrum', 'optimism'],
  BTC: ['ethereum'],
}

const availableChains = computed(() => {
  return chainMap[depositForm.currency] || ['ethereum']
})

const chainIdMap: Record<string, number> = {
  ethereum: 1, bsc: 56, polygon: 137, arbitrum: 42161,
  optimism: 10, avalanche: 43114,
}

const chainNameMap: Record<number, string> = Object.fromEntries(
  Object.entries(chainIdMap).map(([k, v]) => [v, k])
)

// ===== Web3 钱包连接核心 =====

/** 获取 EIP-1193 provider */
function getProvider(walletValue: string): any {
  if (walletValue === 'okx_wallet' && window.okxwallet) return window.okxwallet
  if (walletValue === 'bitget' && window.bitkeep?.ethereum) return window.bitkeep.ethereum
  if (walletValue === 'phantom' && window.phantom?.ethereum) return window.phantom.ethereum
  if (walletValue === 'coinbase' && (window.coinbaseWalletExtension || window.ethereum?.isCoinbaseWallet)) {
    return window.coinbaseWalletExtension || window.ethereum
  }
  return window.ethereum
}

/** 启动 WalletConnect 扫码连接 */
async function startWalletConnect() {
  showQrCode.value = true
  wcConnecting.value = true
  wcURI.value = ''
  try {
    // 动态加载 @walletconnect/ethereum-provider
    const { default: EthereumProvider } = await import('@walletconnect/ethereum-provider')
    const provider = await EthereumProvider.init({
      projectId: 'f1f2a8b3c4d5e6f7a8b9c0d1e2f3a4b5', // WalletConnect Cloud 公共项目 ID
      showQrCode: false, // 手动生成二维码
      chains: [1], // Ethereum mainnet
      optionalChains: [137, 56, 42161, 10, 43114], // Polygon, BSC, Arbitrum, Optimism, Avalanche
      metadata: {
        name: 'TokenHub',
        description: 'AI API Gateway & Token Trading Platform',
        url: 'https://tokenshopai.com',
        icons: ['https://tokenshopai.com/favicon.ico'],
      },
      rpcMap: {
        1: 'https://eth.public-rpc.com',
        137: 'https://polygon-rpc.com',
        56: 'https://bsc-dataseed.binance.org',
      },
    })

    // 获取连接 URI
    const uri = provider.signer?.session?.peer?.metadata?.url || ''
    // WalletConnect v2 通过 events 获取 uri
    provider.on('display_uri', (uri: string) => {
      wcURI.value = uri
      wcConnecting.value = false
      // 生成二维码
      nextTick(async () => {
        if (wcQrCanvas.value && uri) {
          try {
            await QRCode.toCanvas(wcQrCanvas.value, uri, {
              width: 240,
              margin: 2,
              color: { dark: '#1a1a2e', light: '#ffffff' },
            })
          } catch { /* ignore */ }
        }
      })
    })

    // 等待用户扫描
    const accounts = await provider.enable()
    const address = accounts[0]
    if (address) {
      connectedWallet.value = {
        type: 'walletconnect',
        address,
        chain: 'ethereum',
        chainId: 1,
        balance: '0',
        balanceUsd: '',
        provider: provider,
      }
      ElMessage.success(`${t('wallet.connectedSuccess') || '连接成功'}！`)
      showConnectDialog.value = false
      showQrCode.value = false
    }
  } catch (err: any) {
    wcConnecting.value = false
    if (err?.code !== 4001) { // 非用户取消
      ElMessage.error(t('wallet.connectionFailed') || 'WalletConnect 连接失败')
      console.error('WalletConnect error:', err)
    }
  }
}

/** 连接 Web3 钱包 */
async function connectWeb3Wallet(wallet: any) {
  const { value } = wallet

  // WalletConnect 扫码连接
  if (value === 'walletconnect') {
    startWalletConnect()
    return
  }

  connectingType.value = value

  try {
    const prov = getProvider(value)
    if (!prov) {
      // 钱包未安装，跳转到官网下载
      const downloadUrls: Record<string, string> = {
        metamask: 'https://metamask.io/download/',
        coinbase: 'https://www.coinbase.com/wallet/downloads',
        okx_wallet: 'https://www.okx.com/web3',
        bitget: 'https://web3.bitget.com/',
        trustwallet: 'https://trustwallet.com/',
        phantom: 'https://phantom.app/',
        rabby: 'https://rabby.io/',
      }
      const url = downloadUrls[value]
      if (url) {
        window.open(url, '_blank')
        ElMessage.info({ message: '请先安装钱包扩展，安装后刷新页面', duration: 5000 })
      }
      return
    }

    // 1. 请求连接账户
    const accounts = await prov.request({ method: 'eth_requestAccounts' })
    const address = accounts[0]

    if (!address) {
      ElMessage.error(t('wallet.connectionRejected') || '连接被拒绝')
      return
    }

    // 2. 获取链 ID
    const chainIdHex = await prov.request({ method: 'eth_chainId' })
    const chainId = parseInt(chainIdHex, 16)
    const chainName = chainNameMap[chainId] || `chain-${chainId}`

    // 3. 获取余额
    let balance = '0'
    let balanceUsd = ''
    try {
      const balanceHex = await prov.request({
        method: 'eth_getBalance',
        params: [address, 'latest'],
      })
      balance = (parseInt(balanceHex, 16) / 1e18).toFixed(4)
      // 简化的 ETH 价格估算
      const ethPrice = 2500
      balanceUsd = (parseFloat(balance) * ethPrice).toFixed(2)
    } catch { /* 余额获取失败不阻断 */ }

    // 4. 保存连接状态
    connectedWallet.value = {
      type: value,
      address,
      chain: chainName,
      chainId,
      balance,
      balanceUsd,
    }

    // 5. 自动填充充值表单
    depositForm.chainType = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'avalanche'].includes(chainName)
      ? chainName
      : 'ethereum'

    ElMessage.success(`${label} ${t('wallet.connectedSuccess') || '连接成功'}！`)
    showConnectDialog.value = false

    // 6. 注册账户变更监听
    prov.on?.('accountsChanged', handleAccountsChanged)
    prov.on?.('chainChanged', handleChainChanged)

  } catch (err: any) {
    if (err.code === 4001) {
      ElMessage.warning(t('wallet.userRejected') || '用户拒绝了连接请求')
    } else {
      ElMessage.error(t('wallet.connectionFailed') || '钱包连接失败')
      console.error('Wallet connect error:', err)
    }
  } finally {
    connectingType.value = ''
  }
}

/** 账户变更回调 */
function handleAccountsChanged(accounts: string[]) {
  if (accounts.length === 0) {
    connectedWallet.value = null
    ElMessage.info(t('wallet.walletDisconnected') || '钱包已断开连接')
  } else if (connectedWallet.value) {
    connectedWallet.value.address = accounts[0]
    ElMessage.info(t('wallet.accountChanged') || '钱包账户已切换')
  }
}

/** 链变更回调 */
function handleChainChanged(chainIdHex: string) {
  if (connectedWallet.value) {
    const chainId = parseInt(chainIdHex, 16)
    connectedWallet.value.chainId = chainId
    connectedWallet.value.chain = chainNameMap[chainId] || `chain-${chainId}`
    ElMessage.info(`${t('wallet.chainSwitched') || '链已切换'}: ${connectedWallet.value.chain}`)
  }
}

/** 断开钱包连接 */
function disconnectWallet() {
  connectedWallet.value = null
  ElMessage.info(t('wallet.walletDisconnected') || '钱包已断开连接')
}

/** 切换链 */
async function switchChain() {
  if (!connectedWallet.value) return
  const prov = getProvider(connectedWallet.value.type)
  if (!prov) return

  // 循环切换常用链
  const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'avalanche']
  const currentIdx = chains.indexOf(connectedWallet.value.chain)
  const nextChain = chains[(currentIdx + 1) % chains.length]
  const targetChainId = chainIdMap[nextChain]
  if (!targetChainId) return

  const chainIdHex = `0x${targetChainId.toString(16)}`
  try {
    await prov.request({
      method: 'wallet_switchEthereumChain',
      params: [{ chainId: chainIdHex }],
    })
  } catch (switchError: any) {
    // 链未添加，尝试添加
    if (switchError.code === 4902) {
      ElMessage.info(t('wallet.addingChain') || '正在添加链网络...')
    }
  }
}

// ===== 页面跳转 =====

/** 跳转到充值支付页面 */
function goToPayment(channel?: string) {
  if (channel) {
    router.push({ path: '/payment', query: { channel } })
  } else {
    router.push('/payment')
  }
}

/** 从已连接的钱包直接充值 */
function depositFromConnected() {
  if (connectedWallet.value) {
    router.push({
      path: '/payment',
      query: {
        channel: 'crypto',
        wallet_type: connectedWallet.value.type,
        address: connectedWallet.value.address,
        chain: connectedWallet.value.chain,
      },
    })
  } else {
    showConnectDialog.value = true
  }
}

/** 从已绑定钱包充值 */
function depositFromBound(wallet: any) {
  router.push({
    path: '/payment',
    query: {
      channel: 'crypto',
      wallet_type: wallet.wallet_type,
      address: wallet.address,
      chain: wallet.chain_type,
    },
  })
}

// ===== 通用工具函数 =====

function onCurrencyChange() {
  depositForm.chainType = availableChains.value[0] || 'ethereum'
}

function selectWalletType(type: string) {
  bindForm.wallet_type = type
}

function getWalletTagType(type: string) {
  const map: Record<string, string> = {
    metamask: 'warning', trustwallet: 'success', walletconnect: 'info',
    phantom: '', coinbase: 'primary', okx_wallet: 'danger',
  }
  return map[type] || 'info'
}

function getWalletLabel(type: string) {
  const found = walletTypes.find(w => w.value === type)
  return found ? found.label : type
}

function getWalletEmoji(type: string) {
  const found = walletTypes.find(w => w.value === type)
  return found ? found.icon : '💰'
}

function getWalletGradient(type: string) {
  const found = connectableWallets.value.find(w => w.value === type)
  return found?.gradient || 'linear-gradient(135deg, #6366f1, #8b5cf6)'
}

function formatAddress(addr: string) {
  if (!addr) return ''
  if (addr.length > 16) return `${addr.slice(0, 8)}...${addr.slice(-8)}`
  return addr
}

function copyAddress(addr: string) {
  navigator.clipboard.writeText(addr)
  ElMessage.success('Address copied')
}

function getDepositStatusType(status: string) {
  const map: Record<string, string> = {
    completed: 'success', pending: 'warning', failed: 'danger',
    confirming: 'info', detecting: 'info', expired: 'info',
  }
  return map[status] || 'info'
}

// ===== API 调用 =====

async function fetchWallets() {
  try {
    const res = await walletApi.list()
    wallets.value = res.wallets || []
  } catch {
    wallets.value = []
  }
}

async function fetchDepositOrders() {
  try {
    const res = await walletApi.depositOrders()
    depositOrders.value = res.orders || []
  } catch {
    depositOrders.value = []
  }
}

async function fetchExchangeRate() {
  try {
    const res = await walletApi.exchangeRate(depositForm.currency, depositForm.fiatCurrency)
    exchangeRate.value = res.rate
  } catch {
    exchangeRate.value = null
  }
}

async function unbindWallet(address: string) {
  try {
    await walletApi.unbind(address)
    ElMessage.success('Wallet unbound')
    fetchWallets()
  } catch {
    ElMessage.error('Unbind failed')
  }
}

async function createDeposit() {
  depositing.value = true
  try {
    const data: any = {
      currency: depositForm.currency,
      chain_type: depositForm.chainType,
      amount: depositForm.amount,
      fiat_currency: depositForm.fiatCurrency,
    }
    // 如果有已连接钱包，附带地址信息
    if (connectedWallet.value) {
      data.from_address = connectedWallet.value.address
      data.wallet_type = connectedWallet.value.type
    }
    const res = await walletApi.createDeposit(data)
    ElMessage.success('Deposit order created')
    fetchDepositOrders()
  } catch {
    ElMessage.error('Failed to create deposit order')
  } finally {
    depositing.value = false
  }
}

async function verifyAndBind() {
  binding.value = true
  try {
    await walletApi.bind({
      wallet_type: bindForm.wallet_type,
      address: bindForm.address,
      chain_type: bindForm.chain_type,
      label: bindForm.label,
    })
    bindStep.value = 2
    ElMessage.success('Wallet bound successfully')
    fetchWallets()
  } catch {
    ElMessage.error('Bind failed')
  } finally {
    binding.value = false
  }
}

// ===== i18n 快捷 =====
function t(key: string) {
  return (window as any).__vue_i18n__?.global?.t?.(key) || key
}

// ===== 生命周期 =====
onMounted(() => {
  fetchWallets()
  fetchDepositOrders()
  fetchExchangeRate()

  // 检测 URL 参数，自动打开连接对话框
  const route = router.currentRoute.value
  if (route.query.action === 'connect') {
    showConnectDialog.value = true
  }
})

onUnmounted(() => {
  // 清理钱包事件监听
  const prov = window.ethereum
  if (prov?.removeListener) {
    prov.removeListener('accountsChanged', handleAccountsChanged)
    prov.removeListener('chainChanged', handleChainChanged)
  }
})
</script>

<style scoped>
/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.header-actions {
  display: flex;
  gap: 8px;
}

/* 快捷操作 */
.quick-actions-card :deep(.el-card__body) {
  padding: 0;
}
.quick-actions {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0;
}
.action-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  cursor: pointer;
  transition: all 0.2s;
  border-right: 1px solid #f0f0f0;
  border-bottom: 1px solid #f0f0f0;
}
.action-item:nth-child(2n) { border-right: none; }
.action-item:nth-child(n+3) { border-bottom: none; }
.action-item:hover {
  background: #f8fafc;
}
.action-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.action-icon img {
  width: 28px;
  height: 28px;
}
.crypto-icon { background: linear-gradient(135deg, #f7931a20, #f7931a40); }
.alipay-icon { background: linear-gradient(135deg, #1677ff20, #1677ff40); }
.connect-icon { background: linear-gradient(135deg, #6366f120, #6366f140); color: #6366f1; }
.payment-icon { background: linear-gradient(135deg, #10b98120, #10b98140); color: #10b981; }
.action-text { flex: 1; min-width: 0; }
.action-title { font-weight: 600; font-size: 15px; color: #1f2937; }
.action-desc { font-size: 12px; color: #9ca3af; margin-top: 2px; }
.action-arrow { color: #d1d5db; transition: color 0.2s; }
.action-item:hover .action-arrow { color: #6366f1; }

/* 已连接钱包卡片 */
.card-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.connected-wallet-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 20px;
  background: linear-gradient(135deg, #f0f4ff, #eef2ff);
  border-radius: 16px;
  border: 1px solid #c7d2fe;
}
.dark .connected-wallet-card {
  background: linear-gradient(135deg, #1e1b4b, #312e81);
  border-color: #4338ca;
}
.wallet-avatar {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.wallet-emoji { font-size: 28px; }
.wallet-detail { flex: 1; min-width: 0; }
.wallet-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.wallet-name { font-weight: 700; font-size: 16px; }
.chain-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: #e0e7ff;
  color: #4338ca;
  text-transform: uppercase;
  font-weight: 600;
}
.dark .chain-badge { background: #312e81; color: #a5b4fc; }
.wallet-address-row {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  color: #6b7280;
}
.wallet-balance-row {
  margin-top: 6px;
  font-size: 13px;
  color: #4b5563;
}
.wallet-actions-col {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 钱包类型单元格 */
.wallet-type-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}
.wallet-emoji-sm { font-size: 16px; }

/* 来源钱包选择器 */
.from-wallet-selector {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 10px;
  border: 2px solid #e5e7eb;
  cursor: pointer;
  transition: all 0.2s;
}
.from-wallet-selector.selected {
  border-color: #6366f1;
  background: #f5f3ff;
}
.from-wallet-selector.empty {
  border-style: dashed;
  color: #9ca3af;
}
.from-wallet-selector.empty:hover {
  border-color: #6366f1;
  color: #6366f1;
}

/* 连接钱包对话框 */
.connect-wallet-content {
  max-height: 70vh;
  overflow-y: auto;
}
.wallet-connect-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.wallet-connect-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  border: 2px solid #e5e7eb;
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.2s;
}
.wallet-connect-card:hover {
  border-color: #6366f1;
  background: #fafafe;
  transform: translateX(4px);
}
.wallet-connect-card.is-connecting {
  border-color: #6366f1;
  background: #f5f3ff;
}
.wallet-connect-card.is-installed {
  border-color: #d1fae5;
}
.wc-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.wc-emoji { font-size: 22px; }
.wc-info { flex: 1; }
.wc-name { font-weight: 600; font-size: 15px; }
.wc-status { margin-top: 2px; }
.installed-badge {
  font-size: 12px;
  color: #10b981;
  display: flex;
  align-items: center;
  gap: 4px;
}
.not-installed-badge {
  font-size: 12px;
  color: #9ca3af;
}
.wc-arrow { color: #d1d5db; }

/* 二维码区域 */
.qr-connect-section {
  text-align: center;
  margin-top: 12px;
}
.qr-area {
  display: flex;
  justify-content: center;
  margin: 16px 0;
}
.qr-placeholder {
  width: 180px;
  height: 180px;
  border: 2px dashed #d1d5db;
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #9ca3af;
  font-size: 13px;
}
.qr-hint {
  font-size: 12px;
  color: #9ca3af;
}

/* 绑定钱包网格 */
.wallet-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
.wallet-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 8px;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.wallet-option:hover { border-color: #6366f1; }
.wallet-option.selected { border-color: #6366f1; background: #eef2ff; }
.wallet-icon { font-size: 28px; margin-bottom: 4px; }
.wallet-name { font-size: 12px; color: #374151; }
.challenge-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px;
  max-height: 200px;
  overflow-y: auto;
}
.dark .challenge-box { background: #1e293b; border-color: #334155; }

/* WalletConnect 二维码 */
.qr-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px;
  color: #9ca3af;
}
.qr-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.qr-content canvas {
  border-radius: 8px;
  border: 1px solid #e5e7eb;
  padding: 8px;
  background: #fff;
}
.dark .qr-content canvas {
  border-color: #374151;
  background: #1f2937;
}
.qr-tip {
  font-size: 12px;
  color: #9ca3af;
}

/* 响应式 */
@media (max-width: 768px) {
  .quick-actions { grid-template-columns: 1fr; }
  .action-item { border-right: none !important; }
  .action-item:nth-child(n+2) { border-top: 1px solid #f0f0f0; border-bottom: none; }
  .action-item:last-child { border-bottom: none; }
  .connected-wallet-card { flex-direction: column; text-align: center; }
  .wallet-actions-col { flex-direction: row; }
  .wallet-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
