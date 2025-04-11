package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/qrave1/task-track/entity"
	v1 "github.com/qrave1/task-track/repository/v1"
)

var ErrChatNotFound = errors.New("chat not found")

type ChatRepository interface {
	Create(ctx context.Context, chat entity.Chat) error
	GetByID(ctx context.Context, id int64) (entity.Chat, error)
	//List() ([]*entity.Chat, error)
	//Update(task *entity.Chat) error
	//Delete(id int64) error
}

// ChatRepositoryImpl Репозиторий для работы с пользователями бота
type ChatRepositoryImpl struct {
	db *sql.DB
}

func NewChatRepositoryImpl(db *sql.DB) *ChatRepositoryImpl {
	return &ChatRepositoryImpl{db: db}
}

func (c *ChatRepositoryImpl) Create(ctx context.Context, chat entity.Chat) error {
	dbChat, err := v1.NewChatFromEntity(chat)
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(
		ctx,
		`INSERT INTO chats (id, users) VALUES (?, ?)`,
		dbChat.ID,
		dbChat.Users,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *ChatRepositoryImpl) GetByID(ctx context.Context, id int64) (entity.Chat, error) {
	var chat v1.Chat
	err := c.db.QueryRowContext(
		ctx,
		"SELECT id, users FROM chats WHERE id = ?",
		id,
	).Scan(&chat.ID, &chat.Users)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Chat{}, ErrChatNotFound
		}
		return entity.Chat{}, err
	}

	entityChat, err := v1.NewEntityChat(chat)
	if err != nil {
		return entity.Chat{}, err
	}

	return entityChat, nil
}
