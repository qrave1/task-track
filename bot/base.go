package bot

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qrave1/task-track/config"
	"github.com/qrave1/task-track/repository"
)

type Botik struct {
	bot      *tgbotapi.BotAPI
	taskRepo repository.TaskRepository

	updates tgbotapi.UpdatesChannel
}

func NewBotik(cfg *config.Config, taskRepo repository.TaskRepository) (*Botik, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = cfg.Debug
	slog.Info("Authorized on account", "username", bot.Self.UserName)

	return &Botik{bot: bot, taskRepo: taskRepo}, nil
}

func (b *Botik) Start() {
	slog.Info("Starting in debug mode (polling)")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.updates = b.bot.GetUpdatesChan(u)
}
