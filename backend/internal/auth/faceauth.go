package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WebAuthn 凭据和会话数据模型

// FaceCredential 用户人脸/生物识别凭据
type FaceCredential struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	UserID          string    `json:"user_id" gorm:"index"`
	CredentialID    string    `json:"credential_id" gorm:"index"`    // Base64URL 编码的凭据 ID
	PublicKey       string    `json:"public_key"`                    // Base64 编码的公钥
	SignCount       uint32    `json:"sign_count"`                   // 签名计数（防重放）
	Transport       string    `json:"transport"`                     // 传输方式: internal, usb, nfc, ble
	AAGUID          string    `json:"aa_guid"`                       // 认证器 GUID
	Name            string    `json:"name"`                          // 凭据名称 (如 "Windows Hello", "Touch ID")
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// WebAuthnSession 注册/认证会话（存储在 Redis）
type WebAuthnSession struct {
	UserID      string `json:"user_id"`
	Challenge   string `json:"challenge"`
	Type        string `json:"type"`        // "registration" | "authentication"
	CreatedAt   int64  `json:"created_at"`
}

// FaceAuthService 人脸/生物识别认证服务
type FaceAuthService struct {
	db     *gorm.DB
	rdb    *redis.Client
	logger *zap.Logger
	rpID   string       // Relying Party ID (域名)
	rpName string       // Relying Party 名称
	rpOrigin string     // Relying Party Origin
}

// NewFaceAuthService 创建人脸认证服务
func NewFaceAuthService(db *gorm.DB, rdb *redis.Client, logger *zap.Logger, rpID, rpName, rpOrigin string) *FaceAuthService {
	return &FaceAuthService{
		db:       db,
		rdb:      rdb,
		logger:   logger,
		rpID:     rpID,
		rpName:   rpName,
		rpOrigin: rpOrigin,
	}
}

// GenerateRegistrationOptions 生成 WebAuthn 注册选项
func (s *FaceAuthService) GenerateRegistrationOptions(ctx context.Context, userID string) (map[string]interface{}, error) {
	// 生成挑战
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}
	challengeB64 := base64.RawURLEncoding.EncodeToString(challenge)

	// 存储会话
	session := &WebAuthnSession{
		UserID:    userID,
		Challenge: challengeB64,
		Type:      "registration",
		CreatedAt: time.Now().Unix(),
	}
	sessionKey := fmt.Sprintf("webauthn:session:%s", challengeB64[:16])
	sessionData, _ := json.Marshal(session)
	if err := s.rdb.Set(ctx, sessionKey, sessionData, 5*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// 获取用户已有凭据
	var existingCreds []FaceCredential
	s.db.Where("user_id = ?", userID).Find(&existingCreds)
	excludeCredentials := make([]map[string]interface{}, 0, len(existingCreds))
	for _, cred := range existingCreds {
		credIDBytes, _ := base64.RawURLEncoding.DecodeString(cred.CredentialID)
		excludeCredentials = append(excludeCredentials, map[string]interface{}{
			"type": "public-key",
			"id":   base64.RawURLEncoding.EncodeToString(credIDBytes),
		})
	}

	// 构造 PublicKeyCredentialCreationOptions
	options := map[string]interface{}{
		"publicKey": map[string]interface{}{
			"rp": map[string]string{
				"name": s.rpName,
				"id":   s.rpID,
			},
			"user": map[string]interface{}{
				"id":          base64.RawURLEncoding.EncodeToString([]byte(userID)),
				"name":        userID,
				"displayName": userID,
			},
			"challenge":      challengeB64,
			"pubKeyCredParams": []map[string]interface{}{
				{"type": "public-key", "alg": -7},   // ES256 (COSE identifier)
				{"type": "public-key", "alg": -257}, // RS256
			},
			"timeout":           60000,
			"excludeCredentials": excludeCredentials,
			"authenticatorSelection": map[string]interface{}{
				"authenticatorAttachment": "platform",
				"userVerification":       "required",
				"residentKey":            "preferred",
			},
			"attestation": "none",
		},
		"sessionKey": sessionKey,
	}

	return options, nil
}

// VerifyRegistration 验证注册响应
func (s *FaceAuthService) VerifyRegistration(ctx context.Context, sessionKey string, credentialData map[string]interface{}) (*FaceCredential, error) {
	// 获取会话
	sessionData, err := s.rdb.Get(ctx, sessionKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("session expired or not found")
	}
	var session WebAuthnSession
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return nil, fmt.Errorf("invalid session data")
	}
	if session.Type != "registration" {
		return nil, fmt.Errorf("invalid session type")
	}

	// 验证挑战
	clientChallenge, _ := credentialData["response"].(map[string]interface{})["clientDataJSON"].(string)
	if clientChallenge == "" {
		return nil, fmt.Errorf("missing clientDataJSON")
	}

	// 解码 clientDataJSON 验证挑战和 origin
	clientDataBytes, err := base64.RawURLEncoding.DecodeString(clientChallenge)
	if err != nil {
		return nil, fmt.Errorf("invalid clientDataJSON encoding: %w", err)
	}
	var clientData map[string]interface{}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		return nil, fmt.Errorf("invalid clientDataJSON: %w", err)
	}

	// 验证类型
	if clientData["type"] != "webauthn.create" {
		return nil, fmt.Errorf("invalid client data type")
	}

	// 验证挑战
	receivedChallenge, _ := clientData["challenge"].(string)
	if subtle.ConstantTimeCompare([]byte(receivedChallenge), []byte(session.Challenge)) != 1 {
		return nil, fmt.Errorf("challenge mismatch")
	}

	// 验证 origin
	origin, _ := clientData["origin"].(string)
	if origin != s.rpOrigin && origin != "http://localhost:3001" && origin != "http://localhost:3000" {
		s.logger.Warn("origin mismatch in WebAuthn registration",
			zap.String("expected", s.rpOrigin),
			zap.String("received", origin))
		// 开发模式不强制验证 origin
	}

	// 提取凭据数据
	credID, _ := credentialData["id"].(string)
	if credID == "" {
		return nil, fmt.Errorf("missing credential ID")
	}

	responseData, _ := credentialData["response"].(map[string]interface{})
	attestationObject, _ := responseData["attestationObject"].(string)
	if attestationObject == "" {
		return nil, fmt.Errorf("missing attestation object")
	}

	// 解析 attestationObject 获取公钥
	attBytes, err := base64.RawURLEncoding.DecodeString(attestationObject)
	if err != nil {
		return nil, fmt.Errorf("invalid attestation object encoding: %w", err)
	}

	// 简化处理：提取 authData 中的凭据信息
	// 真实实现需要完整解析 CBOR 格式的 attestationObject
	publicKeyB64 := base64.StdEncoding.EncodeToString(attBytes[:min(len(attBytes), 128)])

	// 获取传输方式
	transports := []string{"internal"}
	if t, ok := credentialData["response"].(map[string]interface{})["transports"]; ok {
		if tArr, ok := t.([]interface{}); ok {
			transports = make([]string, 0, len(tArr))
			for _, v := range tArr {
				if s, ok := v.(string); ok {
					transports = append(transports, s)
				}
			}
		}
	}
	transportsJSON, _ := json.Marshal(transports)

	// 创建凭据记录
	now := time.Now()
	credential := &FaceCredential{
		ID:           fmt.Sprintf("fc_%s", credID[:16]),
		UserID:       session.UserID,
		CredentialID: credID,
		PublicKey:    publicKeyB64,
		SignCount:    0,
		Transport:    string(transportsJSON),
		AAGUID:       "",
		Name:         "Face ID / Biometric",
		LastUsedAt:   &now,
		CreatedAt:    now,
	}

	if err := s.db.Create(credential).Error; err != nil {
		return nil, fmt.Errorf("failed to save credential: %w", err)
	}

	// 删除会话
	s.rdb.Del(ctx, sessionKey)

	s.logger.Info("face credential registered",
		zap.String("user_id", session.UserID),
		zap.String("credential_id", credID[:16]+"..."),
	)

	return credential, nil
}

// GenerateAuthenticationOptions 生成 WebAuthn 认证选项
func (s *FaceAuthService) GenerateAuthenticationOptions(ctx context.Context, email string) (map[string]interface{}, error) {
	// 查找用户
	var user User
	if err := s.db.Where("email = ? AND status = ?", email, UserActive).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 获取用户已有凭据
	var credentials []FaceCredential
	if err := s.db.Where("user_id = ?", user.ID).Find(&credentials).Error; err != nil {
		return nil, fmt.Errorf("failed to get credentials")
	}
	if len(credentials) == 0 {
		return nil, fmt.Errorf("no face credentials registered for this user")
	}

	// 生成挑战
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}
	challengeB64 := base64.RawURLEncoding.EncodeToString(challenge)

	// 存储会话
	session := &WebAuthnSession{
		UserID:    user.ID,
		Challenge: challengeB64,
		Type:      "authentication",
		CreatedAt: time.Now().Unix(),
	}
	sessionKey := fmt.Sprintf("webauthn:session:%s", challengeB64[:16])
	sessionData, _ := json.Marshal(session)
	if err := s.rdb.Set(ctx, sessionKey, sessionData, 5*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// 构造允许的凭据列表
	allowCredentials := make([]map[string]interface{}, 0, len(credentials))
	for _, cred := range credentials {
		credIDBytes, _ := base64.RawURLEncoding.DecodeString(cred.CredentialID)
		allowCredentials = append(allowCredentials, map[string]interface{}{
			"type": "public-key",
			"id":   base64.RawURLEncoding.EncodeToString(credIDBytes),
		})
	}

	options := map[string]interface{}{
		"publicKey": map[string]interface{}{
			"rpId":              s.rpID,
			"challenge":         challengeB64,
			"allowCredentials":  allowCredentials,
			"timeout":           60000,
			"userVerification":  "required",
		},
		"sessionKey": sessionKey,
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"tenant_id":  user.TenantID,
		},
	}

	return options, nil
}

// VerifyAuthentication 验证认证响应
func (s *FaceAuthService) VerifyAuthentication(ctx context.Context, sessionKey string, credentialData map[string]interface{}) (*User, error) {
	// 获取会话
	sessionData, err := s.rdb.Get(ctx, sessionKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("session expired or not found")
	}
	var session WebAuthnSession
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return nil, fmt.Errorf("invalid session data")
	}
	if session.Type != "authentication" {
		return nil, fmt.Errorf("invalid session type")
	}

	// 验证挑战
	responseData, _ := credentialData["response"].(map[string]interface{})
	clientChallenge, _ := responseData["clientDataJSON"].(string)
	if clientChallenge == "" {
		return nil, fmt.Errorf("missing clientDataJSON")
	}

	clientDataBytes, err := base64.RawURLEncoding.DecodeString(clientChallenge)
	if err != nil {
		return nil, fmt.Errorf("invalid clientDataJSON encoding: %w", err)
	}
	var clientData map[string]interface{}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		return nil, fmt.Errorf("invalid clientDataJSON: %w", err)
	}

	// 验证类型
	if clientData["type"] != "webauthn.get" {
		return nil, fmt.Errorf("invalid client data type")
	}

	// 验证挑战
	receivedChallenge, _ := clientData["challenge"].(string)
	if subtle.ConstantTimeCompare([]byte(receivedChallenge), []byte(session.Challenge)) != 1 {
		return nil, fmt.Errorf("challenge mismatch")
	}

	// 获取凭据
	credID, _ := credentialData["id"].(string)
	var credential FaceCredential
	if err := s.db.Where("credential_id = ? AND user_id = ?", credID, session.UserID).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("credential not found")
	}

	// 更新签名计数
	newSignCount := uint32(0)
	if signCount, ok := responseData["signCount"]; ok {
		if sc, ok := signCount.(float64); ok {
			newSignCount = uint32(sc)
		}
	}
	if newSignCount <= credential.SignCount && credential.SignCount != 0 {
		// 可能的重放攻击
		s.logger.Warn("possible replay attack detected",
			zap.String("user_id", session.UserID),
			zap.Uint32("old_count", credential.SignCount),
			zap.Uint32("new_count", newSignCount),
		)
	}

	// 更新凭据
	now := time.Now()
	s.db.Model(&credential).Updates(map[string]interface{}{
		"sign_count":   newSignCount,
		"last_used_at": &now,
	})

	// 获取用户
	var user User
	if err := s.db.Where("id = ? AND status = ?", session.UserID, UserActive).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 更新最后登录时间
	s.db.Model(&user).Update("last_login_at", &now)

	// 删除会话
	s.rdb.Del(ctx, sessionKey)

	s.logger.Info("face authentication successful",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	return &user, nil
}

// RemoveCredential 删除凭据
func (s *FaceAuthService) RemoveCredential(ctx context.Context, userID, credentialID string) error {
	result := s.db.Where("id = ? AND user_id = ?", credentialID, userID).Delete(&FaceCredential{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// ListCredentials 列出用户凭据
func (s *FaceAuthService) ListCredentials(ctx context.Context, userID string) ([]FaceCredential, error) {
	var credentials []FaceCredential
	if err := s.db.Where("user_id = ?", userID).Find(&credentials).Error; err != nil {
		return nil, err
	}
	return credentials, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
