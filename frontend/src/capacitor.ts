import { Capacitor } from '@capacitor/core'
import { StatusBar, Style } from '@capacitor/status-bar'
import { SplashScreen } from '@capacitor/splash-screen'
import { App } from '@capacitor/app'

export async function initCapacitor() {
  if (!Capacitor.isNativePlatform()) return

  // 设置状态栏
  try {
    await StatusBar.setStyle({ style: Style.Dark })
    await StatusBar.setBackgroundColor({ color: '#1e40af' })
  } catch (e) {
    console.warn('StatusBar not available:', e)
  }

  // 隐藏启动屏
  try {
    await SplashScreen.hide({ fadeOutDuration: 300 })
  } catch (e) {
    console.warn('SplashScreen not available:', e)
  }

  // Android 返回键处理
  App.addListener('backButton', ({ canGoBack }) => {
    if (canGoBack) {
      window.history.back()
    } else {
      App.exitApp()
    }
  })
}
