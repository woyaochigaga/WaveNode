package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type routeTraceBillingRepo struct {
	OpsRepository
	summary RouteTraceBillingSummary
	err     error
}

func (r *routeTraceBillingRepo) GetRouteTraceBilling(context.Context, string, string) (RouteTraceBillingSummary, error) {
	return r.summary, r.err
}

func TestBuildRouteTraceReturnsRedactedAttemptChain(t *testing.T) {
	accountID := int64(42)
	events, err := json.Marshal([]*OpsUpstreamErrorEvent{
		{
			AccountID:            accountID,
			AccountName:          "sensitive-account-name",
			Platform:             "openai",
			UpstreamStatusCode:   503,
			Kind:                 "failover",
			Stage:                "upstream",
			Reason:               "provider_overloaded sensitive-reason-from-request",
			Message:              "echoed prompt: sensitive-user-prompt",
			ConfigVersion:        "cfg-123",
			PriceVersion:         "price-456",
			UpstreamURL:          "https://secret.example/v1/responses?token=secret",
			UpstreamResponseBody: "secret-response-body",
		},
	})
	require.NoError(t, err)

	trace, err := BuildRouteTrace(&OpsErrorLogDetail{
		OpsErrorLog: OpsErrorLog{
			RequestID:        "req-1",
			ClientRequestID:  "client-1",
			RequestedModel:   "public-model",
			UpstreamModel:    "upstream-model",
			InboundEndpoint:  "/v1/messages",
			UpstreamEndpoint: "/v1/responses",
			StatusCode:       503,
			Phase:            "upstream",
			Message:          "echoed prompt: sensitive-final-prompt",
			AccountID:        &accountID,
		},
		UpstreamErrors: string(events),
	})
	require.NoError(t, err)
	require.Equal(t, "cfg-123", trace.ConfigVersion)
	require.Equal(t, "price-456", trace.PriceVersion)
	require.Equal(t, 1, trace.CandidateCount)
	require.True(t, trace.Retryable)
	require.Equal(t, AnonymousAccountRef(accountID), trace.Attempts[0].AccountRef)
	require.Equal(t, "upstream_unavailable", trace.Attempts[0].Reason)

	encoded, err := json.Marshal(trace)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "sensitive-account-name")
	require.NotContains(t, string(encoded), "secret.example")
	require.NotContains(t, string(encoded), "secret-response-body")
	require.NotContains(t, string(encoded), "sensitive-user-prompt")
	require.NotContains(t, string(encoded), "sensitive-final-prompt")
	require.NotContains(t, string(encoded), "sensitive-reason-from-request")
}

func TestRouteTraceAttemptReasonReturnsFiniteSafeCodes(t *testing.T) {
	tests := []struct {
		name  string
		event *OpsUpstreamErrorEvent
		want  string
	}{
		{name: "rate limit", event: &OpsUpstreamErrorEvent{UpstreamStatusCode: 429, Reason: "arbitrary detail"}, want: "rate_limited"},
		{name: "auth", event: &OpsUpstreamErrorEvent{UpstreamStatusCode: 401}, want: "authentication_failed"},
		{name: "timeout", event: &OpsUpstreamErrorEvent{Reason: "deadline_exceeded"}, want: "upstream_timeout"},
		{name: "unknown", event: &OpsUpstreamErrorEvent{Reason: "raw user supplied value"}, want: "attempt_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, routeTraceAttemptReason(tt.event))
		})
	}
}

func TestEnrichRouteTraceBillingKeepsTraceAvailableOnLookupFailure(t *testing.T) {
	trace := &RouteTrace{RequestID: "req-1", BillingStatus: "unknown"}
	service := &OpsService{opsRepo: &routeTraceBillingRepo{err: errors.New("database unavailable")}}

	service.EnrichRouteTraceBilling(context.Background(), trace)

	require.Equal(t, "lookup_failed", trace.BillingStatus)
}

func TestEnrichRouteTraceBillingAppliesChargedSummary(t *testing.T) {
	trace := &RouteTrace{RequestID: "req-1", BillingStatus: "unknown"}
	service := &OpsService{opsRepo: &routeTraceBillingRepo{summary: RouteTraceBillingSummary{
		Status: "charged", Amount: 1.25, RecordCount: 2,
	}}}

	service.EnrichRouteTraceBilling(context.Background(), trace)

	require.Equal(t, "charged", trace.BillingStatus)
	require.Equal(t, 1.25, trace.BilledAmount)
	require.EqualValues(t, 2, trace.UsageRecordCount)
}
