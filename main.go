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
	subsRepo := repository.NewSubscriberRepository(db)
	rssFetcher := fetcher.NewFetcher(articleRepo, 1*time.Minute, []string{"test", "hey"})
	ntfr := notifier.NewNotifier(
		botAPI,
		userRepo,
		articleRepo,
		subsRepo,
		5*time.Second,
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
			handleUpdate(updateCtx, update, botAPI, userRepo, subsRepo)
			updateCancel()
		case <-ctx.Done():
			return
		}
	}
}

// TODO Solve update problem. With true handling
func handleUpdate(
	ctx context.Context,
	update tgbotapi.Update,
	bot *tgbotapi.BotAPI,
	userRepo *repository.UsersRepository,
	subsRepo *repository.SubscriptionRepository) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ERROR] panic recovered: %v\n%s", p, string(debug.Stack()))
		}
	}()

	if update.Message.Command() == "start" {
		if err := userRepo.AddTgUser(ctx, models.TgUser{
			TgId:     update.Message.From.ID,
			Username: update.Message.From.UserName,
		}); err != nil {
			msg := tgbotapi.NewMessage(update.Message.From.ID, fmt.Sprintf("Error adding user: %v", err))
			bot.Send(msg)
		}
	}

	// TODO Super stupid test code improve this
	switch update.Message.Text {
	case "1":
		log.Printf("Im here!")
		if err := subsRepo.Add(ctx, update.Message.From.ID, 1); err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error adding subscriber: %v", err))
			bot.Send(msg)

		}
	case "2":
		log.Printf("Im here!")
		if err := subsRepo.Add(ctx, update.Message.From.ID, 2); err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error adding subscriber: %v", err))
			bot.Send(msg)
		}
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Invalid input"))
		bot.Send(msg)
	}

	return
}
