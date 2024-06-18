package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	grpcapp "sso/sso/cmd/inter/app/grpc"
	"sso/sso/cmd/inter/services/auth"
	"sso/sso/cmd/inter/storage/sqlite"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	tokenTTL time.Duration,
) *App {

	storage, err := sqlite.New(storagePath)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, storage, storage, tokenTTL)
	grpcApp := grpcapp.New(log, grpcPort, authService)

	return &App{GRPCSrv: grpcApp}
}

type TelegramAuth struct {
	log              *slog.Logger
	telegramProvider TelegramProvider
}

type TelegramProvider interface {
	ConfirmAccountTG(ctx context.Context, telegramName string) int
}

func NewTelegramBot(tga *TelegramAuth) {
	const op = "app.app.NewTelegramBot"

	storage, err := sqlite.New("F:/mainwork/sso/inter/config/storage/sso.db")
	tga.telegramProvider = storage
	bot, err := tgbotapi.NewBotAPI("6709769114:AAEaSj0tZdWduxmu6N5-gYlwYCjI75Sb8nM")
	if err != nil {
		log.Fatalf("%s: %v", op, err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("%s: %v", op, err)
	}

	for update := range updates {
		if update.CallbackQuery != nil {
			// Handle callback query from inline button
			callbackData := update.CallbackQuery.Data
			telegramName := update.CallbackQuery.From.UserName
			fmt.Println(telegramName)
			ctx := context.Background()

			var responseText string
			switch tga.telegramProvider.ConfirmAccountTG(ctx, telegramName) {
			case -1:
				responseText = "You already verified"

			case 0:
				responseText = "You aren't registered yet"
			case 1:
				responseText = "You are now verified"
			default:
				responseText = "An unexpected error occurred"
			}

			callbackConfig := tgbotapi.NewCallback(update.CallbackQuery.ID, callbackData)
			if _, err := bot.AnswerCallbackQuery(callbackConfig); err != nil {
				log.Fatalf("%s: %v", op, err)
			}

			// Respond to the button press
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, responseText)
			if _, err := bot.Send(msg); err != nil {
				log.Fatalf("%s: %v", op, err)
			}
		} else if update.Message != nil {
			// Handle regular message
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Press the button to get your verification code")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Verify me 🔥", "verify_me"),
				),
			)
			if _, err := bot.Send(msg); err != nil {
				log.Fatalf("%s: %v", op, err)
			}
		}
	}
}
