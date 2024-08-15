package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Frozelo/FeedBackManagerBot/fetcher"
	models "github.com/Frozelo/FeedBackManagerBot/model"
	"github.com/Frozelo/FeedBackManagerBot/notifier"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
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

	testId, err := strconv.ParseInt(os.Getenv("TEST_ID"), 10, 64)
	rssFetcher := fetcher.NewFetcher(1*time.Minute, []string{"test", "hey"})
	ntfr := notifier.NewNotifier(
		botAPI,
		10*time.Second,
		[]int64{testId},
		[]models.Article{
			{ID: 1, SourceID: 1, Title: "TestTitle1", Link: "TestLink1"},
			{ID: 2, SourceID: 1, Title: "TestTitle2", Link: "TestLink2"},
			{ID: 3, SourceID: 1, Title: "TestTitle3", Link: "TestLink3"}},
	)

	go func(ctx context.Context) {
		if err = rssFetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("Error starting fetcher: %v", err)
				return
			}
			log.Printf("[INFO] fetcher stopped")
		}
	}(ctx)

	go func(ctx context.Context) {
		if err = ntfr.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("Error starting notifier: %v", err)
				return
			}
			log.Printf("[INFO] notifier stopped")
		}
	}(ctx)

	for {
		select {
		case update := <-updates:
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			handleUpdate(updateCtx, update, botAPI)
			updateCancel()
		case <-ctx.Done():
			return
		}
	}
}

// TODO Solve update problem. With true handling
func handleUpdate(ctx context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(747067942, fmt.Sprintf("Got the new message from %v", update.Message.From.ID))
	bot.Send(msg)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ERROR] panic recovered: %v\n%s", p, string(debug.Stack()))
		}
	}()

	if (update.Message == nil || !update.Message.IsCommand()) && update.CallbackQuery == nil {
		log.Printf("[ERROR] callback query is nil: %v", update.Message)
		return
	}

	if !update.Message.IsCommand() {
		return
	}
	return
}
