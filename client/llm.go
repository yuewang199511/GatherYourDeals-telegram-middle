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
}

func NewLLMClient(baseURL string) *LLMClient {
	return &LLMClient{baseURL: baseURL, http: &http.Client{}}
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
	body, _ := json.Marshal(map[string]interface{}{"messages": messages})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Message string `json:"message"`
		}
		json.Unmarshal(respBody, &e)
		return nil, resp.StatusCode, fmt.Errorf("%s", e.Message)
	}
	var r ChatResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, resp.StatusCode, err
	}
	return &r, resp.StatusCode, nil
}
