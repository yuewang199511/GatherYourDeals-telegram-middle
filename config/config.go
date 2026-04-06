package config

import "os"

type Config struct {
	TelegramBotToken string
	GYDDataURL       string
	GYDEtlURL        string
	GYDLLMChatbotURL string
	RedisURL         string
	Port             string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		GYDDataURL:       os.Getenv("GYD_DATA_URL"),
		GYDEtlURL:        os.Getenv("GYD_ETL_URL"),
		GYDLLMChatbotURL: os.Getenv("GYD_LLM_CHATBOT_URL"),
		RedisURL:         os.Getenv("REDIS_URL"),
		Port:             port,
	}
}
