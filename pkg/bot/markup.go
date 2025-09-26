package bot

import (
	"fmt"
	"github.com/go-telegram/bot/models"
	"workoutbot/pkg/workout"
)

func createStartMarkup() models.InlineKeyboardMarkup {
	var buttonsRow []models.InlineKeyboardButton
	var allButtons [][]models.InlineKeyboardButton

	buttonsRow = append(buttonsRow, models.InlineKeyboardButton{Text: "Мои тренировки", CallbackData: MyTrainingsCallback})
	allButtons = append(allButtons, buttonsRow)

	buttonsRow = []models.InlineKeyboardButton{}
	buttonsRow = append(buttonsRow, models.InlineKeyboardButton{Text: "Статистика", URL: "https://market.yandex.ru/business--metr-v-kube/3697903"})
	allButtons = append(allButtons, buttonsRow)

	buttonsRow = []models.InlineKeyboardButton{}
	buttonsRow = append(buttonsRow, models.InlineKeyboardButton{Text: "Настройки", URL: "https://www.ozon.ru/seller/metr-v-kube-259267"})
	allButtons = append(allButtons, buttonsRow)

	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}

func createTrainingListMarkup(trainings workout.Trainings) models.InlineKeyboardMarkup {
	var buttonsRow []models.InlineKeyboardButton
	var allButtons [][]models.InlineKeyboardButton

	for i := range trainings {
		buttonsRow = []models.InlineKeyboardButton{}
		buttonsRow = append(buttonsRow, models.InlineKeyboardButton{Text: fmt.Sprintf("Тренировка №%d", i), URL: "https://market.yandex.ru/business--metr-v-kube/3697903"})
		allButtons = append(allButtons, buttonsRow)

	}

	buttonsRow = []models.InlineKeyboardButton{}
	buttonsRow = append(buttonsRow, models.InlineKeyboardButton{Text: "Новая тренировка", URL: "https://www.ozon.ru/seller/metr-v-kube-259267"})
	allButtons = append(allButtons, buttonsRow)

	buttonsRow = []models.InlineKeyboardButton{}
	buttonsRow = append(buttonsRow, models.InlineKeyboardButton{Text: "Назад", CallbackData: MyTrainingsCallback})
	allButtons = append(allButtons, buttonsRow)

	return models.InlineKeyboardMarkup{InlineKeyboard: allButtons}
}
