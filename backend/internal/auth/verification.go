package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// VerificationType 验证类型
type VerificationType string

const (
	VerificationTypeSMS   VerificationType = "sms"   // 短信验证码
	VerificationTypeEmail VerificationType = "email"  // 邮箱验证码
)

// SMSSender 短信发送接口
type SMSSender interface {
	Send(ctx context.Context, phoneNumber, code, countryCode string) error
}

// EmailSender 邮件发送接口
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// VerificationService 验证码服务
type VerificationService struct {
	logger    *zap.Logger
	rdb       *redis.Client
	smsSender SMSSender
	emailSender EmailSender
}

// NewVerificationService 创建验证码服务
func NewVerificationService(logger *zap.Logger, rdb *redis.Client, smsSender SMSSender, emailSender EmailSender) *VerificationService {
	return &VerificationService{
		logger:      logger,
		rdb:         rdb,
		smsSender:   smsSender,
		emailSender: emailSender,
	}
}

// VerificationCode 验证码结构
type VerificationCode struct {
	ID        string           `json:"id"`
	Type      VerificationType `json:"type"`
	Target    string           `json:"target"`     // 手机号或邮箱
	Code      string           `json:"code"`
	Purpose   string           `json:"purpose"`    // login, register, reset_password, bind_wallet, bind_phone
	ExpiresAt int64            `json:"expires_at"`
	Used      bool             `json:"used"`
}

// SendCodeRequest 发送验证码请求
type SendCodeRequest struct {
	Type        VerificationType `json:"type" binding:"required"`
	Target      string           `json:"target" binding:"required"`
	CountryCode string           `json:"country_code,omitempty"` // 国际区号，如 +86, +1, +81
	Purpose     string           `json:"purpose" binding:"required"`
	CaptchaToken string          `json:"captcha_token,omitempty"` // 人机验证token
}

// VerifyCodeRequest 验证码验证请求
type VerifyCodeRequest struct {
	Type    VerificationType `json:"type" binding:"required"`
	Target  string           `json:"target" binding:"required"`
	Code    string           `json:"code" binding:"required"`
	Purpose string           `json:"purpose" binding:"required"`
}

// LoginByCodeRequest 验证码登录请求
type LoginByCodeRequest struct {
	Type        VerificationType `json:"type" binding:"required"`
	Target      string           `json:"target" binding:"required"`
	Code        string           `json:"code" binding:"required"`
	CountryCode string           `json:"country_code,omitempty"`
}

// RegisterByCodeRequest 验证码注册请求
type RegisterByCodeRequest struct {
	Type        VerificationType `json:"type" binding:"required"`
	Target      string           `json:"target" binding:"required"`
	Code        string           `json:"code" binding:"required"`
	Username    string           `json:"username,omitempty"`
	CountryCode string           `json:"country_code,omitempty"`
}

// SendVerificationCode 发送验证码
func (s *VerificationService) SendVerificationCode(ctx context.Context, req *SendCodeRequest) error {
	// 1. 频率限制（60秒内同一目标只能发1次，1小时内最多5次）
	rateKey := fmt.Sprintf("verify:rate:%s:%s", req.Type, req.Target)
	count, _ := s.rdb.Get(ctx, rateKey).Int()
	if count >= 5 {
		return fmt.Errorf("too many verification codes sent, please try again later")
	}

	cooldownKey := fmt.Sprintf("verify:cooldown:%s:%s", req.Type, req.Target)
	exists, _ := s.rdb.Exists(ctx, cooldownKey).Result()
	if exists > 0 {
		ttl, _ := s.rdb.TTL(ctx, cooldownKey).Result()
		return fmt.Errorf("please wait %d seconds before requesting again", int(ttl.Seconds()))
	}

	// 2. 生成6位验证码
	code := generateCode(6)

	// 3. 存储验证码（5分钟有效）
	codeKey := fmt.Sprintf("verify:code:%s:%s:%s", req.Type, req.Purpose, req.Target)
	if err := s.rdb.Set(ctx, codeKey, code, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("store verification code: %w", err)
	}

	// 4. 设置冷却
	s.rdb.Set(ctx, cooldownKey, "1", 60*time.Second)

	// 5. 增加频率计数
	pipe := s.rdb.Pipeline()
	pipe.Incr(ctx, rateKey)
	pipe.Expire(ctx, rateKey, time.Hour)
	pipe.Exec(ctx)

	// 6. 发送验证码
	switch req.Type {
	case VerificationTypeSMS:
		countryCode := req.CountryCode
		if countryCode == "" {
			countryCode = "+86" // 默认中国
		}
		phoneNumber := fmt.Sprintf("%s%s", countryCode, req.Target)
		if err := s.smsSender.Send(ctx, phoneNumber, code, countryCode); err != nil {
			return fmt.Errorf("send sms: %w", err)
		}
		s.logger.Info("sms verification code sent",
			zap.String("target", req.Target),
			zap.String("country_code", countryCode),
			zap.String("purpose", req.Purpose),
		)

	case VerificationTypeEmail:
		subject := "TokenHub 验证码" // 实际应根据i18n选择
		body := fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2 style="color: #1a1a2e;">TokenHub 验证码</h2>
				<p>您的验证码为：</p>
				<div style="background: #f0f2f5; padding: 16px; border-radius: 8px; text-align: center; font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #4f46e5;">
					%s
				</div>
				<p style="color: #666; font-size: 14px;">验证码5分钟内有效，请勿泄露给他人。</p>
				<p style="color: #999; font-size: 12px;">如非本人操作，请忽略此邮件。</p>
			</div>
		`, code)
		if err := s.emailSender.Send(ctx, req.Target, subject, body); err != nil {
			return fmt.Errorf("send email: %w", err)
		}
		s.logger.Info("email verification code sent",
			zap.String("target", req.Target),
			zap.String("purpose", req.Purpose),
		)

	default:
		return fmt.Errorf("unsupported verification type: %s", req.Type)
	}

	return nil
}

// VerifyCode 验证验证码
func (s *VerificationService) VerifyCode(ctx context.Context, req *VerifyCodeRequest) (bool, error) {
	codeKey := fmt.Sprintf("verify:code:%s:%s:%s", req.Type, req.Purpose, req.Target)
	storedCode, err := s.rdb.Get(ctx, codeKey).Result()
	if err != nil {
		return false, fmt.Errorf("verification code expired or not found")
	}

	// 错误尝试限制（最多5次）
	attemptKey := fmt.Sprintf("verify:attempt:%s:%s:%s", req.Type, req.Purpose, req.Target)
	attempts, _ := s.rdb.Get(ctx, attemptKey).Int()
	if attempts >= 5 {
		s.rdb.Del(ctx, codeKey)
		s.rdb.Del(ctx, attemptKey)
		return false, fmt.Errorf("too many failed attempts, please request a new code")
	}

	if !strings.EqualFold(storedCode, req.Code) {
		s.rdb.Incr(ctx, attemptKey)
		s.rdb.Expire(ctx, attemptKey, 5*time.Minute)
		return false, fmt.Errorf("invalid verification code")
	}

	// 验证成功，删除验证码和尝试计数
	s.rdb.Del(ctx, codeKey)
	s.rdb.Del(ctx, attemptKey)

	return true, nil
}

// LoginOrRegisterByCode 通过验证码登录或注册
// 如果用户不存在则自动注册
func (s *VerificationService) LoginOrRegisterByCode(ctx context.Context, req *LoginByCodeRequest, jwtManager *JWTManager) (*TokenPair, *UserInfo, error) {
	// 1. 验证验证码
	verifyReq := &VerifyCodeRequest{
		Type:    req.Type,
		Target:  req.Target,
		Code:    req.Code,
		Purpose: "login",
	}
	valid, err := s.VerifyCode(ctx, verifyReq)
	if err != nil || !valid {
		return nil, nil, fmt.Errorf("verification failed: %w", err)
	}

	// 2. 查找用户（实际应从数据库查询）
	userKey := fmt.Sprintf("user:%s:%s", req.Type, req.Target)
	userData, err := s.rdb.HGetAll(ctx, userKey).Result()
	if err != nil && err != redis.Nil {
		return nil, nil, fmt.Errorf("query user: %w", err)
	}

	var userInfo *UserInfo
	if len(userData) == 0 {
		// 用户不存在，自动注册
		userInfo = &UserInfo{
			ID:         uuid.New().String(),
			TenantID:   uuid.New().String(), // 新用户创建默认租户
			Role:       RoleDeveloper,
			Phone:      "",
			Email:      "",
			Username:   fmt.Sprintf("user_%s", generateCode(6)),
			Status:     "active",
			CreatedAt:  time.Now().Unix(),
		}

		switch req.Type {
		case VerificationTypeSMS:
			userInfo.Phone = req.Target
			userInfo.CountryCode = req.CountryCode
		case VerificationTypeEmail:
			userInfo.Email = req.Target
		}

		// 存储用户信息
		s.rdb.HSet(ctx, userKey, map[string]interface{}{
			"id":           userInfo.ID,
			"tenant_id":    userInfo.TenantID,
			"role":         string(userInfo.Role),
			"phone":        userInfo.Phone,
			"email":        userInfo.Email,
			"username":     userInfo.Username,
			"country_code": userInfo.CountryCode,
			"status":       userInfo.Status,
			"created_at":   userInfo.CreatedAt,
		})

		s.logger.Info("user auto-registered via verification code",
			zap.String("type", string(req.Type)),
			zap.String("target", req.Target),
			zap.String("user_id", userInfo.ID),
		)
	} else {
		// 用户已存在
		userInfo = &UserInfo{
			ID:          userData["id"],
			TenantID:    userData["tenant_id"],
			Role:        Role(userData["role"]),
			Phone:       userData["phone"],
			Email:       userData["email"],
			Username:    userData["username"],
			CountryCode: userData["country_code"],
			Status:      userData["status"],
		}
	}

	// 3. 生成JWT Token
	email := userInfo.Email
	if email == "" {
		email = userInfo.Phone + "@phone.tokenhub"
	}
	tokenPair, err := jwtManager.GenerateTokenPair(userInfo.ID, userInfo.TenantID, userInfo.Role, email)
	if err != nil {
		return nil, nil, fmt.Errorf("generate token: %w", err)
	}

	return tokenPair, userInfo, nil
}

// UserInfo 用户信息
type UserInfo struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Role        Role   `json:"role"`
	Username    string `json:"username"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
}

// ==================== SMS 提供者实现 ====================

// TwilioSMSSender Twilio短信发送（国际通用）
type TwilioSMSSender struct {
	accountSID string
	authToken  string
	fromNumber string
}

func NewTwilioSMSSender(accountSID, authToken, fromNumber string) *TwilioSMSSender {
	return &TwilioSMSSender{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
	}
}

func (s *TwilioSMSSender) Send(ctx context.Context, phoneNumber, code, countryCode string) error {
	// 实际实现: 调用 Twilio REST API
	// POST https://api.twilio.com/2010-04-01/Accounts/{SID}/Messages.json
	// Body: "【TokenHub】您的验证码为123456，5分钟内有效。"
	return nil
}

// AliyunSMSSender 阿里云短信发送（中国区）
type AliyunSMSSender struct {
	accessKeyID     string
	accessKeySecret string
	signName        string
	templateCode    string
}

func NewAliyunSMSSender(accessKeyID, accessKeySecret, signName, templateCode string) *AliyunSMSSender {
	return &AliyunSMSSender{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		signName:        signName,
		templateCode:    templateCode,
	}
}

func (s *AliyunSMSSender) Send(ctx context.Context, phoneNumber, code, countryCode string) error {
	// 实际实现: 调用阿里云短信API
	// POST https://dysmsapi.aliyuncs.com/
	// TemplateParam: {"code":"123456"}
	return nil
}

// TencentSMSSender 腾讯云短信发送
type TencentSMSSender struct {
	secretID     string
	secretKey    string
	sdkAppID     string
	signName     string
	templateID   string
}

func NewTencentSMSSender(secretID, secretKey, sdkAppID, signName, templateID string) *TencentSMSSender {
	return &TencentSMSSender{
		secretID:   secretID,
		secretKey:  secretKey,
		sdkAppID:   sdkAppID,
		signName:   signName,
		templateID: templateID,
	}
}

func (s *TencentSMSSender) Send(ctx context.Context, phoneNumber, code, countryCode string) error {
	// 实际实现: 调用腾讯云短信API
	// sms.tencentcloudapi.com SendSms
	return nil
}

// VonageSMSSender Vonage (Nexmo) 短信发送（国际）
type VonageSMSSender struct {
	apiKey    string
	apiSecret string
	fromName  string
}

func NewVonageSMSSender(apiKey, apiSecret, fromName string) *VonageSMSSender {
	return &VonageSMSSender{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		fromName:  fromName,
	}
}

func (s *VonageSMSSender) Send(ctx context.Context, phoneNumber, code, countryCode string) error {
	// 实际实现: 调用 Vonage SMS API
	return nil
}

// ==================== Email 提供者实现 ====================

// SMTPEmailSender SMTP邮件发送
type SMTPEmailSender struct {
	host     string
	port     int
	username string
	password string
	fromAddr string
	fromName string
}

func NewSMTPEmailSender(host string, port int, username, password, fromAddr, fromName string) *SMTPEmailSender {
	return &SMTPEmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		fromAddr: fromAddr,
		fromName: fromName,
	}
}

func (s *SMTPEmailSender) Send(ctx context.Context, to, subject, body string) error {
	// 实际实现: 使用 net/smtp 或 gomail 发送
	return nil
}

// SendGridEmailSender SendGrid邮件发送
type SendGridEmailSender struct {
	apiKey   string
	fromAddr string
	fromName string
}

func NewSendGridEmailSender(apiKey, fromAddr, fromName string) *SendGridEmailSender {
	return &SendGridEmailSender{
		apiKey:   apiKey,
		fromAddr: fromAddr,
		fromName: fromName,
	}
}

func (s *SendGridEmailSender) Send(ctx context.Context, to, subject, body string) error {
	// 实际实现: 调用 SendGrid REST API
	// POST https://api.sendgrid.com/v3/mail/send
	return nil
}

// ==================== 辅助函数 ====================

func generateCode(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = '0' + (b[i] % 10)
	}
	return string(b)
}

// Ensure hex import is used
var _ = hex.EncodeToString
