package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qrave1/task-track/config"
	"github.com/qrave1/task-track/entity"
	"github.com/qrave1/task-track/repository"
)

type TaskBot struct {
	bot      *tgbotapi.BotAPI
	taskRepo repository.TaskRepository
	config   *config.Config
	ctx      context.Context
	cancel   context.CancelFunc
	updates  tgbotapi.UpdatesChannel
}

func NewFamilyTasksBot(cfg *config.Config, taskRepo repository.TaskRepository) (*TaskBot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = cfg.Debug
	slog.Info("Authorized on account", "username", bot.Self.UserName)

	ctx, cancel := context.WithCancel(context.Background())

	return &TaskBot{
		bot:      bot,
		taskRepo: taskRepo,
		config:   cfg,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (b *TaskBot) Start() error {
	if b.config.Debug {
		slog.Info("Starting in debug mode (polling)")
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		b.updates = b.bot.GetUpdatesChan(u)
	} else {
		slog.Info("Starting in production mode (webhooks)")
		wh, err := tgbotapi.NewWebhook(b.config.Telegram.Webhook.URL)
		if err != nil {
			return fmt.Errorf("failed to create webhook: %w", err)
		}

		_, err = b.bot.Request(wh)
		if err != nil {
			return fmt.Errorf("failed to set webhook: %w", err)
		}

		info, err := b.bot.GetWebhookInfo()
		if err != nil {
			return fmt.Errorf("failed to get webhook info: %w", err)
		}

		if info.LastErrorDate != 0 {
			slog.Error("Telegram callback failed", "error", info.LastErrorMessage)
		}

		b.updates = b.bot.ListenForWebhook("/" + b.bot.Token)
	}

	go b.handleUpdates()
	return nil
}

func (b *TaskBot) Stop() {
	b.cancel()
}

func (b *TaskBot) handleUpdates() {
	for update := range b.updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *TaskBot) handleMessage(msg *tgbotapi.Message) {
	if !b.isUserAllowed(msg.From.ID) {
		slog.Warn("Unauthorized access attempt", "user_id", msg.From.ID)
		return
	}

	switch msg.Text {
	case "/start", "/menu":
		b.sendMainMenu(msg.Chat.ID)
	case "/tasks":
		b.showTaskList(msg.Chat.ID, 0)
	default:
		b.handleCommandWithState(msg)
	}
}

func (b *TaskBot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	if !b.isUserAllowed(callback.From.ID) {
		slog.Warn("Unauthorized callback attempt", "user_id", callback.From.ID)
		return
	}

	parts := strings.Split(callback.Data, ":")
	if len(parts) < 1 {
		return
	}

	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	switch parts[0] {
	case "menu":
		b.editMainMenu(chatID, messageID)
	case "list":
		page := 0
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
		b.editTaskList(chatID, messageID, page)
	case "task":
		if len(parts) < 2 {
			return
		}
		taskID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}
		b.showTaskDetails(chatID, messageID, taskID)
	case "create":
		b.setUserState(callback.From.ID, "waiting_for_title")
		b.editMessage(chatID, messageID, "Введите название задания:", b.createCancelKeyboard())
	case "edit":
		if len(parts) < 2 {
			return
		}
		taskID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}
		b.setUserState(callback.From.ID, fmt.Sprintf("editing:%d:title", taskID))
		b.editMessage(chatID, messageID, "Введите новое название задания:", b.createCancelKeyboard())
	case "delete":
		if len(parts) < 2 {
			return
		}
		taskID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}
		b.deleteTask(chatID, messageID, taskID)
	case "cancel":
		b.clearUserState(callback.From.ID)
		b.editMainMenu(chatID, messageID)
	}
}

func (b *TaskBot) handleCommandWithState(msg *tgbotapi.Message) {
	state := b.getUserState(msg.From.ID)
	if state == "" {
		return
	}

	switch state {
	case "waiting_for_title":
		b.setUserState(msg.From.ID, "waiting_for_description")
		b.sendMessage(msg.Chat.ID, "Введите описание задания:", b.createCancelKeyboard())
		b.setUserData(msg.From.ID, "new_task:title", msg.Text)
	case "waiting_for_description":
		b.setUserState(msg.From.ID, "waiting_for_reward")
		b.sendMessage(msg.Chat.ID, "Введите награду за выполнение:", b.createCancelKeyboard())
		b.setUserData(msg.From.ID, "new_task:description", msg.Text)
	case "waiting_for_reward":
		b.setUserState(msg.From.ID, "waiting_for_assignee")
		b.sendMessage(msg.Chat.ID, "Кому назначено задание? (введите имя):", b.createAssigneesKeyboard())
		b.setUserData(msg.From.ID, "new_task:reward", msg.Text)
	case "waiting_for_assignee":
		title := b.getUserData(msg.From.ID, "new_task:title")
		description := b.getUserData(msg.From.ID, "new_task:description")
		reward := b.getUserData(msg.From.ID, "new_task:reward")

		task := &entity.Task{
			Title:       title,
			Description: description,
			Reward:      reward,
			Assignee:    msg.Text,
			CreatedBy:   msg.From.ID,
		}

		id, err := b.taskRepo.Create(task)
		if err != nil {
			slog.Error("Failed to create task", "error", err)
			b.sendMessage(msg.Chat.ID, "Ошибка при создании задания", nil)
			return
		}

		b.clearUserState(msg.From.ID)
		b.clearUserData(msg.From.ID, "new_task:*")
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("Задание создано! ID: %d", id), nil)
		b.showTaskList(msg.Chat.ID, 0)
	default:
		if strings.HasPrefix(state, "editing:") {
			parts := strings.Split(state, ":")
			if len(parts) < 3 {
				return
			}

			taskID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return
			}

			field := parts[2]
			task, err := b.taskRepo.GetByID(taskID)
			if err != nil || task == nil {
				b.sendMessage(msg.Chat.ID, "Задание не найдено", nil)
				return
			}

			switch field {
			case "title":
				task.Title = msg.Text
				b.setUserState(msg.From.ID, fmt.Sprintf("editing:%d:description", taskID))
				b.sendMessage(msg.Chat.ID, "Введите новое описание задания:", b.createCancelKeyboard())
			case "description":
				task.Description = msg.Text
				b.setUserState(msg.From.ID, fmt.Sprintf("editing:%d:reward", taskID))
				b.sendMessage(msg.Chat.ID, "Введите новую награду за выполнение:", b.createCancelKeyboard())
			case "reward":
				task.Reward = msg.Text
				b.setUserState(msg.From.ID, fmt.Sprintf("editing:%d:assignee", taskID))
				b.sendMessage(msg.Chat.ID, "Введите нового исполнителя:", b.createAssigneesKeyboard())
			case "assignee":
				task.Assignee = msg.Text
				if err := b.taskRepo.Update(task); err != nil {
					slog.Error("Failed to update task", "error", err)
					b.sendMessage(msg.Chat.ID, "Ошибка при обновлении задания", nil)
					return
				}

				b.clearUserState(msg.From.ID)
				b.sendMessage(msg.Chat.ID, "Задание успешно обновлено!", nil)
				b.showTaskDetails(msg.Chat.ID, 0, taskID)
				return
			}

			b.setUserData(msg.From.ID, fmt.Sprintf("editing_task:%d", taskID), task)
		}
	}
}

func (b *TaskBot) sendMainMenu(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🏠 Главное меню:")
	msg.ReplyMarkup = b.createMainMenuKeyboard()
	_, err := b.bot.Send(msg)
	if err != nil {
		slog.Error("Failed to send main menu", "error", err)
	}
}

func (b *TaskBot) editMainMenu(chatID int64, messageID int) {
	edit := tgbotapi.NewEditMessageTextAndMarkup(
		chatID,
		messageID,
		"🏠 Главное меню:",
		b.createMainMenuKeyboard(),
	)
	_, err := b.bot.Send(edit)
	if err != nil {
		slog.Error("Failed to edit main menu", "error", err)
	}
}

func (b *TaskBot) showTaskList(chatID int64, page int) {
	tasks, err := b.taskRepo.List()
	if err != nil {
		slog.Error("Failed to get tasks list", "error", err)
		b.sendMessage(chatID, "Ошибка при получении списка заданий", nil)
		return
	}

	if len(tasks) == 0 {
		b.sendMessage(chatID, "Нет созданных заданий", nil)
		return
	}

	// Простая пагинация - показываем по 5 заданий на страницу
	start := page * 5
	if start >= len(tasks) {
		start = 0
		page = 0
	}

	end := start + 5
	if end > len(tasks) {
		end = len(tasks)
	}

	var text strings.Builder
	text.WriteString("📝 Список заданий:\n\n")
	for i := start; i < end; i++ {
		task := tasks[i]
		text.WriteString(fmt.Sprintf("%d. %s (для %s)\n", task.ID, task.Title, task.Assignee))
	}

	keyboard := b.createTaskListKeyboard(tasks, page)
	b.sendMessage(chatID, text.String(), keyboard)
}

func (b *TaskBot) editTaskList(chatID int64, messageID int, page int) {
	tasks, err := b.taskRepo.List()
	if err != nil {
		slog.Error("Failed to get tasks list", "error", err)
		return
	}

	if len(tasks) == 0 {
		b.editMessage(chatID, messageID, "Нет созданных заданий", nil)
		return
	}

	start := page * 5
	if start >= len(tasks) {
		start = 0
		page = 0
	}

	end := start + 5
	if end > len(tasks) {
		end = len(tasks)
	}

	var text strings.Builder
	text.WriteString("📝 Список заданий:\n\n")
	for i := start; i < end; i++ {
		task := tasks[i]
		text.WriteString(fmt.Sprintf("%d. %s (для %s)\n", task.ID, task.Title, task.Assignee))
	}

	keyboard := b.createTaskListKeyboard(tasks, page)
	b.editMessage(chatID, messageID, text.String(), keyboard)
}

func (b *TaskBot) showTaskDetails(chatID int64, messageID int, taskID int64) {
	task, err := b.taskRepo.GetByID(taskID)
	if err != nil || task == nil {
		slog.Error("Failed to get task", "id", taskID, "error", err)
		b.sendMessage(chatID, "Задание не найдено", nil)
		return
	}

	text := fmt.Sprintf(
		"📌 Задание #%d\n\n"+
			"🔹 Название: %s\n"+
			"🔹 Описание: %s\n"+
			"🔹 Награда: %s\n"+
			"🔹 Исполнитель: %s\n"+
			"🔹 Создано: %s\n",
		task.ID,
		task.Title,
		task.Description,
		task.Reward,
		task.Assignee,
		task.CreatedAt.Format("02.01.2006 15:04"),
	)

	keyboard := b.createTaskDetailsKeyboard(taskID)
	if messageID == 0 {
		b.sendMessage(chatID, text, keyboard)
	} else {
		b.editMessage(chatID, messageID, text, keyboard)
	}
}

func (b *TaskBot) deleteTask(chatID int64, messageID int, taskID int64) {
	if err := b.taskRepo.Delete(taskID); err != nil {
		slog.Error("Failed to delete task", "id", taskID, "error", err)
		b.sendMessage(chatID, "Ошибка при удалении задания", nil)
		return
	}

	b.editMessage(chatID, messageID, "Задание успешно удалено!", nil)
	time.Sleep(2 * time.Second)
	b.showTaskList(chatID, 0)
}

func (b *TaskBot) sendMessage(chatID int64, text string, replyMarkup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = replyMarkup
	b.bot.Send(msg)
}

func (b *TaskBot) editMessage(chatID int64, messageID int, text string, replyMarkup interface{}) {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, replyMarkup.(tgbotapi.InlineKeyboardMarkup))
	b.bot.Send(edit)
}

func (b *TaskBot) createMainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Список заданий", "list:0"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать задание", "create"),
		),
	)
}

func (b *TaskBot) createTaskListKeyboard(tasks []*entity.Task, page int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки заданий
	start := page * 5
	end := start + 5
	if end > len(tasks) {
		end = len(tasks)
	}

	for i := start; i < end; i++ {
		task := tasks[i]
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%d. %s", task.ID, task.Title),
				fmt.Sprintf("task:%d", task.ID),
			),
		))
	}

	// Кнопки пагинации
	var paginationRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("list:%d", page-1)))
	}
	if end < len(tasks) {
		paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("list:%d", page+1)))
	}
	if len(paginationRow) > 0 {
		rows = append(rows, paginationRow)
	}

	// Кнопка возврата в меню
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 В меню", "menu"),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *TaskBot) createTaskDetailsKeyboard(taskID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit:%d", taskID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("delete:%d", taskID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 К списку", "list:0"),
		),
	)
}

func (b *TaskBot) createCancelKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)
}

func (b *TaskBot) createAssigneesKeyboard() tgbotapi.InlineKeyboardMarkup {
	// Здесь можно добавить динамическое получение списка возможных исполнителей
	// Для простоты используем фиксированные значения
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Муж", "assignee:Муж"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Жена", "assignee:Жена"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)
}

// Простые методы для хранения состояния пользователя (в реальном приложении лучше использовать Redis или БД)
func (b *TaskBot) setUserState(userID int64, state string) {
	// В реальном приложении сохранять в БД или Redis
}

func (b *TaskBot) getUserState(userID int64) string {
	// В реальном приложении получать из БД или Redis
	return ""
}

func (b *TaskBot) clearUserState(userID int64) {
	// В реальном приложении удалять из БД или Redis
}

func (b *TaskBot) setUserData(userID int64, key string, value string) {
	// В реальном приложении сохранять в БД или Redis
}

func (b *TaskBot) getUserData(userID int64, key string) string {
	// В реальном приложении получать из БД или Redis
	return ""
}

func (b *TaskBot) clearUserData(userID int64, pattern string) {
	// В реальном приложении удалять из БД или Redis
}

func (b *TaskBot) isUserAllowed(userID int64) bool {
	for _, id := range b.config.AllowedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}
