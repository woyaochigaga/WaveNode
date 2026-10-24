package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type officialPricesTestRepo struct {
	SettingRepository
	mu    sync.Mutex
	value string
}

func (r *officialPricesTestRepo) GetValue(context.Context, string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.value == "" {
		return "", ErrSettingNotFound
	}
	return r.value, nil
}

func (r *officialPricesTestRepo) Set(_ context.Context, _, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value = value
	return nil
}

type officialPricesTransport func(*http.Request) (*http.Response, error)

func (f officialPricesTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// 将价格 JSON 拆进多个 RSC chunk，覆盖真实 Next 页面跨 script 的序列化格式。
func officialPricesHTML(t *testing.T, raw string) string {
	t.Helper()
	stream := "0:[\"$\",\"section\",null,{\"app\":" + raw + "}]\n"
	var html strings.Builder
	for _, part := range []string{stream[:len(stream)/2], stream[len(stream)/2:]} {
		chunk, err := json.Marshal([]any{1, part})
		require.NoError(t, err)
		html.WriteString("<script>self.__next_f.push(" + string(chunk) + ")</script>")
	}
	return html.String()
}

const officialPricesTestApp = `{"id":"chatgpt","name":"ChatGPT","subscriptions":[{"name":"Plus monthly","kind":"subscription","billingPeriod":"monthly","prices":[{"country":"us","originalPrice":"$$20.00","priceRmb":134.23,"updatedAt":"2026-10-22T10:00:00Z"},{"country":"jp","originalPrice":"¥2800","priceRmb":126.5,"updatedAt":"2026-10-22T10:00:00Z"}]}]}`

func newOfficialPricesTestService(t *testing.T) (*OfficialPricesService, *officialPricesTestRepo, *config.Config) {
	t.Helper()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = t.TempDir()
	repo := &officialPricesTestRepo{}
	s := NewOfficialPricesService(repo, cfg)
	t.Cleanup(s.Stop)
	s.client.Transport = officialPricesTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(officialPricesHTML(t, officialPricesTestApp)))}, nil
	})
	return s, repo, cfg
}

func TestOfficialPricesParser(t *testing.T) {
	prices, err := parseOfficialPrices([]byte(officialPricesHTML(t, officialPricesTestApp)), "chatgpt")
	require.NoError(t, err)
	require.Len(t, prices, 2)
	require.Equal(t, "$20.00", prices[0].OriginalPrice)
	require.Equal(t, 134.23, prices[0].PriceCNY)
	require.Equal(t, "monthly", *prices[0].BillingPeriod)
	require.Equal(t, officialPricesSource+"/chatgpt", prices[0].SourceURL)
	for _, invalid := range []string{
		strings.Replace(officialPricesTestApp, `"priceRmb":134.23,`, "", 1),
		strings.Replace(officialPricesTestApp, `"priceRmb":134.23`, `"priceRmb":-1`, 1),
		strings.Replace(officialPricesTestApp, `"country":"jp"`, `"country":"us"`, 1),
		strings.Replace(officialPricesTestApp, "2026-10-22T10:00:00Z", "invalid", 1),
	} {
		_, err := parseOfficialPrices([]byte(officialPricesHTML(t, invalid)), "chatgpt")
		require.Error(t, err)
	}
	_, err = parseOfficialPrices([]byte(`<script>throw new Error("must never execute")</script>`), "chatgpt")
	require.Error(t, err)
}

func TestOfficialPricesConfigAndVisibility(t *testing.T) {
	s, _, _ := newOfficialPricesTestService(t)
	ctx := context.Background()
	cfg, err := s.GetConfig(ctx)
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
	require.Error(t, s.Refresh(ctx))
	for _, hours := range []int{5, 18, 24, 168} {
		cfg.RefreshHours = hours
		require.NoError(t, validateOfficialPricesConfig(cfg))
	}
	cfg.RefreshHours = 6
	require.Error(t, s.SaveConfig(ctx, cfg))
	cfg.RefreshHours, cfg.Enabled, cfg.HomeEnabled = 5, true, true
	require.NoError(t, s.SaveConfig(ctx, cfg))
	require.NoError(t, s.Refresh(ctx))

	read := func(home bool) struct {
		Enabled, HomeEnabled bool
		Records              []OfficialPrice
	} {
		value, err := s.Public(ctx, home)
		require.NoError(t, err)
		data, err := json.Marshal(value)
		require.NoError(t, err)
		var out struct {
			Enabled, HomeEnabled bool
			Records              []OfficialPrice
		}
		require.NoError(t, json.Unmarshal(data, &out))
		return out
	}
	require.Len(t, read(true).Records, 6)
	cfg.Apps = []string{"chatgpt"}
	cfg.HomeEnabled = false
	require.NoError(t, s.SaveConfig(ctx, cfg))
	require.Empty(t, read(true).Records)
	require.Len(t, read(false).Records, 2)
	cfg.Enabled = false
	require.NoError(t, s.SaveConfig(ctx, cfg))
	require.Empty(t, read(false).Records)
	status, err := s.Status(ctx)
	require.NoError(t, err)
	require.Nil(t, status.NextRefresh)
	require.Equal(t, 6, status.RecordCount)
}

func TestOfficialPricesFailedSyncRetainsSnapshotAndSurvivesRestart(t *testing.T) {
	s, repo, appConfig := newOfficialPricesTestService(t)
	ctx := context.Background()
	cfg := defaultOfficialPricesConfig()
	cfg.Enabled = true
	require.NoError(t, s.SaveConfig(ctx, cfg))
	require.NoError(t, s.Refresh(ctx))
	first, err := s.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, first.LastSuccess.Add(24*time.Hour), *first.NextRefresh)
	s.client.Transport = officialPricesTransport(func(r *http.Request) (*http.Response, error) {
		body := officialPricesHTML(t, officialPricesTestApp)
		if strings.HasSuffix(r.URL.Path, "/claude") {
			body = "<html>Source unavailable</html>"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	require.Error(t, s.Refresh(ctx))
	failed, err := s.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, first.LastSuccess, failed.LastSuccess)
	require.Equal(t, first.RecordCount, failed.RecordCount)
	require.NotEmpty(t, failed.LastError)
	require.False(t, failed.Syncing)
	restarted := NewOfficialPricesService(repo, appConfig)
	defer restarted.Stop()
	restored, err := restarted.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, first.LastSuccess, restored.LastSuccess)
	require.Equal(t, 6, restored.RecordCount)
}

func TestOfficialPricesWorkerStartsOnEnable(t *testing.T) {
	s, _, _ := newOfficialPricesTestService(t)
	s.Start()
	cfg := defaultOfficialPricesConfig()
	cfg.Enabled = true
	require.NoError(t, s.SaveConfig(context.Background(), cfg))
	require.Eventually(t, func() bool {
		status, err := s.Status(context.Background())
		return err == nil && status.RecordCount == 6 && !status.Syncing
	}, time.Second*3, time.Millisecond*10)
}

func TestOfficialPricesSettingsIndependentOfGatewayVersion(t *testing.T) {
	before := ComputeSettingsVersion(map[string]string{"site_name": "WaveNode"})
	after := ComputeSettingsVersion(map[string]string{"site_name": "WaveNode", officialPricesConfigKey: `{"enabled":true}`})
	require.Equal(t, before, after)
	require.NotEqual(t, before, ComputeSettingsVersion(map[string]string{"site_name": "Changed"}))
}

func TestOfficialPricesFirstFailureBacksOff(t *testing.T) {
	s, _, _ := newOfficialPricesTestService(t)
	s.client.Transport = officialPricesTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
	})
	cfg := defaultOfficialPricesConfig()
	cfg.Enabled = true
	require.NoError(t, s.SaveConfig(context.Background(), cfg))
	require.Error(t, s.Refresh(context.Background()))
	status, err := s.Status(context.Background())
	require.NoError(t, err)
	require.Nil(t, status.LastSuccess)
	require.NotNil(t, status.LastAttempt)
	require.Equal(t, status.LastAttempt.Add(15*time.Minute), *status.NextRefresh)
}

// 联网验收需显式开启，常规测试不依赖第三方网络或实时价格。
func TestOfficialPricesLiveSource(t *testing.T) {
	if os.Getenv("OFFICIAL_PRICES_LIVE_TEST") != "1" {
		t.Skip("set OFFICIAL_PRICES_LIVE_TEST=1 for source integration check")
	}
	appConfig := &config.Config{}
	appConfig.Pricing.DataDir = t.TempDir()
	s := NewOfficialPricesService(&officialPricesTestRepo{}, appConfig)
	t.Cleanup(s.Stop)
	cfg := defaultOfficialPricesConfig()
	cfg.Enabled = true
	require.NoError(t, s.SaveConfig(context.Background(), cfg))
	require.NoError(t, s.Refresh(context.Background()))
	status, err := s.Status(context.Background())
	require.NoError(t, err)
	require.Greater(t, status.RecordCount, 0)
	t.Logf("已解析并保存 %d 条公开报价", status.RecordCount)
}
