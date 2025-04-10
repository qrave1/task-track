package bot

import (
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Botik) HandleUpdates() {
	for update := range b.updates {
		switch {
		case update.Message != nil:
			switch {
			case update.Message.IsCommand():
				slog.Info("got new command")

				b.handleCommand(update.Message)
			default:
				slog.Info("got new message")

				b.handleMessage(update.Message)
			}
		case update.CallbackQuery != nil:
			slog.Info("got new callback query")

			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Botik) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.StartCmd()
	case "help":
		b.HelpCmd()
	}
}

func (b *Botik) handleMessage(msg *tgbotapi.Message) {}

func (b *Botik) handleCallbackQuery(cb *tgbotapi.CallbackQuery) {}
