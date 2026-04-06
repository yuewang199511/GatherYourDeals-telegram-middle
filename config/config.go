package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	TelegramBotToken string
	GYDDataURL       string
	GYDEtlURL        string
	GYDLLMChatbotURL string
	RedisURL         string
	Port             string

	// Circuit breaker settings (shared across all outbound clients).
	// CB_FAILURE_THRESHOLD: consecutive failures to open the circuit (default 5)
	// CB_SUCCESS_THRESHOLD: consecutive successes in half-open to close (default 2)
	// CB_OPEN_TIMEOUT:      duration to stay open before probing, e.g. "30s" (default 30s)
	CBFailureThreshold int
	CBSuccessThreshold int
	CBOpenTimeout      time.Duration
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		GYDDataURL:         os.Getenv("GYD_DATA_URL"),
		GYDEtlURL:          os.Getenv("GYD_ETL_URL"),
		GYDLLMChatbotURL:   os.Getenv("GYD_LLM_CHATBOT_URL"),
		RedisURL:           os.Getenv("REDIS_URL"),
		Port:               port,
		CBFailureThreshold: envInt("CB_FAILURE_THRESHOLD", 5),
		CBSuccessThreshold: envInt("CB_SUCCESS_THRESHOLD", 2),
		CBOpenTimeout:      envDuration("CB_OPEN_TIMEOUT", 30*time.Second),
	}
}

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
