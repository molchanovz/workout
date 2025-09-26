package workout

import (
	"context"
	"github.com/vmkteam/embedlog"
	"time"
	"workoutbot/pkg/db"
)

type TrainingManager struct {
	embedlog.Logger
	dbo db.DB
	tr  db.TrainingRepo
	ur  db.UserRepo
}

func NewTrainingManager(dbo db.DB, log embedlog.Logger) *TrainingManager {
	return &TrainingManager{
		Logger: log,
		dbo:    dbo,
		tr:     db.NewTrainingRepo(dbo.DB),
		ur:     db.NewUserRepo(dbo.DB),
	}
}

// TrainingList get trainings by date
func (tm TrainingManager) TrainingList(ctx context.Context, tgId int, date *time.Time) (Trainings, error) {
	user, err := tm.ur.OneBotUser(ctx, &db.BotUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	trainings, err := tm.tr.TrainingsByFilters(ctx, &db.TrainingSearch{StartedAt: date, BotUserID: &user.ID}, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	return NewTrainings(trainings), err
}
