package errors

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToHTTPAddsConservativeRetryPolicy(t *testing.T) {
	statusCode, body := ToHTTP(ServiceUnavailable("UPSTREAM_BUSY", "try again"))

	require.Equal(t, http.StatusServiceUnavailable, statusCode)
	require.True(t, body.Retryable)
	require.Zero(t, body.RetryAfterSeconds)
}

func TestToHTTPPreservesExplicitRetryDelay(t *testing.T) {
	statusCode, body := ToHTTP(TooManyRequests("RATE_LIMITED", "slow down").WithRetryPolicy(true, 7))

	require.Equal(t, http.StatusTooManyRequests, statusCode)
	require.True(t, body.Retryable)
	require.Equal(t, 7, body.RetryAfterSeconds)
}

func TestToHTTPExplicitNonRetryableOverridesFiveHundredDefault(t *testing.T) {
	_, body := ToHTTP(InternalServer("INVALID_INTERNAL_STATE", "cannot retry").WithRetryPolicy(false, 0))

	require.False(t, body.Retryable)
}
