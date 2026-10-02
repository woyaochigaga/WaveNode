//go:build unit

package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type dingTalkCacheTestEncryptor struct{}

func (dingTalkCacheTestEncryptor) Encrypt(plaintext string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte("cipher:" + plaintext)), nil
}

func (dingTalkCacheTestEncryptor) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	value := string(raw)
	if !strings.HasPrefix(value, "cipher:") {
		return "", fmt.Errorf("invalid ciphertext")
	}
	return strings.TrimPrefix(value, "cipher:"), nil
}

func TestDingTalkAppTokenCache_StoresCiphertextAndProtectsLockOwner(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &dingTalkAppTokenCache{rdb: rdb, encryptor: dingTalkCacheTestEncryptor{}}
	ctx := context.Background()
	const token = "DINGTALK_APP_TOKEN_CANARY"

	require.NoError(t, cache.Set(ctx, "tenant-a", token, time.Minute))
	raw, err := mr.Get(dingTalkAppTokenKeyPrefix + "tenant-a")
	require.NoError(t, err)
	require.NotContains(t, raw, token, "Redis value must never contain the plaintext app token")

	got, ttl, found, err := cache.Get(ctx, "tenant-a")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, token, got)
	require.Greater(t, ttl, time.Duration(0))

	acquired, err := cache.TryAcquireRefresh(ctx, "tenant-a", "operation-a", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, cache.ReleaseRefresh(ctx, "tenant-a", "operation-b"))
	owner, err := mr.Get(dingTalkAppLockKeyPrefix + "tenant-a")
	require.NoError(t, err)
	require.Equal(t, "operation-a", owner)
	require.NoError(t, cache.ReleaseRefresh(ctx, "tenant-a", "operation-a"))
	require.False(t, mr.Exists(dingTalkAppLockKeyPrefix+"tenant-a"))
}
