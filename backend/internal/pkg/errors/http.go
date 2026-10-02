package errors

import "net/http"

// ToHTTP converts an error into an HTTP status code and a JSON-serializable body.
//
// The returned body matches the project's Status shape:
// { code, reason, message, metadata }.
func ToHTTP(err error) (statusCode int, body Status) {
	if err == nil {
		return http.StatusOK, Status{Code: int32(http.StatusOK)}
	}

	appErr := FromError(err)
	if appErr == nil {
		return http.StatusOK, Status{Code: int32(http.StatusOK)}
	}

	retryable := defaultRetryableHTTPStatus(int(appErr.Code))
	if appErr.RetryPolicySet {
		retryable = appErr.Retryable
	}
	body = Status{
		Code:              appErr.Code,
		Reason:            appErr.Reason,
		Message:           appErr.Message,
		Retryable:         retryable,
		RetryAfterSeconds: appErr.RetryAfterSeconds,
	}
	if appErr.Metadata != nil {
		body.Metadata = make(map[string]string, len(appErr.Metadata))
		for k, v := range appErr.Metadata {
			body.Metadata[k] = v
		}
	}
	return int(appErr.Code), body
}

// defaultRetryableHTTPStatus 为尚未声明重试策略的旧错误提供保守默认值。
func defaultRetryableHTTPStatus(status int) bool {
	switch status {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}
