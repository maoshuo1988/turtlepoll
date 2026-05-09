package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.deepseek.com"

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	MaxRetries int
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	Thinking       map[string]any `json:"thinking,omitempty"`
	Stream         bool           `json:"stream"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func New(baseURL, apiKey string, timeout time.Duration, maxRetries int) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		MaxRetries: maxRetries,
	}
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("deepseek api key is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages is required")
	}
	if req.Thinking == nil {
		req.Thinking = map[string]any{"type": "disabled"}
	}
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		out, err := c.doChat(ctx, body)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if attempt < c.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
		}
	}
	return nil, lastErr
}

func (c *Client) doChat(ctx context.Context, body []byte) (*ChatResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("deepseek api status %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return nil, fmt.Errorf("deepseek api returned empty message")
	}
	return &out, nil
}
