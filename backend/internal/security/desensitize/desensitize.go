package desensitize

import (
	"regexp"
	"strings"
)

// Desensitizer 数据脱敏器
type Desensitizer struct {
	rules    []DesensitizeRule
	piiMap   map[string]string // placeholder -> original (用于还原)
	enabled  bool
	maxCache int               // 最大缓存条目数，防止内存无限增长
}

// DesensitizeRule 脱敏规则
type DesensitizeRule struct {
	Name      string
	Pattern   *regexp.Regexp
	Mask      string // 替换模板，支持 $1, $2 等分组引用
	Type      PIIType
}

// PIIType 个人敏感信息类型
type PIIType string

const (
	PIIEmail       PIIType = "email"
	PIIPhone       PIIType = "phone"
	PIIIDCard      PIIType = "id_card"
	PIIBankCard    PIIType = "bank_card"
	PIIName        PIIType = "name"
	PIIAddress     PIIType = "address"
	PIIPassport    PIIType = "passport"
	PIICreditCard  PIIType = "credit_card"
	PIIMedical     PIIType = "medical"
)

// NewDesensitizer 创建脱敏器
func NewDesensitizer(enabled bool) *Desensitizer {
	d := &Desensitizer{
		piiMap:   make(map[string]string),
		enabled:  enabled,
		maxCache: 10000, // 默认最大缓存1万条
	}
	d.initDefaultRules()
	return d
}

// initDefaultRules 初始化默认脱敏规则
func (d *Desensitizer) initDefaultRules() {
	d.rules = []DesensitizeRule{
		{
			Name:    "email",
			Pattern: regexp.MustCompile(`([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`),
			Mask:    "[EMAIL_REDACTED]",
			Type:    PIIEmail,
		},
		{
			Name:    "china_phone",
			Pattern: regexp.MustCompile(`1[3-9]\d{9}`),
			Mask:    "[PHONE_REDACTED]",
			Type:    PIIPhone,
		},
		{
			Name:    "international_phone",
			Pattern: regexp.MustCompile(`\+\d{1,3}[-\s]?\d{4,14}`),
			Mask:    "[PHONE_REDACTED]",
			Type:    PIIPhone,
		},
		{
			Name:    "china_id_card",
			Pattern: regexp.MustCompile(`[1-9]\d{5}(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]`),
			Mask:    "[ID_REDACTED]",
			Type:    PIIIDCard,
		},
		{
			Name:    "bank_card",
			Pattern: regexp.MustCompile(`\b\d{16,19}\b`),
			Mask:    "[BANKCARD_REDACTED]",
			Type:    PIIBankCard,
		},
		{
			Name:    "credit_card_visa",
			Pattern: regexp.MustCompile(`4\d{12}(\d{3})?`),
			Mask:    "[CREDITCARD_REDACTED]",
			Type:    PIICreditCard,
		},
		{
			Name:    "ssn_us",
			Pattern: regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
			Mask:    "[SSN_REDACTED]",
			Type:    PIIIDCard,
		},
	}
}

// Enabled returns whether the desensitizer is enabled
func (d *Desensitizer) Enabled() bool {
	return d.enabled
}

// Desensitize 对文本进行脱敏处理
func (d *Desensitizer) Desensitize(text string) string {
	if !d.enabled {
		return text
	}

	result := text
	for _, rule := range d.rules {
		matches := rule.Pattern.FindAllString(result, -1)
		for _, match := range matches {
			placeholder := d.generatePlaceholder(rule.Type, match)
			result = strings.ReplaceAll(result, match, placeholder)
		}
	}

	// 脱敏后检查缓存大小，超限则清理
	if len(d.piiMap) > d.maxCache {
		d.ClearCache()
	}

	return result
}

// Restore 还原脱敏内容
func (d *Desensitizer) Restore(text string) string {
	if !d.enabled {
		return text
	}

	result := text
	for placeholder, original := range d.piiMap {
		result = strings.ReplaceAll(result, placeholder, original)
	}
	return result
}

// generatePlaceholder 生成占位符
func (d *Desensitizer) generatePlaceholder(piiType PIIType, original string) string {
	// 使用类型+哈希前6位作为占位符
	hash := simpleHash(original)
	placeholder := "<<" + string(piiType) + ":" + hash + ">>"
	d.piiMap[placeholder] = original
	return placeholder
}

// simpleHash 简单哈希（生产环境应使用更安全的哈希）
func simpleHash(s string) string {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	const hex = "0123456789abcdef"
	result := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		result[i] = hex[h&0xf]
		h >>= 4
	}
	return string(result)
}

// AddRule 添加自定义脱敏规则
func (d *Desensitizer) AddRule(name, pattern, mask string, piiType PIIType) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	d.rules = append(d.rules, DesensitizeRule{
		Name:    name,
		Pattern: re,
		Mask:    mask,
		Type:    piiType,
	})
	return nil
}

// ClearCache 清除脱敏缓存（每次请求后应调用）
func (d *Desensitizer) ClearCache() {
	d.piiMap = make(map[string]string)
}
