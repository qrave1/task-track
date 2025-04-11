package bot

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qrave1/task-track/lang"
)

func (b *Botik) handleUpdates() {
	for update := range b.updates {
		switch {
		case update.Message != nil:

			switch {
			case update.Message.IsCommand():
				slog.Info(
					"got new command",
					slog.String("command", update.Message.Command()),
				)

				b.handleCommand(update.Message)
			default:
				slog.Info(
					"got new message",
					slog.String(update.Message.Text, update.Message.Text),
				)

				b.handleMessage(update.Message)
			}
		case update.CallbackQuery != nil:
			slog.Info("got new callback query")

			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Botik) handleMessage(msg *tgbotapi.Message) {
	// События, при добавлении новых участников
	if msg.NewChatMembers != nil {
		b.handleNewChatMember(msg)
	}
}

func (b *Botik) handleCallbackQuery(cb *tgbotapi.CallbackQuery) {

}

func (b *Botik) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case StartCommand:
		b.StartCmd(msg.Chat.ID, msg.MessageID)
	case HelpCommand:
		b.HelpCmd(msg.Chat.ID, msg.MessageID)
	case NewCommand:
		return
	case InitChatCommand:
		b.initChatCmd(msg.Chat.ID, msg.MessageID)
	}
}

func (b *Botik) handleNewChatMember(msg *tgbotapi.Message) {
	for _, member := range msg.NewChatMembers {
		// Если новый пользователь это сам бот
		if member.UserName == b.bot.Self.UserName {
			slog.Info(fmt.Sprintf("added to %s (%s) with ID %d", msg.Chat.Title, msg.Chat.Type, msg.Chat.ID))

			// Отправляем приветственное сообщение
			if err := b.sendText(msg.Chat.ID, lang.BotAddedToGroup); err != nil {
				slog.Error(err.Error())
			}
		}
	}
}
