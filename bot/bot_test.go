package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/yuewang199511/GatherYourDeals-telegram-middle/cache"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/client"
)

// --- mocks ---

type mockSender struct {
	sent []string
}

func (m *mockSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if msg, ok := c.(tgbotapi.MessageConfig); ok {
		m.sent = append(m.sent, msg.Text)
	}
	return tgbotapi.Message{}, nil
}

type mockDataClient struct {
	loginTokens *client.TokenResponse
	loginErr    error
	loginCode   int
	logoutCode  int
	logoutErr   error
	refreshResp *client.TokenResponse
	refreshCode int
	refreshErr  error
}

func (m *mockDataClient) Login(_, _ string) (*client.TokenResponse, int, error) {
	return m.loginTokens, m.loginCode, m.loginErr
}
func (m *mockDataClient) Logout(_, _ string) (int, error) {
	return m.logoutCode, m.logoutErr
}
func (m *mockDataClient) RefreshToken(_ string) (*client.TokenResponse, int, error) {
	return m.refreshResp, m.refreshCode, m.refreshErr
}

type mockETLClient struct {
	resp *client.ETLResponse
	code int
	err  error
}

func (m *mockETLClient) Run(_ string) (*client.ETLResponse, int, error) {
	return m.resp, m.code, m.err
}

type mockLLMClient struct {
	resp *client.ChatResponse
	code int
	err  error
}

func (m *mockLLMClient) Chat(_ string, _ []client.LLMMessage) (*client.ChatResponse, int, error) {
	return m.resp, m.code, m.err
}

type mockStore struct {
	tokens  *cache.Tokens
	history []cache.Message
	err     error
}

func (m *mockStore) GetTokens(_ context.Context, _ int64) (*cache.Tokens, error) {
	return m.tokens, m.err
}
func (m *mockStore) SetTokens(_ context.Context, _ int64, t *cache.Tokens) error {
	m.tokens = t
	return m.err
}
func (m *mockStore) DeleteTokens(_ context.Context, _ int64) error {
	m.tokens = nil
	return m.err
}
func (m *mockStore) GetHistory(_ context.Context, _ int64) ([]cache.Message, error) {
	return m.history, m.err
}
func (m *mockStore) SetHistory(_ context.Context, _ int64, msgs []cache.Message) error {
	m.history = msgs
	return m.err
}
func (m *mockStore) DeleteHistory(_ context.Context, _ int64) error {
	m.history = nil
	return m.err
}

// --- helpers ---

func makeUpdate(chatID int64, text string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			Text: text,
		},
	}
}

func makeJWT(exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{"exp": exp.Unix()})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	return fmt.Sprintf("%s.%s.sig", header, payload)
}

func newBot(sender *mockSender, data *mockDataClient, etl *mockETLClient, llm *mockLLMClient, store *mockStore) *Bot {
	return New(sender, data, etl, llm, store)
}

// --- tests ---

func TestDispatchStart(t *testing.T) {
	sender := &mockSender{}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(makeUpdate(1, "/start"))
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sender.sent))
	}
	want := "This is the gatherYourDeals bot! Right now it is only an alpha version! Try /help to see what you can do!"
	if sender.sent[0] != want {
		t.Errorf("got %q, want %q", sender.sent[0], want)
	}
}

func TestDispatchHelp(t *testing.T) {
	sender := &mockSender{}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(makeUpdate(1, "/help"))
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sender.sent))
	}
	for _, cmd := range []string{"/start", "/help", "/login", "/logout", "/etl"} {
		if !strings.Contains(sender.sent[0], cmd) {
			t.Errorf("help message missing %q", cmd)
		}
	}
}

func TestDispatchNilMessage(t *testing.T) {
	sender := &mockSender{}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(tgbotapi.Update{Message: nil})
	if len(sender.sent) != 0 {
		t.Errorf("expected no messages sent for nil message, got %v", sender.sent)
	}
}

func TestHandleLoginSuccess(t *testing.T) {
	sender := &mockSender{}
	data := &mockDataClient{
		loginTokens: &client.TokenResponse{AccessToken: "acc", RefreshToken: "ref"},
		loginCode:   200,
	}
	store := &mockStore{}
	b := newBot(sender, data, &mockETLClient{}, &mockLLMClient{}, store)
	b.Dispatch(makeUpdate(1, "/login alice password123"))

	if store.tokens == nil {
		t.Fatal("expected tokens to be cached")
	}
	if store.tokens.AccessToken != "acc" {
		t.Errorf("AccessToken = %q, want %q", store.tokens.AccessToken, "acc")
	}
	if store.tokens.RefreshToken != "ref" {
		t.Errorf("RefreshToken = %q, want %q", store.tokens.RefreshToken, "ref")
	}
	if len(sender.sent) == 0 || sender.sent[0] != "Logged in successfully!" {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
}

func TestHandleLoginBadArgs(t *testing.T) {
	sender := &mockSender{}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(makeUpdate(1, "/login onlyuser"))
	if len(sender.sent) == 0 || sender.sent[0] != "Usage: /login <username> <password>" {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
}

func TestHandleLoginFailure(t *testing.T) {
	sender := &mockSender{}
	data := &mockDataClient{loginErr: fmt.Errorf("invalid username or password"), loginCode: 401}
	b := newBot(sender, data, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(makeUpdate(1, "/login alice wrong"))
	if len(sender.sent) == 0 || sender.sent[0] != "Login failed [401]: invalid username or password" {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
}

func TestHandleLogoutSuccess(t *testing.T) {
	sender := &mockSender{}
	store := &mockStore{
		tokens:  &cache.Tokens{AccessToken: "acc", RefreshToken: "ref"},
		history: []cache.Message{{Role: "user", Content: "hello"}},
	}
	data := &mockDataClient{logoutCode: 200}
	b := newBot(sender, data, &mockETLClient{}, &mockLLMClient{}, store)
	b.Dispatch(makeUpdate(1, "/logout"))

	if store.tokens != nil {
		t.Error("expected tokens to be deleted after logout")
	}
	if store.history != nil {
		t.Error("expected history to be cleared after logout")
	}
	if len(sender.sent) != 1 || sender.sent[0] != "Logged out successfully." {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
}

func TestHandleLogoutNotLoggedIn(t *testing.T) {
	sender := &mockSender{}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(makeUpdate(1, "/logout"))
	if len(sender.sent) == 0 || sender.sent[0] != "You are not logged in." {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
}

func TestHandleETLSuccess(t *testing.T) {
	sender := &mockSender{}
	token := makeJWT(time.Now().Add(1 * time.Hour))
	store := &mockStore{tokens: &cache.Tokens{AccessToken: token, RefreshToken: "ref"}}
	etl := &mockETLClient{resp: &client.ETLResponse{Success: true, Message: "done"}, code: 200}
	b := newBot(sender, &mockDataClient{}, etl, &mockLLMClient{}, store)
	b.Dispatch(makeUpdate(1, "/etl https://drive.google.com/folder/abc"))

	if len(sender.sent) < 2 {
		t.Fatalf("expected 2 messages, got %v", sender.sent)
	}
	if sender.sent[0] != "Processing, please wait..." {
		t.Errorf("first message = %q", sender.sent[0])
	}
	if sender.sent[1] != "ETL completed: done" {
		t.Errorf("second message = %q", sender.sent[1])
	}
}

func TestHandleETLNotLoggedIn(t *testing.T) {
	sender := &mockSender{}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, &mockLLMClient{}, &mockStore{})
	b.Dispatch(makeUpdate(1, "/etl https://drive.google.com/folder/abc"))
	if len(sender.sent) == 0 || sender.sent[0] != "please login first with /login <username> <password>" {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
}

func TestHandleChatSuccess(t *testing.T) {
	sender := &mockSender{}
	token := makeJWT(time.Now().Add(1 * time.Hour))
	store := &mockStore{tokens: &cache.Tokens{AccessToken: token, RefreshToken: "ref"}}
	llm := &mockLLMClient{
		resp: &client.ChatResponse{Message: client.LLMMessage{Role: "assistant", Content: "I can help with that."}},
		code: 200,
	}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, llm, store)
	b.Dispatch(makeUpdate(1, "how much did I spend?"))

	if len(sender.sent) != 1 || sender.sent[0] != "I can help with that." {
		t.Errorf("unexpected reply: %v", sender.sent)
	}
	if len(store.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(store.history))
	}
	if store.history[0].Role != "user" || store.history[0].Content != "how much did I spend?" {
		t.Errorf("history[0] = %+v, want {user, 'how much did I spend?'}", store.history[0])
	}
	if store.history[1].Role != "assistant" || store.history[1].Content != "I can help with that." {
		t.Errorf("history[1] = %+v, want {assistant, 'I can help with that.'}", store.history[1])
	}
}

func TestHandleChatHistoryTrimmed(t *testing.T) {
	sender := &mockSender{}
	token := makeJWT(time.Now().Add(1 * time.Hour))
	// pre-fill 10 messages (max)
	history := make([]cache.Message, maxHistory)
	for i := range history {
		history[i] = cache.Message{Role: "user", Content: fmt.Sprintf("msg%d", i)}
	}
	store := &mockStore{
		tokens:  &cache.Tokens{AccessToken: token, RefreshToken: "ref"},
		history: history,
	}
	llm := &mockLLMClient{
		resp: &client.ChatResponse{Message: client.LLMMessage{Role: "assistant", Content: "reply"}},
		code: 200,
	}
	b := newBot(sender, &mockDataClient{}, &mockETLClient{}, llm, store)
	b.Dispatch(makeUpdate(1, "new message"))

	if len(store.history) != maxHistory {
		t.Fatalf("history len = %d, want %d", len(store.history), maxHistory)
	}
	// newest messages must be retained — last entry should be the assistant reply
	last := store.history[len(store.history)-1]
	if last.Role != "assistant" || last.Content != "reply" {
		t.Errorf("last history entry = %+v, want {assistant, 'reply'}", last)
	}
	// oldest messages must be dropped — first entry should NOT be msg0
	if store.history[0].Content == "msg0" {
		t.Error("oldest message was not trimmed from history")
	}
}

func TestJwtExpiry(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	token := makeJWT(future)
	got, err := jwtExpiry(token)
	if err != nil {
		t.Fatalf("jwtExpiry error: %v", err)
	}
	if !got.Equal(future) {
		t.Errorf("got %v, want %v", got, future)
	}
}

func TestJwtExpiryInvalidToken(t *testing.T) {
	_, err := jwtExpiry("not.a.valid.jwt.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}
