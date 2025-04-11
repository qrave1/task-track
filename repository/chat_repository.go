package repository

import (
	"database/sql"

	"github.com/qrave1/task-track/entity"
)

type ChatRepository interface {
	Create(task *entity.Task) (int64, error)
	GetByID(id int64) (*entity.Task, error)
	List() ([]*entity.Task, error)
	Update(task *entity.Task) error
	Delete(id int64) error
}

// ChatRepositoryImpl Репозиторий для работы с пользователями бота
type ChatRepositoryImpl struct {
	db *sql.DB
}

func NewChatRepositoryImpl(db *sql.DB) *ChatRepositoryImpl {
	return &ChatRepositoryImpl{db: db}
}
