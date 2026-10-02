package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

const (
	ConnectionTestStatusSuccess   = "success"
	ConnectionTestStatusFailed    = "failed"
	ConnectionTestStatusCancelled = "cancelled"
)

// ConnectionTestResult 是账号与代理探测共用的终态契约。
// Message 只允许放脱敏后的简短说明，不能包含凭证、请求正文或完整上游响应。
type ConnectionTestResult struct {
	Status              string     `json:"status"`
	Stage               string     `json:"stage"`
	ErrorCode           string     `json:"error_code,omitempty"`
	HTTPStatus          int        `json:"http_status,omitempty"`
	LatencyMs           int64      `json:"latency_ms"`
	TestedAt            time.Time  `json:"tested_at"`
	CredentialExpiresAt *time.Time `json:"credential_expires_at,omitempty"`
	SafeToSchedule      bool       `json:"safe_to_schedule"`
	Message             string     `json:"message,omitempty"`
}

// connectionTestFailure 在旧平台测试函数仍返回 error 的前提下保留机器可读错误信息。
type connectionTestFailure struct {
	Code       string
	Stage      string
	HTTPStatus int
	Message    string
}

func (e *connectionTestFailure) Error() string {
	if e == nil {
		return "connection test failed"
	}
	return e.Message
}

var connectionTestHTTPStatusPattern = regexp.MustCompile(`\b(401|403|407|408|429|5\d\d)\b`)

// classifyConnectionTestFailure 将历史自由文本错误收敛为稳定分类。
// 新代码可传入显式错误码；旧代码继续通过 HTTP 状态和网络错误特征兼容识别。
func classifyConnectionTestFailure(ctx context.Context, err error, explicitCode string) *connectionTestFailure {
	message := "Connection test failed"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = logredact.RedactText(err.Error())
	}

	failure := &connectionTestFailure{
		Code:    strings.TrimSpace(explicitCode),
		Stage:   "upstream",
		Message: message,
	}
	switch failure.Code {
	case "ACCOUNT_NOT_FOUND":
		failure.Stage = "lookup"
	case "PROXY_PROBER_UNAVAILABLE":
		failure.Stage = "proxy"
	case "GROUP_TEST_NO_SCHEDULABLE_ACCOUNTS", "GROUP_TEST_NO_SUPPORTED_MODELS", "GROUP_TEST_MODEL_NOT_ALLOWED", "GROUP_TEST_MODEL_UNAVAILABLE":
		failure.Stage = "routing"
	}
	if typed := (*connectionTestFailure)(nil); errors.As(err, &typed) {
		failure.Code = typed.Code
		failure.Stage = typed.Stage
		failure.HTTPStatus = typed.HTTPStatus
		failure.Message = logredact.RedactText(typed.Message)
		return failure
	}

	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		failure.Code = "REQUEST_CANCELLED"
		failure.Stage = "cancelled"
		return failure
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		failure.Code = "UPSTREAM_TIMEOUT"
		failure.Stage = "transport"
		return failure
	}

	lower := strings.ToLower(message)
	if match := connectionTestHTTPStatusPattern.FindStringSubmatch(lower); len(match) == 2 {
		failure.HTTPStatus, _ = strconv.Atoi(match[1])
	}

	switch failure.HTTPStatus {
	case http.StatusUnauthorized:
		failure.Code = "UPSTREAM_UNAUTHORIZED"
		failure.Stage = "credentials"
	case http.StatusForbidden:
		failure.Code = "UPSTREAM_FORBIDDEN"
		failure.Stage = "credentials"
	case http.StatusProxyAuthRequired:
		failure.Code = "PROXY_AUTH_FAILED"
		failure.Stage = "proxy"
	case http.StatusRequestTimeout:
		failure.Code = "UPSTREAM_TIMEOUT"
		failure.Stage = "transport"
	case http.StatusTooManyRequests:
		failure.Code = "UPSTREAM_RATE_LIMITED"
		failure.Stage = "rate_limit"
	default:
		var networkError net.Error
		switch {
		case strings.Contains(lower, "proxy") && strings.Contains(lower, "auth"):
			failure.Code = "PROXY_AUTH_FAILED"
			failure.Stage = "proxy"
		case strings.Contains(lower, "no such host"), strings.Contains(lower, "dns"), strings.Contains(lower, "name resolution"):
			failure.Code = "DNS_RESOLUTION_FAILED"
			failure.Stage = "dns"
		case strings.Contains(lower, "tls"), strings.Contains(lower, "x509"), strings.Contains(lower, "certificate"):
			failure.Code = "TLS_HANDSHAKE_FAILED"
			failure.Stage = "tls"
		case errors.As(err, &networkError) && networkError.Timeout(), strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"):
			failure.Code = "UPSTREAM_TIMEOUT"
			failure.Stage = "transport"
		case failure.HTTPStatus >= http.StatusInternalServerError:
			failure.Code = "UPSTREAM_UNAVAILABLE"
			failure.Stage = "upstream"
		}
	}

	if failure.Code == "" {
		failure.Code = "CONNECTION_TEST_FAILED"
	}
	return failure
}

func newConnectionTestFailure(ctx context.Context, code, message string) *connectionTestFailure {
	return classifyConnectionTestFailure(ctx, errors.New(message), code)
}

func accountCredentialExpiresAt(account *Account) *time.Time {
	if account == nil {
		return nil
	}
	if expiresAt := account.GetCredentialAsTime("expires_at"); expiresAt != nil {
		return expiresAt
	}
	return account.ExpiresAt
}

func buildAccountConnectionTestResult(ctx context.Context, account *Account, startedAt time.Time, err error) *ConnectionTestResult {
	result := &ConnectionTestResult{
		Status:              ConnectionTestStatusSuccess,
		Stage:               "complete",
		LatencyMs:           time.Since(startedAt).Milliseconds(),
		TestedAt:            time.Now().UTC(),
		CredentialExpiresAt: accountCredentialExpiresAt(account),
		SafeToSchedule:      account != nil && account.IsSchedulable(),
	}
	if err == nil {
		result.Message = "Connection test completed"
		return result
	}

	failure := classifyConnectionTestFailure(ctx, err, "")
	result.Status = ConnectionTestStatusFailed
	result.Stage = failure.Stage
	result.ErrorCode = failure.Code
	result.HTTPStatus = failure.HTTPStatus
	result.Message = failure.Message
	result.SafeToSchedule = false
	if failure.Code == "REQUEST_CANCELLED" {
		result.Status = ConnectionTestStatusCancelled
		// 用户主动取消不能据此判定账号不可调度，沿用测试前的账号状态。
		result.SafeToSchedule = account != nil && account.IsSchedulable()
	}
	return result
}
