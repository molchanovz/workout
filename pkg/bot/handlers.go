package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"workoutbot/pkg/workout"

	"workoutbot/pkg/db"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/vmkteam/embedlog"
)

const (
	StartCommand  = "/start"
	StartCallback = "start"

	MyTrainingsCallback = "myTrainings"
	TrainingCallback    = "training_"
)

type Config struct {
	Token string
}

type Manager struct {
	embedlog.Logger
	bum *workout.BotUserManager
	tm  *workout.TrainingManager
}

func NewManager(dbo db.DB, logger embedlog.Logger) *Manager {
	return &Manager{
		Logger: logger,
		bum:    workout.NewBotUserManager(dbo, logger),
		tm:     workout.NewTrainingManager(dbo, logger),
	}
}

func (bm *Manager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, StartCommand, bot.MatchTypePrefix, bm.startHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, StartCallback, bot.MatchTypePrefix, bm.startHandler)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, MyTrainingsCallback, bot.MatchTypePrefix, bm.sendCalendar)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, TrainingCallback, bot.MatchTypePrefix, bm.trainingList)
}

func (bm *Manager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Нажми %v", StartCommand),
	})
	if err != nil {
		bm.Logger.Errorf("send message failed:%v", err)
		return
	}
}

func (bm *Manager) startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var user *models.User

	if update.Message != nil {
		user = update.Message.From
	} else {
		user = &update.CallbackQuery.From
	}

	hasRegistered, err := bm.bum.RegisterUser(ctx, int(user.ID))
	if err != nil {
		bm.Logger.Errorf("registration failed:%v", err)
		return
	}

	var welcomeMessage string
	if hasRegistered {
		welcomeMessage = fmt.Sprintf("Добро пожаловать, %v!", user.Username)
	} else {
		welcomeMessage = fmt.Sprintf("Привет, %v!", user.Username)
	}

	if update.Message != nil {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      user.ID,
			Text:        welcomeMessage,
			ReplyMarkup: createStartMarkup(),
		})
		if err != nil {
			bm.Logger.Errorf("send message failed:%v", err)
			return
		}
	} else {
		messageID := update.CallbackQuery.Message.Message.ID
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   messageID,
			ChatID:      user.ID,
			Text:        welcomeMessage,
			ReplyMarkup: createStartMarkup(),
		})
		if err != nil {
			bm.Logger.Errorf("edit message failed:%v", err)
			return
		}
	}
}

func (bm *Manager) sendCalendar(ctx context.Context, b *bot.Bot, update *models.Update) {
	var monthStart time.Time

	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		data := update.CallbackQuery.Data
		if strings.HasPrefix(data, "prev:") || strings.HasPrefix(data, "next:") {
			parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(data, "prev:"), "next:"), "-")
			if len(parts) == 2 {
				year, _ := strconv.Atoi(parts[0])
				month, _ := strconv.Atoi(parts[1])
				monthStart = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

				if strings.HasPrefix(data, "prev:") {
					monthStart = monthStart.AddDate(0, -1, 0)
				} else {
					monthStart = monthStart.AddDate(0, 1, 0)
				}
			}
		}
	}

	if monthStart.IsZero() {
		now := time.Now()
		monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	year, month, _ := monthStart.Date()
	firstWeekday := int(monthStart.Weekday())
	if firstWeekday == 0 {
		firstWeekday = 7
	}
	daysInMonth := daysIn(monthStart)

	list, err := bm.tm.TrainingList(ctx, int(update.CallbackQuery.From.ID), nil)
	if err != nil {
		bm.Logger.Errorf("add training failed")
		return
	}

	var rows [][]models.InlineKeyboardButton

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: monthStart.Format("January 2006"), CallbackData: "ignore"},
	})

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "Пн", CallbackData: "ignore"},
		{Text: "Вт", CallbackData: "ignore"},
		{Text: "Ср", CallbackData: "ignore"},
		{Text: "Чт", CallbackData: "ignore"},
		{Text: "Пт", CallbackData: "ignore"},
		{Text: "Сб", CallbackData: "ignore"},
		{Text: "Вс", CallbackData: "ignore"},
	})

	var row []models.InlineKeyboardButton
	for i := 1; i < firstWeekday; i++ {
		row = append(row, models.InlineKeyboardButton{Text: " ", CallbackData: "ignore"})
	}
	for d := 1; d <= daysInMonth; d++ {
		text := fmt.Sprintf("%d", d)
		date := fmt.Sprintf("%d-%02d-%02d", year, month, d)
		if year == time.Now().Year() && month == time.Now().Month() && d == time.Now().Day() {
			text = fmt.Sprintf("[%s]", text)
		} else if _, ok := list.IndexByDate()[date]; ok {
			text = "💪🏻"
		}

		row = append(row, models.InlineKeyboardButton{
			Text:         text,
			CallbackData: TrainingCallback + date,
		})
		if len(row) == 7 {
			rows = append(rows, row)
			row = []models.InlineKeyboardButton{}
		}
	}
	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, models.InlineKeyboardButton{Text: " ", CallbackData: "ignore"})
		}
		rows = append(rows, row)
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "<<", CallbackData: "prev:" + monthStart.Format("2006-01")},
		{Text: ">>", CallbackData: "next:" + monthStart.Format("2006-01")},
	})

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "закрыть", CallbackData: StartCallback},
	})

	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        "Мои тренировки",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	})
	if err != nil {
		fmt.Println("sendCalendar error:", err)
	}
}

func (bm *Manager) trainingList(ctx context.Context, b *bot.Bot, update *models.Update) {
	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 2 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	date, err := time.Parse("2006-01-02", split[1])
	if err != nil {
		bm.Logger.Errorf("parse date failed: %v", err)
		return
	}

	list, err := bm.tm.TrainingList(ctx, int(update.CallbackQuery.From.ID), &date)
	if err != nil {
		return
	}

	markup := createTrainingListMarkup(list)

	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        fmt.Sprintf("Тренировка %v", date.Format("02/01/2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		fmt.Println("send trainings error:", err)
	}
}

func daysIn(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}
