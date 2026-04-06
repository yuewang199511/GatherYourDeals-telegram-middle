package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/yuewang199511/GatherYourDeals-telegram-middle/bot"
)

func Webhook(b *bot.Bot) gin.HandlerFunc {
	return func(c *gin.Context) {
		var update tgbotapi.Update
		if err := c.ShouldBindJSON(&update); err != nil {
			c.Status(http.StatusOK)
			return
		}
		c.Status(http.StatusOK)
		go b.Dispatch(update)
	}
}
