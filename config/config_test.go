package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	os.Setenv("GYD_DATA_URL", "http://data")
	os.Setenv("GYD_ETL_URL", "http://etl")
	os.Setenv("GYD_LLM_CHATBOT_URL", "http://llm")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("GYD_DATA_URL")
		os.Unsetenv("GYD_ETL_URL")
		os.Unsetenv("GYD_LLM_CHATBOT_URL")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("PORT")
	}()

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
	os.Unsetenv("PORT")
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
}
