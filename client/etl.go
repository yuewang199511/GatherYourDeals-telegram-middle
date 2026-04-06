package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ETLClient struct {
	baseURL string
	http    *http.Client
	cb      *CircuitBreaker
}

func NewETLClient(baseURL string, cbCfg CBConfig) *ETLClient {
	return &ETLClient{
		baseURL: baseURL,
		http:    &http.Client{},
		cb:      newCircuitBreaker(cbCfg),
	}
}

type ETLResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (c *ETLClient) Run(accessToken, source string) (*ETLResponse, int, error) {
	if !c.cb.allow() {
		return nil, 0, ErrCircuitOpen
	}
	body, _ := json.Marshal(map[string]string{"source": source})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/etl", bytes.NewReader(body))
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
		return nil, resp.StatusCode, fmt.Errorf("ETL request failed with status %d", resp.StatusCode)
	}
	c.cb.recordSuccess()
	var r ETLResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to parse ETL response")
	}
	return &r, resp.StatusCode, nil
}
