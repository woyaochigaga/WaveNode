package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGatewayCacheLiveCallIdentityAndController(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache, ok := NewGatewayCache(client).(service.LiveCallStore)
	require.True(t, ok)
	otherInstance, ok := NewGatewayCache(client).(service.LiveCallStore)
	require.True(t, ok)
	record := &service.LiveCallRecord{
		CallID:                       "call_secret",
		CallHash:                     HashLiveCallID("call_secret"),
		AccountID:                    11,
		APIKeyID:                     22,
		UserID:                       33,
		GroupID:                      44,
		LeaseID:                      "lease",
		Model:                        "gpt-live-test",
		AttestationCiphertext:        "encrypted-attestation",
		CreatedAt:                    time.Now(),
		ExpiresAt:                    time.Now().Add(time.Hour),
		Controller:                   service.LiveControllerPending,
		BillingGuardStatus:           service.LiveBillingStatusReserved,
		BillingReservationID:         "live:reservation",
		BillingReservedAmount:        0.25,
		BillingRateMultiplier:        1.2,
		BillingAccountRateMultiplier: 0.8,
		BillingRealtimePricePerMin:   0.1,
		BillingRealtimePriceSet:      true,
		BillingAPIKeyQuota:           10,
		BillingAPIKeyHasRates:        true,
		BillingAccountType:           service.AccountTypeOAuth,
		BillingAccountHasQuota:       true,
		BillingPlatform:              service.PlatformOpenAI,
	}
	require.NoError(t, cache.SaveLiveCall(context.Background(), record, time.Hour))

	loaded, err := otherInstance.GetLiveCall(context.Background(), record.CallHash)
	require.NoError(t, err)
	require.Equal(t, record.CallID, loaded.CallID)
	require.Equal(t, record.AccountID, loaded.AccountID)
	require.Equal(t, record.AttestationCiphertext, loaded.AttestationCiphertext)
	require.Equal(t, record.BillingGuardStatus, loaded.BillingGuardStatus)
	require.Equal(t, record.BillingReservationID, loaded.BillingReservationID)
	require.Equal(t, record.BillingReservedAmount, loaded.BillingReservedAmount)
	require.Equal(t, record.BillingRealtimePriceSet, loaded.BillingRealtimePriceSet)
	require.Equal(t, record.BillingPlatform, loaded.BillingPlatform)

	recoveryStore, ok := cache.(service.LiveCallRecoveryStore)
	require.True(t, ok)
	recoverable, err := recoveryStore.ListLiveCallsForRecovery(context.Background(), time.Now().Add(2*time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, recoverable, 1)
	require.Equal(t, record.CallHash, recoverable[0].CallHash)

	claimed, err := cache.ClaimLiveController(context.Background(), record.CallHash, service.LiveControllerObserver, "observer-1")
	require.NoError(t, err)
	require.True(t, claimed)
	claimed, err = cache.ClaimLiveController(context.Background(), record.CallHash, service.LiveControllerProxy, "proxy-1")
	require.NoError(t, err)
	require.True(t, claimed)
	controller, err := cache.GetLiveController(context.Background(), record.CallHash)
	require.NoError(t, err)
	require.Equal(t, service.LiveControllerProxy, controller)

	released, err := cache.ReleaseLiveController(context.Background(), record.CallHash, "proxy-1")
	require.NoError(t, err)
	require.True(t, released)
	closed, err := cache.MarkLiveCallClosed(context.Background(), record.CallHash, time.Hour)
	require.NoError(t, err)
	require.True(t, closed)
	closed, err = cache.MarkLiveCallClosed(context.Background(), record.CallHash, time.Hour)
	require.NoError(t, err)
	require.False(t, closed)
	recoverable, err = recoveryStore.ListLiveCallsForRecovery(context.Background(), time.Now().Add(2*time.Hour), 10)
	require.NoError(t, err)
	require.Empty(t, recoverable, "关闭后的会话必须从恢复索引移除")
}
