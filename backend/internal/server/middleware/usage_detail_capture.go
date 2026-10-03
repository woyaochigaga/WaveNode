package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// UsageDetailCapture 保存成功网关请求的有界、脱敏输入输出，不参与计费主链路。
func UsageDetailCapture(usageService *service.UsageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if usageService == nil || c.Request == nil {
			c.Next()
			return
		}
		// 查询类接口不会产生可关联的请求正文，跳过可减少模型列表、任务轮询等孤立记录。
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}

		requestBody, requestTruncated := captureUsageRequestBody(c)
		originalWriter := c.Writer
		writer := &usageDetailResponseWriter{ResponseWriter: originalWriter}
		c.Writer = writer
		c.Next()
		// 必须在返回外层 Ops 中间件前恢复 writer，避免其释放后留下悬空包装器。
		if c.Writer == writer {
			c.Writer = originalWriter
		}

		// 失败请求由 Ops 错误日志负责；使用详情只关联成功落账记录。
		if writer.Status() >= http.StatusBadRequest {
			return
		}
		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey == nil || apiKey.ID <= 0 || apiKey.UserID <= 0 {
			return
		}
		requestID := usageDetailRequestID(c)
		if requestID == "" {
			return
		}

		responseBody, responseTruncated := writer.captured()
		detail := &service.UsageLogDetail{
			RequestID:           requestID,
			APIKeyID:            apiKey.ID,
			UserID:              apiKey.UserID,
			Method:              c.Request.Method,
			Path:                c.Request.URL.Path,
			StatusCode:          writer.Status(),
			RequestContentType:  c.GetHeader("Content-Type"),
			ResponseContentType: writer.Header().Get("Content-Type"),
			RequestTruncated:    requestTruncated,
			ResponseTruncated:   responseTruncated,
			CreatedAt:           time.Now().UTC(),
		}
		detail.RequestBody = service.SanitizeUsageDetailBody(requestBody, detail.RequestContentType, requestTruncated)
		detail.ResponseBody = service.SanitizeUsageDetailBody(responseBody, detail.ResponseContentType, responseTruncated)

		// 响应已写给客户端，使用独立短超时上下文持久化，避免客户端取消导致详情丢失。
		persistCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := usageService.SaveDetail(persistCtx, detail); err != nil {
			slog.Warn("save usage detail failed", "request_id", requestID, "api_key_id", apiKey.ID, "error", err)
		}
	}
}

func captureUsageRequestBody(c *gin.Context) ([]byte, bool) {
	if c.Request.Body == nil {
		return nil, false
	}
	original := c.Request.Body
	raw, err := io.ReadAll(io.LimitReader(original, service.UsageDetailCaptureLimit+1))
	// 无论读取是否成功都回填已消费前缀，确保后续 handler 仍能读取完整正文。
	c.Request.Body = &usageDetailRestoredBody{
		Reader: io.MultiReader(bytes.NewReader(raw), original),
		closer: original,
	}
	if err != nil {
		return nil, false
	}
	truncated := len(raw) > service.UsageDetailCaptureLimit
	if truncated {
		raw = raw[:service.UsageDetailCaptureLimit]
	}
	return raw, truncated
}

func usageDetailRequestID(c *gin.Context) string {
	if clientID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientID) != "" {
		return "client:" + strings.TrimSpace(clientID)
	}
	if requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return "local:" + strings.TrimSpace(requestID)
	}
	return ""
}

type usageDetailRestoredBody struct {
	io.Reader
	closer io.Closer
}

func (b *usageDetailRestoredBody) Close() error { return b.closer.Close() }

type usageDetailResponseWriter struct {
	gin.ResponseWriter
	mu        sync.Mutex
	body      []byte
	truncated bool
}

func (w *usageDetailResponseWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *usageDetailResponseWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *usageDetailResponseWriter) capture(data []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := service.UsageDetailCaptureLimit - len(w.body)
	if remaining <= 0 {
		if len(data) > 0 {
			w.truncated = true
		}
		return
	}
	if len(data) > remaining {
		w.body = append(w.body, data[:remaining]...)
		w.truncated = true
		return
	}
	w.body = append(w.body, data...)
}

func (w *usageDetailResponseWriter) captured() ([]byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.body...), w.truncated
}

// Unwrap 允许 net/http ResponseController 继续访问底层流式 writer 能力。
func (w *usageDetailResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
