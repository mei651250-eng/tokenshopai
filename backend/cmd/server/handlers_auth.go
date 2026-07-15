package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tokenhub/backend/internal/auth"
	"github.com/tokenhub/backend/internal/config"
	"github.com/tokenhub/backend/internal/subscription"
	"github.com/tokenhub/backend/internal/tenant"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func handleLogin(c *gin.Context, jwtManager *auth.JWTManager, db *gorm.DB, logger *zap.Logger) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从数据库查询用户
	var user auth.User
	if err := db.Where("email = ? AND status = ?", req.Email, auth.UserActive).First(&user).Error; err != nil {
		logger.Warn("login failed: user not found or disabled", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		logger.Warn("login failed: wrong password", zap.String("email", req.Email))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// 更新最后登录时间
	now := time.Now()
	db.Model(&user).Update("last_login_at", &now)

	// 生成JWT
	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, user.TenantID, user.Role, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenPair,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

func handleRegister(c *gin.Context, db *gorm.DB, subSvc *subscription.SubscriptionService, logger *zap.Logger) {
	var req struct {
		Username   string `json:"username"`
		Email      string `json:"email" binding:"required,email"`
		Password   string `json:"password" binding:"required,min=8"`
		TenantID   string `json:"tenant_id"`
		InviteCode string `json:"invite_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查邮箱是否已注册
	var count int64
	db.Model(&auth.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 确定租户ID
	tenantID := req.TenantID
	if tenantID == "" {
		// 分配到默认租户
		var defaultTenant tenant.Tenant
		if err := db.Where("slug = ?", "default").First(&defaultTenant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "default tenant not found"})
			return
		}
		tenantID = defaultTenant.ID
	}

	// 创建用户
	username := req.Username
	if username == "" {
		username = strings.Split(req.Email, "@")[0]
	}
	user := &auth.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Username:     username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         auth.RoleDeveloper,
		Status:       auth.UserActive,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
	}

	if err := db.Create(user).Error; err != nil {
		logger.Error("failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// 处理邀请码
	if req.InviteCode != "" {
		var inviter auth.User
		if err := db.Where("id = ?", strings.TrimPrefix(req.InviteCode, "INV-")).First(&inviter).Error; err == nil {
			// 找到邀请人，创建邀请记录
			db.Create(&ReferralInvite{
				ID:           uuid.New().String(),
				InviterID:    inviter.ID,
				InviteeID:    user.ID,
				InviteCode:   req.InviteCode,
				RewardAmount: 5.0, // 默认奖励 5 元
				Status:       "pending",
				CreatedAt:    time.Now(),
			})
			logger.Info("referral recorded", zap.String("inviter", inviter.ID), zap.String("invitee", user.ID))
		}
	}

	// 自动开通免费试用
	if subSvc != nil {
		trialPlan, err := subSvc.GetDefaultPlan(c.Request.Context(), subscription.PlanTrial)
		if err == nil && trialPlan != nil {
			if _, err := subSvc.Subscribe(c.Request.Context(), user.ID, user.TenantID, trialPlan.ID); err != nil {
				logger.Warn("auto trial subscription failed", zap.String("user_id", user.ID), zap.Error(err))
			} else {
				logger.Info("auto trial assigned", zap.String("user_id", user.ID))
			}
		}
	}

	logger.Info("user registered", zap.String("email", req.Email))
	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"tenant_id": user.TenantID,
		},
	})
}

// handleSendVerificationCode 发送验证码（短信/邮箱）
func handleSendVerificationCode(c *gin.Context, svc *auth.VerificationService, logger *zap.Logger) {
	var req auth.SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.SendVerificationCode(c.Request.Context(), &req); err != nil {
		logger.Warn("send verification code failed", zap.Error(err))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification code sent"})
}

// handleLoginByCode 验证码登录
func handleLoginByCode(c *gin.Context, svc *auth.VerificationService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req auth.LoginByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, userInfo, err := svc.LoginOrRegisterByCode(c.Request.Context(), &req, jwtManager)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":  tokenPair,
		"user":   userInfo,
		"is_new": true, // 简化处理
	})
}

// handleRegisterByCode 验证码注册
func handleRegisterByCode(c *gin.Context, svc *auth.VerificationService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req auth.RegisterByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loginReq := &auth.LoginByCodeRequest{
		Type:        req.Type,
		Target:      req.Target,
		Code:        req.Code,
		CountryCode: req.CountryCode,
	}

	tokenPair, userInfo, err := svc.LoginOrRegisterByCode(c.Request.Context(), loginReq, jwtManager)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": tokenPair,
		"user":  userInfo,
	})
}

// ==================== WebAuthn 人脸识别 ====================

// handleFaceRegisterOptions 获取人脸注册选项（需JWT认证）
func handleFaceRegisterOptions(c *gin.Context, faceSvc *auth.FaceAuthService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	options, err := faceSvc.GenerateRegistrationOptions(c.Request.Context(), userID)
	if err != nil {
		logger.Error("failed to generate face registration options", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, options)
}

// handleFaceRegisterVerify 验证人脸注册
func handleFaceRegisterVerify(c *gin.Context, faceSvc *auth.FaceAuthService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req struct {
		SessionKey string                 `json:"session_key"`
		Credential map[string]interface{} `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credential, err := faceSvc.VerifyRegistration(c.Request.Context(), req.SessionKey, req.Credential)
	if err != nil {
		logger.Warn("face registration verification failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "face credential registered successfully",
		"credential": credential,
	})
}

// handleFaceAuthOptions 获取人脸认证选项（无需JWT，但需提供邮箱）
func handleFaceAuthOptions(c *gin.Context, faceSvc *auth.FaceAuthService, logger *zap.Logger) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	options, err := faceSvc.GenerateAuthenticationOptions(c.Request.Context(), req.Email)
	if err != nil {
		logger.Warn("failed to generate face auth options", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, options)
}

// handleFaceAuthVerify 验证人脸认证
func handleFaceAuthVerify(c *gin.Context, faceSvc *auth.FaceAuthService, jwtManager *auth.JWTManager, logger *zap.Logger) {
	var req struct {
		SessionKey string                 `json:"session_key"`
		Credential map[string]interface{} `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := faceSvc.VerifyAuthentication(c.Request.Context(), req.SessionKey, req.Credential)
	if err != nil {
		logger.Warn("face authentication failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 生成 JWT
	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, user.TenantID, user.Role, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenPair,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"tenant_id": user.TenantID,
		},
	})
}

// handleOAuthRedirect 发起 OAuth 重定向（GitHub / Google）
func handleOAuthRedirect(c *gin.Context, cfg *config.Config, logger *zap.Logger) {
	provider := c.Param("provider")
	var authURL, clientID string
	state := generateSecureRandom(16)

	switch provider {
	case "github":
		clientID = cfg.OAuth.GitHubClientID
		if clientID == "" {
			clientID = os.Getenv("GITHUB_CLIENT_ID")
		}
		if clientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub OAuth not configured"})
			return
		}
		authURL = fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&state=%s&scope=user:email", clientID, state)
	case "google":
		clientID = cfg.OAuth.GoogleClientID
		if clientID == "" {
			clientID = os.Getenv("GOOGLE_CLIENT_ID")
		}
		if clientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Google OAuth not configured"})
			return
		}
		redirectURI := getOAuthRedirectBase(cfg, c.Request.Host) + "/auth/oauth/google/callback"
		authURL = fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&state=%s&scope=openid+email+profile&response_type=code", clientID, redirectURI, state)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// 将 state 存入 Redis 供回调校验（简化：直接用 cookie）
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// handleOAuthCallback 处理 OAuth 回调
func handleOAuthCallback(c *gin.Context, jwtManager *auth.JWTManager, db *gorm.DB, subSvc *subscription.SubscriptionService, logger *zap.Logger, cfg *config.Config) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	// 校验 state
	savedState, _ := c.Cookie("oauth_state")
	if state == "" || state != savedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	var (
		accessToken string
		email       string
		name        string
		avatarURL   string
		providerID  string
	)

	switch provider {
	case "github":
		var err error
		accessToken, email, name, avatarURL, providerID, err = exchangeGitHubCode(code, cfg)
		if err != nil {
			logger.Error("github oauth failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "google":
		var err error
		accessToken, email, name, avatarURL, providerID, err = exchangeGoogleCode(code, c.Request.Host, cfg)
		if err != nil {
			logger.Error("google oauth failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// 查找或创建用户
	var oauthAcc OAuthAccount
	var user auth.User

	if err := db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&oauthAcc).Error; err == nil {
		// 已绑定，直接登录
		db.First(&user, "id = ?", oauthAcc.UserID)
	} else {
		// 未绑定，按邮箱查找或创建
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			// 创建新用户
			var defaultTenant tenant.Tenant
			if err := db.Where("slug = ?", "default").First(&defaultTenant).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "default tenant not found"})
				return
			}
			user = auth.User{
				ID:           uuid.New().String(),
				TenantID:     defaultTenant.ID,
				Username:     name,
				Email:        email,
				PasswordHash: "", // OAuth 用户无密码
				Role:         auth.RoleDeveloper,
				Status:       auth.UserActive,
				Language:     "zh-CN",
				Timezone:     "Asia/Shanghai",
			}
			if err := db.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
			// 自动开通免费试用
			if subSvc != nil {
				trialPlan, err := subSvc.GetDefaultPlan(c.Request.Context(), subscription.PlanTrial)
				if err == nil && trialPlan != nil {
					subSvc.Subscribe(c.Request.Context(), user.ID, user.TenantID, trialPlan.ID)
				}
			}
		}

		// 绑定 OAuth
		db.Create(&OAuthAccount{
			ID:          uuid.New().String(),
			UserID:      user.ID,
			Provider:    provider,
			ProviderID:  providerID,
			AccessToken: accessToken,
			AvatarURL:   avatarURL,
			DisplayName: name,
			CreatedAt:   time.Now(),
		})
	}

	// 生成 JWT
	tokenPair, err := jwtManager.GenerateTokenPair(user.ID, user.TenantID, user.Role, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重定向前端并携带 token
	frontendURL := cfg.Server.FrontendURL
	if frontendURL == "" {
		frontendURL = os.Getenv("FRONTEND_URL")
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3001"
	}
	redirectURL := fmt.Sprintf("%s/#/dashboard?token=%s&refresh_token=%s", frontendURL, tokenPair.AccessToken, tokenPair.RefreshToken)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// exchangeGitHubCode 用 code 换取 GitHub 用户信息
func exchangeGitHubCode(code string, cfg *config.Config) (accessToken, email, name, avatarURL, providerID string, err error) {
	clientID := cfg.OAuth.GitHubClientID
	if clientID == "" {
		clientID = os.Getenv("GITHUB_CLIENT_ID")
	}
	clientSecret := cfg.OAuth.GitHubClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	// 换取 access_token
	resp, err := http.Post(
		"https://github.com/login/oauth/access_token?client_id="+clientID+"&client_secret="+clientSecret+"&code="+code,
		"application/json",
		nil,
	)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	accessToken, _ = tokenResp["access_token"].(string)

	// 获取用户信息
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp2.Body.Close()

	var userInfo map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&userInfo)
	name, _ = userInfo["name"].(string)
	if name == "" {
		name, _ = userInfo["login"].(string)
	}
	avatarURL, _ = userInfo["avatar_url"].(string)
	pid, _ := userInfo["id"].(float64)
	providerID = fmt.Sprintf("%.0f", pid)

	// 获取邮箱
	req2, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	resp3, err := http.DefaultClient.Do(req2)
	if err == nil {
		defer resp3.Body.Close()
		var emails []map[string]interface{}
		json.NewDecoder(resp3.Body).Decode(&emails)
		for _, e := range emails {
			if primary, _ := e["primary"].(bool); primary {
				email, _ = e["email"].(string)
				break
			}
		}
	}
	return
}

// getOAuthRedirectBase 获取 OAuth 回调的基础 URL
// 优先使用配置的 FRONTEND_URL，确保与 Google/GitHub Console 中配置的回调地址一致
func getOAuthRedirectBase(cfg *config.Config, requestHost string) string {
	// 优先使用环境变量或配置文件中的 FRONTEND_URL
	frontendURL := cfg.Server.FrontendURL
	if frontendURL == "" {
		frontendURL = os.Getenv("FRONTEND_URL")
	}
	if frontendURL != "" {
		// 去掉尾部的 / （如果有的话）
		frontendURL = strings.TrimRight(frontendURL, "/")
		return frontendURL
	}

	// 开发模式：使用请求的 Host
	scheme := "http"
	if requestHost != "" && !strings.Contains(requestHost, "localhost") && !strings.Contains(requestHost, "127.0.0.1") {
		scheme = "https"
	}
	return scheme + "://" + requestHost
}

// exchangeGoogleCode 用 code 换取 Google 用户信息
func exchangeGoogleCode(code, host string, cfg *config.Config) (accessToken, email, name, avatarURL, providerID string, err error) {
	clientID := cfg.OAuth.GoogleClientID
	if clientID == "" {
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	clientSecret := cfg.OAuth.GoogleClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	redirectURI := getOAuthRedirectBase(cfg, host) + "/auth/oauth/google/callback"

	// 换取 token
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", map[string][]string{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	accessToken, _ = tokenResp["access_token"].(string)
	idToken, _ := tokenResp["id_token"].(string)
	_ = idToken

	// 获取用户信息
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp2.Body.Close()

	var userInfo map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&userInfo)
	email, _ = userInfo["email"].(string)
	name, _ = userInfo["name"].(string)
	avatarURL, _ = userInfo["picture"].(string)
	pid, _ := userInfo["id"].(string)
	providerID = pid
	return
}
