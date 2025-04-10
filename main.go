package task_track

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/qrave1/task-track/bot"
	"github.com/qrave1/task-track/config"
)

func migrateDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT,
			reward TEXT,
			assignee TEXT NOT NULL,
			created_by INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)

	return err
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.New()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Создание и запуск бота
	b, err := bot.NewFamilyTasksBot(cfg)
	if err != nil {
		slog.Error("Failed to create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer b.Stop()

	if err := b.Start(); err != nil {
		slog.Error("Failed to start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Ожидание завершения (в режиме webhook)
	if !cfg.Debug {
		slog.Info("Server started", "URL", cfg.Telegram.Webhook.URL)
		err = http.ListenAndServe(":"+strconv.Itoa(cfg.Telegram.Webhook.Port), nil)
		if err != nil {
			slog.Error("failed to start webhook mode", slog.String("error", err.Error()))
			os.Exit(1)
		}
	} else {
		// В режиме polling просто ждем сигнала завершения
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		slog.Info("Shutting down...")
	}
}
