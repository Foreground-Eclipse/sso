package telegram

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func SendTokenViaTelegram(ctx context.Context, telegramname string) (bool, error) {
	bot, err := tgbotapi.NewBotAPI("6709769114:AAEaSj0tZdWduxmu6N5-gYlwYCjI75Sb8nM")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	// Replace '@non' with the actual chat ID of the user
	msg := tgbotapi.NewMessageToChannel("@Diabolikha", "Your message here")

	// Send the message
	_, err = bot.Send(msg)
	if err != nil {
		log.Panic(err)
	}
	return true, nil
}
