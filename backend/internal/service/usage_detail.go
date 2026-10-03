package service

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	// UsageDetailCaptureLimit 限制单侧正文采集大小，防止流式响应或媒体内容占满内存和数据库。
	UsageDetailCaptureLimit = 256 * 1024
	// UsageDetailRetention 是详情正文的默认保留期；计费汇总记录不受影响。
	UsageDetailRetention = 7 * 24 * time.Hour
)

// UsageLogDetail 保存一次网关调用的脱敏输入输出，只通过单条详情接口读取。
type UsageLogDetail struct {
	ID                  int64     `json:"id"`
	RequestID           string    `json:"request_id"`
	APIKeyID            int64     `json:"api_key_id"`
	UserID              int64     `json:"user_id"`
	Method              string    `json:"method"`
	Path                string    `json:"path"`
	StatusCode          int       `json:"status_code"`
	RequestContentType  string    `json:"request_content_type"`
	ResponseContentType string    `json:"response_content_type"`
	RequestBody         string    `json:"request_body"`
	ResponseBody        string    `json:"response_body"`
	RequestTruncated    bool      `json:"request_truncated"`
	ResponseTruncated   bool      `json:"response_truncated"`
	CreatedAt           time.Time `json:"created_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// UsageLogDetailRepository 隔离大字段读写，避免扩大现有 UsageLogRepository 的测试桩接口。
type UsageLogDetailRepository interface {
	UpsertUsageLogDetail(ctx context.Context, detail *UsageLogDetail) error
	GetUsageLogDetail(ctx context.Context, requestID string, apiKeyID int64) (*UsageLogDetail, error)
}

// SaveDetail 保存中间件已完成脱敏的请求详情。
func (s *UsageService) SaveDetail(ctx context.Context, detail *UsageLogDetail) error {
	if s == nil || s.detailRepo == nil || detail == nil {
		return nil
	}
	return s.detailRepo.UpsertUsageLogDetail(ctx, detail)
}

// GetDetailForUsage 先读取计费行，再按其幂等键获取独立详情。
func (s *UsageService) GetDetailForUsage(ctx context.Context, usageID int64) (*UsageLog, *UsageLogDetail, error) {
	usage, err := s.GetByID(ctx, usageID)
	if err != nil {
		return nil, nil, err
	}
	if s.detailRepo == nil {
		return usage, nil, nil
	}
	detail, err := s.detailRepo.GetUsageLogDetail(ctx, usage.RequestID, usage.APIKeyID)
	if err != nil {
		return nil, nil, err
	}
	return usage, detail, nil
}

var usageSecretTextPattern = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._~+/-]+|sk-[a-z0-9._-]{8,}`)

// SanitizeUsageDetailBody 保留模型输入输出结构，同时移除凭证、大段二进制和签名 URL 参数。
func SanitizeUsageDetailBody(raw []byte, contentType string, truncated bool) string {
	if len(raw) == 0 {
		return ""
	}
	if truncated && !json.Valid(raw) && !strings.Contains(strings.ToLower(contentType), "event-stream") {
		return "<body omitted: exceeds 262144 bytes>"
	}
	contentType = strings.ToLower(contentType)
	if strings.Contains(contentType, "event-stream") {
		return sanitizeUsageSSE(raw)
	}
	if strings.Contains(contentType, "json") || json.Valid(raw) {
		var value any
		if json.Unmarshal(raw, &value) == nil {
			encoded, err := json.Marshal(sanitizeUsageValue(value, 0))
			if err == nil {
				return string(encoded)
			}
		}
	}
	if strings.Contains(contentType, "multipart") || strings.Contains(contentType, "octet-stream") || strings.HasPrefix(contentType, "audio/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "image/") {
		return "<binary body omitted>"
	}
	return usageSecretTextPattern.ReplaceAllString(string(raw), "***")
}

func sanitizeUsageSSE(raw []byte) string {
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		prefix := ""
		payload := line
		if strings.HasPrefix(line, "data:") {
			prefix = "data:"
			payload = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if payload == "" || payload == "[DONE]" || !json.Valid([]byte(payload)) {
			lines[i] = usageSecretTextPattern.ReplaceAllString(line, "***")
			continue
		}
		var value any
		if json.Unmarshal([]byte(payload), &value) != nil {
			continue
		}
		encoded, err := json.Marshal(sanitizeUsageValue(value, 0))
		if err == nil {
			lines[i] = prefix + string(encoded)
		}
	}
	return strings.Join(lines, "\n")
}

func sanitizeUsageValue(value any, depth int) any {
	if depth > 32 {
		return "<depth limit exceeded>"
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isUsageDetailSecretKey(key) {
				out[key] = "***"
				continue
			}
			out[key] = sanitizeUsageValue(item, depth+1)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = sanitizeUsageValue(item, depth+1)
		}
		return out
	case string:
		return sanitizeUsageString(typed)
	default:
		return value
	}
}

func isUsageDetailSecretKey(key string) bool {
	normalized := auditNormalizeBodyKey(key)
	switch normalized {
	case "authorization", "cookie", "xapikey", "apikey", "apitoken", "token", "password", "passwd", "secret",
		"accesstoken", "refreshtoken", "idtoken", "sessionkey", "privatekey", "serviceaccountjson":
		return true
	default:
		return strings.Contains(normalized, "password") || strings.Contains(normalized, "privatekey") || strings.Contains(normalized, "secretkey")
	}
}

func sanitizeUsageString(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "data:") && strings.Contains(trimmed, ";base64,") {
		return "<data URI omitted>"
	}
	if len(trimmed) > 4096 && looksLikeBase64(trimmed) {
		return "<base64 content omitted>"
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" && parsed.Host != "" && parsed.RawQuery != "" {
		// 查询参数可能包含云厂商签名等非标准凭证，详情只保留资源路径。
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String() + "?<query omitted>"
	}
	return usageSecretTextPattern.ReplaceAllString(value, "***")
}

func looksLikeBase64(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' || r == '\n' || r == '\r' {
			continue
		}
		return false
	}
	return true
}
