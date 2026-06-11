import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh'
import enUS from './locales/en'
import jaJP from './locales/ja'
import koKR from './locales/ko'
import zhTW from './locales/zh-TW'
import frFR from './locales/fr'
import deDE from './locales/de'
import esES from './locales/es'
import ptBR from './locales/pt-BR'
import itIT from './locales/it'
import ruRU from './locales/ru'
import arSA from './locales/ar'
import hiIN from './locales/hi'
import idID from './locales/id'
import viVN from './locales/vi'
import thTH from './locales/th'
import trTR from './locales/tr'
import nlNL from './locales/nl'

const i18n = createI18n({
  legacy: false,
  locale: localStorage.getItem('locale') || 'zh-CN',
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
    'ja-JP': jaJP,
    'ko-KR': koKR,
    'zh-TW': zhTW,
    'fr-FR': frFR,
    'de-DE': deDE,
    'es-ES': esES,
    'pt-BR': ptBR,
    'it-IT': itIT,
    'ru-RU': ruRU,
    'ar-SA': arSA,
    'hi-IN': hiIN,
    'id-ID': idID,
    'vi-VN': viVN,
    'th-TH': thTH,
    'tr-TR': trTR,
    'nl-NL': nlNL,
  },
})

export default i18n
