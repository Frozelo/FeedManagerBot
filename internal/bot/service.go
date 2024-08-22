package bot

import (
	"context"
	"fmt"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"sort"
	"strconv"
	"strings"
)

type ViewFunc func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error
type CallBackFunc func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error

type SourceRepository interface {
	Add(ctx context.Context, source models.Source) error
	Sources(ctx context.Context) ([]models.Source, error)
}
type SubsRepo interface {
	Add(ctx context.Context, userId int64, sourceId int64) error
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
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		sources, err := sourceRepo.Sources(ctx)
		if err != nil {
			return err
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("На какую категорию вы хотите подписаться?"))
		var rows [][]tgbotapi.InlineKeyboardButton
		for _, source := range sources {
			log.Printf("The sources is %v", source)
			button := tgbotapi.NewInlineKeyboardButtonData(source.Name, fmt.Sprintf("source_add:%d", source.ID))
			row := tgbotapi.NewInlineKeyboardRow(button)
			rows = append(rows, row)
		}
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
		bot.Send(msg)
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

func CallbackAddSource(subsRepo SubsRepo) CallBackFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		callbackData := update.CallbackQuery.Data
		parts := strings.Split(callbackData, ":")
		if len(parts) != 2 || parts[0] != "source_add" {
			return fmt.Errorf("invalid callback data")
		}
		sourceID, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}

		if err = subsRepo.Add(ctx, update.CallbackQuery.Message.Chat.ID, int64(sourceID)); err != nil {
			return err
		}
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Вы успешно подписались на источник с ID %d", sourceID))
		if _, err := bot.Send(msg); err != nil {
			return err
		}
		return nil
	}
}
