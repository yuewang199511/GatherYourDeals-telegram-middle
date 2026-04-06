package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("GYD_DATA_URL", "http://data")
	t.Setenv("GYD_ETL_URL", "http://etl")
	t.Setenv("GYD_LLM_CHATBOT_URL", "http://llm")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("PORT", "9090")

	cfg := Load()

	if cfg.TelegramBotToken != "test-token" {
		t.Errorf("TelegramBotToken = %q, want %q", cfg.TelegramBotToken, "test-token")
	}
	if cfg.GYDDataURL != "http://data" {
		t.Errorf("GYDDataURL = %q, want %q", cfg.GYDDataURL, "http://data")
	}
	if cfg.GYDEtlURL != "http://etl" {
		t.Errorf("GYDEtlURL = %q, want %q", cfg.GYDEtlURL, "http://etl")
	}
	if cfg.GYDLLMChatbotURL != "http://llm" {
		t.Errorf("GYDLLMChatbotURL = %q, want %q", cfg.GYDLLMChatbotURL, "http://llm")
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("RedisURL = %q, want %q", cfg.RedisURL, "redis://localhost:6379")
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
}

func TestLoadDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
}
