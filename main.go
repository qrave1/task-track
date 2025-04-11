package main

import (
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "modernc.org/sqlite"

	"github.com/qrave1/task-track/bot"
	"github.com/qrave1/task-track/config"
	"github.com/qrave1/task-track/repository"
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
		);
		
		CREATE TABLE IF NOT EXISTS chats (
		    id INTEGER PRIMARY KEY,
		    users TEXT NOT NULL
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

	db, err := sql.Open("sqlite", cfg.Database.Path)
	if err != nil {
		slog.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	err = migrateDB(db)
	if err != nil {
		slog.Error("failed to migrate database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	taskRepo := repository.NewTaskRepositoryImpl(db)
	chatRepo := repository.NewChatRepositoryImpl(db)

	b, err := bot.NewBotik(cfg, taskRepo, chatRepo)
	if err != nil {
		slog.Error("failed to create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	b.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	slog.Info("Shutting down...")
}
