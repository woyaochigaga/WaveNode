package logredact

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactText_JSONLike(t *testing.T) {
	in := `{"access_token":"ya29.a0AfH6SMDUMMY","refresh_token":"1//0gDUMMY","other":"ok"}`
	out := RedactText(in)
	if out == in {
		t.Fatalf("expected redaction, got unchanged")
	}
	if want := `"access_token":"***"`; !strings.Contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
	if want := `"refresh_token":"***"`; !strings.Contains(out, want) {
		t.Fatalf("expected %q in %q", want, out)
	}
}

func TestRedactText_QueryLike(t *testing.T) {
	in := "access_token=ya29.a0AfH6SMDUMMY refresh_token=1//0gDUMMY"
	out := RedactText(in)
	if strings.Contains(out, "ya29") || strings.Contains(out, "1//0") {
		t.Fatalf("expected tokens redacted, got %q", out)
	}
}

func TestRedactText_GOCSPX(t *testing.T) {
	in := "client_secret=GOCSPX-your-client-secret"
	out := RedactText(in)
	if strings.Contains(out, "your-client-secret") {
		t.Fatalf("expected secret redacted, got %q", out)
	}
	if !strings.Contains(out, "client_secret=***") {
		t.Fatalf("expected key redacted, got %q", out)
	}
}

func TestRedactText_ExtraKeyCacheUsesNormalizedSortedKey(t *testing.T) {
	clearExtraTextPatternCache()

	out1 := RedactText("custom_secret=abc", "Custom_Secret", " custom_secret ")
	out2 := RedactText("custom_secret=xyz", "custom_secret")
	if !strings.Contains(out1, "custom_secret=***") {
		t.Fatalf("expected custom key redacted in first call, got %q", out1)
	}
	if !strings.Contains(out2, "custom_secret=***") {
		t.Fatalf("expected custom key redacted in second call, got %q", out2)
	}

	if got := countExtraTextPatternCacheEntries(); got != 1 {
		t.Fatalf("expected 1 cached pattern set, got %d", got)
	}
}

func TestRedactText_DefaultPathDoesNotUseExtraCache(t *testing.T) {
	clearExtraTextPatternCache()

	out := RedactText("access_token=abc")
	if !strings.Contains(out, "access_token=***") {
		t.Fatalf("expected default key redacted, got %q", out)
	}
	if got := countExtraTextPatternCacheEntries(); got != 0 {
		t.Fatalf("expected extra cache to remain empty, got %d", got)
	}
}

func TestSensitiveLeakageCanaryRedactsHighRiskFormats(t *testing.T) {
	const canary = "sub2api-canary-secret-7f19"
	privateKey := "-----BEGIN PRIVATE KEY-----\n" + canary + "\n-----END PRIVATE KEY-----"

	tests := []struct {
		name  string
		input string
	}{
		{name: "authorization", input: "Authorization: Bearer " + canary},
		{name: "proxy_authorization", input: "Proxy-Authorization: Basic " + canary},
		{name: "cookie", input: "Cookie: session=" + canary + "; theme=dark"},
		{name: "proxy_url", input: "proxy=http://worker:" + canary + "@127.0.0.1:8080"},
		{name: "refresh_token", input: "refreshToken=" + canary},
		{name: "raw_prompt", input: "raw_prompt=" + canary},
		{name: "private_key", input: privateKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RedactText(tt.input)
			if strings.Contains(output, canary) {
				t.Fatalf("完整 canary 不应出现在脱敏结果中: %q", output)
			}
		})
	}
}

func TestSensitiveLeakageCanaryRedactsNestedStructuredValues(t *testing.T) {
	const canary = "sub2api-canary-secret-structured"
	input := map[string]any{
		"safe": "ok",
		"payload": map[string]any{
			"apiKey": canary,
			"nested": []any{map[string]any{"proxy-password": canary}},
		},
		"detail": "Authorization: Bearer " + canary,
	}

	output := RedactMap(input)
	encoded := string(mustJSON(t, output))
	if strings.Contains(encoded, canary) {
		t.Fatalf("嵌套结构泄漏完整 canary: %s", encoded)
	}
	if output["safe"] != "ok" {
		t.Fatalf("非敏感字段不应被改写: %#v", output)
	}
}

func TestSensitiveLeakageCanaryRedactsTypedStructuredValues(t *testing.T) {
	type credentialPayload struct {
		AccessToken string `json:"accessToken"`
		Label       string `json:"label"`
	}
	const canary = "sub2api-canary-secret-typed"

	output := RedactValue(credentialPayload{AccessToken: canary, Label: "primary"})
	encoded := string(mustJSON(t, output))
	if strings.Contains(encoded, canary) {
		t.Fatalf("具名结构泄漏完整 canary: %s", encoded)
	}
	if !strings.Contains(encoded, "primary") {
		t.Fatalf("非敏感字段不应丢失: %s", encoded)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return encoded
}

func clearExtraTextPatternCache() {
	extraTextPatternCache.Range(func(key, value any) bool {
		extraTextPatternCache.Delete(key)
		return true
	})
}

func countExtraTextPatternCacheEntries() int {
	count := 0
	extraTextPatternCache.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
