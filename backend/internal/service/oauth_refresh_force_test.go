//go:build unit

package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type forceRefreshRepo struct {
	refreshAPIAccountRepo
	mu sync.Mutex
}

func (r *forceRefreshRepo) GetByID(_ context.Context, _ int64) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return snapshotOAuthRefreshAccount(r.account), nil
}

func (r *forceRefreshRepo) UpdateOAuthCredentialsIfUnchanged(
	_ context.Context,
	id int64,
	expectedCredentials map[string]any,
	expectedProxyID *int64,
	credentials map[string]any,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.account == nil || r.account.ID != id ||
		!reflect.DeepEqual(r.account.Credentials, expectedCredentials) ||
		!reflect.DeepEqual(r.account.ProxyID, expectedProxyID) {
		return false, nil
	}
	r.account.Credentials = shallowCopyMap(credentials)
	return true, nil
}

type sharedRefreshLeaseCache struct {
	mu         sync.Mutex
	owners     map[string]string
	renewCalls int
}

func newSharedRefreshLeaseCache() *sharedRefreshLeaseCache {
	return &sharedRefreshLeaseCache{owners: make(map[string]string)}
}

func (c *sharedRefreshLeaseCache) GetAccessToken(context.Context, string) (string, error) {
	return "", nil
}
func (c *sharedRefreshLeaseCache) SetAccessToken(context.Context, string, string, time.Duration) error {
	return nil
}
func (c *sharedRefreshLeaseCache) DeleteAccessToken(context.Context, string) error { return nil }
func (c *sharedRefreshLeaseCache) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return false, errors.New("legacy lock must not be used")
}
func (c *sharedRefreshLeaseCache) ReleaseRefreshLock(context.Context, string) error { return nil }

func (c *sharedRefreshLeaseCache) TryAcquireRefreshLease(_ context.Context, key, owner string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owners[key] != "" {
		return false, nil
	}
	c.owners[key] = owner
	return true, nil
}

func (c *sharedRefreshLeaseCache) RenewRefreshLease(_ context.Context, key, owner string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owners[key] != owner {
		return false, nil
	}
	c.renewCalls++
	return true, nil
}

func (c *sharedRefreshLeaseCache) ReleaseRefreshLease(_ context.Context, key, owner string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owners[key] == owner {
		delete(c.owners, key)
	}
	return nil
}

type blockingForceRefreshExecutor struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	calls   int
	err     error
}

func (e *blockingForceRefreshExecutor) CanRefresh(*Account) bool                  { return true }
func (e *blockingForceRefreshExecutor) NeedsRefresh(*Account, time.Duration) bool { return false }
func (e *blockingForceRefreshExecutor) CacheKey(account *Account) string {
	return "force:test:" + account.Platform
}
func (e *blockingForceRefreshExecutor) Refresh(ctx context.Context, _ *Account) (map[string]any, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	e.once.Do(func() { close(e.started) })
	if e.release != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-e.release:
		}
	}
	if e.err != nil {
		return nil, e.err
	}
	return map[string]any{"access_token": "new-access", "refresh_token": "new-refresh"}, nil
}

func TestForceRefresh_TwoInstancesCallUpstreamOnceAndRenewLease(t *testing.T) {
	account := &Account{
		ID:       42,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"access_token":  "old-access",
			"refresh_token": "old-refresh",
		},
	}
	repo := &forceRefreshRepo{refreshAPIAccountRepo: refreshAPIAccountRepo{account: account}}
	cache := newSharedRefreshLeaseCache()
	executor := &blockingForceRefreshExecutor{started: make(chan struct{}), release: make(chan struct{})}
	firstAPI := NewOAuthRefreshAPI(repo, cache, 300*time.Millisecond)
	secondAPI := NewOAuthRefreshAPI(repo, cache, 300*time.Millisecond)

	type refreshCallResult struct {
		result *OAuthRefreshResult
		err    error
	}
	firstDone := make(chan refreshCallResult, 1)
	go func() {
		result, err := firstAPI.ForceRefresh(context.Background(), account, executor)
		firstDone <- refreshCallResult{result: result, err: err}
	}()
	<-executor.started
	time.Sleep(130 * time.Millisecond) // 覆盖至少一次 100ms 续租周期。

	secondResult, secondErr := secondAPI.ForceRefresh(context.Background(), account, executor)
	require.NoError(t, secondErr)
	require.True(t, secondResult.LockHeld)
	close(executor.release)
	first := <-firstDone
	require.NoError(t, first.err)
	require.True(t, first.result.Refreshed)

	executor.mu.Lock()
	require.Equal(t, 1, executor.calls)
	executor.mu.Unlock()
	cache.mu.Lock()
	require.GreaterOrEqual(t, cache.renewCalls, 1)
	cache.mu.Unlock()
	require.Equal(t, "new-access", repo.account.GetCredential("access_token"))
}

func TestForceRefresh_UpstreamFailurePreservesOldCredentials(t *testing.T) {
	account := &Account{
		ID:          43,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Credentials: map[string]any{"access_token": "still-valid", "refresh_token": "old-refresh"},
	}
	repo := &forceRefreshRepo{refreshAPIAccountRepo: refreshAPIAccountRepo{account: account}}
	cache := newSharedRefreshLeaseCache()
	executor := &blockingForceRefreshExecutor{
		started: make(chan struct{}),
		err:     errors.New("provider unavailable"),
	}

	result, err := NewOAuthRefreshAPI(repo, cache).ForceRefresh(context.Background(), account, executor)
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, "still-valid", repo.account.GetCredential("access_token"))
	require.Equal(t, "old-refresh", repo.account.GetCredential("refresh_token"))
}
