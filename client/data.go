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
}

func NewDataClient(baseURL string) *DataClient {
	return &DataClient{baseURL: baseURL, http: &http.Client{}}
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *DataClient) Login(username, password string) (*TokenResponse, int, error) {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := c.http.Post(c.baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &e)
		return nil, resp.StatusCode, fmt.Errorf("%s", e.Error)
	}
	var t TokenResponse
	if err := json.Unmarshal(respBody, &t); err != nil {
		return nil, resp.StatusCode, err
	}
	return &t, resp.StatusCode, nil
}

func (c *DataClient) Logout(accessToken, refreshToken string) (int, error) {
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func (c *DataClient) RefreshToken(refreshToken string) (*TokenResponse, int, error) {
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	resp, err := c.http.Post(c.baseURL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &e)
		return nil, resp.StatusCode, fmt.Errorf("%s", e.Error)
	}
	var t TokenResponse
	if err := json.Unmarshal(respBody, &t); err != nil {
		return nil, resp.StatusCode, err
	}
	return &t, resp.StatusCode, nil
}
