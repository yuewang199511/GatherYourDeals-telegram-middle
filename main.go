package main

import (
	"log"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/redis/go-redis/v9"

	"github.com/yuewang199511/GatherYourDeals-telegram-middle/bot"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/cache"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/client"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/config"
	"github.com/yuewang199511/GatherYourDeals-telegram-middle/handler"
)

func main() {
	cfg := config.Load()

	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("telegram bot init failed: %v", err)
	}

	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("invalid redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpt)

	store := cache.NewStore(redisClient)
	dataClient := client.NewDataClient(cfg.GYDDataURL)
	etlClient := client.NewETLClient(cfg.GYDEtlURL)
	llmClient := client.NewLLMClient(cfg.GYDLLMChatbotURL)

	b := bot.New(botAPI, dataClient, etlClient, llmClient, store)

	r := gin.Default()
	r.GET("/health", handler.Health)
	r.POST("/webhook", handler.Webhook(b))

	log.Printf("starting server on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
