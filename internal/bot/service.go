package bot

import (
	"context"
	"fmt"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sort"
	"strings"
)

type ViewFunc func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error

type SourceRepository interface {
	Add(ctx context.Context, source models.Source) error
	Sources(ctx context.Context) ([]models.Source, error)
}

type UserRepository interface {
	AddTgUser(ctx context.Context, tgUser models.TgUser) error
}

func CmdStart(userRepo UserRepository) ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		if err := userRepo.AddTgUser(ctx, models.TgUser{
			TgId:     update.Message.Chat.ID,
			Username: update.Message.From.UserName,
		}); err != nil {
			return err
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Hi there!"))
		if _, err := bot.Send(msg); err != nil {
			return err
		}
		return nil

	}
}

func CmdAddSource(sourceRepo SourceRepository) ViewFunc {
	type addSourceArgs struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Priority int    `json:"priority"`
	}
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		args, err := ParseJSON[addSourceArgs](update.Message.CommandArguments())
		if err != nil {
			return err
		}
		source := models.Source{
			Name:     args.Name,
			FeedURL:  args.URL,
			Priority: args.Priority,
		}
		if err = sourceRepo.Add(ctx, source); err != nil {
			return err
		}
		var (
			msgText = fmt.Sprintf(
				"Источник добавлен %v",
				source,
			)
			reply = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		)
		reply.ParseMode = "MarkdownV2"
		if _, err = bot.Send(reply); err != nil {
			return err
		}
		return nil
	}
}

func CmdListSource(sourceRepo SourceRepository) ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		sources, err := sourceRepo.Sources(ctx)
		if err != nil {
			return err
		}
		sort.SliceStable(sources, func(i, j int) bool {
			return sources[i].Name < sources[j].Name
		})
		var msgText string
		for _, source := range sources {
			msgText = fmt.Sprintf("Список источников \\(всего %d\\)\n\n%s",
				len(sources),
				strings.TrimSuffix(source.Name, "\\"),
			)

		}

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		reply.ParseMode = "MarkdownV2"

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil

	}
}
