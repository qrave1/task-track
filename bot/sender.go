package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MessageOption определяет тип функции-опции
type MessageOption func(*tgbotapi.MessageConfig)

// WithReply добавляет опцию ответа на сообщение
func WithReply(messageID int) MessageOption {
	return func(msg *tgbotapi.MessageConfig) {
		msg.ReplyToMessageID = messageID
	}
}

// WithParseMode добавляет опцию режима парсинга (Markdown/HTML)
func WithParseMode(mode string) MessageOption {
	return func(msg *tgbotapi.MessageConfig) {
		msg.ParseMode = mode
	}
}

func (b *Botik) sendText(chatID int64, text string, opts ...MessageOption) error {
	msg := tgbotapi.NewMessage(chatID, text)

	// Применяем все переданные опции
	for _, opt := range opts {
		opt(&msg)
	}

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}
