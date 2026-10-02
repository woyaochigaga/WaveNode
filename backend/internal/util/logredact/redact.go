package logredact

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// maxRedactDepth 限制递归深度以防止栈溢出
const maxRedactDepth = 32

var defaultSensitiveKeyList = []string{
	"authorization",
	"proxy_authorization",
	"authorization_code",
	"code",
	"code_verifier",
	"access_token",
	"refresh_token",
	"id_token",
	"client_secret",
	"password",
	"proxy_password",
	"proxy_key",
	"api_key",
	"x_api_key",
	"token",
	"secret",
	"cookie",
	"set_cookie",
	"session",
	"session_key",
	"private_key",
	"service_account_json",
	"credential",
	"credentials",
	"prompt",
	"raw_prompt",
	"request_body",
	"response_body",
}

var defaultSensitiveKeys = func() map[string]struct{} {
	keys := make(map[string]struct{}, len(defaultSensitiveKeyList))
	for _, key := range defaultSensitiveKeyList {
		keys[normalizeKey(key)] = struct{}{}
	}
	return keys
}()

type textRedactPatterns struct {
	reJSONLike  *regexp.Regexp
	reQueryLike *regexp.Regexp
	rePlain     *regexp.Regexp
}

var (
	reGOCSPX = regexp.MustCompile(`GOCSPX-[0-9A-Za-z_-]{24,}`)
	reAIza   = regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`)
	// Header 值可能包含空格，通用 key:value 规则只能遮住 Bearer/Basic 字样，
	// 因此先单独处理完整认证头和 Cookie 行。
	reAuthorizationHeader = regexp.MustCompile(`(?i)\b((?:proxy[-_ ]?)?authorization\s*[:=]\s*(?:bearer|basic)\s+)[^\s,;]+`)
	reCookieHeader        = regexp.MustCompile(`(?i)\b((?:set[-_ ]?)?cookie\s*[:=]\s*)[^\r\n]+`)
	reProxyCredentials    = regexp.MustCompile(`(?i)\b((?:https?|socks5h?|socks4a?)://[^:\s/@]+:)[^@\s]+(@)`)
	rePrivateKey          = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)

	defaultTextRedactPatterns = compileTextRedactPatterns(nil)
	extraTextPatternCache     sync.Map // map[string]*textRedactPatterns
)

func RedactMap(input map[string]any, extraKeys ...string) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	keys := buildKeySet(extraKeys)
	redacted, ok := redactValueWithDepth(input, keys, 0).(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return redacted
}

// RedactValue 递归复制并脱敏任意结构化值，供日志、审计和错误响应等统一出口使用。
// map 的敏感键会整体替换，普通字符串仍会继续检查 Header、URL 和私钥等文本模式。
func RedactValue(input any, extraKeys ...string) any {
	return redactValueWithDepth(input, buildKeySet(extraKeys), 0)
}

func RedactJSON(raw []byte, extraKeys ...string) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "<non-json payload redacted>"
	}
	keys := buildKeySet(extraKeys)
	redacted := redactValueWithDepth(value, keys, 0)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		return "<redacted>"
	}
	return string(encoded)
}

// RedactText 对非结构化文本做轻量脱敏。
//
// 规则：
// - 如果文本本身是 JSON，则按 RedactJSON 处理。
// - 否则尝试对常见 key=value / key:"value" 片段做脱敏。
//
// 注意：该函数用于日志/错误信息兜底，不保证覆盖所有格式。
func RedactText(input string, extraKeys ...string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	raw := []byte(input)
	if json.Valid(raw) {
		return RedactJSON(raw, extraKeys...)
	}

	patterns := getTextRedactPatterns(extraKeys)

	out := input
	out = rePrivateKey.ReplaceAllString(out, "<private key redacted>")
	out = reAuthorizationHeader.ReplaceAllString(out, `${1}***`)
	out = reCookieHeader.ReplaceAllString(out, `${1}***`)
	out = reProxyCredentials.ReplaceAllString(out, `${1}***${2}`)
	out = reGOCSPX.ReplaceAllString(out, "GOCSPX-***")
	out = reAIza.ReplaceAllString(out, "AIza***")
	out = patterns.reJSONLike.ReplaceAllString(out, `$1***$3`)
	out = patterns.reQueryLike.ReplaceAllString(out, `$1=***`)
	out = patterns.rePlain.ReplaceAllString(out, `$1$2***`)
	return out
}

func compileTextRedactPatterns(extraKeys []string) *textRedactPatterns {
	keyAlt := buildKeyAlternation(extraKeys)
	return &textRedactPatterns{
		// JSON-like: "access_token":"..."
		reJSONLike: regexp.MustCompile(`(?i)("(?:` + keyAlt + `)"\s*:\s*")([^"]*)(")`),
		// Query-like: access_token=...
		reQueryLike: regexp.MustCompile(`(?i)\b((?:` + keyAlt + `))=([^&\s]+)`),
		// Plain: access_token: ... / access_token = ...
		rePlain: regexp.MustCompile(`(?i)\b((?:` + keyAlt + `))\b(\s*[:=]\s*)([^,\s]+)`),
	}
}

func getTextRedactPatterns(extraKeys []string) *textRedactPatterns {
	normalizedExtraKeys := normalizeAndSortExtraKeys(extraKeys)
	if len(normalizedExtraKeys) == 0 {
		return defaultTextRedactPatterns
	}

	cacheKey := strings.Join(normalizedExtraKeys, ",")
	if cached, ok := extraTextPatternCache.Load(cacheKey); ok {
		if patterns, ok := cached.(*textRedactPatterns); ok {
			return patterns
		}
	}

	// 缓存键折叠大小写和首尾空白，正则保留调用方的字段分隔方式。
	compiled := compileTextRedactPatterns(extraKeys)
	actual, _ := extraTextPatternCache.LoadOrStore(cacheKey, compiled)
	if patterns, ok := actual.(*textRedactPatterns); ok {
		return patterns
	}
	return compiled
}

func normalizeAndSortExtraKeys(extraKeys []string) []string {
	if len(extraKeys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(extraKeys))
	keys := make([]string, 0, len(extraKeys))
	for _, key := range extraKeys {
		// 正则缓存只合并大小写和首尾空白相同的写法；分隔符不同的扩展键
		// 单独编译，避免先缓存 customsecret 后漏掉 custom_secret。
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		keys = append(keys, normalized)
	}
	sort.Strings(keys)
	return keys
}

func buildKeyAlternation(extraKeys []string) string {
	seen := make(map[string]struct{}, len(defaultSensitiveKeyList)+len(extraKeys))
	keys := make([]string, 0, len(defaultSensitiveKeyList)+len(extraKeys))
	for _, k := range defaultSensitiveKeyList {
		n := normalizeKey(k)
		seen[n] = struct{}{}
		keys = append(keys, sensitiveKeyPattern(k))
	}
	for _, k := range extraKeys {
		n := normalizeKey(k)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		keys = append(keys, sensitiveKeyPattern(k))
	}
	return strings.Join(keys, "|")
}

// sensitiveKeyPattern 允许 snake_case、kebab-case、空格和 camelCase 共用一条规则。
func sensitiveKeyPattern(key string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(key)), func(r rune) bool {
		switch r {
		case '_', '-', '.', ' ':
			return true
		default:
			return false
		}
	})
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	return strings.Join(parts, `[\s_.-]*`)
}

func buildKeySet(extraKeys []string) map[string]struct{} {
	keys := make(map[string]struct{}, len(defaultSensitiveKeys)+len(extraKeys))
	for k := range defaultSensitiveKeys {
		keys[k] = struct{}{}
	}
	for _, key := range extraKeys {
		normalized := normalizeKey(key)
		if normalized == "" {
			continue
		}
		keys[normalized] = struct{}{}
	}
	return keys
}

func redactValueWithDepth(value any, keys map[string]struct{}, depth int) any {
	if depth > maxRedactDepth {
		return "<depth limit exceeded>"
	}

	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			if isSensitiveKey(k, keys) {
				out[k] = "***"
				continue
			}
			out[k] = redactValueWithDepth(val, keys, depth+1)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = redactValueWithDepth(item, keys, depth+1)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, val := range v {
			if isSensitiveKey(k, keys) {
				out[k] = "***"
				continue
			}
			out[k] = RedactText(val)
		}
		return out
	case []string:
		out := make([]string, len(v))
		for i, item := range v {
			out[i] = RedactText(item)
		}
		return out
	case string:
		return RedactText(v)
	case []byte:
		return []byte(RedactText(string(v)))
	default:
		if redacted, ok := redactCompositeValue(value, keys, depth); ok {
			return redacted
		}
		return value
	}
}

// redactCompositeValue 通过 JSON 语义处理具名 map/slice、DTO、指针和 RawMessage。
// 无法安全展开的复合值直接替换，按 fail-close 原则避免自定义类型绕过结构化脱敏。
func redactCompositeValue(value any, keys map[string]struct{}, depth int) (any, bool) {
	if value == nil {
		return nil, false
	}
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct, reflect.Pointer, reflect.Interface:
	default:
		return nil, false
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return "<unserializable value redacted>", true
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return "<unserializable value redacted>", true
	}
	return redactValueWithDepth(generic, keys, depth+1), true
}

func isSensitiveKey(key string, keys map[string]struct{}) bool {
	_, ok := keys[normalizeKey(key)]
	return ok
}

func normalizeKey(key string) string {
	var normalized strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(key)) {
		switch r {
		case '_', '-', '.', ' ':
			continue
		default:
			_, _ = normalized.WriteRune(r)
		}
	}
	return normalized.String()
}
