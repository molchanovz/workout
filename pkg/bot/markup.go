package bot

import (
	"fmt"
	"github.com/go-telegram/bot/models"
	"strconv"
	"workout/pkg/db"
	"workout/pkg/workout"
)

func startMarkup() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "Мои тренировки", CallbackData: MyTrainingsCallback}},
		{{Text: "Статистика", CallbackData: StatisticsCallback}},
		{{Text: "Настройки", CallbackData: SettingsCallback}},
	}}
}

func trainingListMarkup(trainings workout.Trainings, date string) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton

	for i := range trainings {
		trainingId := trainings[i].ID
		allButtons = append(allButtons, []models.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("Тренировка №%d", i+1),
				CallbackData: fmt.Sprintf("%v_%v_%v", ExerciseListCallback, date, trainingId),
			},
			{
				Text:         "🗑",
				CallbackData: fmt.Sprintf("%v_%v_%v", DeleteTrainingCallback, date, trainingId),
			},
		})
	}

	allButtons = append(allButtons,
		[]models.InlineKeyboardButton{
			{Text: "Новая тренировка", CallbackData: fmt.Sprintf("%v_%v", NewTrainingCallback, date)},
		},
		[]models.InlineKeyboardButton{
			{Text: "Назад", CallbackData: MyTrainingsCallback},
		},
	)

	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

// exerciseListMarkup shows unique exercises in a training.
// format: exerciseList_date_TID
func exerciseListMarkup(exercises []db.Exercise, date string, trainingId int) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton

	for _, e := range exercises {
		allButtons = append(allButtons, []models.InlineKeyboardButton{
			{
				Text:         e.Title,
				CallbackData: fmt.Sprintf("%v_%v_%v_%v", ApproachListCallback, e.ID, date, trainingId),
			},
		})
	}

	allButtons = append(allButtons,
		[]models.InlineKeyboardButton{
			{Text: "+ Добавить упражнение", CallbackData: fmt.Sprintf("%v_%v_%v", NewExerciseCallback, date, trainingId)},
		},
		[]models.InlineKeyboardButton{
			{Text: "Назад", CallbackData: fmt.Sprintf("%v_%v", TrainingListCallback, date)},
		},
	)

	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

// approachListMarkup shows approaches for a specific exercise.
// format: approachList_EID_date_TID
func approachListMarkup(approaches workout.Approaches, exerciseId, trainingId int, date string) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton

	for i := range approaches {
		a := approaches[i]

		allButtons = append(allButtons,
			[]models.InlineKeyboardButton{
				{Text: formatApproach(i, a.Approach), CallbackData: "ignore"},
			},

			[]models.InlineKeyboardButton{
				{
					Text:         "✏️ Изменить",
					CallbackData: fmt.Sprintf("%v_%v_%v_%v_%v", EditApproachCallback, a.ID, exerciseId, date, trainingId),
				},
				{
					Text:         "🗑 Удалить",
					CallbackData: fmt.Sprintf("%v_%v_%v_%v_%v", DeleteApproachCallback, a.ID, exerciseId, date, trainingId),
				},
			},
		)
	}

	allButtons = append(allButtons,
		[]models.InlineKeyboardButton{
			{
				Text:         "+ Новый подход",
				CallbackData: fmt.Sprintf("%v_%v_%v_%v", NewApproachCallback, exerciseId, date, trainingId),
			},
		},
		[]models.InlineKeyboardButton{
			{Text: "Назад", CallbackData: fmt.Sprintf("%v_%v_%v", ExerciseListCallback, date, trainingId)},
		},
	)

	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

// categoryMarkup shows a list of categories. backCallback is the "Назад" button target.
func categoryMarkup(categories []db.Category, date string, trainingId int, backCallback string) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton
	for _, c := range categories {
		allButtons = append(allButtons, []models.InlineKeyboardButton{
			{
				Text:         c.Title,
				CallbackData: fmt.Sprintf("%v_%v_%v_%v", SelectCategoryCallback, c.ID, date, trainingId),
			},
		})
	}
	allButtons = append(allButtons, []models.InlineKeyboardButton{
		{Text: "Назад", CallbackData: backCallback},
	})
	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

// exerciseMarkup shows a list of exercises. backCallback is the "Назад" button target.
func exerciseMarkup(exercises []db.Exercise, date string, trainingId int, backCallback string) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton
	for _, e := range exercises {
		allButtons = append(allButtons, []models.InlineKeyboardButton{
			{
				Text:         e.Title,
				CallbackData: fmt.Sprintf("%v_%v_%v_%v", SelectExerciseCallback, e.ID, date, trainingId),
			},
		})
	}
	allButtons = append(allButtons, []models.InlineKeyboardButton{
		{Text: "Назад", CallbackData: backCallback},
	})
	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

func settingsMarkup() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "Добавить категорию", CallbackData: AddCategoryCallback}},
		{{Text: "Добавить упражнение", CallbackData: AddExerciseCallback}},
		{{Text: "Назад", CallbackData: StartCallback}},
	}}
}

func parentCategoryMarkup(categories []db.Category) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton
	allButtons = append(allButtons, []models.InlineKeyboardButton{
		{Text: "Без родителя", CallbackData: fmt.Sprintf("%v_0", SelectParentCategoryCallback)},
	})
	for _, c := range categories {
		allButtons = append(allButtons, []models.InlineKeyboardButton{
			{Text: c.Title, CallbackData: fmt.Sprintf("%v_%v", SelectParentCategoryCallback, c.ID)},
		})
	}
	allButtons = append(allButtons, []models.InlineKeyboardButton{
		{Text: "Назад", CallbackData: SettingsCallback},
	})
	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

func formatApproach(i int, a db.Approach) string {
	if a.Duration != nil {
		return fmt.Sprintf("%d. %s", i+1, formatDuration(*a.Duration))
	}
	return fmt.Sprintf("%d. %s пов × %s кг", i+1,
		strconv.Itoa(workout.Deref(a.Reps, 0)),
		strconv.Itoa(workout.Deref(a.Weight, 0)))
}

func formatDuration(secs int) string {
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func exerciseTypeMarkup() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{
			{Text: "Силовое", CallbackData: fmt.Sprintf("%v_1", SelectExerciseTypeCallback)},
			{Text: "Временное", CallbackData: fmt.Sprintf("%v_2", SelectExerciseTypeCallback)},
		},
	}}
}

func exerciseCategoryMarkup(categories []db.Category) models.InlineKeyboardMarkup {
	var allButtons [][]models.InlineKeyboardButton
	for _, c := range categories {
		allButtons = append(allButtons, []models.InlineKeyboardButton{
			{Text: c.Title, CallbackData: fmt.Sprintf("%v_%v", SelectExerciseCategoryCallback, c.ID)},
		})
	}
	allButtons = append(allButtons, []models.InlineKeyboardButton{
		{Text: "Назад", CallbackData: SettingsCallback},
	})
	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}
