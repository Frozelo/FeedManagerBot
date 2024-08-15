package main

import (
	"context"
	"errors"
	"github.com/Frozelo/FeedBackManagerBot/fetcher"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	botAPI, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))

	if err != nil {
		log.Panic(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := botAPI.GetUpdatesChan(u)

	rssFetcher := fetcher.NewFetcher(1*time.Minute, []string{"test", "hey"})

	go func(ctx context.Context) {
		if err = rssFetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("Error starting fetcher: %v", err)
				return
			}
			log.Printf("[INFO] fetcher stopped")
		}
	}(ctx)

	for update := range updates {
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID
		}
	}

}
