package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/yuewang199511/GatherYourDeals-telegram-middle/cache"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/client"
)

const maxHistory = 10 // 5 rounds × 2 messages

// Sender abstracts tgbotapi.BotAPI for testing.
type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

// DataClientIface abstracts client.DataClient for testing.
type DataClientIface interface {
	Login(username, password string) (*client.TokenResponse, int, error)
	Logout(accessToken, refreshToken string) (int, error)
	RefreshToken(refreshToken string) (*client.TokenResponse, int, error)
}

// ETLClientIface abstracts client.ETLClient for testing.
type ETLClientIface interface {
	Run(accessToken, source string) (*client.ETLResponse, int, error)
}

// LLMClientIface abstracts client.LLMClient for testing.
type LLMClientIface interface {
	Chat(accessToken string, messages []client.LLMMessage) (*client.ChatResponse, int, error)
}

// StoreIface abstracts cache.Store for testing.
type StoreIface interface {
	GetTokens(ctx context.Context, chatID int64) (*cache.Tokens, error)
	SetTokens(ctx context.Context, chatID int64, t *cache.Tokens) error
	DeleteTokens(ctx context.Context, chatID int64) error
	GetHistory(ctx context.Context, chatID int64) ([]cache.Message, error)
	SetHistory(ctx context.Context, chatID int64, msgs []cache.Message) error
	DeleteHistory(ctx context.Context, chatID int64) error
}

type Bot struct {
	api        Sender
	dataClient DataClientIface
	etlClient  ETLClientIface
	llmClient  LLMClientIface
	store      StoreIface
}

func New(api Sender, data DataClientIface, etl ETLClientIface, llm LLMClientIface, store StoreIface) *Bot {
	return &Bot{api: api, dataClient: data, etlClient: etl, llmClient: llm, store: store}
}

func (b *Bot) Dispatch(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	switch {
	case text == "/start":
		b.send(chatID, "This is the gatherYourDeals bot! Right now it is only an alpha version! Try /help to see what you can do!")
	case text == "/help":
		b.handleHelp(chatID)
	case strings.HasPrefix(text, "/login"):
		b.handleLogin(chatID, text)
	case text == "/logout":
		b.handleLogout(chatID)
	case strings.HasPrefix(text, "/etl"):
		b.handleETL(chatID, text)
	default:
		b.handleChat(chatID, text)
	}
}

func (b *Bot) handleHelp(chatID int64) {
	b.send(chatID,
		"Available commands:\n"+
			"/start - Welcome message\n"+
			"/help - Show this help\n"+
			"/login <username> <password> - Log in to GatherYourDeals\n"+
			"/logout - Log out\n"+
			"/etl <url> - Process a Google Drive link into purchase records\n"+
			"Any other message - Ask questions about your purchase records",
	)
}

func (b *Bot) handleLogin(chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) != 3 {
		b.send(chatID, "Usage: /login <username> <password>")
		return
	}
	username, password := parts[1], parts[2]

	tokens, statusCode, err := b.dataClient.Login(username, password)
	if err != nil {
		b.send(chatID, fmt.Sprintf("Login failed [%d]: %s", statusCode, err.Error()))
		return
	}

	ctx := context.Background()
	if err := b.store.SetTokens(ctx, chatID, &cache.Tokens{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}); err != nil {
		log.Printf("failed to cache tokens for chat %d: %v", chatID, err)
		b.send(chatID, "Login succeeded but failed to save session. Please try again.")
		return
	}
	b.send(chatID, "Logged in successfully!")
}

func (b *Bot) handleLogout(chatID int64) {
	ctx := context.Background()
	tokens, err := b.store.GetTokens(ctx, chatID)
	if err != nil || tokens == nil {
		b.send(chatID, "You are not logged in.")
		return
	}

	statusCode, err := b.dataClient.Logout(tokens.AccessToken, tokens.RefreshToken)
	if err != nil {
		b.send(chatID, fmt.Sprintf("Logout failed [%d]: %s", statusCode, err.Error()))
		return
	}

	if err := b.store.DeleteTokens(ctx, chatID); err != nil {
		log.Printf("failed to delete tokens for chat %d: %v", chatID, err)
	}
	if err := b.store.DeleteHistory(ctx, chatID); err != nil {
		log.Printf("failed to delete history for chat %d: %v", chatID, err)
	}
	b.send(chatID, "Logged out successfully.")
}

func (b *Bot) handleETL(chatID int64, text string) {
	ctx := context.Background()
	accessToken, err := b.requireAuth(ctx, chatID)
	if err != nil {
		b.send(chatID, err.Error())
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		b.send(chatID, "Usage: /etl <url>")
		return
	}
	source := parts[1]

	b.send(chatID, "Processing, please wait...")

	result, statusCode, err := b.etlClient.Run(accessToken, source)
	if err != nil {
		b.send(chatID, fmt.Sprintf("ETL failed [%d]: %s", statusCode, err.Error()))
		return
	}
	if !result.Success {
		b.send(chatID, fmt.Sprintf("ETL failed [%d]: %s", statusCode, result.Message))
		return
	}
	b.send(chatID, "ETL completed: "+result.Message)
}

func (b *Bot) handleChat(chatID int64, text string) {
	ctx := context.Background()
	accessToken, err := b.requireAuth(ctx, chatID)
	if err != nil {
		b.send(chatID, err.Error())
		return
	}

	history, err := b.store.GetHistory(ctx, chatID)
	if err != nil {
		log.Printf("failed to load history for chat %d: %v", chatID, err)
		history = []cache.Message{}
	}

	history = append(history, cache.Message{Role: "user", Content: text})

	llmMsgs := make([]client.LLMMessage, len(history))
	for i, m := range history {
		llmMsgs[i] = client.LLMMessage{Role: m.Role, Content: m.Content}
	}

	resp, statusCode, err := b.llmClient.Chat(accessToken, llmMsgs)
	if err != nil {
		b.send(chatID, fmt.Sprintf("Chat error [%d]: %s", statusCode, err.Error()))
		return
	}

	history = append(history, cache.Message{Role: resp.Message.Role, Content: resp.Message.Content})
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	if err := b.store.SetHistory(ctx, chatID, history); err != nil {
		log.Printf("failed to save history for chat %d: %v", chatID, err)
	}

	b.send(chatID, resp.Message.Content)
}

// requireAuth returns a valid access token, refreshing if needed.
func (b *Bot) requireAuth(ctx context.Context, chatID int64) (string, error) {
	tokens, err := b.store.GetTokens(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("session error, please try again")
	}
	if tokens == nil {
		return "", fmt.Errorf("please login first with /login <username> <password>")
	}

	expiry, err := jwtExpiry(tokens.AccessToken)
	if err != nil {
		log.Printf("[requireAuth] chat %d: failed to parse JWT expiry: %v", chatID, err)
	} else {
		log.Printf("[requireAuth] chat %d: token expires at %v (in %v)", chatID, expiry, time.Until(expiry).Round(time.Second))
	}
	if err != nil || time.Until(expiry) < 30*time.Minute {
		if err == nil {
			log.Printf("[requireAuth] chat %d: token expiring soon, attempting refresh", chatID)
		} else {
			log.Printf("[requireAuth] chat %d: attempting refresh due to JWT parse error", chatID)
		}
		newTokens, statusCode, err := b.dataClient.RefreshToken(tokens.RefreshToken)
		if err != nil {
			log.Printf("[requireAuth] chat %d: refresh failed with status %d: %v", chatID, statusCode, err)
			if err := b.store.DeleteTokens(ctx, chatID); err != nil {
				log.Printf("failed to delete tokens for chat %d: %v", chatID, err)
			}
			return "", fmt.Errorf("session expired [%d]: please login again with /login <username> <password>", statusCode)
		}
		log.Printf("[requireAuth] chat %d: token refreshed successfully", chatID)
		tokens = &cache.Tokens{
			AccessToken:  newTokens.AccessToken,
			RefreshToken: newTokens.RefreshToken,
		}
		if err := b.store.SetTokens(ctx, chatID, tokens); err != nil {
			log.Printf("failed to save refreshed tokens for chat %d: %v", chatID, err)
			// The old refresh token was already consumed by the data service.
			// Losing the new one here means the next refresh will get a 401.
			// Force re-login so the user gets a fresh token pair.
			return "", fmt.Errorf("session error: failed to save refreshed tokens, please login again with /login <username> <password>")
		}
	}

	return tokens.AccessToken, nil
}

func (b *Bot) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("failed to send message to chat %d: %v", chatID, err)
	}
}

// jwtExpiry decodes the JWT payload and returns the expiry time without verifying the signature.
func jwtExpiry(tokenStr string) (time.Time, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, err
	}
	return time.Unix(claims.Exp, 0), nil
}
