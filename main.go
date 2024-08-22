package main

import (
	"context"
	"errors"
	"github.com/Frozelo/FeedBackManagerBot/internal/bot"
	"github.com/Frozelo/FeedBackManagerBot/internal/config"
	"github.com/Frozelo/FeedBackManagerBot/internal/fetcher"
	"github.com/Frozelo/FeedBackManagerBot/internal/notifier"
	"github.com/Frozelo/FeedBackManagerBot/internal/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os/signal"
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

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	db, err := pgxpool.New(ctx, "postgres://admin:123123@localhost:5432/feedbot")
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUsersRepository(db)
	sourceRepo := repository.NewSourceRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	subsRepo := repository.NewSubscriberRepository(db)
	rssFetcher := fetcher.NewFetcher(articleRepo, 1*time.Minute, []string{"test", "hey"})
	ntfr := notifier.NewNotifier(
		botAPI,
		userRepo,
		articleRepo,
		subsRepo,
		30*time.Second,
	)
	feedBot := bot.New(botAPI)
	feedBot.RegisterCmd(
		"addsource",
		bot.CmdAddSource(sourceRepo),
	)

	feedBot.RegisterCmd(
		"listsources",
		bot.CmdListSource(sourceRepo),
	)

	feedBot.RegisterCmd(
		"start",
		bot.CmdStart(userRepo),
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

	if err := feedBot.Start(ctx); err != nil {
		log.Printf("[ERROR] failed to run botkit: %v", err)
	}

}
