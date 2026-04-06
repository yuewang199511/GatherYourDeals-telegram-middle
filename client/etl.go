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
}

func NewETLClient(baseURL string) *ETLClient {
	return &ETLClient{baseURL: baseURL, http: &http.Client{}}
}

type ETLResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (c *ETLClient) Run(source string) (*ETLResponse, int, error) {
	body, _ := json.Marshal(map[string]string{"source": source})
	resp, err := c.http.Post(c.baseURL+"/etl", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var r ETLResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to parse ETL response")
	}
	return &r, resp.StatusCode, nil
}
