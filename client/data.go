package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DataClient struct {
	baseURL string
	http    *http.Client
	cb      *CircuitBreaker
}

func NewDataClient(baseURL string, cbCfg CBConfig) *DataClient {
	return &DataClient{
		baseURL: baseURL,
		http:    &http.Client{},
		cb:      newCircuitBreaker(cbCfg),
	}
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *DataClient) Login(username, password string) (*TokenResponse, int, error) {
	if !c.cb.allow() {
		return nil, 0, ErrCircuitOpen
	}
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.http.Post(c.baseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		c.cb.recordFailure()
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		c.cb.recordFailure()
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &e)
		return nil, resp.StatusCode, fmt.Errorf("%s", e.Error)
	}
	c.cb.recordSuccess()
	var t TokenResponse
	if err := json.Unmarshal(respBody, &t); err != nil {
		return nil, resp.StatusCode, err
	}
	return &t, resp.StatusCode, nil
}

func (c *DataClient) Logout(accessToken, refreshToken string) (int, error) {
	if !c.cb.allow() {
		return 0, ErrCircuitOpen
	}
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		c.cb.recordFailure()
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		c.cb.recordFailure()
		return resp.StatusCode, nil
	}
	c.cb.recordSuccess()
	return resp.StatusCode, nil
}

func (c *DataClient) RefreshToken(refreshToken string) (*TokenResponse, int, error) {
	if !c.cb.allow() {
		return nil, 0, ErrCircuitOpen
	}
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	resp, err := c.http.Post(c.baseURL+"/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		c.cb.recordFailure()
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		c.cb.recordFailure()
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &e)
		return nil, resp.StatusCode, fmt.Errorf("%s", e.Error)
	}
	c.cb.recordSuccess()
	var t TokenResponse
	if err := json.Unmarshal(respBody, &t); err != nil {
		return nil, resp.StatusCode, err
	}
	return &t, resp.StatusCode, nil
}
