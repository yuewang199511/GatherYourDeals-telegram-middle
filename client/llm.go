package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LLMClient struct {
	baseURL string
	http    *http.Client
	cb      *CircuitBreaker
}

func NewLLMClient(baseURL string, cbCfg CBConfig) *LLMClient {
	return &LLMClient{
		baseURL: baseURL,
		http:    &http.Client{},
		cb:      newCircuitBreaker(cbCfg),
	}
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Message    LLMMessage `json:"message"`
	StopReason string     `json:"stop_reason"`
}

func (c *LLMClient) Chat(accessToken string, messages []LLMMessage) (*ChatResponse, int, error) {
	if !c.cb.allow() {
		return nil, 0, ErrCircuitOpen
	}
	body, _ := json.Marshal(map[string]any{"messages": messages})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		c.cb.recordFailure()
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		c.cb.recordFailure()
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &e)
		return nil, resp.StatusCode, fmt.Errorf("%s", e.Message)
	}
	c.cb.recordSuccess()
	var r ChatResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, resp.StatusCode, err
	}
	return &r, resp.StatusCode, nil
}
