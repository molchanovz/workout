package workout

import (
	"context"
	"github.com/vmkteam/embedlog"
	"time"
	"workoutbot/pkg/db"
)

type BotUserManager struct {
	embedlog.Logger
	dbo      db.DB
	userRepo db.UserRepo
}

func NewBotUserManager(dbo db.DB, log embedlog.Logger) *BotUserManager {
	return &BotUserManager{
		Logger:   log,
		dbo:      dbo,
		userRepo: db.NewUserRepo(dbo.DB),
	}
}

func (m BotUserManager) RegisterUser(ctx context.Context, tgId int) (bool, error) {
	user, err := m.userRepo.OneBotUser(ctx, &db.BotUserSearch{TgID: &tgId})
	if err != nil {
		return false, err
	}

	if user != nil {
		return false, nil
	}

	_, err = m.userRepo.AddBotUser(ctx, &db.BotUser{
		TgID:      tgId,
		CreatedAt: time.Now(),
		StatusID:  db.StatusEnabled,
	})

	return true, err
}
