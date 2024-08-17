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

type ArticleRepo interface {
	MarkAsPosted(ctx context.Context, article models.Article) error
	GetAllNotPosted(ctx context.Context) ([]models.Article, error)
	GetAll(ctx context.Context) ([]models.Article, error)
}

type Notifier struct {
	bot          *tgbotapi.BotAPI
	articleRepo  ArticleRepo
	userRepo     UserRepo
	sendInterval time.Duration
}

func NewNotifier(bot *tgbotapi.BotAPI, userRepo UserRepo, articles ArticleRepo, sendInterval time.Duration) *Notifier {
	return &Notifier{bot: bot, userRepo: userRepo, articleRepo: articles, sendInterval: sendInterval}
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
	articles, err := n.articleRepo.GetAllNotPosted(ctx)
	if err != nil {
		return err
	}
	articleToSend := articles[0]

	subscribers, err := n.userRepo.GetAllUsers(ctx)
	if err != nil {
		return err
	}

	if err = n.Send(ctx, articleToSend, subscribers); err != nil {
		return err
	}
	return nil
}

func (n *Notifier) Send(ctx context.Context, article models.Article, subs []models.TgUser) error {
	var wg sync.WaitGroup
	log.Printf("subscribers: %v", subs)

	msg := fmt.Sprintf(
		"*Title:* %s\n*Link:* [%s](%s)\n*Published At:* %s",
		article.Title,
		article.Title,
		article.Link,
		article.PublishedAt.Format("2006-01-02 15:04:05"),
	)

	for _, subscriber := range subs {
		wg.Add(1)
		go func(userId int64) {
			defer wg.Done()
			// Super test stupid code (temporary)
			userSource := make(map[int64]int64)
			userSource[subscriber.TgId] = 2
			if userSource[subscriber.TgId] == article.SourceID {
				log.Printf("[INFO] sending message to user %d", userId)
				telegramMsg := tgbotapi.NewMessage(userId, msg)
				telegramMsg.ParseMode = "Markdown"
				if _, err := n.bot.Send(telegramMsg); err != nil {
					log.Printf("[ERROR] failed to send message to user with such %v id", userId)
					log.Printf("[ERROR] %s", err.Error())
				}
			}

		}(subscriber.TgId)
	}

	wg.Wait()

	// После успешной отправки всем пользователям, помечаем статью как отправленную
	if err := n.articleRepo.MarkAsPosted(ctx, article); err != nil {
		log.Printf("[ERROR] failed to mark article as posted: %s", err.Error())
		return err
	}

	return nil
}
