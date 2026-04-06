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

type Bot struct {
	api        *tgbotapi.BotAPI
	dataClient *client.DataClient
	etlClient  *client.ETLClient
	llmClient  *client.LLMClient
	store      *cache.Store
}

func New(api *tgbotapi.BotAPI, data *client.DataClient, etl *client.ETLClient, llm *client.LLMClient, store *cache.Store) *Bot {
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

	b.store.DeleteTokens(ctx, chatID)
	b.store.DeleteHistory(ctx, chatID)
	b.send(chatID, "Logged out successfully.")
}

func (b *Bot) handleETL(chatID int64, text string) {
	ctx := context.Background()
	if _, err := b.requireAuth(ctx, chatID); err != nil {
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

	result, statusCode, err := b.etlClient.Run(source)
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
	if err != nil || time.Until(expiry) < 5*time.Minute {
		newTokens, statusCode, err := b.dataClient.RefreshToken(tokens.RefreshToken)
		if err != nil {
			b.store.DeleteTokens(ctx, chatID)
			return "", fmt.Errorf("session expired [%d]: please login again with /login <username> <password>", statusCode)
		}
		tokens = &cache.Tokens{
			AccessToken:  newTokens.AccessToken,
			RefreshToken: newTokens.RefreshToken,
		}
		b.store.SetTokens(ctx, chatID, tokens)
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
