/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

/** EIP-1193 Ethereum Provider 接口 */
interface EIP1193Provider {
  request(args: { method: string; params?: any[] }): Promise<any>
  on(event: string, handler: (...args: any[]) => void): void
  removeListener(event: string, handler: (...args: any[]) => void): void
  isMetaMask?: boolean
  isCoinbaseWallet?: boolean
  isTrust?: boolean
  isRabby?: boolean
}

interface Window {
  ethereum?: EIP1193Provider & {
    isMetaMask?: boolean
    isCoinbaseWallet?: boolean
    isTrust?: boolean
    isRabby?: boolean
  }
  okxwallet?: EIP1193Provider
  bitkeep?: {
    ethereum: EIP1193Provider
  }
  phantom?: {
    ethereum: EIP1193Provider
  }
  solana?: {
    isPhantom?: boolean
  }
  coinbaseWalletExtension?: EIP1193Provider
}
