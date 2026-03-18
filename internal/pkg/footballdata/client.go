package footballdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	DefaultBaseURL = "https://api.football-data.org/v4"
	HeaderAuth     = "X-Auth-Token"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type MatchesResponse struct {
	Filters   any     `json:"filters"`
	ResultSet any     `json:"resultSet"`
	Matches   []Match `json:"matches"`
}

type Match struct {
	ID          int64       `json:"id"`
	UtcDate     time.Time   `json:"utcDate"`
	Status      string      `json:"status"`
	Matchday    int         `json:"matchday"`
	Stage       string      `json:"stage"`
	Group       string      `json:"group"`
	HomeTeam    Team        `json:"homeTeam"`
	AwayTeam    Team        `json:"awayTeam"`
	Competition Competition `json:"competition"`
	Season      Season      `json:"season"`
}

type Team struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Competition struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Season struct {
	StartDate       string `json:"startDate"`
	EndDate         string `json:"endDate"`
	CurrentMatchday int    `json:"currentMatchday"`
	Year            int    `json:"year"`
}

func (c *Client) GetCompetitionMatches(ctx context.Context, competitionCode string, season int) (*MatchesResponse, error) {
	if c == nil {
		return nil, errors.New("footballdata client is nil")
	}
	if c.APIKey == "" {
		return nil, errors.New("football-data api key is empty")
	}
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base + "/competitions/" + url.PathEscape(competitionCode) + "/matches")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if season > 0 {
		q.Set("season", strconv.Itoa(season))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderAuth, c.APIKey)
	req.Header.Set("Accept", "application/json")

	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("football-data api status=%d body=%s", resp.StatusCode, string(b))
	}
	var out MatchesResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
