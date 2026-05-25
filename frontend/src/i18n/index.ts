import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh'
import enUS from './locales/en'
import jaJP from './locales/ja'
import koKR from './locales/ko'

const i18n = createI18n({
  legacy: false,
  locale: localStorage.getItem('locale') || 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
    'ja-JP': jaJP,
    'ko-KR': koKR,
  },
})

export default i18n
