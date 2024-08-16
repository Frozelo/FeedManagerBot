package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Frozelo/FeedBackManagerBot/internal/config"
	"github.com/Frozelo/FeedBackManagerBot/internal/fetcher"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/Frozelo/FeedBackManagerBot/internal/notifier"
	"github.com/Frozelo/FeedBackManagerBot/internal/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

const cfgPath = "internal/config/config.yaml"

func main() {
	cfg, err := config.New(cfgPath)

	if err != nil {
		log.Fatal(err)
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramBot.Token)

	if err != nil {
		log.Panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log.Printf("Connected to the database")
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := botAPI.GetUpdatesChan(u)

	db, err := pgxpool.New(ctx, "postgres://admin:123123@localhost:5432/feedbot")
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUsersRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	rssFetcher := fetcher.NewFetcher(articleRepo, 1*time.Minute, []string{"test", "hey"})
	ntfr := notifier.NewNotifier(
		botAPI,
		userRepo,
		articleRepo,
		15*time.Second,
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
			handleUpdate(updateCtx, update, botAPI, userRepo)
			updateCancel()
		case <-ctx.Done():
			return
		}
	}
}

// TODO Solve update problem. With true handling
func handleUpdate(ctx context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI, userRepo *repository.UsersRepository) {
	msg := tgbotapi.NewMessage(747067942, fmt.Sprintf("Got the new message from %v", update.Message.From.ID))
	if err := userRepo.AddTgUser(ctx, models.TgUser{
		TgId:     update.Message.From.ID,
		Username: update.Message.From.UserName,
	}); err != nil {
		msg = tgbotapi.NewMessage(update.Message.From.ID, fmt.Sprintf("Error adding user: %v", err))
		bot.Send(msg)
	}

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
