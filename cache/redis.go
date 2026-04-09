package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Store struct {
	rdb        *redis.Client
	tokenTTL   time.Duration
	historyTTL time.Duration
}

func NewStore(rdb *redis.Client, tokenTTL, historyTTL time.Duration) *Store {
	return &Store{rdb: rdb, tokenTTL: tokenTTL, historyTTL: historyTTL}
}

func tokenKey(chatID int64) string {
	return fmt.Sprintf("tokens:%d", chatID)
}

func historyKey(chatID int64) string {
	return fmt.Sprintf("history:%d", chatID)
}

func (s *Store) GetTokens(ctx context.Context, chatID int64) (*Tokens, error) {
	val, err := s.rdb.Get(ctx, tokenKey(chatID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var t Tokens
	if err := json.Unmarshal([]byte(val), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) SetTokens(ctx context.Context, chatID int64, t *Tokens) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, tokenKey(chatID), data, s.tokenTTL).Err()
}

func (s *Store) DeleteTokens(ctx context.Context, chatID int64) error {
	return s.rdb.Del(ctx, tokenKey(chatID)).Err()
}

func (s *Store) GetHistory(ctx context.Context, chatID int64) ([]Message, error) {
	val, err := s.rdb.Get(ctx, historyKey(chatID)).Result()
	if err == redis.Nil {
		return []Message{}, nil
	}
	if err != nil {
		return nil, err
	}
	var msgs []Message
	if err := json.Unmarshal([]byte(val), &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (s *Store) SetHistory(ctx context.Context, chatID int64, msgs []Message) error {
	data, err := json.Marshal(msgs)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, historyKey(chatID), data, s.historyTTL).Err()
}

func (s *Store) DeleteHistory(ctx context.Context, chatID int64) error {
	return s.rdb.Del(ctx, historyKey(chatID)).Err()
}
