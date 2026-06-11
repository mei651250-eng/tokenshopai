package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Gateway      GatewayConfig      `mapstructure:"gateway"`
	Billing      BillingConfig      `mapstructure:"billing"`
	Wallet       WalletConfig       `mapstructure:"wallet"`
	Payment      PaymentConfig      `mapstructure:"payment"`
	Verification VerificationConfig `mapstructure:"verification"`
	Monitor      MonitorConfig      `mapstructure:"monitor"`
	Security     SecurityConfig     `mapstructure:"security"`
	I18n         I18nConfig         `mapstructure:"i18n"`
	KeyVault     KeyVaultConfig     `mapstructure:"keyvault"`
	Admin        AdminConfig        `mapstructure:"admin"`
	OAuth        OAuthConfig        `mapstructure:"oauth"`
}

type OAuthConfig struct {
	GoogleClientID     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`
	GitHubClientID     string `mapstructure:"github_client_id"`
	GitHubClientSecret string `mapstructure:"github_client_secret"`
}

type KeyVaultConfig struct {
	MasterKey string `mapstructure:"master_key"`
}

type AdminConfig struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // debug, release, test
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	GracefulWait time.Duration `mapstructure:"graceful_wait"`
	FrontendURL  string        `mapstructure:"frontend_url"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level"`
}

type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	Expire     time.Duration `mapstructure:"expire"`
	Issuer     string        `mapstructure:"issuer"`
	RefreshExp time.Duration `mapstructure:"refresh_exp"`
}

type GatewayConfig struct {
	MaxRetries      int           `mapstructure:"max_retries"`
	Timeout         time.Duration `mapstructure:"timeout"`
	StreamTimeout   time.Duration `mapstructure:"stream_timeout"`
	MaxConcurrent   int           `mapstructure:"max_concurrent"`
	RateLimitPerSec int           `mapstructure:"rate_limit_per_sec"`
}

type BillingConfig struct {
	DefaultCurrency  string `mapstructure:"default_currency"`
	MinBalance       int64  `mapstructure:"min_balance"` // 最小余额（分）
	TokenGranularity int    `mapstructure:"token_granularity"`
}

type MonitorConfig struct {
	EnableMetrics  bool          `mapstructure:"enable_metrics"`
	MetricsPath    string        `mapstructure:"metrics_path"`
	ReportInterval time.Duration `mapstructure:"report_interval"`
	AlertWebhook   string        `mapstructure:"alert_webhook"`
}

type SecurityConfig struct {
	EnableWAF       bool       `mapstructure:"enable_waf"`
	EnableDesensitize bool     `mapstructure:"enable_desensitize"`
	BlockedIPs      []string   `mapstructure:"blocked_ips"`
	MaxRequestBody  int64      `mapstructure:"max_request_body"`
	CORS            CORSConfig `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

type I18nConfig struct {
	DefaultLocale  string   `mapstructure:"default_locale"`
	SupportedLocales []string `mapstructure:"supported_locales"`
}

// WalletConfig Web3钱包与加密货币充值配置
type WalletConfig struct {
	DepositAddresses map[string]string    `mapstructure:"deposit_addresses"`
	ChainMonitor     ChainMonitorConfig   `mapstructure:"chain_monitor"`
}

type ChainMonitorConfig struct {
	PollInterval time.Duration `mapstructure:"poll_interval"`
	BatchSize    int           `mapstructure:"batch_size"`
}

// PaymentConfig 支付渠道配置
type PaymentConfig struct {
	Alipay     AlipayConfig     `mapstructure:"alipay"`
	AlipayHK   AlipayHKConfig   `mapstructure:"alipay_hk"`
	WeChatPay  WeChatPayConfig  `mapstructure:"wechat_pay"`
	PayPal     PayPalConfig     `mapstructure:"paypal"`
	WorldFirst WorldFirstConfig `mapstructure:"worldfirst"`
	Payoneer   PayoneerConfig   `mapstructure:"payoneer"`
	Wise       WiseConfig       `mapstructure:"wise"`
	Stripe     StripeConfig     `mapstructure:"stripe"`
}

type AlipayConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	AppID      string `mapstructure:"app_id"`
	PrivateKey string `mapstructure:"private_key"`
	PublicKey  string `mapstructure:"public_key"`
	NotifyURL  string `mapstructure:"notify_url"`
	IsSandbox  bool   `mapstructure:"is_sandbox"`
	LogoURL    string `mapstructure:"logo_url"`
}

type AlipayHKConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	AppID      string `mapstructure:"app_id"`
	PrivateKey string `mapstructure:"private_key"`
	PublicKey  string `mapstructure:"public_key"`
	NotifyURL  string `mapstructure:"notify_url"`
	IsSandbox  bool   `mapstructure:"is_sandbox"`
	LogoURL    string `mapstructure:"logo_url"`
}

type WeChatPayConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	AppID    string `mapstructure:"app_id"`
	MchID    string `mapstructure:"mch_id"`
	APIKey   string `mapstructure:"api_key"`
	CertPath string `mapstructure:"cert_path"`
	NotifyURL string `mapstructure:"notify_url"`
	IsSandbox bool  `mapstructure:"is_sandbox"`
	LogoURL   string `mapstructure:"logo_url"`
}

type PayPalConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	Mode         string `mapstructure:"mode"`
	NotifyURL    string `mapstructure:"notify_url"`
	LogoURL      string `mapstructure:"logo_url"`
}

type WorldFirstConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	MerchantID  string `mapstructure:"merchant_id"`
	APIKey      string `mapstructure:"api_key"`
	APISecret   string `mapstructure:"api_secret"`
	NotifyURL   string `mapstructure:"notify_url"`
	LogoURL     string `mapstructure:"logo_url"`
}

type PayoneerConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	MerchantID  string `mapstructure:"merchant_id"`
	APIKey      string `mapstructure:"api_key"`
	APISecret   string `mapstructure:"api_secret"`
	NotifyURL   string `mapstructure:"notify_url"`
	LogoURL     string `mapstructure:"logo_url"`
}

type WiseConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	APIKey        string `mapstructure:"api_key"`
	ProfileID     string `mapstructure:"profile_id"`
	WebhookSecret string `mapstructure:"webhook_secret"`
	NotifyURL     string `mapstructure:"notify_url"`
	LogoURL       string `mapstructure:"logo_url"`
}

type StripeConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	PublishableKey string `mapstructure:"publishable_key"`
	SecretKey      string `mapstructure:"secret_key"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	NotifyURL      string `mapstructure:"notify_url"`
	LogoURL        string `mapstructure:"logo_url"`
}

// VerificationConfig 验证码配置
type VerificationConfig struct {
	SMS   SMSVerificationConfig   `mapstructure:"sms"`
	Email EmailVerificationConfig `mapstructure:"email"`
}

type SMSVerificationConfig struct {
	Provider       string              `mapstructure:"provider"`
	Aliyun         AliyunSMSConfig     `mapstructure:"aliyun"`
	Tencent        TencentSMSConfig    `mapstructure:"tencent"`
	Twilio         TwilioSMSConfig     `mapstructure:"twilio"`
	Vonage         VonageSMSConfig    `mapstructure:"vonage"`
	CodeLength     int                 `mapstructure:"code_length"`
	ExpireMinutes  int                 `mapstructure:"expire_minutes"`
	CooldownSeconds int               `mapstructure:"cooldown_seconds"`
	MaxSendPerHour int                 `mapstructure:"max_send_per_hour"`
	MaxVerifyAttempts int              `mapstructure:"max_verify_attempts"`
}

type AliyunSMSConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	SignName        string `mapstructure:"sign_name"`
	TemplateCode    string `mapstructure:"template_code"`
}

type TencentSMSConfig struct {
	SecretID   string `mapstructure:"secret_id"`
	SecretKey  string `mapstructure:"secret_key"`
	SDKAppID   string `mapstructure:"sdk_app_id"`
	SignName   string `mapstructure:"sign_name"`
	TemplateID string `mapstructure:"template_id"`
}

type TwilioSMSConfig struct {
	AccountSID string `mapstructure:"account_sid"`
	AuthToken  string `mapstructure:"auth_token"`
	FromNumber string `mapstructure:"from_number"`
}

type VonageSMSConfig struct {
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	FromName  string `mapstructure:"from_name"`
}

type EmailVerificationConfig struct {
	Provider      string            `mapstructure:"provider"`
	SMTP          SMTPConfig        `mapstructure:"smtp"`
	SendGrid      SendGridConfig    `mapstructure:"sendgrid"`
	CodeLength    int               `mapstructure:"code_length"`
	ExpireMinutes int               `mapstructure:"expire_minutes"`
	CooldownSeconds int             `mapstructure:"cooldown_seconds"`
	MaxSendPerHour int              `mapstructure:"max_send_per_hour"`
}

type SMTPConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	FromAddr  string `mapstructure:"from_address"`
	FromName  string `mapstructure:"from_name"`
}

type SendGridConfig struct {
	APIKey    string `mapstructure:"api_key"`
	FromAddr  string `mapstructure:"from_address"`
	FromName  string `mapstructure:"from_name"`
}

// Load 加载配置，支持环境变量覆盖
func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()
	// 环境变量前缀映射，方便 Kubernetes/Docker 部署
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("keyvault.master_key", "KEYVAULT_MASTER_KEY")
	viper.BindEnv("oauth.google_client_id", "GOOGLE_CLIENT_ID")
	viper.BindEnv("oauth.google_client_secret", "GOOGLE_CLIENT_SECRET")
	viper.BindEnv("oauth.github_client_id", "GITHUB_CLIENT_ID")
	viper.BindEnv("oauth.github_client_secret", "GITHUB_CLIENT_SECRET")
	viper.BindEnv("server.frontend_url", "FRONTEND_URL")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 环境变量覆盖关键敏感配置
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("KEYVAULT_MASTER_KEY"); v != "" {
		cfg.KeyVault.MasterKey = v
	}

	return &cfg, nil
}

// Validate 校验关键配置项，返回警告和错误列表
func (c *Config) Validate() (warnings []string, errors []string) {
	// JWT Secret 必须设置且足够长
	if c.JWT.Secret == "" {
		errors = append(errors, "jwt.secret is required (set JWT_SECRET env var)")
	} else if len(c.JWT.Secret) < 32 {
		warnings = append(warnings, "jwt.secret should be at least 32 characters for security")
	}

	// KeyVault 主密钥检查
	if c.KeyVault.MasterKey == "" {
		warnings = append(warnings, "keyvault.master_key is empty (set KEYVAULT_MASTER_KEY env var); encrypted key storage will be disabled")
	} else if len(c.KeyVault.MasterKey) < 32 {
		errors = append(errors, "keyvault.master_key must be at least 32 bytes")
	}

	// 数据库密码检查
	if c.Database.Password == "" {
		warnings = append(warnings, "database.password is empty; not recommended for production")
	}

	// Redis 密码检查
	if c.Redis.Password == "" {
		warnings = append(warnings, "redis.password is empty; not recommended for production")
	}

	// CORS 检查
	if len(c.Security.CORS.AllowedOrigins) == 0 {
		warnings = append(warnings, "security.cors.allowed_origins is empty; CORS will allow all origins (not safe for production)")
	}

	// Admin 密码强度检查（仅当使用默认密码时警告）
	if c.Server.Mode == "release" {
		if c.Database.Password == "tokenhub_secret" {
			errors = append(errors, "database.password is using default value 'tokenhub_secret'; must change in production")
		}
	}

	return warnings, errors
}
