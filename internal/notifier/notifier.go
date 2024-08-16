package notifier

import (
	"context"
	"fmt"
	"github.com/Frozelo/FeedBackManagerBot/internal/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"sync"
	"time"
)

type UserRepo interface {
	GetAllUsers(ctx context.Context) ([]models.TgUser, error)
	AddTgUser(ctx context.Context, tgUser models.TgUser) error
}

type Notifier struct {
	bot          *tgbotapi.BotAPI
	sendInterval time.Duration
	channelId    int64
	userRepo     UserRepo
	articles     []models.Article
}

func NewNotifier(bot *tgbotapi.BotAPI, sendInterval time.Duration, userRepo UserRepo, articles []models.Article) *Notifier {
	return &Notifier{bot: bot, sendInterval: sendInterval, userRepo: userRepo, articles: articles}
}

func (n *Notifier) Start(ctx context.Context) error {
	ticker := time.NewTicker(n.sendInterval)
	defer ticker.Stop()

	if err := n.Notify(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			if err := n.Notify(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Notifier) Notify(ctx context.Context) error {
	if len(n.articles) == 0 {
		return nil
	}
	articleToSend := n.articles[0]
	if err := n.Send(ctx, articleToSend); err != nil {
		return err
	}
	return nil
}

func (n *Notifier) Send(ctx context.Context, article models.Article) error {
	var wg sync.WaitGroup
	subscribers, err := n.userRepo.GetAllUsers(ctx)
	log.Printf("subscribers: %v", subscribers)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(
		article.Title,
	)
	for _, subscriber := range subscribers {
		wg.Add(1)
		go func(userId int64) {
			defer wg.Done()
			log.Printf("Sending message to user %d", userId)
			telegramMsg := tgbotapi.NewMessage(userId, msg)
			if _, err := n.bot.Send(telegramMsg); err != nil {
				log.Printf("[ERROR] failed to send message to user with such %v id", userId)
				log.Printf("[ERROR] %s", err.Error())
			}
		}(subscriber.TgId)
	}
	wg.Wait()
	return nil

}
