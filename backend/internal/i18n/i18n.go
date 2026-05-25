package i18n

import (
	"sync"
)

// I18n 国际化服务
type I18n struct {
	mu       sync.RWMutex
	messages map[string]map[string]string // locale -> key -> message
	defaultLocale string
	supportedLocales []string
}

// NewI18n 创建国际化服务
func NewI18n(defaultLocale string, supportedLocales []string) *I18n {
	i := &I18n{
		messages:         make(map[string]map[string]string),
		defaultLocale:    defaultLocale,
		supportedLocales: supportedLocales,
	}
	i.loadDefaultMessages()
	return i
}

// loadDefaultMessages 加载默认消息
func (i *I18n) loadDefaultMessages() {
	// 中文
	i.messages["zh-CN"] = map[string]string{
		"app.name":           "Token中站站",
		"app.description":    "企业级AI API网关与多租户SaaS管理平台",
		"model.list":         "模型列表",
		"model.create":       "创建模型",
		"model.edit":         "编辑模型",
		"model.delete":       "删除模型",
		"model.route":        "模型路由",
		"model.enabled":      "已启用",
		"model.disabled":     "已禁用",
		"billing.balance":    "账户余额",
		"billing.topup":      "充值",
		"billing.history":    "交易记录",
		"billing.export":     "导出对账单",
		"billing.insufficient": "余额不足",
		"tenant.list":        "租户列表",
		"tenant.create":      "创建租户",
		"tenant.plan":        "套餐",
		"tenant.users":       "用户管理",
		"security.waf":       "安全防火墙",
		"security.audit":     "审计日志",
		"security.desensitize": "数据脱敏",
		"monitor.dashboard":  "监控大屏",
		"monitor.qps":        "请求/秒",
		"monitor.latency":    "响应延迟",
		"monitor.tokens":     "Token消耗",
		"monitor.success_rate": "成功率",
		"alert.high_latency": "高延迟告警",
		"alert.error_rate":   "错误率告警",
		"alert.balance_low":  "余额不足告警",
		"alert.circuit_open": "熔断器开启告警",
		"common.save":        "保存",
		"common.cancel":      "取消",
		"common.delete":      "删除",
		"common.search":      "搜索",
		"common.export":      "导出",
		"common.loading":     "加载中...",
		"common.success":     "操作成功",
		"common.error":       "操作失败",
		"error.unauthorized": "未授权访问",
		"error.forbidden":    "权限不足",
		"error.not_found":    "资源不存在",
		"error.rate_limit":   "请求频率超限",
		"error.internal":     "内部错误",
	}

	// 英文
	i.messages["en-US"] = map[string]string{
		"app.name":           "Token Hub",
		"app.description":    "Enterprise AI API Gateway & Multi-tenant SaaS Platform",
		"model.list":         "Model List",
		"model.create":      "Create Model",
		"model.edit":         "Edit Model",
		"model.delete":       "Delete Model",
		"model.route":        "Model Route",
		"model.enabled":      "Enabled",
		"model.disabled":     "Disabled",
		"billing.balance":    "Account Balance",
		"billing.topup":      "Top Up",
		"billing.history":    "Transaction History",
		"billing.export":     "Export Statement",
		"billing.insufficient": "Insufficient Balance",
		"tenant.list":        "Tenant List",
		"tenant.create":      "Create Tenant",
		"tenant.plan":        "Plan",
		"tenant.users":       "User Management",
		"security.waf":       "Security Firewall",
		"security.audit":     "Audit Log",
		"security.desensitize": "Data Desensitization",
		"monitor.dashboard":  "Monitor Dashboard",
		"monitor.qps":        "Requests/sec",
		"monitor.latency":    "Latency",
		"monitor.tokens":     "Token Usage",
		"monitor.success_rate": "Success Rate",
		"alert.high_latency": "High Latency Alert",
		"alert.error_rate":   "Error Rate Alert",
		"alert.balance_low":  "Low Balance Alert",
		"alert.circuit_open": "Circuit Breaker Open Alert",
		"common.save":        "Save",
		"common.cancel":      "Cancel",
		"common.delete":      "Delete",
		"common.search":      "Search",
		"common.export":      "Export",
		"common.loading":     "Loading...",
		"common.success":     "Success",
		"common.error":       "Error",
		"error.unauthorized": "Unauthorized",
		"error.forbidden":    "Forbidden",
		"error.not_found":    "Not Found",
		"error.rate_limit":   "Rate Limit Exceeded",
		"error.internal":     "Internal Error",
	}

	// 日文
	i.messages["ja-JP"] = map[string]string{
		"app.name":           "Token Hub",
		"app.description":    "エンタープライズAI APIゲートウェイ＆マルチテナントSaaSプラットフォーム",
		"model.list":         "モデル一覧",
		"billing.balance":    "アカウント残高",
		"tenant.list":        "テナント一覧",
		"security.waf":       "セキュリティファイアウォール",
		"monitor.dashboard":  "モニターダッシュボード",
		"common.save":        "保存",
		"common.cancel":      "キャンセル",
	}

	// 韩文
	i.messages["ko-KR"] = map[string]string{
		"app.name":           "Token Hub",
		"app.description":    "엔터프라이즈 AI API 게이트웨이 및 멀티테넌트 SaaS 플랫폼",
		"model.list":         "모델 목록",
		"billing.balance":    "계정 잔액",
		"tenant.list":        "테넌트 목록",
		"common.save":        "저장",
		"common.cancel":      "취소",
	}
}

// T 翻译
func (i *I18n) T(locale, key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if msgs, ok := i.messages[locale]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	// 回退到默认语言
	if msgs, ok := i.messages[i.defaultLocale]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	return key
}

// GetSupportedLocales 获取支持的语言
func (i *I18n) GetSupportedLocales() []string {
	return i.supportedLocales
}

// AddMessages 添加自定义翻译
func (i *I18n) AddMessages(locale string, messages map[string]string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.messages[locale]; !ok {
		i.messages[locale] = make(map[string]string)
	}
	for k, v := range messages {
		i.messages[locale][k] = v
	}
}
