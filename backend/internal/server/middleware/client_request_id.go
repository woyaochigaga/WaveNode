package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const clientRequestIDHeader = "X-Client-Request-ID"

const (
	gatewayErrorCodeHeader = "X-Sub2API-Error-Code"
	gatewayRetryableHeader = "X-Sub2API-Retryable"
)

type gatewayErrorMetadataWriter struct {
	gin.ResponseWriter
}

// WriteHeader 在协议响应体写出前补充安全诊断头，不改变 OpenAI、Anthropic 或 Gemini 的 JSON 结构。
func (w *gatewayErrorMetadataWriter) WriteHeader(statusCode int) {
	if statusCode >= http.StatusBadRequest {
		w.Header().Set(gatewayErrorCodeHeader, safeGatewayErrorCode(statusCode))
		retryable := gatewayStatusRetryable(statusCode)
		w.Header().Set(gatewayRetryableHeader, strconv.FormatBool(retryable))
		if statusCode == http.StatusTooManyRequests && w.Header().Get("Retry-After") == "" {
			w.Header().Set("Retry-After", "1")
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// ClientRequestID ensures every request has a unique client_request_id in request.Context().
//
// This is used by the Ops monitoring module for end-to-end request correlation.
func ClientRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil {
			c.Next()
			return
		}

		if _, wrapped := c.Writer.(*gatewayErrorMetadataWriter); !wrapped {
			c.Writer = &gatewayErrorMetadataWriter{ResponseWriter: c.Writer}
		}

		if v, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(v) != "" {
			var valid bool
			v, valid = normalizeCorrelationID(v)
			if !valid {
				v = uuid.New().String()
			}
			c.Header(clientRequestIDHeader, v)
			ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, v)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		id := uuid.New().String()
		c.Header(clientRequestIDHeader, id)
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, id)
		requestLogger := logger.FromContext(ctx).With(zap.String("client_request_id", strings.TrimSpace(id)))
		ctx = logger.IntoContext(ctx, requestLogger)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func gatewayStatusRetryable(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func safeGatewayErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest, http.StatusConflict, http.StatusUnprocessableEntity:
		return "request_invalid"
	case http.StatusUnauthorized:
		return "auth_invalid"
	case http.StatusForbidden:
		return "permission_denied"
	case http.StatusNotFound:
		return "model_unavailable"
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return "upstream_timeout"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusTooEarly, http.StatusBadGateway, http.StatusServiceUnavailable:
		return "upstream_overloaded"
	default:
		return "internal_error"
	}
}
