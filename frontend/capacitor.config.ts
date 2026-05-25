import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.tokenhub.app',
  appName: 'TokenHub',
  webDir: 'dist',
  server: {
    // 开发时可指向本地开发服务器，生产环境留空使用打包文件
    // url: 'http://你的局域网IP:3000',
    // cleartext: true,
    androidScheme: 'https',
  },
  plugins: {
    SplashScreen: {
      launchShowDuration: 2000,
      launchAutoHide: true,
      backgroundColor: '#1e40af',
      showSpinner: true,
      spinnerColor: '#ffffff',
    },
    StatusBar: {
      style: 'DARK',
      backgroundColor: '#1e40af',
    },
  },
  ios: {
    contentInset: 'automatic',
    // 允许混合内容
    allowsLinkPreview: false,
  },
}

export default config
