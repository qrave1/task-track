package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/qrave1/task-track/entity"
)

type TaskRepository interface {
	Create(ctx context.Context, task *entity.Task) error
	GetByID(ctx context.Context, id int64) (*entity.Task, error)
	List(ctx context.Context, ) ([]*entity.Task, error)
	Update(ctx context.Context, task *entity.Task) error
	Delete(ctx context.Context, id int64) error
}

// TaskRepositoryImpl Репозиторий для работы с заданиями
type TaskRepositoryImpl struct {
	db *sql.DB
}

func NewTaskRepositoryImpl(db *sql.DB) *TaskRepositoryImpl {
	return &TaskRepositoryImpl{db: db}
}

func (r *TaskRepositoryImpl) Create(ctx context.Context, task *entity.Task) error {
	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO tasks (title, description, reward, assignee, created_by) VALUES (?, ?, ?, ?, ?)",
		task.Title, task.Description, task.Reward, task.Assignee, task.CreatedBy,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *TaskRepositoryImpl) GetByID(ctx context.Context, id int64) (*entity.Task, error) {
	var task entity.Task
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, title, description, reward, assignee, created_by, created_at FROM tasks WHERE id = ?",
		id,
	).Scan(&task.ID, &task.Title, &task.Description, &task.Reward, &task.Assignee, &task.CreatedBy, &task.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepositoryImpl) List(ctx context.Context, ) ([]*entity.Task, error) {
	rows, err := r.db.QueryContext(
		ctx,
		"SELECT id, title, description, reward, assignee, created_by, created_at FROM tasks ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*entity.Task
	for rows.Next() {
		var task entity.Task
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Reward, &task.Assignee, &task.CreatedBy, &task.CreatedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (r *TaskRepositoryImpl) Update(ctx context.Context, task *entity.Task) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE tasks SET title = ?, description = ?, reward = ?, assignee = ? WHERE id = ?",
		task.Title, task.Description, task.Reward, task.Assignee, task.ID,
	)
	return err
}

func (r *TaskRepositoryImpl) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	return err
}
