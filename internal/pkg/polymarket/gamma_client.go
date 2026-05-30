package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 8 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   12 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &GammaClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

type gammaHTTPError struct {
	StatusCode int
	Body       string
}

func (e *gammaHTTPError) Error() string {
	if e == nil {
		return "polymarket gamma api unknown error"
	}
	return fmt.Sprintf("polymarket gamma api status=%d body=%s", e.StatusCode, e.Body)
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

	ClobTokenIds  StringArray `json:"clobTokenIds"`
	OutcomePrices StringArray `json:"outcomePrices"`

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

type StringArray []string

func (a *StringArray) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*a = nil
		return nil
	}
	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*a = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		*a = nil
		return nil
	}
	if err := json.Unmarshal([]byte(s), &arr); err == nil {
		*a = arr
		return nil
	}
	*a = []string{s}
	return nil
}

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
	ID     any    `json:"id"` // 同样可能 number/string
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Winner bool   `json:"winner"`
}

type Event struct {
	ID    any    `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type MarketsKeysetPage struct {
	Markets    []Market `json:"markets"`
	NextCursor string   `json:"next_cursor"`
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

// ListMarketsKeyset 拉取 markets/keyset（Gamma 推荐的 cursor/keyset 分页）。
func (c *GammaClient) ListMarketsKeyset(ctx context.Context, limit int, afterCursor string, params map[string]string) ([]Market, string, error) {
	u, err := url.Parse(c.baseURL() + "/markets/keyset")
	if err != nil {
		return nil, "", err
	}
	q := u.Query()
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if afterCursor != "" {
		q.Set("after_cursor", afterCursor)
	}
	for k, v := range params {
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	var page MarketsKeysetPage
	if err := c.getJSON(ctx, u.String(), &page); err != nil {
		return nil, "", err
	}
	return page.Markets, page.NextCursor, nil
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

func (c *GammaClient) GetMarketByID(ctx context.Context, id string) (*Market, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("market id is required")
	}
	u, err := url.Parse(c.baseURL() + "/markets/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	var out Market
	if err := c.getJSON(ctx, u.String(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *GammaClient) getJSON(ctx context.Context, urlStr string, out any) error {
	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	const maxAttempts = 4
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = c.getJSONOnce(ctx, hc, urlStr, out)
		if lastErr == nil {
			return nil
		}
		if attempt == maxAttempts || !shouldRetryGammaError(ctx, lastErr) {
			return lastErr
		}
		if err := waitRetryBackoff(ctx, attempt); err != nil {
			return lastErr
		}
	}
	return lastErr
}

func (c *GammaClient) getJSONOnce(ctx context.Context, hc *http.Client, urlStr string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

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
		return &gammaHTTPError{StatusCode: resp.StatusCode, Body: string(b)}
	}
	if len(b) == 0 {
		return errors.New("polymarket gamma api empty body")
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return nil
}

func shouldRetryGammaError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if ctx.Err() != nil {
		return false
	}

	var httpErr *gammaHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == 429 || httpErr.StatusCode >= 500
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "tls handshake timeout") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "eof") {
		return true
	}

	return false
}

func waitRetryBackoff(ctx context.Context, attempt int) error {
	backoff := 300 * time.Millisecond
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= 2*time.Second {
			backoff = 2 * time.Second
			break
		}
	}
	jitterMs := time.Now().UnixNano()%200 + 1
	wait := backoff + time.Duration(jitterMs)*time.Millisecond

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
