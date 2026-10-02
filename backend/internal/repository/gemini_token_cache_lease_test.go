//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGeminiTokenCache_RefreshLeaseChecksOwner(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &geminiTokenCache{rdb: rdb}
	ctx := context.Background()

	acquired, err := cache.TryAcquireRefreshLease(ctx, "account:42", "operation-a", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)

	acquired, err = cache.TryAcquireRefreshLease(ctx, "account:42", "operation-b", time.Minute)
	require.NoError(t, err)
	require.False(t, acquired)

	renewed, err := cache.RenewRefreshLease(ctx, "account:42", "operation-a", 2*time.Minute)
	require.NoError(t, err)
	require.True(t, renewed)

	// 模拟 A 过期后 B 已接管：A 的迟到续租和释放都不能影响 B。
	key := oauthRefreshLockKeyPrefix + "account:42"
	require.NoError(t, rdb.Set(ctx, key, "operation-b", time.Minute).Err())
	renewed, err = cache.RenewRefreshLease(ctx, "account:42", "operation-a", 2*time.Minute)
	require.NoError(t, err)
	require.False(t, renewed)
	require.NoError(t, cache.ReleaseRefreshLease(ctx, "account:42", "operation-a"))
	owner, err := mr.Get(key)
	require.NoError(t, err)
	require.Equal(t, "operation-b", owner)
}
