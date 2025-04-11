package v1

import (
	"encoding/json"

	"github.com/qrave1/task-track/entity"
)

type Chat struct {
	ID    int64  // ID чата
	Users string // ID пользователей в чате в виде json массива
}

func NewChatFromEntity(c entity.Chat) (Chat, error) {
	rawUsers, err := json.Marshal(c.Users)
	if err != nil {
		return Chat{}, err
	}

	return Chat{
		ID:    c.ID,
		Users: string(rawUsers),
	}, nil
}

func NewEntityChat(c Chat) (entity.Chat, error) {
	var users []int64
	if err := json.Unmarshal([]byte(c.Users), &users); err != nil {
		return entity.Chat{}, err
	}

	return entity.Chat{
		ID:    c.ID,
		Users: users,
	}, nil
}
