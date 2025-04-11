package entity

type Chat struct {
	ID    int64   // ID чата
	Users []int64 // ID пользователей в чате
}

func NewChat(ID int64, users []int64) Chat {
	return Chat{ID: ID, Users: users}
}
