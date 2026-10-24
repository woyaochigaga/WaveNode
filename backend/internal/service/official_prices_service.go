package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const officialPricesConfigKey = "official_subscription_prices_config"

// OfficialPricesConfig 将后台同步与首页展示分开；地区仅限制首页摘要，详情始终展示所有地区。
type OfficialPricesConfig struct {
	Enabled       bool     `json:"enabled"`
	HomeEnabled   bool     `json:"home_enabled"`
	RefreshHours  int      `json:"refresh_hours"`
	Apps          []string `json:"apps"`
	HomeCountries []string `json:"home_countries"`
}

// OfficialPricesSnapshot 是最近一次完整抓取结果，持久化到既有 data 挂载目录。
type OfficialPricesSnapshot struct {
	FetchedAt time.Time       `json:"fetched_at"`
	Records   []OfficialPrice `json:"records"`
}

// OfficialPricesStatus 只向管理员提供抓取状态；错误不进入公开接口。
type OfficialPricesStatus struct {
	Config      OfficialPricesConfig `json:"config"`
	Syncing     bool                 `json:"syncing"`
	LastAttempt *time.Time           `json:"last_attempt"`
	LastSuccess *time.Time           `json:"last_success"`
	NextRefresh *time.Time           `json:"next_refresh"`
	LastError   string               `json:"last_error"`
	RecordCount int                  `json:"record_count"`
}

var officialPriceApps = []string{"chatgpt", "claude", "grok"}

// OfficialPriceCountries 是来源支持的地区及中国大陆占位，缺失报价不使用其他地区替代。
var OfficialPriceCountries = []string{"cn", "us", "jp", "gb", "sg", "tw", "au", "ca", "de", "in", "id", "ph", "eg", "pk", "vn", "tr", "ng", "my", "ar", "mx", "co", "kz", "br", "th", "il", "ae", "cl", "ch", "za", "sa", "kr", "dk", "no"}

type OfficialPricesService struct {
	repo      SettingRepository
	client    *http.Client
	file      string
	mu        sync.RWMutex // 保护快照与状态，发布后的快照不再原地修改。
	snapshot  *OfficialPricesSnapshot
	lastTry   *time.Time
	lastError string
	syncing   bool
	refreshMu sync.Mutex // 自动同步与手动刷新互斥，避免重复请求和覆盖。
	wake      chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	startOnce sync.Once
}

func NewOfficialPricesService(repo SettingRepository, cfg *config.Config) *OfficialPricesService {
	ctx, cancel := context.WithCancel(context.Background())
	// 独立连接池仅服务价格同步，停止任务时不会关闭网关使用的共享连接。
	transport := http.DefaultTransport.(*http.Transport).Clone()
	s := &OfficialPricesService{repo: repo, file: filepath.Join(cfg.Pricing.DataDir, "official_subscription_prices.json"),
		wake: make(chan struct{}, 1), ctx: ctx, cancel: cancel,
		client: &http.Client{Timeout: 20 * time.Second, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 固定公开来源，禁止跳转到任意主机或内网地址。
			if len(via) > 3 || req.URL.Scheme != "https" || (req.URL.Host != "www.openprice.cc" && req.URL.Host != "openprice.cc") {
				return fmt.Errorf("价格来源发生了不受支持的跳转")
			}
			return nil
		}},
	}
	if data, err := os.ReadFile(s.file); err == nil {
		var snapshot OfficialPricesSnapshot
		if json.Unmarshal(data, &snapshot) == nil && !snapshot.FetchedAt.IsZero() && len(snapshot.Records) > 0 {
			s.snapshot = &snapshot
		}
	}
	return s
}

func defaultOfficialPricesConfig() OfficialPricesConfig {
	return OfficialPricesConfig{RefreshHours: 24, Apps: slices.Clone(officialPriceApps), HomeCountries: []string{"us", "jp", "gb"}}
}

func (s *OfficialPricesService) GetConfig(ctx context.Context) (OfficialPricesConfig, error) {
	cfg := defaultOfficialPricesConfig()
	value, err := s.repo.GetValue(ctx, officialPricesConfigKey)
	if errors.Is(err, ErrSettingNotFound) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return cfg, fmt.Errorf("读取订阅价格配置失败: %w", err)
	}
	return cfg, validateOfficialPricesConfig(cfg)
}

func validateOfficialPricesConfig(cfg OfficialPricesConfig) error {
	if !slices.Contains([]int{5, 18, 24, 168}, cfg.RefreshHours) {
		return fmt.Errorf("刷新周期必须为 5、18、24 或 168 小时")
	}
	if len(cfg.Apps) == 0 || len(cfg.Apps) > len(officialPriceApps) || len(cfg.HomeCountries) > 8 {
		return fmt.Errorf("请选择至少一个产品及最多八个首页代表地区")
	}
	for _, group := range []struct{ values, allowed []string }{{cfg.Apps, officialPriceApps}, {cfg.HomeCountries, OfficialPriceCountries}} {
		seen := make(map[string]bool)
		for _, value := range group.values {
			if seen[value] || !slices.Contains(group.allowed, value) {
				return fmt.Errorf("产品或地区选项无效或重复")
			}
			seen[value] = true
		}
	}
	return nil
}

func (s *OfficialPricesService) SaveConfig(ctx context.Context, cfg OfficialPricesConfig) error {
	if err := validateOfficialPricesConfig(cfg); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := s.repo.Set(ctx, officialPricesConfigKey, string(data)); err != nil {
		return err
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return nil
}

func (s *OfficialPricesService) Status(ctx context.Context) (*OfficialPricesStatus, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := &OfficialPricesStatus{Config: cfg, Syncing: s.syncing, LastAttempt: s.lastTry, LastError: s.lastError}
	if s.snapshot != nil {
		status.LastSuccess = &s.snapshot.FetchedAt
		status.RecordCount = len(s.snapshot.Records)
	}
	if cfg.Enabled {
		next := time.Now()
		if status.LastSuccess != nil {
			next = status.LastSuccess.Add(time.Duration(cfg.RefreshHours) * time.Hour)
		}
		// 失败后最多每 15 分钟重试一次，避免源站故障时连续抓取。
		if s.lastError != "" && s.lastTry != nil && s.lastTry.Add(15*time.Minute).After(next) {
			next = s.lastTry.Add(15 * time.Minute)
		}
		status.NextRefresh = &next
	}
	return status, nil
}

// Public 直接读取本地快照，访问首页不会触发爬虫。关闭总开关时不返回价格。
func (s *OfficialPricesService) Public(ctx context.Context, home bool) (any, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	result := struct {
		Enabled       bool            `json:"enabled"`
		HomeEnabled   bool            `json:"home_enabled"`
		HomeCountries []string        `json:"home_countries"`
		FetchedAt     *time.Time      `json:"fetched_at"`
		Records       []OfficialPrice `json:"records"`
	}{Enabled: cfg.Enabled, HomeEnabled: cfg.Enabled && cfg.HomeEnabled, HomeCountries: cfg.HomeCountries, Records: []OfficialPrice{}}
	if !cfg.Enabled || (home && !cfg.HomeEnabled) {
		return result, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.snapshot != nil {
		result.FetchedAt = &s.snapshot.FetchedAt
		for _, price := range s.snapshot.Records {
			if slices.Contains(cfg.Apps, price.Slug) {
				result.Records = append(result.Records, price)
			}
		}
	}
	return result, nil
}

func (s *OfficialPricesService) Start() {
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				status, err := s.Status(s.ctx)
				if err == nil && status.Config.Enabled && status.NextRefresh != nil && !time.Now().Before(*status.NextRefresh) {
					if err := s.Refresh(s.ctx); err != nil && s.ctx.Err() == nil {
						slog.Warn("订阅价格同步失败", "error", err)
					}
				}
				select {
				case <-s.ctx.Done():
					return
				case <-ticker.C:
				case <-s.wake:
				}
			}
		}()
	})
}

func (s *OfficialPricesService) Stop() {
	if s == nil {
		return
	}
	s.cancel()
	s.wg.Wait()
	s.client.CloseIdleConnections()
}

// Refresh 完整抓取三个产品，任何页面失败都保留旧快照，防止发布不完整的最低价排行。
func (s *OfficialPricesService) Refresh(ctx context.Context) (err error) {
	if !s.refreshMu.TryLock() {
		return fmt.Errorf("价格同步正在进行，请稍后查看结果")
	}
	defer s.refreshMu.Unlock()
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return fmt.Errorf("请先启用并保存订阅价格功能")
	}
	ctx, cancel := context.WithTimeout(ctx, 70*time.Second)
	defer cancel()
	now := time.Now().UTC()
	s.mu.Lock()
	s.syncing = true
	s.lastTry = &now
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.syncing = false
		if err != nil {
			s.lastError = err.Error()
		} else {
			s.lastError = ""
		}
	}()
	snapshot := &OfficialPricesSnapshot{}
	for _, slug := range officialPriceApps {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, officialPricesSource+"/"+slug, nil)
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("User-Agent", "WaveNode-OfficialPrices/1.0")
		resp, reqErr := s.client.Do(req)
		if reqErr != nil {
			return fmt.Errorf("%s: 抓取失败: %w", slug, reqErr)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || readErr != nil || len(body) > 4*1024*1024 {
			return fmt.Errorf("%s: 来源响应无效（HTTP %d）", slug, resp.StatusCode)
		}
		prices, parseErr := parseOfficialPrices(body, slug)
		if parseErr != nil {
			return parseErr
		}
		snapshot.Records = append(snapshot.Records, prices...)
	}
	// 抓取期间管理员可能关闭功能；提交前再次检查，防止旧任务覆盖关闭后的状态。
	cfg, err = s.GetConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return fmt.Errorf("功能已关闭，本次同步已停止")
	}
	snapshot.FetchedAt = time.Now().UTC()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(s.file), 0750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.file), ".official-prices-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp.Name(), s.file); err != nil {
		return err
	}
	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()
	return nil
}
