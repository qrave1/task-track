package entity

import "time"

type Task struct {
	ID          int64
	Title       string
	Description string
	Reward      string
	Assignee    string
	CreatedBy   int64
	CreatedAt   time.Time
}
