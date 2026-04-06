package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDataClientLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "acc", RefreshToken: "ref"})
	}))
	defer srv.Close()

	c := NewDataClient(srv.URL, testCBConfig())
	tokens, code, err := c.Login("alice", "password123")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
	if tokens.AccessToken != "acc" || tokens.RefreshToken != "ref" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}
}

func TestDataClientLoginFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid username or password"})
	}))
	defer srv.Close()

	c := NewDataClient(srv.URL, testCBConfig())
	_, code, err := c.Login("alice", "wrong")
	if code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", code)
	}
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestDataClientLogout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer acc" {
			t.Errorf("Authorization header = %q, want %q", r.Header.Get("Authorization"), "Bearer acc")
		}
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body.RefreshToken != "ref" {
			t.Errorf("refresh_token = %q, want %q", body.RefreshToken, "ref")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewDataClient(srv.URL, testCBConfig())
	code, err := c.Logout("acc", "ref")
	if err != nil {
		t.Fatalf("Logout error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
}

func TestDataClientRefreshToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "new-acc", RefreshToken: "new-ref"})
	}))
	defer srv.Close()

	c := NewDataClient(srv.URL, testCBConfig())
	tokens, code, err := c.RefreshToken("ref")
	if err != nil {
		t.Fatalf("RefreshToken error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
	if tokens.AccessToken != "new-acc" {
		t.Errorf("AccessToken = %q, want %q", tokens.AccessToken, "new-acc")
	}
}
