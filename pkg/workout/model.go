package workout

import (
	"workoutbot/pkg/db"
)

type Training struct {
	db.Training
	Date string
}

func NewTraining(in *db.Training) *Training {
	if in == nil {
		return nil
	}

	return &Training{
		Training: *in,
		Date:     in.StartedAt.Format("2006-01-02"),
	}
}

type BotUser struct {
	db.BotUser
}

func NewBotUser(in *db.BotUser) *BotUser {
	if in == nil {
		return nil
	}

	return &BotUser{
		BotUser: *in,
	}
}

func (u BotUser) ToDB() db.BotUser {
	return db.BotUser{
		ID:             u.ID,
		TgID:           u.TgID,
		CreatedAt:      u.CreatedAt,
		LastActivityAt: u.LastActivityAt,
		StatisticID:    u.StatisticID,
		StatusID:       u.StatusID,
	}
}
