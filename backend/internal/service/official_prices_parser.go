package service

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

const officialPricesSource = "https://www.openprice.cc/official-prices"

// OfficialPrice 是来源网站的一条套餐/地区报价，人民币金额沿用来源的换算结果。
type OfficialPrice struct {
	App             string    `json:"app"`
	AppID           string    `json:"app_id"`
	Slug            string    `json:"slug"`
	Plan            string    `json:"plan"`
	Kind            string    `json:"kind"`
	BillingPeriod   *string   `json:"billing_period"`
	CountryCode     string    `json:"country_code"`
	OriginalPrice   string    `json:"original_price"`
	PriceCNY        float64   `json:"price_cny"`
	SourceUpdatedAt time.Time `json:"source_updated_at"`
	SourceURL       string    `json:"source_url"`
}

var officialPriceScripts = regexp.MustCompile(`(?s)<script\b[^>]*>(.*?)</script>`)
var officialPriceCountry = regexp.MustCompile(`^[a-z]{2}$`)

// parseOfficialPrices 仅解析公开 HTML 中的 JSON 数据，不执行第三方 JavaScript。
func parseOfficialPrices(body []byte, slug string) ([]OfficialPrice, error) {
	var stream strings.Builder
	for _, script := range officialPriceScripts.FindAllSubmatch(body, -1) {
		text := strings.TrimSpace(string(script[1]))
		if !strings.HasPrefix(text, "self.__next_f.push(") {
			continue
		}
		payload := strings.TrimSuffix(strings.TrimSuffix(text, ";"), ")")
		payload = strings.TrimPrefix(payload, "self.__next_f.push(")
		var chunk []json.RawMessage
		if json.Unmarshal([]byte(payload), &chunk) != nil || len(chunk) != 2 || string(chunk[0]) != "1" {
			continue
		}
		var part string
		if json.Unmarshal(chunk[1], &part) == nil {
			stream.WriteString(part)
		}
	}
	for _, line := range strings.Split(stream.String(), "\n") {
		_, payload, ok := strings.Cut(line, ":")
		if !ok || !json.Valid([]byte(payload)) {
			continue
		}
		var tree any
		if json.Unmarshal([]byte(payload), &tree) != nil {
			continue
		}
		if raw := findOfficialPriceApp(tree); raw != nil {
			encoded, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			return decodeOfficialPriceApp(encoded, slug)
		}
	}
	return nil, fmt.Errorf("%s: 来源页面未包含完整价格数据", slug)
}

// findOfficialPriceApp 在 React 序列化树中查找传给价格组件的 app 属性。
func findOfficialPriceApp(value any) any {
	switch node := value.(type) {
	case map[string]any:
		if app, ok := node["app"].(map[string]any); ok {
			if _, ok := app["subscriptions"].([]any); ok {
				return app
			}
		}
		for _, child := range node {
			if found := findOfficialPriceApp(child); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range node {
			if found := findOfficialPriceApp(child); found != nil {
				return found
			}
		}
	}
	return nil
}

func decodeOfficialPriceApp(data []byte, slug string) ([]OfficialPrice, error) {
	var app struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Subscriptions []struct {
			Name          string  `json:"name"`
			Kind          string  `json:"kind"`
			BillingPeriod *string `json:"billingPeriod"`
			Prices        []struct {
				Country       string    `json:"country"`
				OriginalPrice string    `json:"originalPrice"`
				PriceRMB      *float64  `json:"priceRmb"`
				UpdatedAt     time.Time `json:"updatedAt"`
			} `json:"prices"`
		} `json:"subscriptions"`
	}
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("%s: 价格字段格式不正确: %w", slug, err)
	}
	if app.ID == "" || app.Name == "" || len(app.Subscriptions) == 0 {
		return nil, fmt.Errorf("%s: 缺少应用或套餐数据", slug)
	}
	var records []OfficialPrice
	seen := make(map[string]bool)
	for _, plan := range app.Subscriptions {
		if plan.Name == "" || len(plan.Prices) == 0 || (plan.Kind != "subscription" && plan.Kind != "credit") {
			return nil, fmt.Errorf("%s: 套餐数据不完整", slug)
		}
		for _, price := range plan.Prices {
			country := strings.ToLower(price.Country)
			key := plan.Name + ":" + country
			if seen[key] || !officialPriceCountry.MatchString(country) || price.OriginalPrice == "" || price.PriceRMB == nil || *price.PriceRMB < 0 || math.IsNaN(*price.PriceRMB) || math.IsInf(*price.PriceRMB, 0) || price.UpdatedAt.IsZero() {
				return nil, fmt.Errorf("%s: 存在重复或无效的地区报价", slug)
			}
			seen[key] = true
			records = append(records, OfficialPrice{
				App: app.Name, AppID: app.ID, Slug: slug, Plan: plan.Name,
				Kind: plan.Kind, BillingPeriod: plan.BillingPeriod, CountryCode: country,
				OriginalPrice: decodeOfficialPriceText(price.OriginalPrice), PriceCNY: *price.PriceRMB,
				SourceUpdatedAt: price.UpdatedAt, SourceURL: officialPricesSource + "/" + slug,
			})
		}
	}
	return records, nil
}

// React 用 $$ 转义开头的美元符号，展示时必须还原，不能把 $20.00 导出为 $$20.00。
func decodeOfficialPriceText(value string) string {
	if strings.HasPrefix(value, "$$") {
		return value[1:]
	}
	return value
}
