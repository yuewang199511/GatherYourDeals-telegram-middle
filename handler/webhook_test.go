package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/yuewang199511/GatherYourDeals-telegram-middle/bot"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/cache"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/client"
)

// minimal mocks duplicated here to avoid test helper packages

type noopSender struct{}

func (n *noopSender) Send(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
	return tgbotapi.Message{}, nil
}

type noopDataClient struct{}

func (n *noopDataClient) Login(_, _ string) (*client.TokenResponse, int, error) { return nil, 0, nil }
func (n *noopDataClient) Logout(_, _ string) (int, error)                       { return 200, nil }
func (n *noopDataClient) RefreshToken(_ string) (*client.TokenResponse, int, error) {
	return nil, 0, nil
}

type noopETLClient struct{}

func (n *noopETLClient) Run(_ string) (*client.ETLResponse, int, error) { return nil, 0, nil }

type noopLLMClient struct{}

func (n *noopLLMClient) Chat(_ string, _ []client.LLMMessage) (*client.ChatResponse, int, error) {
	return nil, 0, nil
}

type noopStore struct{}

func (n *noopStore) GetTokens(_ context.Context, _ int64) (*cache.Tokens, error) { return nil, nil }
func (n *noopStore) SetTokens(_ context.Context, _ int64, _ *cache.Tokens) error { return nil }
func (n *noopStore) DeleteTokens(_ context.Context, _ int64) error                { return nil }
func (n *noopStore) GetHistory(_ context.Context, _ int64) ([]cache.Message, error) {
	return nil, nil
}
func (n *noopStore) SetHistory(_ context.Context, _ int64, _ []cache.Message) error { return nil }
func (n *noopStore) DeleteHistory(_ context.Context, _ int64) error                  { return nil }

func newTestBot() *bot.Bot {
	return bot.New(&noopSender{}, &noopDataClient{}, &noopETLClient{}, &noopLLMClient{}, &noopStore{})
}

func TestWebhookReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhook", Webhook(newTestBot()))

	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 1},
			Text: "/help",
			Date: int(time.Now().Unix()),
		},
	}
	body, _ := json.Marshal(update)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhookInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhook", Webhook(newTestBot()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// still returns 200 — we never want to signal errors back to Telegram
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 even for invalid JSON", w.Code)
	}
}
