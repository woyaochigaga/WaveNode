package service

import (
	"context"
	"time"
)

// DingTalkAppTokenCache 保存钉钉企业 appToken 的加密共享副本，并提供跨实例互斥。
// 实现层必须保证 Redis 中只出现密文，不能保存 token 原文。
type DingTalkAppTokenCache interface {
	Get(ctx context.Context, cacheKey string) (token string, ttl time.Duration, found bool, err error)
	Set(ctx context.Context, cacheKey, token string, ttl time.Duration) error
	Delete(ctx context.Context, cacheKey string) error
	TryAcquireRefresh(ctx context.Context, cacheKey, owner string, ttl time.Duration) (bool, error)
	ReleaseRefresh(ctx context.Context, cacheKey, owner string) error
}
