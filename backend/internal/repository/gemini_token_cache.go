package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/redis/go-redis/v9"
)

const (
	oauthTokenKeyPrefix       = "oauth:token:"
	oauthRefreshLockKeyPrefix = "oauth:refresh_lock:"
)

var (
	// 续租必须校验 owner，防止已经失去租约的旧进程延长新进程的锁。
	oauthRefreshLeaseRenewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)
	// 释放同样采用 compare-and-delete，过期锁的旧持有者不能删除新租约。
	oauthRefreshLeaseReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)
)

type geminiTokenCache struct {
	rdb *redis.Client
}

func NewGeminiTokenCache(rdb *redis.Client) service.GeminiTokenCache {
	return &geminiTokenCache{rdb: rdb}
}

func (c *geminiTokenCache) GetAccessToken(ctx context.Context, cacheKey string) (string, error) {
	key := fmt.Sprintf("%s%s", oauthTokenKeyPrefix, cacheKey)
	return c.rdb.Get(ctx, key).Result()
}

func (c *geminiTokenCache) SetAccessToken(ctx context.Context, cacheKey string, token string, ttl time.Duration) error {
	key := fmt.Sprintf("%s%s", oauthTokenKeyPrefix, cacheKey)
	return c.rdb.Set(ctx, key, token, ttl).Err()
}

func (c *geminiTokenCache) DeleteAccessToken(ctx context.Context, cacheKey string) error {
	key := fmt.Sprintf("%s%s", oauthTokenKeyPrefix, cacheKey)
	return c.rdb.Del(ctx, key).Err()
}

func (c *geminiTokenCache) AcquireRefreshLock(ctx context.Context, cacheKey string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("%s%s", oauthRefreshLockKeyPrefix, cacheKey)
	return c.rdb.SetNX(ctx, key, 1, ttl).Result()
}

func (c *geminiTokenCache) ReleaseRefreshLock(ctx context.Context, cacheKey string) error {
	key := fmt.Sprintf("%s%s", oauthRefreshLockKeyPrefix, cacheKey)
	return c.rdb.Del(ctx, key).Err()
}

func (c *geminiTokenCache) TryAcquireRefreshLease(ctx context.Context, lockKey, owner string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("%s%s", oauthRefreshLockKeyPrefix, lockKey)
	return c.rdb.SetNX(ctx, key, owner, ttl).Result()
}

func (c *geminiTokenCache) RenewRefreshLease(ctx context.Context, lockKey, owner string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("%s%s", oauthRefreshLockKeyPrefix, lockKey)
	result, err := oauthRefreshLeaseRenewScript.Run(ctx, c.rdb, []string{key}, owner, ttl.Milliseconds()).Int64()
	return result == 1, err
}

func (c *geminiTokenCache) ReleaseRefreshLease(ctx context.Context, lockKey, owner string) error {
	key := fmt.Sprintf("%s%s", oauthRefreshLockKeyPrefix, lockKey)
	return oauthRefreshLeaseReleaseScript.Run(ctx, c.rdb, []string{key}, owner).Err()
}

var _ service.OAuthRefreshLeaseCache = (*geminiTokenCache)(nil)
