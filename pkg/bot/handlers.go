package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"workout/pkg/workout"
	"workout/pkg/workout/training"

	"workout/pkg/db"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/vmkteam/embedlog"
)

const (
	StartCommand  = "/start"
	StartCallback = "start"

	MyTrainingsCallback    = "myTrainings"
	TrainingListCallback   = "trainingList"   // _date                    (2 parts)
	NewTrainingCallback    = "newTraining"    // _date                    (2 parts)
	ExerciseListCallback   = "exerciseList"   // _date_TID                (3 parts)
	NewExerciseCallback    = "newExercise"    // _date_TID                (3 parts)
	ApproachListCallback   = "approachList"   // _EID_date_TID            (4 parts)
	NewApproachCallback    = "newApproach"    // _EID_date_TID            (4 parts)
	EditApproachCallback   = "editApproach"   // _AID_EID_date_TID     (5 parts)
	DeleteApproachCallback = "deleteApproach" // _AID_EID_date_TID     (5 parts)
	DeleteTrainingCallback = "deleteTraining" // _date_TID             (3 parts)

	SelectCategoryCallback = "selectCategory" // _CID_date_TID         (4 parts)
	SelectExerciseCallback = "selectExercise" // _EID_date_TID         (4 parts)

	StatisticsCallback  = "statistics"
	SettingsCallback    = "settings"
	AddCategoryCallback = "addCategory"
	AddExerciseCallback = "addExercise"

	SelectParentCategoryCallback   = "selectParentCategory"   // _CID (2 parts)
	SelectExerciseCategoryCallback = "selectExerciseCategory" // _CID (2 parts)
	SelectExerciseTypeCallback     = "selectExerciseType"     // _TYPE (2 parts)
)

type Config struct {
	Token    string
	ProxyURL string
}

type Manager struct {
	embedlog.Logger
	bum    *workout.SiteUserManager
	tm     *training.Manager
	cm     *workout.CategoryManager
	sm     *workout.StatsManager
	states *StateStore
}

func NewManager(dbo db.DB, logger embedlog.Logger) *Manager {
	return &Manager{
		Logger: logger,
		bum:    workout.NewSiteUserManager(dbo, logger),
		tm:     training.NewTrainingManager(dbo, logger),
		cm:     workout.NewCategoryManager(dbo, logger),
		sm:     workout.NewStatsManager(dbo, logger),
		states: NewStateStore(),
	}
}

func (bm *Manager) RegisterBotHandlers(b *bot.Bot) {
	b.RegisterHandler(bot.HandlerTypeMessageText, StartCommand, bot.MatchTypePrefix, bm.startHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, StartCallback, bot.MatchTypePrefix, bm.startHandler)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, MyTrainingsCallback, bot.MatchTypePrefix, bm.sendCalendar)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "prev:", bot.MatchTypePrefix, bm.sendCalendar)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "next:", bot.MatchTypePrefix, bm.sendCalendar)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, TrainingListCallback, bot.MatchTypePrefix, bm.trainingList)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, NewTrainingCallback, bot.MatchTypePrefix, bm.newTraining)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, ExerciseListCallback, bot.MatchTypePrefix, bm.exerciseList)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, NewExerciseCallback, bot.MatchTypePrefix, bm.newExercise)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, ApproachListCallback, bot.MatchTypePrefix, bm.approachList)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, NewApproachCallback, bot.MatchTypePrefix, bm.newApproach)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, SelectCategoryCallback, bot.MatchTypePrefix, bm.selectCategory)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, SelectExerciseCategoryCallback, bot.MatchTypePrefix, bm.selectExerciseCategory)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, SelectExerciseTypeCallback, bot.MatchTypePrefix, bm.selectExerciseTypeHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, SelectExerciseCallback, bot.MatchTypePrefix, bm.selectExercise)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, EditApproachCallback, bot.MatchTypePrefix, bm.editApproach)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, DeleteApproachCallback, bot.MatchTypePrefix, bm.deleteApproach)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, DeleteTrainingCallback, bot.MatchTypePrefix, bm.deleteTraining)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, StatisticsCallback, bot.MatchTypePrefix, bm.statisticsHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, SettingsCallback, bot.MatchTypePrefix, bm.settingsHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, AddCategoryCallback, bot.MatchTypePrefix, bm.addCategoryHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, AddExerciseCallback, bot.MatchTypePrefix, bm.addExerciseHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, SelectParentCategoryCallback, bot.MatchTypePrefix, bm.selectParentCategory)
}

func (bm *Manager) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	tgId := update.Message.From.ID
	if state := bm.states.Get(tgId); state != nil {
		bm.textInputHandler(ctx, b, update, state)
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Нажми %v", StartCommand),
	})
	if err != nil {
		bm.Logger.Errorf("send message failed:%v", err)
	}
}

func (bm *Manager) textInputHandler(ctx context.Context, b *bot.Bot, update *models.Update, state *UserState) {
	tgId := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	bm.deleteMessages(ctx, b, chatID, update.Message.ID, state.PromptMessageID)

	switch state.Step {
	case StepEnterReps:
		reps, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || reps <= 0 {
			bm.sendText(ctx, b, chatID, "Введите корректное число повторений:")
			return
		}
		state.Reps = reps
		state.Step = StepEnterWeight
		bm.states.Set(tgId, state)
		state.PromptMessageID = bm.sendForceReply(ctx, b, chatID, "Введите вес (кг):")
		bm.states.Set(tgId, state)

	case StepEnterWeight:
		weight, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || weight < 0 {
			bm.sendText(ctx, b, chatID, "Введите корректный вес:")
			return
		}

		var execErr error
		if state.ApproachID != 0 {
			execErr = bm.tm.UpdateApproach(ctx, int(tgId), state.ApproachID, state.Reps, weight)
		} else {
			_, execErr = bm.tm.AddApproach(ctx, int(tgId), state.TrainingID, state.ExerciseID, state.Reps, weight)
		}
		if execErr != nil {
			bm.Logger.Errorf("save approach failed: %v", execErr)
			bm.sendText(ctx, b, chatID, "Ошибка при сохранении подхода.")
			bm.states.Delete(tgId)
			return
		}

		botMsgID := state.BotMessageID
		bm.states.Delete(tgId)
		bm.sendApproachListMessage(ctx, b, chatID, int(tgId), state.ExerciseID, state.TrainingID, state.Date, botMsgID)

	case StepEnterCategoryName:
		var parentID *int
		if state.CategoryID != 0 {
			parentID = &state.CategoryID
		}
		_, err := bm.cm.AddCategory(ctx, int(tgId), strings.TrimSpace(text), parentID)
		if err != nil {
			bm.Logger.Errorf("add category failed: %v", err)
			bm.sendText(ctx, b, chatID, "Ошибка при создании категории.")
			bm.states.Delete(tgId)
			return
		}
		botMsgID := state.BotMessageID
		bm.states.Delete(tgId)
		bm.sendSettingsMessage(ctx, b, chatID, "Категория создана!", botMsgID)

	case StepEnterExerciseName:
		botMsgID := state.BotMessageID
		bm.states.Set(tgId, &UserState{
			Step:         StepSelectExerciseType,
			CategoryID:   state.CategoryID,
			Date:         strings.TrimSpace(text),
			BotMessageID: botMsgID,
		})
		markup := exerciseTypeMarkup()
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   botMsgID,
			ChatID:      chatID,
			Text:        "Выберите тип упражнения:",
			ReplyMarkup: &markup,
		})
		if err != nil {
			bm.Logger.Errorf("send exercise type markup failed: %v", err)
		}

	case StepEnterDuration:
		duration, err := parseDuration(strings.TrimSpace(text))
		if err != nil || duration <= 0 {
			bm.sendText(ctx, b, chatID, "Введите корректное время (например 90 или 1:30):")
			return
		}

		var execErr error
		if state.ApproachID != 0 {
			execErr = bm.tm.UpdateTimedApproach(ctx, int(tgId), state.ApproachID, duration)
		} else {
			_, execErr = bm.tm.AddTimedApproach(ctx, int(tgId), state.TrainingID, state.ExerciseID, duration)
		}
		if execErr != nil {
			bm.Logger.Errorf("save timed approach failed: %v", execErr)
			bm.sendText(ctx, b, chatID, "Ошибка при сохранении подхода.")
			bm.states.Delete(tgId)
			return
		}
		botMsgID := state.BotMessageID
		bm.states.Delete(tgId)
		bm.sendApproachListMessage(ctx, b, chatID, int(tgId), state.ExerciseID, state.TrainingID, state.Date, botMsgID)
	}
}

func (bm *Manager) startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var user *models.User

	if update.Message != nil {
		user = update.Message.From
	} else {
		user = &update.CallbackQuery.From
	}

	tgId := user.ID

	hasRegistered, err := bm.bum.RegisterUser(ctx, int(tgId))
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
			ChatID:      tgId,
			Text:        welcomeMessage,
			ReplyMarkup: startMarkup(),
		})
	} else {
		messageID := update.CallbackQuery.Message.Message.ID
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   messageID,
			ChatID:      tgId,
			Text:        welcomeMessage,
			ReplyMarkup: startMarkup(),
		})
	}
	if err != nil {
		bm.Logger.Errorf("start handler send failed:%v", err)
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
		bm.Logger.Errorf("training list failed: %v", err)
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
			CallbackData: fmt.Sprintf("%v_%v", TrainingListCallback, date),
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
	rows = append(rows,
		[]models.InlineKeyboardButton{
			{Text: "<<", CallbackData: "prev:" + monthStart.Format("2006-01")},
			{Text: ">>", CallbackData: "next:" + monthStart.Format("2006-01")},
		},
		[]models.InlineKeyboardButton{
			{Text: "закрыть", CallbackData: StartCallback},
		},
	)

	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        "Мои тренировки",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	})
	if err != nil {
		bm.Logger.Errorf("sendCalendar error: %v", err)
	}
}

func (bm *Manager) trainingList(ctx context.Context, b *bot.Bot, update *models.Update) {
	// trainingList_date (2 parts)
	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 2 {
		bm.Logger.Errorf("bad callbackQuery data: %v", update.CallbackQuery.Data)
		return
	}

	dateStr := split[1]
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		bm.Logger.Errorf("parse date failed: %v", err)
		return
	}

	list, err := bm.tm.TrainingList(ctx, int(update.CallbackQuery.From.ID), &date)
	if err != nil {
		return
	}

	markup := trainingListMarkup(list, dateStr)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        fmt.Sprintf("Тренировки %v", date.Format("02/01/2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send training list error: %v", err)
	}
}

func (bm *Manager) newTraining(ctx context.Context, b *bot.Bot, update *models.Update) {
	// newTraining_date (2 parts)
	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 2 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	dateStr := split[1]
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		bm.Logger.Errorf("parse date failed: %v", err)
		return
	}

	training, err := bm.tm.NewTraining(ctx, int(update.CallbackQuery.From.ID), date)
	if err != nil {
		bm.Logger.Errorf("create training failed: %v", err)
		return
	}

	markup := exerciseListMarkup(nil, dateStr, training.ID)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        fmt.Sprintf("Упражнения %v", date.Format("02/01/2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("show exercise list failed: %v", err)
	}
}

func (bm *Manager) exerciseList(ctx context.Context, b *bot.Bot, update *models.Update) {
	// exerciseList_date_TID (3 parts)
	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 3 {
		bm.Logger.Errorf("bad callbackQuery data: %v", update.CallbackQuery.Data)
		return
	}

	dateStr := split[1]
	trainingId, err := strconv.Atoi(split[2])
	if err != nil {
		return
	}

	date, _ := time.Parse("2006-01-02", dateStr)
	exercises, err := bm.tm.ExerciseList(ctx, int(update.CallbackQuery.From.ID), trainingId)
	if err != nil {
		bm.Logger.Errorf("get exercise list failed: %v", err)
		return
	}

	markup := exerciseListMarkup(exercises, dateStr, trainingId)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        fmt.Sprintf("Упражнения %v", date.Format("02/01/2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send exercise list failed: %v", err)
	}
}

func (bm *Manager) newExercise(ctx context.Context, b *bot.Bot, update *models.Update) {
	// newExercise_date_TID (3 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 3 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	dateStr := split[1]
	trainingId, err := strconv.Atoi(split[2])
	if err != nil {
		return
	}

	categories, err := bm.cm.RootCategories(ctx, int(tgId))
	if err != nil {
		bm.Logger.Errorf("get categories failed: %v", err)
		return
	}

	bm.states.Set(tgId, &UserState{
		Step:       StepSelectCategory,
		TrainingID: trainingId,
		Date:       dateStr,
	})

	backCallback := fmt.Sprintf("%v_%v_%v", ExerciseListCallback, dateStr, trainingId)
	markup := categoryMarkup(categories, dateStr, trainingId, backCallback)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        "Выберите категорию:",
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send categories failed: %v", err)
	}
}

func (bm *Manager) approachList(ctx context.Context, b *bot.Bot, update *models.Update) {
	// approachList_EID_date_TID (4 parts)
	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 4 {
		bm.Logger.Errorf("bad callbackQuery data: %v", update.CallbackQuery.Data)
		return
	}

	exerciseId, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}
	dateStr := split[2]
	trainingId, err := strconv.Atoi(split[3])
	if err != nil {
		return
	}

	d, _ := time.Parse("2006-01-02", dateStr)
	tgId := int(update.CallbackQuery.From.ID)

	list, err := bm.tm.ApproachList(ctx, tgId, trainingId, exerciseId)
	if err != nil {
		bm.Logger.Errorf("get approach list failed: %v", err)
		return
	}

	// Get exercise name for the header
	exerciseName := ""
	if len(list) > 0 && list[0].Exercise != nil {
		exerciseName = list[0].Exercise.Title
	}

	markup := approachListMarkup(list, exerciseId, trainingId, dateStr)
	messageID := update.CallbackQuery.Message.Message.ID
	text := fmt.Sprintf("Подходы: %v (%v)", exerciseName, d.Format("02/01/2006"))
	if exerciseName == "" {
		text = fmt.Sprintf("Подходы (%v)", d.Format("02/01/2006"))
	}
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send approach list failed: %v", err)
	}
}

func (bm *Manager) newApproach(ctx context.Context, b *bot.Bot, update *models.Update) {
	// newApproach_EID_date_TID (4 parts) — exercise is already known
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 4 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	exerciseId, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}
	dateStr := split[2]
	trainingId, err := strconv.Atoi(split[3])
	if err != nil {
		return
	}

	exType, _ := bm.tm.ExerciseType(ctx, int(tgId), exerciseId)
	botMsgID := int(update.CallbackQuery.Message.Message.ID)
	state := &UserState{
		TrainingID:   trainingId,
		Date:         dateStr,
		ExerciseID:   exerciseId,
		ExerciseType: exType,
		BotMessageID: botMsgID,
	}
	if exType == training.ExerciseTypeTimed {
		state.Step = StepEnterDuration
		bm.states.Set(tgId, state)
		state.PromptMessageID = bm.sendForceReply(ctx, b, tgId, "Введите время (секунды или MM:SS):")
		bm.states.Set(tgId, state)
	} else {
		state.Step = StepEnterReps
		bm.states.Set(tgId, state)
		state.PromptMessageID = bm.sendForceReply(ctx, b, tgId, "Введите количество повторений:")
		bm.states.Set(tgId, state)
	}
}

func (bm *Manager) selectCategory(ctx context.Context, b *bot.Bot, update *models.Update) {
	// selectCategory_CID_date_TID (4 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 4 {
		bm.Logger.Errorf("bad callbackQuery data: %v", update.CallbackQuery.Data)
		return
	}

	categoryID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}
	dateStr := split[2]
	trainingId, err := strconv.Atoi(split[3])
	if err != nil {
		return
	}

	state := bm.states.Get(tgId)

	// Check for subcategories
	subs, err := bm.cm.SubCategories(ctx, int(tgId), categoryID)
	if err != nil {
		bm.Logger.Errorf("get subcategories failed: %v", err)
		return
	}

	messageID := update.CallbackQuery.Message.Message.ID

	if len(subs) > 0 {
		var backCallback string
		if state != nil {
			backCallback = fmt.Sprintf("%v_%v_%v", ExerciseListCallback, dateStr, trainingId)
		} else {
			backCallback = fmt.Sprintf("%v_%v_%v_%v", ApproachListCallback, 0, dateStr, trainingId)
		}
		markup := categoryMarkup(subs, dateStr, trainingId, backCallback)
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   messageID,
			ChatID:      tgId,
			Text:        "Выберите подкатегорию:",
			ReplyMarkup: &markup,
		})
		if err != nil {
			bm.Logger.Errorf("send subcategories failed: %v", err)
		}
		return
	}

	// Show exercises
	exercises, err := bm.cm.ExercisesByCategory(ctx, int(tgId), categoryID)
	if err != nil {
		bm.Logger.Errorf("get exercises failed: %v", err)
		return
	}

	var backCallback string
	if state != nil {
		backCallback = fmt.Sprintf("%v_%v_%v", ExerciseListCallback, dateStr, trainingId)
	} else {
		backCallback = fmt.Sprintf("%v_%v_%v_%v", ApproachListCallback, 0, dateStr, trainingId)
	}

	markup := exerciseMarkup(exercises, dateStr, trainingId, backCallback)
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        "Выберите упражнение:",
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send exercises failed: %v", err)
	}
}

func (bm *Manager) selectExercise(ctx context.Context, b *bot.Bot, update *models.Update) {
	// selectExercise_EID_date_TID (4 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 4 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	exerciseID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}
	dateStr := split[2]
	trainingId, err := strconv.Atoi(split[3])
	if err != nil {
		return
	}

	bm.states.Delete(tgId)
	exercises, err := bm.tm.ApproachList(ctx, int(tgId), trainingId, exerciseID)
	if err != nil {
		bm.Logger.Errorf("get approach list failed: %v", err)
		return
	}
	markup := approachListMarkup(exercises, exerciseID, trainingId, dateStr)
	d, _ := time.Parse("2006-01-02", dateStr)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        fmt.Sprintf("Подходы (%v)", d.Format("02/01/2006")),
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send approach list failed: %v", err)
	}
}

func (bm *Manager) editApproach(ctx context.Context, b *bot.Bot, update *models.Update) {
	// editApproach_AID_EID_date_TID (5 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 5 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	approachID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}
	exerciseID, err := strconv.Atoi(split[2])
	if err != nil {
		return
	}
	dateStr := split[3]
	trainingId, err := strconv.Atoi(split[4])
	if err != nil {
		return
	}

	exType, _ := bm.tm.ExerciseType(ctx, int(tgId), exerciseID)
	botMsgID := int(update.CallbackQuery.Message.Message.ID)
	state := &UserState{
		TrainingID:   trainingId,
		Date:         dateStr,
		ApproachID:   approachID,
		ExerciseID:   exerciseID,
		ExerciseType: exType,
		BotMessageID: botMsgID,
	}
	if exType == training.ExerciseTypeTimed {
		state.Step = StepEnterDuration
		bm.states.Set(tgId, state)
		state.PromptMessageID = bm.sendForceReply(ctx, b, tgId, "Введите новое время (секунды или MM:SS):")
		bm.states.Set(tgId, state)
	} else {
		state.Step = StepEnterReps
		bm.states.Set(tgId, state)
		state.PromptMessageID = bm.sendForceReply(ctx, b, tgId, "Введите новое количество повторений:")
		bm.states.Set(tgId, state)
	}
}

func (bm *Manager) deleteApproach(ctx context.Context, b *bot.Bot, update *models.Update) {
	// deleteApproach_AID_EID_date_TID (5 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 5 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	approachID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}
	exerciseID, err := strconv.Atoi(split[2])
	if err != nil {
		return
	}
	dateStr := split[3]
	trainingId, err := strconv.Atoi(split[4])
	if err != nil {
		return
	}

	err = bm.tm.DeleteApproach(ctx, int(tgId), trainingId, approachID)
	if err != nil {
		bm.Logger.Errorf("delete approach failed: %v", err)
		return
	}

	messageID := int(update.CallbackQuery.Message.Message.ID)
	bm.sendApproachListMessage(ctx, b, tgId, int(tgId), exerciseID, trainingId, dateStr, messageID)
}

func (bm *Manager) deleteTraining(ctx context.Context, b *bot.Bot, update *models.Update) {
	// deleteTraining_date_TID (3 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 3 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	dateStr := split[1]
	trainingId, err := strconv.Atoi(split[2])
	if err != nil {
		return
	}

	err = bm.tm.DeleteTraining(ctx, int(tgId), trainingId)
	if err != nil {
		bm.Logger.Errorf("delete training failed: %v", err)
		return
	}

	date, _ := time.Parse("2006-01-02", dateStr)
	list, err := bm.tm.TrainingList(ctx, int(tgId), &date)
	if err != nil {
		bm.Logger.Errorf("training list failed: %v", err)
		return
	}

	markup := trainingListMarkup(list, dateStr)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        fmt.Sprintf("Тренировки %v", date.Format("02/01/2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("delete training send failed: %v", err)
	}
}

func (bm *Manager) statisticsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgId := update.CallbackQuery.From.ID

	stats, err := bm.sm.UserStats(ctx, int(tgId))
	if err != nil {
		bm.Logger.Errorf("get stats failed: %v", err)
		return
	}

	leaderboard, err := bm.sm.Leaderboard(ctx)
	if err != nil {
		bm.Logger.Errorf("get leaderboard failed: %v", err)
		return
	}

	text := fmt.Sprintf(
		"📊 <b>Статистика</b>\n\nТренировок в этом месяце: %d\nТренировок в этом году: %d\nОбщий тоннаж: %.0f кг\n\n<b>Лидерборд</b>\n",
		stats.MonthCount, stats.YearCount, stats.TotalTonnage,
	)
	for i, us := range leaderboard {
		if i >= 10 {
			break
		}
		text += fmt.Sprintf("%d. TgID %d — %.0f кг\n", i+1, us.SiteUser.TgID, us.Stats.TotalTonnage)
	}

	markup := models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "Назад", CallbackData: StartCallback}},
	}}
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send statistics failed: %v", err)
	}
}

func (bm *Manager) settingsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	markup := settingsMarkup()
	messageID := update.CallbackQuery.Message.Message.ID
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      update.CallbackQuery.From.ID,
		Text:        "⚙️ Настройки",
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send settings failed: %v", err)
	}
}

func (bm *Manager) addCategoryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgId := update.CallbackQuery.From.ID

	categories, err := bm.cm.RootCategories(ctx, int(tgId))
	if err != nil {
		bm.Logger.Errorf("get categories failed: %v", err)
		return
	}

	markup := parentCategoryMarkup(categories)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        "Выберите родительскую категорию:",
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send parent category list failed: %v", err)
	}
}

func (bm *Manager) selectParentCategory(ctx context.Context, b *bot.Bot, update *models.Update) {
	// selectParentCategory_CID (2 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 2 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	parentID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}

	botMsgID := int(update.CallbackQuery.Message.Message.ID)
	state := &UserState{
		Step:         StepEnterCategoryName,
		CategoryID:   parentID,
		BotMessageID: botMsgID,
	}
	bm.states.Set(tgId, state)
	state.PromptMessageID = bm.sendForceReply(ctx, b, tgId, "Введите название категории:")
	bm.states.Set(tgId, state)
}

func (bm *Manager) addExerciseHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	tgId := update.CallbackQuery.From.ID

	categories, err := bm.cm.RootCategories(ctx, int(tgId))
	if err != nil {
		bm.Logger.Errorf("get categories failed: %v", err)
		return
	}

	markup := exerciseCategoryMarkup(categories)
	messageID := update.CallbackQuery.Message.Message.ID
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		MessageID:   messageID,
		ChatID:      tgId,
		Text:        "Выберите категорию для упражнения:",
		ReplyMarkup: &markup,
	})
	if err != nil {
		bm.Logger.Errorf("send exercise category list failed: %v", err)
	}
}

func (bm *Manager) selectExerciseCategory(ctx context.Context, b *bot.Bot, update *models.Update) {
	// selectExerciseCategory_CID (2 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 2 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	categoryID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}

	botMsgID := int(update.CallbackQuery.Message.Message.ID)
	state := &UserState{
		Step:         StepEnterExerciseName,
		CategoryID:   categoryID,
		BotMessageID: botMsgID,
	}
	bm.states.Set(tgId, state)
	state.PromptMessageID = bm.sendForceReply(ctx, b, tgId, "Введите название упражнения:")
	bm.states.Set(tgId, state)
}

func (bm *Manager) selectExerciseTypeHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// selectExerciseType_TYPE (2 parts)
	tgId := update.CallbackQuery.From.ID

	split := strings.Split(update.CallbackQuery.Data, "_")
	if len(split) != 2 {
		bm.Logger.Errorf("bad callbackQuery data")
		return
	}

	typeID, err := strconv.Atoi(split[1])
	if err != nil {
		return
	}

	state := bm.states.Get(tgId)
	if state == nil || state.Step != StepSelectExerciseType {
		return
	}

	name := state.Date // exercise name was stored in Date field
	_, err = bm.cm.AddExercise(ctx, int(tgId), name, state.CategoryID, typeID)
	if err != nil {
		bm.Logger.Errorf("add exercise failed: %v", err)
		bm.sendText(ctx, b, tgId, "Ошибка при создании упражнения.")
		bm.states.Delete(tgId)
		return
	}
	bm.states.Delete(tgId)
	messageID := int(update.CallbackQuery.Message.Message.ID)
	bm.sendSettingsMessage(ctx, b, tgId, "Упражнение создано!", messageID)
}

// --- helpers ---

func (bm *Manager) sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text})
	if err != nil {
		bm.Logger.Errorf("send message failed: %v", err)
	}
}

func (bm *Manager) deleteMessages(ctx context.Context, b *bot.Bot, chatID int64, messageIDs ...int) {
	for _, id := range messageIDs {
		if id == 0 {
			continue
		}
		_, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: chatID, MessageID: id})
		if err != nil {
			bm.Logger.Errorf("delete message %d failed: %v", id, err)
		}
	}
}

func (bm *Manager) sendForceReply(ctx context.Context, b *bot.Bot, chatID int64, text string) int {
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
		ReplyMarkup: &models.ForceReply{
			ForceReply: true,
			Selective:  true,
		},
	})
	if err != nil {
		bm.Logger.Errorf("send force reply failed: %v", err)
		return 0
	}
	return msg.ID
}

func (bm *Manager) sendApproachListMessage(ctx context.Context, b *bot.Bot, chatID int64, tgId, exerciseId, trainingId int, dateStr string, editMessageID int) {
	list, err := bm.tm.ApproachList(ctx, tgId, trainingId, exerciseId)
	if err != nil {
		bm.Logger.Errorf("get approach list failed: %v", err)
		return
	}

	exerciseName := ""
	if len(list) > 0 && list[0].Exercise != nil {
		exerciseName = list[0].Exercise.Title
	}

	d, _ := time.Parse("2006-01-02", dateStr)
	markup := approachListMarkup(list, exerciseId, trainingId, dateStr)
	text := fmt.Sprintf("Подходы: %v (%v)", exerciseName, d.Format("02/01/2006"))
	if exerciseName == "" {
		text = fmt.Sprintf("Подходы (%v)", d.Format("02/01/2006"))
	}
	if editMessageID != 0 {
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   editMessageID,
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: &markup,
		})
	} else {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: &markup,
		})
	}
	if err != nil {
		bm.Logger.Errorf("send approach list failed: %v", err)
	}
}

func (bm *Manager) sendSettingsMessage(ctx context.Context, b *bot.Bot, chatID int64, prefix string, editMessageID int) {
	markup := settingsMarkup()
	text := prefix + "\n\n⚙️ Настройки"
	if prefix == "" {
		text = "⚙️ Настройки"
	}
	var err error
	if editMessageID != 0 {
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID:   editMessageID,
			ChatID:      chatID,
			Text:        text,
			ReplyMarkup: &markup,
		})
	} else {
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ReplyMarkup: &markup,
		})
	}
	if err != nil {
		bm.Logger.Errorf("send settings failed: %v", err)
	}
}

func parseDuration(s string) (int, error) {
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		secs, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		return minutes*60 + secs, nil
	}
	return strconv.Atoi(s)
}

func daysIn(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}
