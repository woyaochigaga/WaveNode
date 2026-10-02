package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	dingTalkAppTokenKeyPrefix = "oauth:dingtalk:app_token:"
	dingTalkAppLockKeyPrefix  = "oauth:dingtalk:app_token_lock:"
)

var dingTalkAppTokenLockReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

type dingTalkAppTokenCache struct {
	rdb       *redis.Client
	encryptor service.SecretEncryptor
}

// NewDingTalkAppTokenCache 使用项目统一 AES-GCM 加密器保存 appToken，保证 Redis
// dump、监控和排障输出中不会出现可直接使用的 token 原文。
func NewDingTalkAppTokenCache(rdb *redis.Client, encryptor service.SecretEncryptor) service.DingTalkAppTokenCache {
	return &dingTalkAppTokenCache{rdb: rdb, encryptor: encryptor}
}

func (c *dingTalkAppTokenCache) Get(ctx context.Context, cacheKey string) (string, time.Duration, bool, error) {
	ciphertext, err := c.rdb.Get(ctx, dingTalkAppTokenKeyPrefix+cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	token, err := c.encryptor.Decrypt(ciphertext)
	if err != nil {
		return "", 0, false, err
	}
	ttl, err := c.rdb.TTL(ctx, dingTalkAppTokenKeyPrefix+cacheKey).Result()
	if err != nil {
		return "", 0, false, err
	}
	if ttl <= 0 {
		return "", 0, false, nil
	}
	return token, ttl, true, nil
}

func (c *dingTalkAppTokenCache) Set(ctx context.Context, cacheKey, token string, ttl time.Duration) error {
	ciphertext, err := c.encryptor.Encrypt(token)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, dingTalkAppTokenKeyPrefix+cacheKey, ciphertext, ttl).Err()
}

func (c *dingTalkAppTokenCache) Delete(ctx context.Context, cacheKey string) error {
	return c.rdb.Del(ctx, dingTalkAppTokenKeyPrefix+cacheKey).Err()
}

func (c *dingTalkAppTokenCache) TryAcquireRefresh(ctx context.Context, cacheKey, owner string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, dingTalkAppLockKeyPrefix+cacheKey, owner, ttl).Result()
}

func (c *dingTalkAppTokenCache) ReleaseRefresh(ctx context.Context, cacheKey, owner string) error {
	return dingTalkAppTokenLockReleaseScript.Run(ctx, c.rdb, []string{dingTalkAppLockKeyPrefix + cacheKey}, owner).Err()
}

var _ service.DingTalkAppTokenCache = (*dingTalkAppTokenCache)(nil)
