package bot

import (
	"context"
	"errors"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qrave1/task-track/entity"
	"github.com/qrave1/task-track/lang"
	"github.com/qrave1/task-track/repository"
)

const (
	StartCommand    = "start"
	HelpCommand     = "help"
	NewCommand      = "new"
	InitChatCommand = "init_chat"
)

func (b *Botik) StartCmd(chatID int64, msgID int) {
	if err := b.sendText(chatID, lang.Start, WithReply(msgID)); err != nil {
		slog.Error("handle /start command", slog.String("error", err.Error()))
	}
}

func (b *Botik) HelpCmd(chatID int64, msgID int) {
	if err := b.sendText(chatID, lang.Help, WithReply(msgID)); err != nil {
		slog.Error("handle /help command", slog.String("error", err.Error()))
	}
}

func (b *Botik) NewCmd(chatID int64, msgID int) {

}

func (b *Botik) initChatCmd(chatID int64, msgID int) {
	sentStub := false
	defer func() {
		if sentStub {
			err := b.sendText(chatID, lang.FailedStub, WithReply(msgID))
			if err != nil {
				slog.Error(err.Error())
			}
		}
	}()

	chat, err := b.chatRepo.GetByID(context.Background(), chatID)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			chatMembers, err := b.bot.GetChatAdministrators(
				tgbotapi.ChatAdministratorsConfig{
					ChatConfig: tgbotapi.ChatConfig{ChatID: chatID},
				},
			)
			if err != nil {
				slog.Error("failed to get chat members: %v", err)
				sentStub = true
				return
			}

			var chatUsers []int64
			for _, chatMember := range chatMembers {
				chatUsers = append(chatUsers, chatMember.User.ID)
			}

			chat = entity.NewChat(chatID, chatUsers)

			err = b.chatRepo.Create(context.Background(), chat)
			if err != nil {
				slog.Error("failed to create chat", slog.String("error", err.Error()))
				sentStub = true
				return
			}
		} else {
			slog.Error("failed to get chat by ID", slog.String("error", err.Error()))
			sentStub = true
			return
		}
	}
}
