package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewStore(rdb, 7*24*time.Hour, 7*24*time.Hour), mr
}

func TestGetSetTokens(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	got, err := store.GetTokens(ctx, 1)
	if err != nil || got != nil {
		t.Fatalf("expected nil tokens for missing key, got %v, %v", got, err)
	}

	tokens := &Tokens{AccessToken: "access", RefreshToken: "refresh"}
	if err := store.SetTokens(ctx, 1, tokens); err != nil {
		t.Fatalf("SetTokens: %v", err)
	}

	got, err = store.GetTokens(ctx, 1)
	if err != nil {
		t.Fatalf("GetTokens: %v", err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" {
		t.Errorf("got %+v, want %+v", got, tokens)
	}
}

func TestDeleteTokens(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	if err := store.SetTokens(ctx, 1, &Tokens{AccessToken: "a", RefreshToken: "b"}); err != nil {
		t.Fatalf("SetTokens: %v", err)
	}
	if err := store.DeleteTokens(ctx, 1); err != nil {
		t.Fatalf("DeleteTokens: %v", err)
	}

	got, err := store.GetTokens(ctx, 1)
	if err != nil || got != nil {
		t.Errorf("expected nil after delete, got %v, %v", got, err)
	}
}

func TestGetSetHistory(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	got, err := store.GetHistory(ctx, 1)
	if err != nil || len(got) != 0 {
		t.Fatalf("expected empty history for missing key, got %v, %v", got, err)
	}

	msgs := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	if err := store.SetHistory(ctx, 1, msgs); err != nil {
		t.Fatalf("SetHistory: %v", err)
	}

	got, err = store.GetHistory(ctx, 1)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(got) != 2 || got[0].Role != "user" || got[1].Content != "hi" {
		t.Errorf("got %+v", got)
	}
}

func TestDeleteHistory(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	if err := store.SetHistory(ctx, 1, []Message{{Role: "user", Content: "hello"}}); err != nil {
		t.Fatalf("SetHistory: %v", err)
	}
	if err := store.DeleteHistory(ctx, 1); err != nil {
		t.Fatalf("DeleteHistory: %v", err)
	}

	got, err := store.GetHistory(ctx, 1)
	if err != nil || len(got) != 0 {
		t.Errorf("expected empty after delete, got %v, %v", got, err)
	}
}
