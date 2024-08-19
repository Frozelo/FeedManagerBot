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
	GetAllNotPostedByUserSources(ctx context.Context, userID int64) ([]models.Article, error)
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

// TODO Decompose this method
func (n *Notifier) Notify(ctx context.Context) error {
	subscribers, err := n.userRepo.GetAllUsers(ctx)
	if err != nil {
		return err
	}
	log.Printf("subscribers: %v", subscribers)

	var wg sync.WaitGroup
	errChan := make(chan error, len(subscribers))
	mu := &sync.Mutex{}
	articlesToSend := make(map[int64]models.Article)

	for _, subscriber := range subscribers {
		wg.Add(1)

		go func(userID int64) {
			defer wg.Done()
			articles, err := n.articleRepo.GetAllNotPostedByUserSources(ctx, userID)
			if err != nil {
				errChan <- err
				return
			}
			if len(articles) == 0 {
				return
			}
			articleToSend := articles[0]
			mu.Lock()
			articlesToSend[articleToSend.ID] = articleToSend
			mu.Unlock()
			if err = n.Send(ctx, articleToSend, subscriber); err != nil {
				errChan <- err
			}
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

	for _, article := range articlesToSend {
		log.Println("trying to mark article", article.ID)
		if err := n.articleRepo.MarkAsPosted(ctx, article); err != nil {
			log.Printf("[ERROR] failed to mark article as posted: %s", err.Error())
			sendErrs = append(sendErrs, err)
		}
		log.Println("article marked", article.ID)
	}

	if len(sendErrs) > 0 {
		return fmt.Errorf("encountered errors during notification: %v", sendErrs)
	}

	return nil
}

func (n *Notifier) Send(ctx context.Context, article models.Article, subscriber models.TgUser) error {
	msg := n.formatMessage(article)

	log.Printf("Sending message: %v", msg)

	return nil
}

func (n *Notifier) sendMessageToUser(ctx context.Context, userId int64, article models.Article, msg string) error {
	telegramMsg := tgbotapi.NewMessage(userId, msg)
	telegramMsg.ParseMode = "Markdown"

	_, err := n.bot.Send(telegramMsg)
	if err != nil {
		log.Printf("[ERROR] failed to send message to user %d: %s", userId, err.Error())
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
