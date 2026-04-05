package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultGammaBaseURL = "https://gamma-api.polymarket.com"
)

type GammaClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewGammaClient(baseURL string) *GammaClient {
	if baseURL == "" {
		baseURL = DefaultGammaBaseURL
	}
	return &GammaClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

type Tag struct {
	// Gamma API 的 id 可能是 number 或 string（不同环境/版本可能不一致）
	ID   any    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type Market struct {
	ID       any    `json:"id"` // 可能是 number 或 string
	Slug     string `json:"slug"`
	Question string `json:"question"`
	Title    string `json:"title"`

	Active bool `json:"active"`
	Closed bool `json:"closed"`

	EndDate    string `json:"endDate"`   // 常见 ISO8601
	CloseDate  string `json:"closeDate"` // 可选
	Resolved   bool   `json:"resolved"`
	ResolvedAt string `json:"resolvedAt"`
	Resolution string `json:"resolution"` // 有些市场直接给出赢家文本/Key

	// outcomes 在 Gamma 返回里可能是：
	// 1) 数组：[{id,name,slug}, ...]
	// 2) 字符串："[\"Yes\", \"No\"]"（历史/部分接口）
	// 为了不让反序列化失败，这里做兼容解析。
	Outcomes Outcomes `json:"outcomes"`
	Tags     []Tag    `json:"tags"`
	Event    *Event   `json:"event"`
	EventID  any      `json:"eventId"`
}

type Outcomes []Outcome

func (o *Outcomes) UnmarshalJSON(b []byte) error {
	// null
	if len(b) == 0 || string(b) == "null" {
		*o = nil
		return nil
	}

	// 1) 直接数组
	var arr []Outcome
	if err := json.Unmarshal(b, &arr); err == nil {
		*o = arr
		return nil
	}

	// 2) string 包了一层 JSON（例如："[\"Yes\",\"No\"]")
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		*o = nil
		return nil
	}

	// 2.1) 期望是 ["Yes","No"]
	var names []string
	if err := json.Unmarshal([]byte(s), &names); err == nil {
		out := make([]Outcome, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			out = append(out, Outcome{ID: name, Name: name, Slug: strings.ToLower(name)})
		}
		*o = out
		return nil
	}

	// 2.2) 或者 s 自己就是 outcomes 对象数组
	if err := json.Unmarshal([]byte(s), &arr); err == nil {
		*o = arr
		return nil
	}

	// 保底：不阻断整体解析
	*o = nil
	return nil
}

type Outcome struct {
	ID   any    `json:"id"` // 同样可能 number/string
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Event struct {
	ID    any    `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

func (c *GammaClient) ListTags(ctx context.Context) ([]Tag, error) {
	u, err := url.Parse(c.baseURL() + "/tags")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("limit", "200")
	u.RawQuery = q.Encode()

	var out []Tag
	if err := c.getJSON(ctx, u.String(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListMarkets 拉取 markets（Gamma 支持 limit/offset）。
// 说明：为了兼容接口差异，这里允许传一些常用筛选参数；不保证所有参数都生效。
func (c *GammaClient) ListMarkets(ctx context.Context, limit, offset int, params map[string]string) ([]Market, error) {
	u, err := url.Parse(c.baseURL() + "/markets")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	for k, v := range params {
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	var out []Market
	if err := c.getJSON(ctx, u.String(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *GammaClient) getJSON(ctx context.Context, urlStr string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("polymarket gamma api status=%d body=%s", resp.StatusCode, string(b))
	}
	if len(b) == 0 {
		return errors.New("polymarket gamma api empty body")
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return nil
}

func (c *GammaClient) baseURL() string {
	if c == nil {
		return DefaultGammaBaseURL
	}
	if c.BaseURL == "" {
		return DefaultGammaBaseURL
	}
	return c.BaseURL
}
