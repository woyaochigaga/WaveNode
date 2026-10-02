package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// OAuthRefreshExecutor 各平台实现的 OAuth 刷新执行器
// TokenRefresher 接口的超集：增加了 CacheKey 方法用于分布式锁
type OAuthRefreshExecutor interface {
	TokenRefresher

	// CacheKey 返回用于分布式锁的缓存键（与 TokenProvider 使用的一致）
	CacheKey(account *Account) string
}

// GrokOAuthRefreshSuccessRepository is the persistence boundary for a
// provider-issued Grok credential rotation. Implementations must compare the
// complete credential document and proxy used by the upstream attempt, and
// atomically publish scheduler invalidation with a successful update.
type GrokOAuthRefreshSuccessRepository interface {
	UpdateGrokOAuthCredentialsIfUnchanged(
		ctx context.Context,
		id int64,
		expectedCredentials map[string]any,
		expectedProxyID *int64,
		credentials map[string]any,
	) (bool, error)
}

// OAuthCredentialsCASRepository prevents a completed provider refresh from
// overwriting credentials or proxy configuration changed while the upstream
// request was in flight.
type OAuthCredentialsCASRepository interface {
	UpdateOAuthCredentialsIfUnchanged(
		ctx context.Context,
		id int64,
		expectedCredentials map[string]any,
		expectedProxyID *int64,
		credentials map[string]any,
	) (bool, error)
}

// OAuthRefreshOutcomeExecutor is an optional extension for providers that need
// to return a non-fatal operator warning in addition to refreshed credentials.
type OAuthRefreshOutcomeExecutor interface {
	RefreshWithOutcome(ctx context.Context, account *Account) (credentials map[string]any, warning string, err error)
}

const (
	defaultRefreshLockTTL                   = 60 * time.Second
	defaultRefreshLockReleaseTimeout        = 2 * time.Second
	defaultRefreshPostPersistCleanupTimeout = 2 * time.Second
)

var (
	errOAuthRefreshAccountRereadFailed = errors.New("oauth refresh account reread failed")
	errOAuthRefreshAccountStateChanged = errors.New("oauth refresh account state changed")
	errOAuthRefreshCredentialPersist   = errors.New("oauth refresh credential persistence failed")
	errOAuthRefreshLeaseUnavailable    = errors.New("oauth refresh distributed lease unavailable")
	errOAuthRefreshLeaseLost           = errors.New("oauth refresh distributed lease lost")
)

type oauthRefreshRequestPathKey struct{}

func withOAuthRefreshRequestPath(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauthRefreshRequestPathKey{}, true)
}

func isOAuthRefreshRequestPath(ctx context.Context) bool {
	requestPath, _ := ctx.Value(oauthRefreshRequestPathKey{}).(bool)
	return requestPath
}

type contextMutex struct {
	token chan struct{}
}

// Keep the request-path credential mutation lock API introduced by #4212
// while sharing the context-aware mutex implementation used by pool refresh.
type oauthRefreshLocalLock = contextMutex

func newOAuthRefreshLocalLock() *oauthRefreshLocalLock {
	return newContextMutex()
}

type oauthRefreshStateUnavailableError struct {
	err error
}

func (e *oauthRefreshStateUnavailableError) Error() string {
	return "OAuth refresh account state is unavailable"
}

func (e *oauthRefreshStateUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newContextMutex() *contextMutex {
	return &contextMutex{token: make(chan struct{}, 1)}
}

func (m *contextMutex) Lock(ctx context.Context) error {
	select {
	case m.token <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *contextMutex) Unlock() {
	<-m.token
}

// OAuthRefreshResult 统一刷新结果
type OAuthRefreshResult struct {
	Refreshed      bool           // 实际执行了刷新
	NewCredentials map[string]any // 刷新后的 credentials（nil 表示未刷新）
	Account        *Account       // 成功时为最新 account；刷新错误时为实际尝试的凭据快照
	LockHeld       bool           // 锁被其他 worker 持有（未执行刷新）
	Warning        string         // 非致命警告，例如 Antigravity 暂时缺少 project_id
}

func snapshotOAuthRefreshAccount(account *Account) *Account {
	if account == nil {
		return nil
	}
	snapshot := *account
	snapshot.Credentials = shallowCopyMap(account.Credentials)
	if account.ProxyID != nil {
		proxyID := *account.ProxyID
		snapshot.ProxyID = &proxyID
	}
	return &snapshot
}

// OAuthRefreshAPI 统一的 OAuth Token 刷新入口
// 封装分布式锁、进程内互斥锁、DB 重读、已刷新检查、竞争恢复等通用逻辑
type OAuthRefreshAPI struct {
	accountRepo AccountRepository
	tokenCache  GeminiTokenCache // 可选，nil = 无分布式锁
	lockTTL     time.Duration
	localLocks  sync.Map // key: cacheKey string -> value: *contextMutex
}

// NewOAuthRefreshAPI 创建统一刷新 API
// 可选传入 lockTTL 覆盖默认的 60s 分布式锁 TTL
func NewOAuthRefreshAPI(accountRepo AccountRepository, tokenCache GeminiTokenCache, lockTTL ...time.Duration) *OAuthRefreshAPI {
	ttl := defaultRefreshLockTTL
	if len(lockTTL) > 0 && lockTTL[0] > 0 {
		ttl = lockTTL[0]
	}
	return &OAuthRefreshAPI{
		accountRepo: accountRepo,
		tokenCache:  tokenCache,
		lockTTL:     ttl,
	}
}

// getLocalLock 返回指定 cacheKey 的进程内互斥锁
func (api *OAuthRefreshAPI) getLocalLock(cacheKey string) *contextMutex {
	actual, _ := api.localLocks.LoadOrStore(cacheKey, newContextMutex())
	mu, ok := actual.(*contextMutex)
	if !ok {
		mu = newContextMutex()
		api.localLocks.Store(cacheKey, mu)
	}
	return mu
}

// RefreshIfNeeded 在分布式锁保护下按需刷新 OAuth token
//
// 流程:
//  1. 获取分布式锁
//  2. 从 DB 重读最新 account（防止使用过时的 refresh_token）
//  3. 二次检查是否仍需刷新
//  4. 调用 executor.Refresh() 执行平台特定刷新逻辑
//  5. 设置 _token_version + 更新 DB
//  6. 释放锁
func (api *OAuthRefreshAPI) RefreshIfNeeded(
	ctx context.Context,
	account *Account,
	executor OAuthRefreshExecutor,
	refreshWindow time.Duration,
) (*OAuthRefreshResult, error) {
	return api.refresh(ctx, account, executor, refreshWindow, false)
}

// ForceRefresh 执行管理员显式触发的刷新。它不会受 token 到期窗口限制，且要求
// Redis 提供带所有权的租约；协调层不可用时宁可拒绝，也不进行无锁上游刷新。
func (api *OAuthRefreshAPI) ForceRefresh(
	ctx context.Context,
	account *Account,
	executor OAuthRefreshExecutor,
) (*OAuthRefreshResult, error) {
	return api.refresh(ctx, account, executor, 0, true)
}

func (api *OAuthRefreshAPI) refresh(
	ctx context.Context,
	account *Account,
	executor OAuthRefreshExecutor,
	refreshWindow time.Duration,
	force bool,
) (*OAuthRefreshResult, error) {
	if api == nil || api.accountRepo == nil {
		return nil, errors.New("oauth refresh account repository is not configured")
	}
	if account == nil {
		return nil, errors.New("oauth refresh account is nil")
	}
	if executor == nil {
		return nil, errors.New("oauth refresh executor is nil")
	}
	requestPath := isOAuthRefreshRequestPath(ctx)
	tokenCacheKey := executor.CacheKey(account)
	lockKey := oauthRefreshAccountLockKey(account.ID)
	var activeLeaseLost *atomic.Bool
	var localMu *contextMutex
	if !force {
		// 按需刷新保留原有“先本地等待、再重读数据库”的语义，避免同进程
		// 竞争被误判为跨实例冲突。
		localMu = api.getLocalLock(tokenCacheKey)
		if err := localMu.Lock(ctx); err != nil {
			return nil, fmt.Errorf("oauth refresh local lock: %w", err)
		}
		defer localMu.Unlock()
	}

	// 1. 先获取跨实例租约。顺序早于进程内锁，避免同一实例的第二个强制刷新
	// 等待首个请求结束后再次调用上游。
	if api.tokenCache == nil && force {
		return nil, fmt.Errorf("%w: token cache is not configured", errOAuthRefreshLeaseUnavailable)
	}
	if api.tokenCache != nil {
		if leaseCache, ok := api.tokenCache.(OAuthRefreshLeaseCache); ok {
			owner := uuid.NewString()
			acquired, lockErr := leaseCache.TryAcquireRefreshLease(ctx, lockKey, owner, api.lockTTL)
			if lockErr != nil {
				if force {
					return nil, fmt.Errorf("%w: %v", errOAuthRefreshLeaseUnavailable, lockErr)
				}
				slog.Warn("oauth_refresh_lease_failed_degraded",
					"account_id", account.ID,
					"error", lockErr,
				)
			} else if !acquired {
				return &OAuthRefreshResult{LockHeld: true}, nil
			} else {
				leaseCtx, leaseLost, stopRenewal := api.startRefreshLeaseRenewal(ctx, leaseCache, lockKey, owner, account.ID)
				activeLeaseLost = leaseLost
				ctx = leaseCtx
				defer func() {
					stopRenewal()
					api.releaseOwnedRefreshLease(ctx, leaseCache, lockKey, owner)
				}()
				defer func() {
					if leaseLost.Load() {
						slog.Warn("oauth_refresh_lease_lost", "account_id", account.ID)
					}
				}()
			}
		} else if force {
			return nil, fmt.Errorf("%w: owner-aware lease is not configured", errOAuthRefreshLeaseUnavailable)
		} else {
			// 兼容旧测试实现；生产 Redis cache 实现 OAuthRefreshLeaseCache。
			acquired, lockErr := api.tokenCache.AcquireRefreshLock(ctx, lockKey, api.lockTTL)
			if lockErr != nil {
				slog.Warn("oauth_refresh_lock_failed_degraded", "account_id", account.ID, "error", lockErr)
			} else if !acquired {
				return &OAuthRefreshResult{LockHeld: true}, nil
			} else {
				defer api.releaseRefreshLock(ctx, lockKey)
			}
		}
	}

	// 2. 强制刷新在分布式租约之后获取本地锁；第二个请求会在 Redis 层立即
	// 得到“正在刷新”，而不会等待后再次执行一次强制刷新。
	if force {
		localMu = api.getLocalLock(lockKey)
		if err := localMu.Lock(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, errOAuthRefreshLeaseLost
			}
			return nil, fmt.Errorf("oauth refresh local lock: %w", err)
		}
		defer localMu.Unlock()
	}

	// 3. 从 DB 重读最新 account（锁保护下，确保使用最新的 refresh_token）
	freshAccount, err := api.accountRepo.GetByID(ctx, account.ID)
	if err != nil {
		if requestPath {
			return nil, fmt.Errorf("%w: %v", errOAuthRefreshAccountRereadFailed, err)
		}
		return nil, &oauthRefreshStateUnavailableError{err: err}
	}
	if freshAccount == nil {
		if requestPath {
			return nil, fmt.Errorf("%w: account not found", errOAuthRefreshAccountStateChanged)
		}
		return nil, &oauthRefreshStateUnavailableError{err: fmt.Errorf("account not found")}
	}
	if freshAccount.ID != account.ID {
		return nil, fmt.Errorf("%w: account identity mismatch", errOAuthRefreshAccountRereadFailed)
	}
	if !force && !freshAccount.IsActive() {
		if requestPath {
			return nil, fmt.Errorf("%w: account is not active", errOAuthRefreshAccountStateChanged)
		}
		return &OAuthRefreshResult{Account: freshAccount}, nil
	}
	if requestPath && freshAccount.Platform == PlatformGrok {
		if eligibilityErr := grokOAuthRequestAccountEligibilityError(freshAccount); eligibilityErr != nil {
			return nil, withGrokCredentialFailureSnapshot(eligibilityErr, freshAccount)
		}
	}
	if !executor.CanRefresh(freshAccount) {
		if requestPath && freshAccount.IsGrokOAuth() && strings.TrimSpace(freshAccount.GetGrokRefreshToken()) == "" {
			return nil, withGrokCredentialFailureSnapshot(errGrokOAuthRefreshTokenMissing, freshAccount)
		}
		if requestPath {
			return nil, fmt.Errorf("%w: account is no longer refreshable", errOAuthRefreshAccountStateChanged)
		}
		return &OAuthRefreshResult{Account: freshAccount}, nil
	}

	// 4. 后台路径二次检查是否仍需刷新；管理员强制刷新跳过到期窗口。
	if !force && !executor.NeedsRefresh(freshAccount, refreshWindow) {
		return &OAuthRefreshResult{
			Account: freshAccount,
		}, nil
	}

	// 5. 执行平台特定刷新逻辑
	attemptedAccount := snapshotOAuthRefreshAccount(freshAccount)
	var warning string
	var newCredentials map[string]any
	var refreshErr error
	if outcomeExecutor, ok := executor.(OAuthRefreshOutcomeExecutor); ok {
		newCredentials, warning, refreshErr = outcomeExecutor.RefreshWithOutcome(ctx, freshAccount)
	} else {
		newCredentials, refreshErr = executor.Refresh(ctx, freshAccount)
	}
	if activeLeaseLost != nil && activeLeaseLost.Load() {
		return &OAuthRefreshResult{Account: attemptedAccount}, errOAuthRefreshLeaseLost
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		// A provider implementation may ignore cancellation and return late
		// credentials. Never persist them after the attempt/cycle boundary.
		return nil, ctxErr
	}
	if refreshErr != nil {
		// 竞争恢复：invalid_grant 可能是另一个 worker 已消费了旧 refresh_token
		// 重新读取 DB，如果 refresh_token 已更新则说明是竞争，返回成功
		if isInvalidGrantError(refreshErr) {
			if recoveredAccount, recovered := api.tryRecoverFromRefreshRace(ctx, freshAccount); recovered {
				if requestPath && recoveredAccount.Platform == PlatformGrok {
					if eligibilityErr := grokOAuthRequestAccountEligibilityError(recoveredAccount); eligibilityErr != nil {
						return nil, withGrokCredentialFailureSnapshot(eligibilityErr, recoveredAccount)
					}
				}
				slog.Info("oauth_refresh_race_recovered",
					"account_id", freshAccount.ID,
					"platform", freshAccount.Platform,
				)
				return &OAuthRefreshResult{
					Account: recoveredAccount,
				}, nil
			}
		}
		// Preserve the exact account snapshot used by the failed upstream call.
		// Callers can then conditionally mutate only that credential version and
		// avoid quarantining a concurrently reauthorized account.
		result := &OAuthRefreshResult{Account: attemptedAccount}
		if requestPath && attemptedAccount.Platform == PlatformGrok {
			return result, withGrokCredentialFailureSnapshot(refreshErr, attemptedAccount)
		}
		return result, refreshErr
	}

	// 6. 设置版本号 + 更新 DB
	if newCredentials != nil {
		newCredentials["_token_version"] = time.Now().UnixMilli()
		if freshAccount.IsGrokOAuth() {
			conditionalRepo, ok := api.accountRepo.(GrokOAuthRefreshSuccessRepository)
			if !ok {
				return nil, &providerConfigurationRefreshError{
					err: fmt.Errorf("grok OAuth refresh success CAS repository is not configured"),
				}
			}
			applied, updateErr := conditionalRepo.UpdateGrokOAuthCredentialsIfUnchanged(
				ctx,
				freshAccount.ID,
				attemptedAccount.Credentials,
				attemptedAccount.ProxyID,
				newCredentials,
			)
			if updateErr != nil {
				slog.Error("oauth_refresh_update_failed",
					"account_id", freshAccount.ID,
					"platform", freshAccount.Platform,
					"error", updateErr,
				)
				// The provider may have rotated and consumed the refresh token.
				// Retrying after an ambiguous local persistence result can turn a
				// healthy account into invalid_grant, so contain this provider cycle.
				return nil, &providerCycleContainmentRefreshError{
					err: fmt.Errorf("OAuth refresh succeeded but credential persistence failed: %w", updateErr),
				}
			}
			if !applied {
				currentAccount, readErr := api.accountRepo.GetByID(ctx, freshAccount.ID)
				if readErr != nil || currentAccount == nil {
					if readErr == nil {
						readErr = fmt.Errorf("account not found after Grok OAuth success CAS miss")
					}
					return nil, &providerCycleContainmentRefreshError{
						err: fmt.Errorf("grok OAuth success CAS lost and current state is unavailable: %w", readErr),
					}
				}
				slog.Info("oauth_refresh_success_cas_skipped_stale_credentials",
					"account_id", freshAccount.ID,
					"platform", freshAccount.Platform,
				)
				return &OAuthRefreshResult{Account: currentAccount}, nil
			}
			durableAccount, readErr := api.loadGrokDurableAccountAfterPersist(ctx, tokenCacheKey, freshAccount.ID)
			if readErr != nil || durableAccount == nil {
				if readErr == nil {
					readErr = fmt.Errorf("account not found after Grok OAuth success CAS")
				}
				return nil, &providerCycleContainmentRefreshError{
					err: fmt.Errorf("grok OAuth success persisted but durable account state is unavailable: %w", readErr),
				}
			}
			// The CAS changes credentials only. A concurrent admin or scheduler
			// mutation may have changed status, schedulability, or cooldown fields
			// while the provider call was in flight. Return the durable row so
			// post-refresh cache publication cannot restore that stale snapshot.
			freshAccount = durableAccount
		} else if conditionalRepo, ok := api.accountRepo.(OAuthCredentialsCASRepository); ok {
			applied, updateErr := conditionalRepo.UpdateOAuthCredentialsIfUnchanged(
				ctx,
				freshAccount.ID,
				attemptedAccount.Credentials,
				attemptedAccount.ProxyID,
				newCredentials,
			)
			if updateErr != nil {
				slog.Error("oauth_refresh_update_failed", "account_id", freshAccount.ID, "error", updateErr)
				return nil, fmt.Errorf("%w: %v", errOAuthRefreshCredentialPersist, updateErr)
			}
			currentAccount, readErr := api.accountRepo.GetByID(ctx, freshAccount.ID)
			if readErr != nil || currentAccount == nil {
				if readErr == nil {
					readErr = errors.New("account not found after OAuth credential CAS")
				}
				return nil, fmt.Errorf("%w: %v", errOAuthRefreshCredentialPersist, readErr)
			}
			if !applied {
				slog.Info("oauth_refresh_success_cas_skipped_stale_credentials",
					"account_id", freshAccount.ID,
					"platform", freshAccount.Platform,
				)
				return &OAuthRefreshResult{Account: currentAccount}, nil
			}
			freshAccount = currentAccount
		} else if force {
			return nil, fmt.Errorf("%w: OAuth credential CAS repository is not configured", errOAuthRefreshCredentialPersist)
		} else if updateErr := persistAccountCredentials(ctx, api.accountRepo, freshAccount, newCredentials); updateErr != nil {
			slog.Error("oauth_refresh_update_failed", "account_id", freshAccount.ID, "error", updateErr)
			return nil, fmt.Errorf("%w: %v", errOAuthRefreshCredentialPersist, updateErr)
		}
	}

	if requestPath && freshAccount.Platform == PlatformGrok {
		if eligibilityErr := grokOAuthRequestAccountEligibilityError(freshAccount); eligibilityErr != nil {
			return nil, withGrokCredentialFailureSnapshot(eligibilityErr, freshAccount)
		}
	}

	return &OAuthRefreshResult{
		Refreshed:      true,
		NewCredentials: newCredentials,
		Account:        freshAccount,
		Warning:        warning,
	}, nil
}

func oauthRefreshAccountLockKey(accountID int64) string {
	return "account:" + strconv.FormatInt(accountID, 10)
}

// startRefreshLeaseRenewal 在上游刷新期间定时延长租约。续租失败意味着当前请求
// 已不能证明自己仍是唯一持有者，因此取消请求并禁止后续凭证落库。
func (api *OAuthRefreshAPI) startRefreshLeaseRenewal(
	parent context.Context,
	cache OAuthRefreshLeaseCache,
	lockKey string,
	owner string,
	accountID int64,
) (context.Context, *atomic.Bool, func()) {
	leaseCtx, cancel := context.WithCancel(parent)
	lost := &atomic.Bool{}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	interval := api.lockTTL / 3
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}

	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				renewTimeout := defaultRefreshLockReleaseTimeout
				if halfTTL := api.lockTTL / 2; halfTTL > 0 && halfTTL < renewTimeout {
					renewTimeout = halfTTL
				}
				renewCtx, renewCancel := context.WithTimeout(leaseCtx, renewTimeout)
				ok, err := cache.RenewRefreshLease(renewCtx, lockKey, owner, api.lockTTL)
				renewCancel()
				if err != nil || !ok {
					lost.Store(true)
					cancel()
					slog.Warn("oauth_refresh_lease_renew_failed",
						"account_id", accountID,
						"error", err,
					)
					return
				}
			}
		}
	}()

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			close(stopCh)
			cancel()
		})
		<-doneCh
	}
	return leaseCtx, lost, stop
}

func (api *OAuthRefreshAPI) releaseOwnedRefreshLease(parent context.Context, cache OAuthRefreshLeaseCache, lockKey, owner string) {
	cleanupParent := context.Background()
	if parent != nil {
		cleanupParent = context.WithoutCancel(parent)
	}
	ctx, cancel := context.WithTimeout(cleanupParent, defaultRefreshLockReleaseTimeout)
	defer cancel()
	if err := cache.ReleaseRefreshLease(ctx, lockKey, owner); err != nil {
		slog.Warn("oauth_refresh_lease_release_failed", "lock_key", lockKey, "error", err)
	}
}

func (api *OAuthRefreshAPI) releaseRefreshLock(parent context.Context, cacheKey string) {
	cleanupParent := context.Background()
	if parent != nil {
		cleanupParent = context.WithoutCancel(parent)
	}
	ctx, cancel := context.WithTimeout(cleanupParent, defaultRefreshLockReleaseTimeout)
	defer cancel()
	if err := api.tokenCache.ReleaseRefreshLock(ctx, cacheKey); err != nil {
		slog.Warn("oauth_refresh_lock_release_failed", "cache_key", cacheKey, "error", err)
	}
}

func (api *OAuthRefreshAPI) loadGrokDurableAccountAfterPersist(parent context.Context, cacheKey string, accountID int64) (*Account, error) {
	cleanupParent := context.Background()
	if parent != nil {
		cleanupParent = context.WithoutCancel(parent)
	}
	ctx, cancel := context.WithTimeout(cleanupParent, defaultRefreshPostPersistCleanupTimeout)
	defer cancel()

	// A successful rotation can revoke the access token still cached from the
	// pre-rotation credential document. Trigger deletion at the commit boundary,
	// even if the attempt/parent context was canceled immediately after CAS.
	if api.tokenCache != nil {
		if err := api.tokenCache.DeleteAccessToken(ctx, cacheKey); err != nil {
			slog.Warn("oauth_refresh_post_persist_cache_delete_failed",
				"account_id", accountID,
				"cache_key", cacheKey,
				"error", err,
			)
		}
	}

	return api.accountRepo.GetByID(ctx, accountID)
}

// isInvalidGrantError 检查错误是否为 invalid_grant
func isInvalidGrantError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "invalid_grant")
}

// tryRecoverFromRefreshRace 在 invalid_grant 错误后尝试竞争恢复
// 重新读取 DB，如果 refresh_token 已改变（说明另一个 worker 成功刷新），则返回更新后的 account
func (api *OAuthRefreshAPI) tryRecoverFromRefreshRace(ctx context.Context, usedAccount *Account) (*Account, bool) {
	if api.accountRepo == nil {
		return nil, false
	}
	reReadAccount, err := api.accountRepo.GetByID(ctx, usedAccount.ID)
	if err != nil || reReadAccount == nil {
		return nil, false
	}
	usedRT := usedAccount.GetCredential("refresh_token")
	currentRT := reReadAccount.GetCredential("refresh_token")
	if usedRT == "" || currentRT == "" {
		return nil, false
	}
	// refresh_token 不同 → 另一个 worker 已成功刷新
	if usedRT != currentRT {
		return reReadAccount, true
	}
	return nil, false
}

// MergeCredentials 将旧 credentials 中不存在于新 map 的字段保留到新 map 中
func MergeCredentials(oldCreds, newCreds map[string]any) map[string]any {
	if newCreds == nil {
		newCreds = make(map[string]any)
	}
	for k, v := range oldCreds {
		if _, exists := newCreds[k]; !exists {
			newCreds[k] = v
		}
	}
	return newCreds
}

// BuildClaudeAccountCredentials 为 Claude 平台构建 OAuth credentials map
// 消除 Claude 平台没有 BuildAccountCredentials 方法的问题
func BuildClaudeAccountCredentials(tokenInfo *TokenInfo) map[string]any {
	creds := map[string]any{
		"access_token": tokenInfo.AccessToken,
		"token_type":   tokenInfo.TokenType,
		"expires_in":   strconv.FormatInt(tokenInfo.ExpiresIn, 10),
		"expires_at":   strconv.FormatInt(tokenInfo.ExpiresAt, 10),
	}
	if tokenInfo.RefreshToken != "" {
		creds["refresh_token"] = tokenInfo.RefreshToken
	}
	if tokenInfo.Scope != "" {
		creds["scope"] = tokenInfo.Scope
	}
	return creds
}
