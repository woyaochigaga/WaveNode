//go:build unit

package admin

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type grokRefreshOAuthStub struct {
	account *service.Account
	info    *service.GrokTokenInfo
	calls   int
}

func (s *grokRefreshOAuthStub) RefreshAccountToken(_ context.Context, account *service.Account) (*service.GrokTokenInfo, error) {
	s.calls++
	s.account = account
	return s.info, nil
}

func (s *grokRefreshOAuthStub) BuildAccountCredentials(info *service.GrokTokenInfo) map[string]any {
	return map[string]any{
		"access_token":  info.AccessToken,
		"refresh_token": info.RefreshToken,
		"expires_at":    info.ExpiresAt,
		"base_url":      "https://api.x.ai/v1",
	}
}

type grokRefreshAdminService struct {
	*stubAdminService
	refreshCalls int
	result       *service.AccountCredentialRefreshResult
}

func (s *grokRefreshAdminService) RefreshAccountCredentials(_ context.Context, _ int64) (*service.AccountCredentialRefreshResult, error) {
	s.refreshCalls++
	return s.result, nil
}

func TestRefreshSingleAccountRoutesGrokThroughUnifiedAdminRefresh(t *testing.T) {
	t.Parallel()

	updatedAccount := &service.Account{
		ID:       4227,
		Platform: service.PlatformGrok,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Credentials: map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
		},
	}
	adminSvc := &grokRefreshAdminService{
		stubAdminService: newStubAdminService(),
		result:           &service.AccountCredentialRefreshResult{Account: updatedAccount, Refreshed: true},
	}
	grokOAuth := &grokRefreshOAuthStub{info: &service.GrokTokenInfo{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		ExpiresAt:    1_800_000_000,
	}}
	handler := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		grokOAuth,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	account := &service.Account{
		ID:       4227,
		Platform: service.PlatformGrok,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "old-access",
			"refresh_token":      "old-refresh",
			"base_url":           "https://example.invalid/v1",
			"subscription_tier":  "SUPER_GROK",
			"entitlement_status": "ACTIVE",
		},
	}

	updated, warning, err := handler.refreshSingleAccount(context.Background(), account)
	require.NoError(t, err)
	require.Empty(t, warning)
	require.Equal(t, 1, adminSvc.refreshCalls)
	require.Zero(t, grokOAuth.calls, "handler must not bypass the unified refresh coordinator")
	require.Same(t, updatedAccount, updated)
}
