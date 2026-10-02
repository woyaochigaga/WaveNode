package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dingTalkSharedCacheStub struct {
	mu          sync.Mutex
	token       string
	ttl         time.Duration
	lockOwner   string
	deleteCalls int
}

func (c *dingTalkSharedCacheStub) Get(context.Context, string) (string, time.Duration, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token, c.ttl, c.token != "", nil
}

func (c *dingTalkSharedCacheStub) Set(_ context.Context, _ string, token string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	c.ttl = ttl
	return nil
}

func (c *dingTalkSharedCacheStub) Delete(context.Context, string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.ttl = 0
	c.deleteCalls++
	return nil
}

func (c *dingTalkSharedCacheStub) TryAcquireRefresh(_ context.Context, _ string, owner string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lockOwner != "" {
		return false, nil
	}
	c.lockOwner = owner
	return true, nil
}

func (c *dingTalkSharedCacheStub) ReleaseRefresh(_ context.Context, _ string, owner string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lockOwner == owner {
		c.lockOwner = ""
	}
	return nil
}

func TestDingTalkClient_ExchangeCodeForUserToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "/v1.0/oauth2/userAccessToken", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"USER_TOKEN_X","expireIn":7200,"refreshToken":"R","corpId":"dingABC"}`))
	}))
	defer server.Close()

	cli := &DingTalkClient{
		cfg: dingTalkClientConfig{
			ClientID: "k", ClientSecret: "s",
			TokenURL: server.URL + "/v1.0/oauth2/userAccessToken",
		},
		httpClient: server.Client(),
	}
	resp, err := cli.ExchangeCodeForUserToken(context.Background(), "AUTH_CODE")
	require.NoError(t, err)
	require.Equal(t, "USER_TOKEN_X", resp.AccessToken)
	require.Equal(t, "dingABC", resp.CorpID)
}

func TestDingTalkClient_GetUnionIdByUserToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "USER_TOKEN_X", r.Header.Get("x-acs-dingtalk-access-token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nick":"张三","unionId":"UID_AAA","openId":"OPEN","avatarUrl":"http://x"}`))
	}))
	defer server.Close()

	cli := &DingTalkClient{
		cfg:        dingTalkClientConfig{UserInfoURL: server.URL + "/v1.0/contact/users/me"},
		httpClient: server.Client(),
	}
	unionID, nick, err := cli.GetUnionIdByUserToken(context.Background(), "USER_TOKEN_X")
	require.NoError(t, err)
	require.Equal(t, "UID_AAA", unionID)
	require.Equal(t, "张三", nick)
}

func TestDingTalkClient_GetAppToken_Cached(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_, _ = w.Write([]byte(`{"accessToken":"APP_TKN","expireIn":7200}`))
	}))
	defer server.Close()

	cli := &DingTalkClient{
		cfg:        dingTalkClientConfig{ClientID: "k", ClientSecret: "s", TokenURL: server.URL + "/gettoken"},
		httpClient: server.Client(),
	}
	t1, err := cli.GetAppToken(context.Background())
	require.NoError(t, err)
	t2, err := cli.GetAppToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, t1, t2)
	require.Equal(t, 1, callCount, "second call should hit cache")
}

func TestDingTalkClient_GetAppToken_TwoInstancesFetchOnce(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(`{"accessToken":"SHARED_APP_TOKEN","expireIn":7200}`))
	}))
	defer server.Close()

	cache := &dingTalkSharedCacheStub{}
	newClient := func() *DingTalkClient {
		return &DingTalkClient{
			cfg: dingTalkClientConfig{
				ClientID: "shared-app", ClientSecret: "secret", TokenURL: server.URL + "/gettoken",
			},
			httpClient:  server.Client(),
			sharedCache: cache,
		}
	}
	first, second := newClient(), newClient()
	results := make(chan string, 2)
	errs := make(chan error, 2)
	for _, client := range []*DingTalkClient{first, second} {
		go func(cli *DingTalkClient) {
			token, err := cli.GetAppToken(context.Background())
			results <- token
			errs <- err
		}(client)
	}
	for range 2 {
		require.NoError(t, <-errs)
		require.Equal(t, "SHARED_APP_TOKEN", <-results)
	}
	mu.Lock()
	require.Equal(t, 1, callCount)
	mu.Unlock()
}

func TestDingTalkClient_InvalidAppTokenInvalidatesAndRetriesOnce(t *testing.T) {
	tokenCalls := 0
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		_, _ = w.Write([]byte(`{"accessToken":"NEW_APP_TOKEN","expireIn":7200}`))
	}))
	defer tokenServer.Close()
	businessCalls := 0
	businessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		businessCalls++
		if businessCalls == 1 {
			_, _ = w.Write([]byte(`{"errcode":40014,"errmsg":"invalid access token"}`))
			return
		}
		_, _ = w.Write([]byte(`{"errcode":0,"result":{"userid":"USER-42"}}`))
	}))
	defer businessServer.Close()

	cache := &dingTalkSharedCacheStub{token: "STALE_APP_TOKEN", ttl: time.Hour}
	client := &DingTalkClient{
		cfg: dingTalkClientConfig{
			ClientID: "app", ClientSecret: "secret",
			TokenURL: tokenServer.URL + "/gettoken", UserInfoURL: businessServer.URL + "/stub",
		},
		httpClient:  businessServer.Client(),
		sharedCache: cache,
	}
	// token 获取和业务调用使用不同测试服务，因此自定义 transport 按 host 正常路由。
	client.httpClient = &http.Client{Timeout: time.Second}

	userID, err := client.GetUserIdByUnionId(context.Background(), "UNION-42")
	require.NoError(t, err)
	require.Equal(t, "USER-42", userID)
	require.Equal(t, 1, tokenCalls)
	require.Equal(t, 2, businessCalls)
	cache.mu.Lock()
	require.Equal(t, 1, cache.deleteCalls)
	require.Equal(t, "NEW_APP_TOKEN", cache.token)
	cache.mu.Unlock()
}

func TestDingTalkClient_GetUserIdByUnionId_60011(t *testing.T) {
	appTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessToken":"APP_TKN","expireIn":7200}`))
	}))
	defer appTokenServer.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":60011,"errmsg":"not in directory"}`))
	}))
	defer server.Close()

	cli := &DingTalkClient{
		cfg:        dingTalkClientConfig{TokenURL: appTokenServer.URL + "/gettoken"},
		httpClient: server.Client(),
	}
	cli.appToken = "APP_TKN"
	cli.appTokenExp = time.Now().Add(time.Hour)
	cli.cfg.UserInfoURL = server.URL + "/v1.0/contact/users/byUnionId"

	_, err := cli.GetUserIdByUnionId(context.Background(), "UID_AAA")
	require.Error(t, err)
	apiErr, ok := err.(*DingTalkAPIError)
	require.True(t, ok)
	require.Equal(t, "60011", apiErr.Code)
}

// TestDingTalkClient_GetDeptInfo_Success 验证 GetDeptInfo 正常情况返回部门信息。
func TestDingTalkClient_GetDeptInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok","result":{"dept_id":42,"name":"AI数据","parent_id":1}}`))
	}))
	defer server.Close()

	cli := &DingTalkClient{
		cfg: dingTalkClientConfig{
			UserInfoURL: server.URL + "/stub", // 不含 /contact/users/me，走 test stub 路径
		},
		httpClient: server.Client(),
	}
	cli.appToken = "APP_TKN"
	cli.appTokenExp = time.Now().Add(time.Hour)

	info, err := cli.GetDeptInfo(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, int64(42), info.DeptID)
	require.Equal(t, "AI数据", info.Name)
	require.Equal(t, int64(1), info.ParentID)
}

// TestDingTalkClient_GetDeptInfo_ErrCode60003 验证 errcode=60003（部门不存在）时返回错误。
func TestDingTalkClient_GetDeptInfo_ErrCode60003(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":60003,"errmsg":"dept not found"}`))
	}))
	defer server.Close()

	cli := &DingTalkClient{
		cfg:        dingTalkClientConfig{UserInfoURL: server.URL + "/stub"},
		httpClient: server.Client(),
	}
	cli.appToken = "APP_TKN"
	cli.appTokenExp = time.Now().Add(time.Hour)

	_, err := cli.GetDeptInfo(context.Background(), 999)
	require.Error(t, err)
	apiErr, ok := err.(*DingTalkAPIError)
	require.True(t, ok)
	require.Equal(t, "60003", apiErr.Code)
}
