package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     Role   `json:"role"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secret     []byte
	expire     time.Duration
	refreshExp time.Duration
	issuer     string
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secret string, expire, refreshExp time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		expire:     expire,
		refreshExp: refreshExp,
		issuer:     issuer,
	}
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GenerateTokenPair 生成Token对
func (m *JWTManager) GenerateTokenPair(userID, tenantID string, role Role, email string) (*TokenPair, error) {
	now := time.Now()

	// Access Token
	accessClaims := JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(m.secret)
	if err != nil {
		return nil, err
	}

	// Refresh Token
	refreshClaims := JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExp)),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(m.secret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresIn:    int64(m.expire.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken 验证Token
func (m *JWTManager) ValidateToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
