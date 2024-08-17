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

type SubsRepo interface {
	GetSourcesByUserID(ctx context.Context, userID int64) ([]models.Source, error)
}

type Notifier struct {
	bot          *tgbotapi.BotAPI
	articleRepo  ArticleRepo
	userRepo     UserRepo
	subsRepo     SubsRepo
	sendInterval time.Duration
}

func NewNotifier(bot *tgbotapi.BotAPI, userRepo UserRepo, articles ArticleRepo, subs SubsRepo, sendInterval time.Duration) *Notifier {
	return &Notifier{bot: bot, userRepo: userRepo, articleRepo: articles, subsRepo: subs, sendInterval: sendInterval}
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
	if len(articles) == 0 {
		return nil
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
	msg := n.formatMessage(article)

	sendErrs := n.sendMessagesToAllUsers(ctx, article, subs, msg)

	if len(sendErrs) > 0 {
		return fmt.Errorf("encountered errors during message sending: %v", sendErrs)
	}

	if err := n.articleRepo.MarkAsPosted(ctx, article); err != nil {
		log.Printf("[ERROR] failed to mark article as posted: %s", err.Error())
		return err
	}

	return nil
}

func (n *Notifier) formatMessage(article models.Article) string {
	return fmt.Sprintf(
		"*Title:* %s\n*Link:* [%s](%s)\n*Published At:* %s",
		article.Title,
		article.Title,
		article.Link,
		article.PublishedAt.Format("2006-01-02 15:04:05"),
	)
}

func (n *Notifier) sendMessagesToAllUsers(ctx context.Context, article models.Article, subs []models.TgUser, msg string) []error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(subs))

	for _, subscriber := range subs {
		wg.Add(1)

		go func(userId int64) {
			defer wg.Done()
			n.sendMessageToUser(ctx, userId, article, msg, errChan)
		}(subscriber.TgId)
	}

	wg.Wait()
	close(errChan)

	var sendErrs []error
	for err := range errChan {
		if err != nil {
			log.Println(err)
			sendErrs = append(sendErrs, err)
		}
	}

	return sendErrs
}

func (n *Notifier) sendMessageToUser(ctx context.Context, userId int64, article models.Article, msg string, errChan chan<- error) {
	userSources, err := n.subsRepo.GetSourcesByUserID(ctx, userId)
	if err != nil {
		errChan <- fmt.Errorf("[ERROR] failed to find user subs for user %v id: %w", userId, err)
		return
	}

	for _, source := range userSources {
		if source.ID == article.SourceID {
			telegramMsg := tgbotapi.NewMessage(userId, msg)
			telegramMsg.ParseMode = "Markdown"

			if _, err := n.bot.Send(telegramMsg); err != nil {
				errChan <- fmt.Errorf("[ERROR] failed to send message to user with id %v: %w", userId, err)
			}
			break
		}
	}
}
