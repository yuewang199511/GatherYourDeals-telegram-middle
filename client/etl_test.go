package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestETLClientRunSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/etl" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ETLResponse{Success: true, Message: "ETL completed successfully"})
	}))
	defer srv.Close()

	c := NewETLClient(srv.URL, testCBConfig())
	resp, code, err := c.Run("https://drive.google.com/folder/abc")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
	if !resp.Success {
		t.Error("expected success = true")
	}
}

func TestETLClientRunFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(ETLResponse{Success: false, Message: "Failed to parse data from source"})
	}))
	defer srv.Close()

	c := NewETLClient(srv.URL, testCBConfig())
	_, code, err := c.Run("https://bad-source.example.com")
	if err == nil {
		t.Fatal("expected error for 422 response, got nil")
	}
	if code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", code)
	}
}
