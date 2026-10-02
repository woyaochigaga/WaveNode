package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrRouteTraceNotFound = infraerrors.NotFound("ROUTE_TRACE_NOT_FOUND", "route trace not found or expired")

// RouteTrace 是管理员可查看的请求路由说明，只包含路由元数据和脱敏后的上游尝试摘要。
type RouteTrace struct {
	RequestID         string              `json:"request_id"`
	ClientRequestID   string              `json:"client_request_id,omitempty"`
	RequestedModel    string              `json:"requested_model,omitempty"`
	FinalModel        string              `json:"final_model,omitempty"`
	InboundEndpoint   string              `json:"inbound_endpoint,omitempty"`
	UpstreamEndpoint  string              `json:"upstream_endpoint,omitempty"`
	ConfigVersion     string              `json:"config_version,omitempty"`
	PriceVersion      string              `json:"price_version"`
	CandidateCount    int                 `json:"candidate_count"`
	RetryCount        int                 `json:"retry_count"`
	DowngradeReason   string              `json:"downgrade_reason,omitempty"`
	FinalAccountRef   string              `json:"final_account_ref,omitempty"`
	FinalErrorCode    string              `json:"final_error_code,omitempty"`
	FinalErrorMessage string              `json:"final_error_message,omitempty"`
	Retryable         bool                `json:"retryable"`
	RetryAfterSeconds int                 `json:"retry_after,omitempty"`
	BillingStatus     string              `json:"billing_status"`
	BilledAmount      float64             `json:"billed_amount"`
	UsageRecordCount  int64               `json:"usage_record_count"`
	Attempts          []RouteTraceAttempt `json:"attempts"`
}

type RouteTraceAttempt struct {
	AtUnixMs      int64  `json:"at_unix_ms,omitempty"`
	AccountRef    string `json:"account_ref,omitempty"`
	Platform      string `json:"platform,omitempty"`
	StatusCode    int    `json:"status_code,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Stage         string `json:"stage,omitempty"`
	Scope         string `json:"scope,omitempty"`
	Reason        string `json:"reason,omitempty"`
	ConfigVersion string `json:"config_version,omitempty"`
	PriceVersion  string `json:"price_version,omitempty"`
}

// RouteTraceBillingSummary 是按请求 ID 聚合后的计费结果，不包含用户或凭据信息。
type RouteTraceBillingSummary struct {
	Status      string
	Amount      float64
	RecordCount int64
}

// OpsRouteTraceBillingReader 是 Ops 仓储的可选能力，避免扩大已有大接口并影响测试桩。
type OpsRouteTraceBillingReader interface {
	GetRouteTraceBilling(ctx context.Context, requestID, clientRequestID string) (RouteTraceBillingSummary, error)
}

// BuildRouteTrace 将已有错误详情转换为路由追踪结构；上游事件在落库前已经完成脱敏。
func BuildRouteTrace(detail *OpsErrorLogDetail) (*RouteTrace, error) {
	if detail == nil {
		return nil, ErrRouteTraceNotFound
	}
	trace := &RouteTrace{
		RequestID:         strings.TrimSpace(detail.RequestID),
		ClientRequestID:   strings.TrimSpace(detail.ClientRequestID),
		RequestedModel:    strings.TrimSpace(detail.RequestedModel),
		FinalModel:        strings.TrimSpace(detail.UpstreamModel),
		InboundEndpoint:   strings.TrimSpace(detail.InboundEndpoint),
		UpstreamEndpoint:  strings.TrimSpace(detail.UpstreamEndpoint),
		PriceVersion:      "not_captured",
		BillingStatus:     "unknown",
		FinalErrorCode:    routeTraceErrorCode(detail),
		FinalErrorMessage: routeTraceSafeErrorMessage(routeTraceErrorCode(detail)),
	}
	trace.Retryable, trace.RetryAfterSeconds = routeTraceRetryPolicy(detail.StatusCode, detail.Phase)
	if detail.AccountID != nil && *detail.AccountID > 0 {
		trace.FinalAccountRef = AnonymousAccountRef(*detail.AccountID)
	}
	if strings.EqualFold(detail.Phase, "billing") || strings.Contains(strings.ToLower(detail.Type), "billing") {
		trace.BillingStatus = "billing_blocked"
	}
	if detail.UpstreamErrors != "" {
		events, err := ParseOpsUpstreamErrors(detail.UpstreamErrors)
		if err != nil {
			return nil, err
		}
		trace.Attempts = make([]RouteTraceAttempt, 0, len(events))
		for _, event := range events {
			if event == nil {
				continue
			}
			safeReason := routeTraceAttemptReason(event)
			attempt := RouteTraceAttempt{
				AtUnixMs:      event.AtUnixMs,
				AccountRef:    AnonymousAccountRef(event.AccountID),
				Platform:      event.Platform,
				StatusCode:    event.UpstreamStatusCode,
				Kind:          event.Kind,
				Stage:         event.Stage,
				Scope:         event.Scope,
				Reason:        safeReason,
				ConfigVersion: event.ConfigVersion,
				PriceVersion:  event.PriceVersion,
			}
			trace.Attempts = append(trace.Attempts, attempt)
			if trace.ConfigVersion == "" && event.ConfigVersion != "" {
				trace.ConfigVersion = event.ConfigVersion
			}
			if trace.PriceVersion == "not_captured" && event.PriceVersion != "" {
				trace.PriceVersion = event.PriceVersion
			}
			if attempt.Reason == "downgraded" && trace.DowngradeReason == "" {
				trace.DowngradeReason = attempt.Reason
			}
		}
	}
	trace.CandidateCount = len(trace.Attempts)
	if trace.CandidateCount == 0 && trace.FinalAccountRef != "" {
		trace.CandidateCount = 1
	}
	if trace.CandidateCount > 1 {
		trace.RetryCount = trace.CandidateCount - 1
	}
	if trace.ConfigVersion == "" {
		trace.ConfigVersion = "not_captured"
	}
	if len(trace.Attempts) == 0 {
		trace.Attempts = []RouteTraceAttempt{}
	}
	return trace, nil
}

// routeTraceAttemptReason 只输出平台定义的有限原因码，不透传可能包含请求内容的原始 reason。
func routeTraceAttemptReason(event *OpsUpstreamErrorEvent) string {
	if event == nil {
		return "attempt_failed"
	}
	context := strings.ToLower(strings.Join([]string{event.Kind, event.Stage, event.Scope, event.Reason}, " "))
	switch {
	case strings.Contains(context, "downgrade"), strings.Contains(context, "fallback"):
		return "downgraded"
	case event.UpstreamStatusCode == http.StatusUnauthorized, event.UpstreamStatusCode == http.StatusForbidden,
		strings.Contains(context, "auth"), strings.Contains(context, "credential"), strings.Contains(context, "token"):
		return "authentication_failed"
	case event.UpstreamStatusCode == http.StatusTooManyRequests,
		strings.Contains(context, "rate_limit"), strings.Contains(context, "quota"), strings.Contains(context, "exhaust"):
		return "rate_limited"
	case event.UpstreamStatusCode == http.StatusRequestTimeout, event.UpstreamStatusCode == http.StatusGatewayTimeout,
		strings.Contains(context, "timeout"), strings.Contains(context, "deadline"):
		return "upstream_timeout"
	case strings.Contains(context, "proxy"), strings.Contains(context, "network"), strings.Contains(context, "transport"):
		return "network_error"
	case strings.Contains(context, "safety"), strings.Contains(context, "content"), strings.Contains(context, "policy"):
		return "content_policy"
	case strings.Contains(context, "model"), strings.Contains(context, "unsupported"):
		return "model_unavailable"
	case event.UpstreamStatusCode >= http.StatusInternalServerError,
		strings.Contains(context, "overload"), strings.Contains(context, "unavailable"):
		return "upstream_unavailable"
	default:
		return "attempt_failed"
	}
}

// EnrichRouteTraceBilling 使用用量表确认该请求是否已经落账。
// 查询失败不会遮蔽路由追踪主体，只把状态标记为 lookup_failed 供管理员识别。
func (s *OpsService) EnrichRouteTraceBilling(ctx context.Context, trace *RouteTrace) {
	if s == nil || trace == nil || s.opsRepo == nil {
		return
	}
	reader, ok := s.opsRepo.(OpsRouteTraceBillingReader)
	if !ok {
		return
	}
	summary, err := reader.GetRouteTraceBilling(ctx, trace.RequestID, trace.ClientRequestID)
	if err != nil {
		trace.BillingStatus = "lookup_failed"
		return
	}
	trace.BilledAmount = summary.Amount
	trace.UsageRecordCount = summary.RecordCount
	if trace.BillingStatus != "billing_blocked" || summary.RecordCount > 0 {
		trace.BillingStatus = summary.Status
	}
}

// AnonymousAccountRef 把账号 ID 转成稳定匿名引用，便于关联尝试但不暴露账号名称或凭据。
func AnonymousAccountRef(accountID int64) string {
	if accountID <= 0 {
		return ""
	}
	digest := sha256.Sum256([]byte("sub2api-account:" + strconv.FormatInt(accountID, 10)))
	return "acct_" + hex.EncodeToString(digest[:])[:12]
}

func routeTraceErrorCode(detail *OpsErrorLogDetail) string {
	if detail == nil {
		return "internal_error"
	}
	typ := strings.ToLower(strings.TrimSpace(detail.Type))
	phase := strings.ToLower(strings.TrimSpace(detail.Phase))
	switch {
	case strings.Contains(typ, "billing") || strings.Contains(phase, "billing"):
		return "billing_blocked"
	case detail.StatusCode == http.StatusTooManyRequests:
		return "rate_limited"
	case detail.StatusCode == http.StatusRequestTimeout || detail.StatusCode == http.StatusGatewayTimeout:
		return "upstream_timeout"
	case detail.StatusCode >= 500 || phase == "upstream" || phase == "network":
		return "upstream_overloaded"
	case phase == "auth" || phase == "account_auth":
		return "auth_invalid"
	case phase == "routing":
		return "model_unavailable"
	default:
		return "request_failed"
	}
}

// routeTraceSafeErrorMessage 只返回平台定义的说明，避免上游错误文本回显请求内容。
func routeTraceSafeErrorMessage(code string) string {
	switch code {
	case "billing_blocked":
		return "request was blocked before billing completed"
	case "rate_limited":
		return "request was rate limited"
	case "upstream_timeout":
		return "upstream request timed out"
	case "upstream_overloaded":
		return "upstream service was unavailable"
	case "auth_invalid":
		return "upstream authentication failed"
	case "model_unavailable":
		return "no eligible route was available for the requested model"
	default:
		return "request failed"
	}
}

func routeTraceRetryPolicy(statusCode int, phase string) (bool, int) {
	if statusCode == http.StatusTooManyRequests {
		return true, 1
	}
	if statusCode == http.StatusRequestTimeout || statusCode == http.StatusBadGateway || statusCode == http.StatusServiceUnavailable || statusCode == http.StatusGatewayTimeout || strings.EqualFold(phase, "network") || strings.EqualFold(phase, "upstream") {
		return true, 0
	}
	return false, 0
}
