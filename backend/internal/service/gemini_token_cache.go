package service

import (
	"context"
	"time"
)

// GeminiTokenCache stores short-lived access tokens and coordinates refresh to avoid stampedes.
type GeminiTokenCache interface {
	// cacheKey should be stable for the token scope; for GeminiCli OAuth we primarily use project_id.
	GetAccessToken(ctx context.Context, cacheKey string) (string, error)
	SetAccessToken(ctx context.Context, cacheKey string, token string, ttl time.Duration) error
	DeleteAccessToken(ctx context.Context, cacheKey string) error

	AcquireRefreshLock(ctx context.Context, cacheKey string, ttl time.Duration) (bool, error)
	ReleaseRefreshLock(ctx context.Context, cacheKey string) error
}

// OAuthRefreshLeaseCache 为跨实例刷新提供带所有权的租约。
// owner 是一次刷新操作的唯一 ID；续租和释放只有在 owner 仍匹配时才会生效，
// 避免旧请求在租约过期后误删新请求持有的锁。
type OAuthRefreshLeaseCache interface {
	TryAcquireRefreshLease(ctx context.Context, lockKey, owner string, ttl time.Duration) (bool, error)
	RenewRefreshLease(ctx context.Context, lockKey, owner string, ttl time.Duration) (bool, error)
	ReleaseRefreshLease(ctx context.Context, lockKey, owner string) error
}
