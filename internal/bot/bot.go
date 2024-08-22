package bot

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"runtime/debug"
	"time"
)

type Bot struct {
	bot *tgbotapi.BotAPI
	cmd map[string]ViewFunc
}

func New(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{bot: bot}
}

func (b *Bot) RegisterCmd(cmd string, viewFunc ViewFunc) {
	if b.cmd == nil {
		b.cmd = make(map[string]ViewFunc)
	}
	b.cmd[cmd] = viewFunc
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)
	for {
		select {
		case update := <-updates:
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			b.handleUpdate(updateCtx, update)
			updateCancel()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in handleUpdate", r, string(debug.Stack()))
		}
	}()

	if update.Message != nil {
		b.handleMessage(ctx, update)
	}
}

func (b *Bot) handleMessage(ctx context.Context, update tgbotapi.Update) {

	var view ViewFunc
	cmd := update.Message.Command()
	cmdView, ok := b.cmd[cmd]
	if !ok {
		return
	}
	view = cmdView

	if err := view(ctx, b.bot, update); err != nil {
		log.Printf("[ERROR] failed to execute view: %v", err)

		if _, err := b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Internal error")); err != nil {
			log.Printf("[ERROR] failed to send error message: %v", err)
		}
	}
}
