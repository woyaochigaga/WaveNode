//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestClassifyConnectionTestFailure(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantStage  string
		wantStatus int
	}{
		{name: "unauthorized", err: errors.New("API returned 401: invalid token"), wantCode: "UPSTREAM_UNAUTHORIZED", wantStage: "credentials", wantStatus: 401},
		{name: "forbidden", err: errors.New("API returned 403: subscription denied"), wantCode: "UPSTREAM_FORBIDDEN", wantStage: "credentials", wantStatus: 403},
		{name: "rate limited", err: errors.New("request failed with status 429"), wantCode: "UPSTREAM_RATE_LIMITED", wantStage: "rate_limit", wantStatus: 429},
		{name: "proxy auth", err: errors.New("proxy authentication required (407)"), wantCode: "PROXY_AUTH_FAILED", wantStage: "proxy", wantStatus: 407},
		{name: "dns", err: errors.New("dial tcp: lookup api.example: no such host"), wantCode: "DNS_RESOLUTION_FAILED", wantStage: "dns"},
		{name: "tls", err: errors.New("tls: failed to verify certificate: x509 unknown authority"), wantCode: "TLS_HANDSHAKE_FAILED", wantStage: "tls"},
		{name: "timeout", err: errors.New("upstream i/o timeout"), wantCode: "UPSTREAM_TIMEOUT", wantStage: "transport"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyConnectionTestFailure(context.Background(), tt.err, "")
			require.Equal(t, tt.wantCode, result.Code)
			require.Equal(t, tt.wantStage, result.Stage)
			require.Equal(t, tt.wantStatus, result.HTTPStatus)
		})
	}
}

func TestClassifyConnectionTestFailureCancellationAndRedaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cancelled := classifyConnectionTestFailure(ctx, errors.New("request interrupted"), "")
	require.Equal(t, "REQUEST_CANCELLED", cancelled.Code)
	require.Equal(t, "cancelled", cancelled.Stage)

	redacted := classifyConnectionTestFailure(
		context.Background(),
		errors.New(`request failed: {"api_key":"canary-connection-secret"}`),
		"",
	)
	require.NotContains(t, redacted.Message, "canary-connection-secret")
}

func TestAccountConnectionEmitsBackwardCompatibleSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:          701,
		Name:        "synthetic-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Extra:       map[string]any{"synthetic_ui_test": true},
		Credentials: map[string]any{"expires_at": expiresAt.Format(time.RFC3339)},
	}
	repo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{account.ID: account}}
	service := &AccountTestService{accountRepo: repo}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/test", nil)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	result, err := service.TestAccountConnectionWithResult(ctx, account.ID, "test-model", "", AccountTestModeDefault)

	require.NoError(t, err)
	require.Equal(t, ConnectionTestStatusSuccess, result.Status)
	require.True(t, result.SafeToSchedule)
	require.NotNil(t, result.CredentialExpiresAt)
	require.WithinDuration(t, expiresAt, *result.CredentialExpiresAt, time.Second)
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
	require.Contains(t, recorder.Body.String(), `"type":"summary"`)
	require.Equal(t, 1, strings.Count(recorder.Body.String(), `"type":"summary"`))
}

type connectionTestProxyRepo struct {
	proxyRepoStub
	proxy *Proxy
}

func (r *connectionTestProxyRepo) GetByID(context.Context, int64) (*Proxy, error) {
	return r.proxy, nil
}

type connectionTestProxyProber struct {
	exit      *ProxyExitInfo
	latencyMs int64
	err       error
}

func (p *connectionTestProxyProber) ProbeProxy(context.Context, string) (*ProxyExitInfo, int64, error) {
	return p.exit, p.latencyMs, p.err
}

func TestAdminServiceTestProxyReturnsStructuredResult(t *testing.T) {
	proxy := &Proxy{ID: 9, Protocol: "http", Host: "127.0.0.1", Port: 8080, Status: StatusActive}
	service := &adminServiceImpl{
		proxyRepo: &connectionTestProxyRepo{proxy: proxy},
		proxyProber: &connectionTestProxyProber{
			latencyMs: 23,
			err:       errors.New("proxy authentication failed with status 407"),
		},
	}

	result, err := service.TestProxy(context.Background(), proxy.ID)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Equal(t, ConnectionTestStatusFailed, result.Status)
	require.Equal(t, "PROXY_AUTH_FAILED", result.ErrorCode)
	require.Equal(t, 407, result.HTTPStatus)
	require.Equal(t, int64(23), result.LatencyMs)
	require.False(t, result.SafeToSchedule)
	require.WithinDuration(t, time.Now().UTC(), result.TestedAt, time.Second)

	encoded, marshalErr := json.Marshal(result)
	require.NoError(t, marshalErr)
	require.Contains(t, string(encoded), `"status":"failed"`)
	require.Contains(t, string(encoded), `"success":false`)
	require.Contains(t, string(encoded), `"message":"proxy authentication failed with status 407"`)
}

func TestLegacyConnectionTestEntrypointsDoNotReturnFakeSuccess(t *testing.T) {
	accountRepo := &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{
		11: {ID: 11, Status: StatusActive},
	}}
	accountService := NewAccountService(accountRepo, nil)
	accountErr := accountService.TestCredentials(context.Background(), 11)
	require.ErrorContains(t, accountErr, "AccountTestService.TestAccountConnection")

	proxy := &Proxy{ID: 12, Status: StatusActive}
	proxyRepo := &connectionTestProxyRepo{proxy: proxy}
	proxyService := NewProxyService(proxyRepo)
	proxyErr := proxyService.TestConnection(context.Background(), proxy.ID)
	require.ErrorContains(t, proxyErr, "AdminService.TestProxy")
}
