package waf

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// WAF Web应用防火墙
type WAF struct {
	mu            sync.RWMutex
	ipBlacklist   map[string]bool
	ipWhitelist   map[string]bool
	rateLimits    map[string]*rateLimiter
	promptRules   []PromptRule
	enabled       bool
}

// PromptRule Prompt安全规则
type PromptRule struct {
	Name      string
	Pattern   *regexp.Regexp
	Action    Action
	Severity  Severity
}

type Action string

const (
	ActionBlock   Action = "block"   // 拦截请求
	ActionWarn    Action = "warn"    // 告警但放行
	ActionSanitize Action = "sanitize" // 清理后放行
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type rateLimiter struct {
	tokens    float64
	maxTokens float64
	rate      float64 // tokens/second
	lastTime  time.Time
}

// WAFResult WAF检查结果
type WAFResult struct {
	Blocked      bool
	Reason       string
	Severity     Severity
	MatchedRule  string
	Sanitized    string // 清理后的内容
}

// NewWAF 创建WAF
func NewWAF(enabled bool) *WAF {
	waf := &WAF{
		ipBlacklist: make(map[string]bool),
		ipWhitelist: make(map[string]bool),
		rateLimits:  make(map[string]*rateLimiter),
		enabled:     enabled,
	}
	waf.initDefaultPromptRules()
	return waf
}

// initDefaultPromptRules 初始化默认Prompt安全规则
func (w *WAF) initDefaultPromptRules() {
	w.promptRules = []PromptRule{
		{
			Name:    "prompt_injection_ignore_instructions",
			Pattern: regexp.MustCompile(`(?i)(ignore\s+(previous|above|all)\s+instructions|disregard\s+(your|the)\s+(training|instructions))`),
			Action:  ActionBlock,
			Severity: SeverityHigh,
		},
		{
			Name:    "prompt_injection_role_override",
			Pattern: regexp.MustCompile(`(?i)(you\s+are\s+now|pretend\s+you\s+(are|can)|roleplay\s+as|act\s+as\s+(?:a\s+)?(?:malicious|harmful|evil|criminal))`),
			Action:  ActionWarn,
			Severity: SeverityMedium,
		},
		{
			Name:    "prompt_injection_data_extraction",
			Pattern: regexp.MustCompile(`(?i)(reveal\s+(your|the)\s+(system|initial)\s+prompt|show\s+me\s+(your|the)\s+instructions|output\s+your\s+system\s+prompt)`),
			Action:  ActionBlock,
			Severity: SeverityCritical,
		},
		{
			Name:    "prompt_injection_dan",
			Pattern: regexp.MustCompile(`(?i)(DAN\s+mode|jailbreak|bypass\s+(safety|filter|restriction))`),
			Action:  ActionBlock,
			Severity: SeverityCritical,
		},
		{
			Name:    "sql_injection",
			Pattern: regexp.MustCompile(`(?i)((UNION\s+SELECT|DROP\s+TABLE|INSERT\s+INTO|DELETE\s+FROM|OR\s+1\s*=\s*1|;\s*--))`),
			Action:  ActionBlock,
			Severity: SeverityHigh,
		},
		{
			Name:    "code_execution",
			Pattern: regexp.MustCompile(`(?i)(exec\s*\(|eval\s*\(|system\s*\(|os\.system|subprocess|__import__)`),
			Action:  ActionWarn,
			Severity: SeverityMedium,
		},
	}
}

// CheckRequest 检查请求
func (w *WAF) CheckRequest(ip string, content string) *WAFResult {
	if !w.enabled {
		return &WAFResult{Blocked: false}
	}

	// 1. IP白名单检查
	w.mu.RLock()
	if w.ipWhitelist[ip] {
		w.mu.RUnlock()
		return &WAFResult{Blocked: false}
	}

	// 2. IP黑名单检查
	if w.ipBlacklist[ip] {
		w.mu.RUnlock()
		return &WAFResult{
			Blocked:     true,
			Reason:      "IP blocked",
			Severity:    SeverityHigh,
			MatchedRule: "ip_blacklist",
		}
	}
	w.mu.RUnlock()

	// 3. 限流检查
	if !w.checkRateLimit(ip) {
		return &WAFResult{
			Blocked:     true,
			Reason:      "rate limit exceeded",
			Severity:    SeverityMedium,
			MatchedRule: "rate_limit",
		}
	}

	// 4. Prompt安全检查
	return w.checkPrompt(content)
}

// checkPrompt 检查Prompt安全性
func (w *WAF) checkPrompt(content string) *WAFResult {
	for _, rule := range w.promptRules {
		if rule.Pattern.MatchString(content) {
			result := &WAFResult{
				Blocked:     rule.Action == ActionBlock,
				Reason:      "matched security rule: " + rule.Name,
				Severity:    rule.Severity,
				MatchedRule: rule.Name,
			}

			if rule.Action == ActionSanitize {
				result.Sanitized = rule.Pattern.ReplaceAllString(content, "[REDACTED]")
			}

			return result
		}
	}
	return &WAFResult{Blocked: false}
}

// checkRateLimit 限流检查
func (w *WAF) checkRateLimit(ip string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	limiter, ok := w.rateLimits[ip]
	if !ok {
		w.rateLimits[ip] = &rateLimiter{
			tokens:    99, // 留一个给当前请求
			maxTokens: 100,
			rate:      10, // 10 tokens/sec
			lastTime:  time.Now(),
		}
		return true
	}

	now := time.Now()
	elapsed := now.Sub(limiter.lastTime).Seconds()
	limiter.tokens = min(limiter.maxTokens, limiter.tokens+elapsed*limiter.rate)
	limiter.lastTime = now

	if limiter.tokens < 1 {
		return false
	}
	limiter.tokens--
	return true
}

// AddIPBlacklist 添加IP黑名单
func (w *WAF) AddIPBlacklist(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ipBlacklist[ip] = true
}

// RemoveIPBlacklist 移除IP黑名单
func (w *WAF) RemoveIPBlacklist(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.ipBlacklist, ip)
}

// AddIPWhitelist 添加IP白名单
func (w *WAF) AddIPWhitelist(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ipWhitelist[ip] = true
}

// IsEnabled 是否启用
func (w *WAF) IsEnabled() bool { return w.enabled }

// AddPromptRule 添加Prompt安全规则
func (w *WAF) AddPromptRule(name, pattern string, action Action, severity Severity) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	w.promptRules = append(w.promptRules, PromptRule{
		Name:     name,
		Pattern:  re,
		Action:   action,
		Severity: severity,
	})
	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// IsIPBlocked 检查IP是否在黑名单中
func (w *WAF) IsIPBlocked(ip string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.ipBlacklist[ip]
}

// GetBlockedIPs 获取黑名单IP列表
func (w *WAF) GetBlockedIPs() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var ips []string
	for ip := range w.ipBlacklist {
		ips = append(ips, ip)
	}
	return ips
}

// ParseAction 字符串转Action
func ParseAction(s string) Action {
	switch strings.ToLower(s) {
	case "block":
		return ActionBlock
	case "warn":
		return ActionWarn
	case "sanitize":
		return ActionSanitize
	default:
		return ActionWarn
	}
}
