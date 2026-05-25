package keyvault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// KeyVault 上游API密钥安全存储
// 密钥使用 AES-256-GCM 加密后存储，运行时解密
type KeyVault struct {
	logger    *zap.Logger
	rdb       *redis.Client
	masterKey []byte // 主加密密钥，从环境变量加载
	cache     sync.Map
}

// NewKeyVault 创建密钥保险库
func NewKeyVault(logger *zap.Logger, rdb *redis.Client, masterKey string) *KeyVault {
	// 确保 masterKey 是 32 字节（AES-256）
	mk := []byte(masterKey)
	if len(mk) < 32 {
		padded := make([]byte, 32)
		copy(padded, mk)
		mk = padded
	}
	if len(mk) > 32 {
		mk = mk[:32]
	}

	return &KeyVault{
		logger:    logger,
		rdb:       rdb,
		masterKey: mk,
	}
}

// StoreKey 存储加密后的API密钥
func (kv *KeyVault) StoreKey(ctx context.Context, keyID, plaintextKey string) error {
	encrypted, err := kv.encrypt(plaintextKey)
	if err != nil {
		return fmt.Errorf("encrypt key: %w", err)
	}

	// 存储到 Redis（加密后的值）
	storageKey := fmt.Sprintf("keyvault:%s", keyID)
	if err := kv.rdb.Set(ctx, storageKey, encrypted, 0).Err(); err != nil {
		return fmt.Errorf("store encrypted key: %w", err)
	}

	// 清除缓存
	kv.cache.Delete(keyID)

	kv.logger.Info("key stored securely",
		zap.String("key_id", keyID),
	)
	return nil
}

// GetKey 获取解密后的API密钥
func (kv *KeyVault) GetKey(ctx context.Context, keyID string) (string, error) {
	// 先检查内存缓存
	if cached, ok := kv.cache.Load(keyID); ok {
		return cached.(string), nil
	}

	// 从 Redis 读取加密后的值
	storageKey := fmt.Sprintf("keyvault:%s", keyID)
	encrypted, err := kv.rdb.Get(ctx, storageKey).Result()
	if err != nil {
		return "", fmt.Errorf("get encrypted key: %w", err)
	}

	// 解密
	plaintext, err := kv.decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt key: %w", err)
	}

	// 缓存到内存（短时间，5分钟）
	kv.cache.Store(keyID, plaintext)

	return plaintext, nil
}

// DeleteKey 删除API密钥
func (kv *KeyVault) DeleteKey(ctx context.Context, keyID string) error {
	storageKey := fmt.Sprintf("keyvault:%s", keyID)
	if err := kv.rdb.Del(ctx, storageKey).Err(); err != nil {
		return err
	}
	kv.cache.Delete(keyID)
	return nil
}

// RotateKey 轮换API密钥
func (kv *KeyVault) RotateKey(ctx context.Context, keyID, newPlaintextKey string) error {
	return kv.StoreKey(ctx, keyID, newPlaintextKey)
}

// ListKeyIDs 列出所有密钥ID
func (kv *KeyVault) ListKeyIDs(ctx context.Context, prefix string) ([]string, error) {
	pattern := fmt.Sprintf("keyvault:%s*", prefix)
	keys, err := kv.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, k := range keys {
		ids = append(ids, k[len("keyvault:"):])
	}
	return ids, nil
}

// ==================== 加密/解密 ====================

// encrypt 使用 AES-256-GCM 加密
func (kv *KeyVault) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(kv.masterKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 使用 AES-256-GCM 解密
func (kv *KeyVault) decrypt(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(kv.masterKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
