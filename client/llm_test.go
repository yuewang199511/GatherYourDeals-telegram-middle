package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMClientChatSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Message:    LLMMessage{Role: "assistant", Content: "Hello!"},
			StopReason: "end_turn",
		})
	}))
	defer srv.Close()

	c := NewLLMClient(srv.URL)
	resp, code, err := c.Chat("test-token", []LLMMessage{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
	if resp.Message.Content != "Hello!" {
		t.Errorf("content = %q, want %q", resp.Message.Content, "Hello!")
	}
}

func TestLLMClientChatUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"code": "unauthorized", "message": "Missing or invalid Authorization header"})
	}))
	defer srv.Close()

	c := NewLLMClient(srv.URL)
	_, code, err := c.Chat("bad-token", []LLMMessage{{Role: "user", Content: "Hi"}})
	if code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", code)
	}
	if err == nil {
		t.Error("expected error, got nil")
	}
}
