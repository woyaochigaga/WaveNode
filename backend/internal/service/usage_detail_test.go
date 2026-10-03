package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeUsageDetailBodyPreservesContentAndTokenCounts(t *testing.T) {
	raw := []byte(`{
		"model":"gpt-test",
		"messages":[{"role":"user","content":"hello"}],
		"input_tokens":123,
		"authorization":"Bearer secret-value",
		"api_key":"sk-super-secret-value"
	}`)
	out := SanitizeUsageDetailBody(raw, "application/json", false)

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("sanitized output is not JSON: %v", err)
	}
	if decoded["authorization"] != "***" || decoded["api_key"] != "***" {
		t.Fatalf("credentials were not redacted: %s", out)
	}
	if decoded["input_tokens"] != float64(123) || !strings.Contains(out, "hello") {
		t.Fatalf("usage content was unexpectedly removed: %s", out)
	}
}

func TestSanitizeUsageDetailBodyClassifiesSSEAndOmitsDataURI(t *testing.T) {
	raw := []byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\n" +
		"data: {\"type\":\"image\",\"image_url\":\"data:image/png;base64,AAAA\"}\n\n")
	out := SanitizeUsageDetailBody(raw, "text/event-stream", false)
	if !strings.Contains(out, `"delta":"hi"`) {
		t.Fatalf("SSE text delta missing: %s", out)
	}
	if strings.Contains(out, "base64,AAAA") || !strings.Contains(out, `\u003cdata URI omitted\u003e`) {
		t.Fatalf("data URI was not omitted: %s", out)
	}
}

func TestSanitizeUsageDetailBodyRemovesSignedURLQuery(t *testing.T) {
	raw := []byte(`{"image_url":"https://cdn.example.com/file.png?X-Amz-Signature=secret&expires=123"}`)
	out := SanitizeUsageDetailBody(raw, "application/json", false)
	if strings.Contains(out, "secret") || strings.Contains(out, "expires=123") {
		t.Fatalf("signed URL query was not removed: %s", out)
	}
	if !strings.Contains(out, "cdn.example.com/file.png") {
		t.Fatalf("signed URL resource path should be preserved: %s", out)
	}
}

func TestSanitizeUsageDetailBodyRejectsTruncatedJSON(t *testing.T) {
	out := SanitizeUsageDetailBody([]byte(`{"messages":[`), "application/json", true)
	if !strings.Contains(out, "exceeds") {
		t.Fatalf("truncated JSON should use an omission marker: %s", out)
	}
}
