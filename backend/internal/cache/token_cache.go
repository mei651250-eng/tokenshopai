package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// TokenCache Token缓存层
// 缓存相似请求的响应，降低上游API调用成本
type TokenCache struct {
	logger    *zap.Logger
	rdb       *redis.Client
	ttl       time.Duration
	enabled   bool
	hitRate   float64
	hits      int64
	misses    int64
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled    bool          `json:"enabled"`
	TTL        time.Duration `json:"ttl"`         // 默认缓存时间
	MaxEntries int64         `json:"max_entries"` // 最大缓存条目数
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Key         string `json:"key"`
	Model       string `json:"model"`
	Response    string `json:"response"`
	InputHash   string `json:"input_hash"`
	Tokens      int    `json:"tokens"`      // 缓存的Token数
	SavedCalls  int    `json:"saved_calls"` // 节省的调用次数
	CreatedAt   int64  `json:"created_at"`
	ExpiresAt   int64  `json:"expires_at"`
}

// NewTokenCache 创建Token缓存
func NewTokenCache(logger *zap.Logger, rdb *redis.Client, cfg CacheConfig) *TokenCache {
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	return &TokenCache{
		logger:  logger,
		rdb:     rdb,
		ttl:     ttl,
		enabled: cfg.Enabled,
	}
}

// GenerateKey 生成缓存键
// 基于模型名+消息内容的SHA256哈希
func (tc *TokenCache) GenerateKey(model string, messages []map[string]string, temperature float64) string {
	// 构建缓存键的内容
	content := fmt.Sprintf("%s:%v:%.2f", model, messages, temperature)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Get 获取缓存
func (tc *TokenCache) Get(ctx context.Context, key string) (string, bool) {
	if !tc.enabled {
		tc.misses++
		return "", false
	}

	cacheKey := fmt.Sprintf("token_cache:%s", key)
	data, err := tc.rdb.HGetAll(ctx, cacheKey).Result()
	if err != nil || len(data) == 0 {
		tc.misses++
		tc.updateHitRate()
		return "", false
	}

	// 检查过期
	expiresAt := parseInt64(data["expires_at"])
	if time.Now().Unix() > expiresAt {
		tc.rdb.Del(ctx, cacheKey)
		tc.misses++
		tc.updateHitRate()
		return "", false
	}

	tc.hits++
	// 更新节省调用次数
	tc.rdb.HIncrBy(ctx, cacheKey, "saved_calls", 1)
	tc.updateHitRate()

	tc.logger.Debug("cache hit",
		zap.String("key", key[:16]+"..."),
		zap.Int("saved_calls", parseInt(data["saved_calls"])+1),
	)

	return data["response"], true
}

// Set 设置缓存
func (tc *TokenCache) Set(ctx context.Context, key, model, response string, tokens int) error {
	if !tc.enabled {
		return nil
	}

	now := time.Now().Unix()
	expiresAt := now + int64(tc.ttl.Seconds())

	cacheKey := fmt.Sprintf("token_cache:%s", key)
	if err := tc.rdb.HSet(ctx, cacheKey, map[string]interface{}{
		"key":          key,
		"model":        model,
		"response":     response,
		"tokens":       tokens,
		"saved_calls":  0,
		"created_at":   now,
		"expires_at":   expiresAt,
	}).Err(); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	// 设置过期时间
	tc.rdb.Expire(ctx, cacheKey, tc.ttl)

	// 添加到缓存索引
	tc.rdb.ZAdd(ctx, "token_cache:index", &redis.Z{
		Score:  float64(now),
		Member: key,
	})

	return nil
}

// Invalidate 使缓存失效
func (tc *TokenCache) Invalidate(ctx context.Context, key string) error {
	cacheKey := fmt.Sprintf("token_cache:%s", key)
	tc.rdb.Del(ctx, cacheKey)
	tc.rdb.ZRem(ctx, "token_cache:index", key)
	return nil
}

// ClearAll 清空所有缓存
func (tc *TokenCache) ClearAll(ctx context.Context) error {
	// 获取所有缓存键
	keys, err := tc.rdb.ZRange(ctx, "token_cache:index", 0, -1).Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		cacheKey := fmt.Sprintf("token_cache:%s", key)
		tc.rdb.Del(ctx, cacheKey)
	}

	tc.rdb.Del(ctx, "token_cache:index")
	tc.hits = 0
	tc.misses = 0
	tc.hitRate = 0

	return nil
}

// GetStats 获取缓存统计
func (tc *TokenCache) GetStats(ctx context.Context) map[string]interface{} {
	total := tc.hits + tc.misses

	// 获取缓存条目数
	cacheSize, _ := tc.rdb.ZCard(ctx, "token_cache:index").Result()

	// 计算节省的Token和成本
	var totalSavedTokens int64
	keys, _ := tc.rdb.ZRange(ctx, "token_cache:index", 0, -1).Result()
	for _, key := range keys {
		cacheKey := fmt.Sprintf("token_cache:%s", key)
		data, err := tc.rdb.HGetAll(ctx, cacheKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		savedCalls := parseInt64(data["saved_calls"])
		tokens := parseInt64(data["tokens"])
		totalSavedTokens += savedCalls * tokens
	}

	return map[string]interface{}{
		"enabled":            tc.enabled,
		"cache_size":        cacheSize,
		"hits":              tc.hits,
		"misses":            tc.misses,
		"hit_rate":          tc.hitRate,
		"total_requests":    total,
		"total_saved_tokens": totalSavedTokens,
		"ttl_seconds":       tc.ttl.Seconds(),
	}
}

// CleanupExpired 清理过期缓存
func (tc *TokenCache) CleanupExpired(ctx context.Context) (int, error) {
	now := time.Now().Unix()
	var cleaned int

	keys, err := tc.rdb.ZRange(ctx, "token_cache:index", 0, -1).Result()
	if err != nil {
		return 0, err
	}

	for _, key := range keys {
		cacheKey := fmt.Sprintf("token_cache:%s", key)
		expiresAt, err := tc.rdb.HGet(ctx, cacheKey, "expires_at").Int64()
		if err != nil {
			continue
		}

		if now > expiresAt {
			tc.rdb.Del(ctx, cacheKey)
			tc.rdb.ZRem(ctx, "token_cache:index", key)
			cleaned++
		}
	}

	if cleaned > 0 {
		tc.logger.Info("expired cache entries cleaned",
			zap.Int("cleaned", cleaned),
		)
	}

	return cleaned, nil
}

func (tc *TokenCache) updateHitRate() {
	total := tc.hits + tc.misses
	if total > 0 {
		tc.hitRate = float64(tc.hits) / float64(total)
	}
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
