/**
 * IP 地理位置检测服务
 * 用于根据用户 IP 自动切换登录方式和界面语言
 */

export interface IPInfo {
  ip: string
  country: string
  country_code: string
  region: string
  city: string
  timezone: string
  is_china: boolean
  is_eu: boolean
  continent: string
}

export interface LoginMethods {
  primary: LoginMethod[]
  secondary: LoginMethod[]
  recommended: LoginMethod
}

export type LoginMethod = 
  | 'phone' 
  | 'email' 
  | 'alipay' 
  | 'wechat' 
  | 'google' 
  | 'github' 
  | 'apple'
  | 'facebook'
  | 'twitter'
  | 'microsoft'
  | 'web3'

// 国家手机号区号映射
export const countryPhoneCodes: Record<string, { code: string; name: string; pattern: string }> = {
  CN: { code: '+86', name: '中国', pattern: '^1[3-9]\\d{9}$' },
  US: { code: '+1', name: '美国', pattern: '^\\d{10}$' },
  GB: { code: '+44', name: '英国', pattern: '^\\d{10,11}$' },
  JP: { code: '+81', name: '日本', pattern: '^\\d{10,11}$' },
  KR: { code: '+82', name: '韩国', pattern: '^\\d{10,11}$' },
  DE: { code: '+49', name: '德国', pattern: '^\\d{10,11}$' },
  FR: { code: '+33', name: '法国', pattern: '^\\d{9}$' },
  AU: { code: '+61', name: '澳大利亚', pattern: '^\\d{9}$' },
  CA: { code: '+1', name: '加拿大', pattern: '^\\d{10}$' },
  IN: { code: '+91', name: '印度', pattern: '^\\d{10}$' },
  BR: { code: '+55', name: '巴西', pattern: '^\\d{10,11}$' },
  RU: { code: '+7', name: '俄罗斯', pattern: '^\\d{10}$' },
  MX: { code: '+52', name: '墨西哥', pattern: '^\\d{10}$' },
  ID: { code: '+62', name: '印尼', pattern: '^\\d{10,12}$' },
  TR: { code: '+90', name: '土耳其', pattern: '^\\d{10}$' },
  SA: { code: '+966', name: '沙特', pattern: '^\\d{9}$' },
  AE: { code: '+971', name: '阿联酋', pattern: '^\\d{9}$' },
  SG: { code: '+65', name: '新加坡', pattern: '^\\d{8}$' },
  MY: { code: '+60', name: '马来西亚', pattern: '^\\d{9,10}$' },
  TH: { code: '+66', name: '泰国', pattern: '^\\d{9}$' },
  VN: { code: '+84', name: '越南', pattern: '^\\d{9,10}$' },
  PH: { code: '+63', name: '菲律宾', pattern: '^\\d{10}$' },
  TW: { code: '+886', name: '台湾', pattern: '^\\d{9,10}$' },
  HK: { code: '+852', name: '香港', pattern: '^\\d{8}$' },
  IT: { code: '+39', name: '意大利', pattern: '^\\d{10}$' },
  ES: { code: '+34', name: '西班牙', pattern: '^\\d{9}$' },
  NL: { code: '+31', name: '荷兰', pattern: '^\\d{9}$' },
  PL: { code: '+48', name: '波兰', pattern: '^\\d{9}$' },
  SE: { code: '+46', name: '瑞典', pattern: '^\\d{9}$' },
  NO: { code: '+47', name: '挪威', pattern: '^\\d{8}$' },
  DK: { code: '+45', name: '丹麦', pattern: '^\\d{8}$' },
  FI: { code: '+358', name: '芬兰', pattern: '^\\d{9,10}$' },
  BE: { code: '+32', name: '比利时', pattern: '^\\d{9}$' },
  AT: { code: '+43', name: '奥地利', pattern: '^\\d{10,13}$' },
  CH: { code: '+41', name: '瑞士', pattern: '^\\d{9}$' },
  PT: { code: '+351', name: '葡萄牙', pattern: '^\\d{9}$' },
  IE: { code: '+353', name: '爱尔兰', pattern: '^\\d{9}$' },
  NZ: { code: '+64', name: '新西兰', pattern: '^\\d{9,10}$' },
  ZA: { code: '+27', name: '南非', pattern: '^\\d{9}$' },
  NG: { code: '+234', name: '尼日利亚', pattern: '^\\d{10}$' },
  EG: { code: '+20', name: '埃及', pattern: '^\\d{10}$' },
  IL: { code: '+972', name: '以色列', pattern: '^\\d{9}$' },
  AR: { code: '+54', name: '阿根廷', pattern: '^\\d{10}$' },
  CL: { code: '+56', name: '智利', pattern: '^\\d{9}$' },
  CO: { code: '+57', name: '哥伦比亚', pattern: '^\\d{10}$' },
  PE: { code: '+51', name: '秘鲁', pattern: '^\\d{9}$' },
  PK: { code: '+92', name: '巴基斯坦', pattern: '^\\d{10}$' },
  BD: { code: '+880', name: '孟加拉', pattern: '^\\d{10,11}$' },
}

// 缓存 IP 信息
let cachedIPInfo: IPInfo | null = null

/**
 * 检测用户 IP 地理位置
 */
export async function detectIP(): Promise<IPInfo> {
  if (cachedIPInfo) {
    return cachedIPInfo
  }

  try {
    // 使用 ipapi.co 免费 API（每月 1000 次免费）
    const response = await fetch('https://ipapi.co/json/', {
      headers: {
        'Accept': 'application/json',
      },
    })
    
    if (!response.ok) {
      throw new Error('IP detection failed')
    }

    const data = await response.json()
    
    cachedIPInfo = {
      ip: data.ip || '',
      country: data.country_name || '',
      country_code: data.country || 'US',
      region: data.region || '',
      city: data.city || '',
      timezone: data.timezone || '',
      is_china: data.country === 'CN',
      is_eu: data.in_eu || false,
      continent: data.continent_code || '',
    }

    return cachedIPInfo
  } catch (error) {
    console.warn('IP detection failed, using default:', error)
    // 默认返回美国
    return {
      ip: '',
      country: 'United States',
      country_code: 'US',
      region: '',
      city: '',
      timezone: 'America/New_York',
      is_china: false,
      is_eu: false,
      continent: 'NA',
    }
  }
}

/**
 * 根据国家代码获取推荐的登录方式
 */
export function getRecommendedLoginMethods(countryCode: string): LoginMethods {
  // 中国大陆 - 仅显示国内登录方式（支付宝/微信/手机号/邮箱）
  if (countryCode === 'CN') {
    return {
      primary: ['phone', 'alipay', 'wechat'],
      secondary: [], // 国内不显示任何国外登录方式
      recommended: 'phone',
    }
  }
  
  // 港澳台 - 手机号 + OAuth
  if (['HK', 'TW', 'MO'].includes(countryCode)) {
    return {
      primary: ['phone', 'email', 'google'],
      secondary: ['wechat', 'github', 'web3'],
      recommended: 'phone',
    }
  }
  
  // 欧美国家 - Email + OAuth 为主
  if (['US', 'GB', 'CA', 'AU', 'NZ', 'DE', 'FR', 'IT', 'ES', 'NL'].includes(countryCode)) {
    return {
      primary: ['email', 'google', 'github', 'apple'],
      secondary: ['phone', 'microsoft', 'facebook'],
      recommended: 'email',
    }
  }
  
  // 日韩 - 手机号 + OAuth
  if (['JP', 'KR'].includes(countryCode)) {
    return {
      primary: ['phone', 'email', 'google', 'apple'],
      secondary: ['github', 'twitter', 'web3'],
      recommended: 'phone',
    }
  }
  
  // 东南亚 - 手机号为主
  if (['SG', 'MY', 'TH', 'VN', 'ID', 'PH'].includes(countryCode)) {
    return {
      primary: ['phone', 'email', 'google'],
      secondary: ['github', 'facebook', 'web3'],
      recommended: 'phone',
    }
  }
  
  // 中东 - 手机号 + OAuth
  if (['SA', 'AE', 'IL', 'TR', 'EG'].includes(countryCode)) {
    return {
      primary: ['phone', 'email', 'google', 'apple'],
      secondary: ['github', 'twitter'],
      recommended: 'phone',
    }
  }
  
  // 其他国家默认
  return {
    primary: ['email', 'google', 'phone'],
    secondary: ['github', 'apple', 'web3'],
    recommended: 'email',
  }
}

/**
 * 获取登录方式的显示信息
 */
export function getLoginMethodInfo(method: LoginMethod): { 
  name: string
  icon: string
  color: string
  bgColor: string
  iconType: 'svg' | 'emoji'
} {
  const info: Record<LoginMethod, { name: string; icon: string; color: string; bgColor: string; iconType: 'svg' | 'emoji' }> = {
    phone:    { name: '手机号',   icon: 'phone',     color: '#10B981', bgColor: '#ECFDF5', iconType: 'svg' },
    email:    { name: '邮箱',     icon: 'email',     color: '#3B82F6', bgColor: '#EFF6FF', iconType: 'svg' },
    alipay:   { name: '支付宝',   icon: 'alipay',    color: '#1677FF', bgColor: '#E6F4FF', iconType: 'svg' },
    wechat:   { name: '微信',     icon: 'wechat',    color: '#07C160', bgColor: '#E8F8EE', iconType: 'svg' },
    google:   { name: 'Google',   icon: 'google',    color: '#DB4437', bgColor: '#FEE2E2', iconType: 'svg' },
    github:   { name: 'GitHub',   icon: 'github',    color: '#24292F', bgColor: '#F3F4F6', iconType: 'svg' },
    apple:    { name: 'Apple',    icon: 'apple',     color: '#000000', bgColor: '#F3F4F6', iconType: 'svg' },
    facebook: { name: 'Facebook', icon: 'facebook',  color: '#1877F2', bgColor: '#E0E7FF', iconType: 'svg' },
    twitter:  { name: 'Twitter',  icon: 'twitter',   color: '#1DA1F2', bgColor: '#E0F2FE', iconType: 'svg' },
    microsoft:{ name: 'Microsoft',icon: 'microsoft', color: '#00A4EF', bgColor: '#E0F2FE', iconType: 'svg' },
    web3:     { name: 'Web3钱包', icon: 'wallet',    color: '#F6851B', bgColor: '#FEF3C7', iconType: 'svg' },
  }
  
  return info[method] || info.email
}

/**
 * 获取国家默认手机区号
 */
export function getDefaultPhoneCode(countryCode: string): string {
  return countryPhoneCodes[countryCode]?.code || '+1'
}

/**
 * 验证手机号格式
 */
export function validatePhone(phone: string, countryCode: string): boolean {
  const config = countryPhoneCodes[countryCode]
  if (!config) {
    // 默认验证：至少8位数字
    return /^\d{8,15}$/.test(phone.replace(/[\s-]/g, ''))
  }
  
  const pattern = new RegExp(config.pattern)
  return pattern.test(phone.replace(/[\s-]/g, ''))
}

/**
 * 清除 IP 缓存（用于测试）
 */
export function clearIPCache(): void {
  cachedIPInfo = null
}
