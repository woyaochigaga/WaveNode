package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

const (
	dingTalkAppTokenSafetyMargin = 200 * time.Second
	dingTalkLocalTokenMaxTTL     = 30 * time.Second
	dingTalkAppTokenLockTTL      = 15 * time.Second
	dingTalkAppTokenWaitTimeout  = 10 * time.Second
)

// dingTalkClientConfig 是 DingTalkClient 需要的最小配置子集
type dingTalkClientConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	UserInfoURL  string
}

type DingTalkClient struct {
	cfg         dingTalkClientConfig
	appToken    string
	appTokenExp time.Time // 钉钉 7200s，留 200s 余量 → 7000s
	mu          sync.Mutex
	httpClient  *http.Client
	sharedCache service.DingTalkAppTokenCache
}

type DingTalkUserTokenResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpireIn     int64  `json:"expireIn"`
	CorpID       string `json:"corpId"`
}

func (c *DingTalkClient) ExchangeCodeForUserToken(ctx context.Context, code string) (*DingTalkUserTokenResp, error) {
	body := map[string]string{
		"clientId":     c.cfg.ClientID,
		"clientSecret": c.cfg.ClientSecret,
		"code":         code,
		"grantType":    "authorization_code",
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, parseDingTalkErr(raw, resp.StatusCode)
	}
	var out DingTalkUserTokenResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, parseDingTalkErr(raw, resp.StatusCode)
	}
	return &out, nil
}

type DingTalkAPIError struct {
	Code    string
	Message string
	HTTP    int
}

func (e *DingTalkAPIError) Error() string {
	return fmt.Sprintf("dingtalk api error code=%s msg=%s http=%d", e.Code, e.Message, e.HTTP)
}

func parseDingTalkErr(raw []byte, status int) error {
	var v struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(raw, &v)
	code := v.Code
	if code == "" && v.ErrCode != 0 {
		code = fmt.Sprintf("%d", v.ErrCode)
	}
	msg := v.Message
	if msg == "" {
		msg = v.ErrMsg
	}
	return &DingTalkAPIError{Code: code, Message: msg, HTTP: status}
}

// GetUnionIdByUserToken 调用 /v1.0/contact/users/me 返回 unionId 与用户自设昵称 nick。
// nick 来自钉钉新版 OIDC 接口（用户在 App 个人资料填的昵称），与旧版 user/get.nickname 不同源。
func (c *DingTalkClient) GetUnionIdByUserToken(ctx context.Context, userToken string) (unionID string, nick string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.UserInfoURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("x-acs-dingtalk-access-token", userToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", parseDingTalkErr(raw, resp.StatusCode)
	}
	var v struct {
		UnionID string `json:"unionId"`
		Nick    string `json:"nick"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(v.UnionID) == "" {
		return "", "", parseDingTalkErr(raw, resp.StatusCode)
	}
	return v.UnionID, v.Nick, nil
}

type DingTalkStaffInfo struct {
	UserID   string
	Name     string // 企业内真实姓名（钉钉企业管理后台配置）
	Nickname string // 钉钉个人昵称（用户自己设置）
	Email    string
	DeptIDs  []int64
	// CorpID 不来自 staff 接口，来自 userToken；不在此 struct
}

// dingTalkOAPIBase 推导钉钉旧版 OAPI base URL（host: api.dingtalk.com → oapi.dingtalk.com）。
// getbyunionid 与 topapi/v2/user/get 仅在旧版 OAPI 提供，不在 v1.0 OpenAPI。
func (c *DingTalkClient) dingTalkOAPIBase() string {
	u, err := url.Parse(c.cfg.UserInfoURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "https://oapi.dingtalk.com"
	}
	host := u.Host
	if strings.HasPrefix(host, "api.") {
		host = "oapi." + strings.TrimPrefix(host, "api.")
	}
	return u.Scheme + "://" + host
}

func (c *DingTalkClient) GetAppToken(ctx context.Context) (string, error) {
	cacheKey := c.appTokenCacheKey()
	var sharedErr error
	if c.sharedCache != nil {
		token, ttl, found, err := c.sharedCache.Get(ctx, cacheKey)
		if err == nil && found && strings.TrimSpace(token) != "" {
			c.rememberLocalAppToken(token, ttl)
			return token, nil
		}
		sharedErr = err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Redis 不可用时才使用短时本地副本；Redis 明确 miss 通常表示另一个实例
	// 已执行失效，此时不能继续信任旧 token。
	if (c.sharedCache == nil || sharedErr != nil) && c.localAppTokenValid() {
		return c.appToken, nil
	}
	if c.sharedCache != nil && sharedErr == nil {
		token, ttl, found, err := c.sharedCache.Get(ctx, cacheKey)
		if err == nil && found && strings.TrimSpace(token) != "" {
			c.setLocalAppTokenLocked(token, ttl)
			return token, nil
		}
		if err != nil {
			sharedErr = err
			if c.localAppTokenValid() {
				return c.appToken, nil
			}
		}
	}

	var lockOwner string
	lockAcquired := false
	if c.sharedCache != nil && sharedErr == nil {
		lockOwner = uuid.NewString()
		acquired, err := c.sharedCache.TryAcquireRefresh(ctx, cacheKey, lockOwner, dingTalkAppTokenLockTTL)
		if err != nil {
			sharedErr = err
		} else if !acquired {
			token, ttl, waitErr := c.waitForSharedAppToken(ctx, cacheKey)
			if waitErr != nil {
				return "", waitErr
			}
			c.setLocalAppTokenLocked(token, ttl)
			return token, nil
		} else {
			lockAcquired = true
			defer c.releaseSharedAppTokenLock(cacheKey, lockOwner)
		}
	}

	token, ttl, err := c.fetchAppToken(ctx)
	if err != nil {
		return "", err
	}
	c.setLocalAppTokenLocked(token, ttl)
	if c.sharedCache != nil && (lockAcquired || sharedErr == nil) {
		if err := c.sharedCache.Set(ctx, cacheKey, token, ttl); err != nil {
			slog.Warn("dingtalk_app_token_cache_set_failed", "error", err)
		}
	}
	return token, nil
}

func (c *DingTalkClient) fetchAppToken(ctx context.Context) (string, time.Duration, error) {
	body := map[string]string{"appKey": c.cfg.ClientID, "appSecret": c.cfg.ClientSecret}
	payload, _ := json.Marshal(body)
	// 钉钉新版 v1.0 企业内部应用 access_token: POST /v1.0/oauth2/accessToken
	// 此 token 也可作为旧版 OAPI 的 access_token 使用（钉钉文档已说明）
	appTokenURL := strings.Replace(c.cfg.TokenURL, "/oauth2/userAccessToken", "/oauth2/accessToken", 1)
	if !strings.Contains(appTokenURL, "accessToken") && !strings.Contains(appTokenURL, "gettoken") {
		appTokenURL = c.cfg.TokenURL // fallback for test stub
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appTokenURL, bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", 0, parseDingTalkErr(raw, resp.StatusCode)
	}
	var v struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int64  `json:"expireIn"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", 0, err
	}
	if v.AccessToken == "" {
		return "", 0, parseDingTalkErr(raw, resp.StatusCode)
	}
	ttl := time.Duration(v.ExpireIn) * time.Second
	if ttl > dingTalkAppTokenSafetyMargin {
		ttl -= dingTalkAppTokenSafetyMargin
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	return v.AccessToken, ttl, nil
}

func (c *DingTalkClient) appTokenCacheKey() string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(c.cfg.ClientID) + "|" + strings.TrimSpace(c.cfg.TokenURL)))
	return fmt.Sprintf("%x", digest[:])
}

func (c *DingTalkClient) localAppTokenValid() bool {
	return c.appToken != "" && time.Now().Before(c.appTokenExp)
}

func (c *DingTalkClient) rememberLocalAppToken(token string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocalAppTokenLocked(token, ttl)
}

func (c *DingTalkClient) setLocalAppTokenLocked(token string, ttl time.Duration) {
	if ttl > dingTalkLocalTokenMaxTTL {
		ttl = dingTalkLocalTokenMaxTTL
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	c.appToken = token
	c.appTokenExp = time.Now().Add(ttl)
}

func (c *DingTalkClient) waitForSharedAppToken(ctx context.Context, cacheKey string) (string, time.Duration, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(dingTalkAppTokenWaitTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", 0, ctx.Err()
		case <-timer.C:
			return "", 0, fmt.Errorf("dingtalk app token refresh is still in progress")
		case <-ticker.C:
			token, ttl, found, err := c.sharedCache.Get(ctx, cacheKey)
			if err != nil {
				return "", 0, fmt.Errorf("read shared dingtalk app token: %w", err)
			}
			if found && strings.TrimSpace(token) != "" {
				return token, ttl, nil
			}
		}
	}
}

func (c *DingTalkClient) releaseSharedAppTokenLock(cacheKey, owner string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.sharedCache.ReleaseRefresh(ctx, cacheKey, owner); err != nil {
		slog.Warn("dingtalk_app_token_lock_release_failed", "error", err)
	}
}

// doAppTokenRequest 统一处理旧版 OAPI 请求。若钉钉明确告知 appToken 无效，
// 会清除本地和共享缓存、重新取 token，并且只重试一次，避免无限循环。
func (c *DingTalkClient) doAppTokenRequest(
	ctx context.Context,
	payload []byte,
	buildURL func(appToken string) string,
) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		appToken, err := c.GetAppToken(ctx)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildURL(appToken), bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		apiErr := dingTalkResponseError(raw, resp.StatusCode)
		if apiErr == nil {
			return raw, nil
		}
		if attempt == 0 && isDingTalkAppTokenInvalid(apiErr) {
			c.invalidateAppToken(ctx, appToken)
			continue
		}
		return nil, apiErr
	}
	return nil, fmt.Errorf("dingtalk app token retry exhausted")
}

func dingTalkResponseError(raw []byte, status int) error {
	if status != http.StatusOK {
		return parseDingTalkErr(raw, status)
	}
	var envelope struct {
		ErrCode int `json:"errcode"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.ErrCode != 0 {
		return parseDingTalkErr(raw, status)
	}
	return nil
}

func isDingTalkAppTokenInvalid(err error) bool {
	var apiErr *DingTalkAPIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case "40014", "50015", "88":
		return true
	default:
		return false
	}
}

func (c *DingTalkClient) invalidateAppToken(ctx context.Context, expectedToken string) {
	c.mu.Lock()
	if c.appToken == expectedToken {
		c.appToken = ""
		c.appTokenExp = time.Time{}
	}
	c.mu.Unlock()
	if c.sharedCache != nil {
		if err := c.sharedCache.Delete(ctx, c.appTokenCacheKey()); err != nil {
			slog.Warn("dingtalk_app_token_cache_delete_failed", "error", err)
		}
	}
}

func (c *DingTalkClient) GetUserIdByUnionId(ctx context.Context, unionID string) (string, error) {
	body := map[string]string{"unionid": unionID}
	payload, _ := json.Marshal(body)
	// 钉钉旧版 OAPI: POST https://oapi.dingtalk.com/topapi/user/getbyunionid?access_token=XXX
	// access_token 通过 query string 传递（不是 header）
	raw, err := c.doAppTokenRequest(ctx, payload, func(appToken string) string {
		if strings.Contains(c.cfg.UserInfoURL, "/contact/users/me") {
			return c.dingTalkOAPIBase() + "/topapi/user/getbyunionid?access_token=" + url.QueryEscape(appToken)
		}
		return c.cfg.UserInfoURL
	})
	if err != nil {
		return "", err
	}
	var v struct {
		Result struct {
			UserID string `json:"userid"`
		} `json:"result"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	if strings.TrimSpace(v.Result.UserID) == "" {
		return "", parseDingTalkErr(raw, http.StatusOK)
	}
	return v.Result.UserID, nil
}

// DingTalkDeptInfo 部门信息（topapi/v2/department/get 返回子集）
type DingTalkDeptInfo struct {
	DeptID   int64
	Name     string
	ParentID int64
}

// GetDeptInfo 查询单个部门信息（用于递归拼部门路径）。
// 调用钉钉旧版 OAPI: POST /topapi/v2/department/get?access_token=XXX
func (c *DingTalkClient) GetDeptInfo(ctx context.Context, deptID int64) (*DingTalkDeptInfo, error) {
	body := map[string]any{"dept_id": deptID, "language": "zh_CN"}
	payload, _ := json.Marshal(body)
	raw, err := c.doAppTokenRequest(ctx, payload, func(appToken string) string {
		if strings.Contains(c.cfg.UserInfoURL, "/contact/users/me") {
			return c.dingTalkOAPIBase() + "/topapi/v2/department/get?access_token=" + url.QueryEscape(appToken)
		}
		return c.cfg.UserInfoURL
	})
	if err != nil {
		return nil, err
	}
	var v struct {
		Result struct {
			DeptID   int64  `json:"dept_id"`
			Name     string `json:"name"`
			ParentID int64  `json:"parent_id"`
		} `json:"result"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &DingTalkDeptInfo{
		DeptID:   v.Result.DeptID,
		Name:     v.Result.Name,
		ParentID: v.Result.ParentID,
	}, nil
}

func (c *DingTalkClient) GetStaffInfoByUserId(ctx context.Context, userID string) (*DingTalkStaffInfo, error) {
	body := map[string]string{"userid": userID}
	payload, _ := json.Marshal(body)
	// 钉钉旧版 OAPI: POST https://oapi.dingtalk.com/topapi/v2/user/get?access_token=XXX
	raw, err := c.doAppTokenRequest(ctx, payload, func(appToken string) string {
		if strings.Contains(c.cfg.UserInfoURL, "/contact/users/me") {
			return c.dingTalkOAPIBase() + "/topapi/v2/user/get?access_token=" + url.QueryEscape(appToken)
		}
		return c.cfg.UserInfoURL
	})
	if err != nil {
		return nil, err
	}
	var v struct {
		Result struct {
			UserID    string  `json:"userid"`
			Name      string  `json:"name"`
			Nickname  string  `json:"nickname"`
			Email     string  `json:"email"`
			OrgEmail  string  `json:"org_email"`
			Extension string  `json:"extension"`
			DeptID    []int64 `json:"dept_id_list"`
		} `json:"result"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	if strings.TrimSpace(v.Result.UserID) == "" {
		return nil, parseDingTalkErr(raw, http.StatusOK)
	}
	// 邮箱三级 fallback：org_email > email > extension["企业邮箱"]（钉钉自定义扩展字段，JSON string）
	email := strings.TrimSpace(v.Result.OrgEmail)
	emailSource := "org_email"
	if email == "" {
		email = strings.TrimSpace(v.Result.Email)
		emailSource = "email"
	}
	extensionParsed := false
	if email == "" && strings.TrimSpace(v.Result.Extension) != "" {
		var ext map[string]string
		if err := json.Unmarshal([]byte(v.Result.Extension), &ext); err == nil {
			extensionParsed = true
			if v, ok := ext["企业邮箱"]; ok {
				email = strings.TrimSpace(v)
				emailSource = "extension.企业邮箱"
			}
		}
	}
	if email == "" {
		emailSource = "none"
	}
	slog.Info("dingtalk staff fetched",
		"userid", v.Result.UserID,
		"name_present", v.Result.Name != "",
		"nickname_present", v.Result.Nickname != "",
		"name_eq_nickname", v.Result.Name != "" && v.Result.Name == v.Result.Nickname,
		"email_present", v.Result.Email != "",
		"org_email_present", v.Result.OrgEmail != "",
		"extension_present", v.Result.Extension != "",
		"extension_parsed", extensionParsed,
		"email_source", emailSource,
		"dept_count", len(v.Result.DeptID),
	)
	return &DingTalkStaffInfo{
		UserID:   v.Result.UserID,
		Name:     v.Result.Name,
		Nickname: v.Result.Nickname,
		Email:    email,
		DeptIDs:  v.Result.DeptID,
	}, nil
}
