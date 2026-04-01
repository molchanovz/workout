package workout

import (
	"workout/pkg/db"
)

type Approach struct {
	db.Approach
}

func NewApproach(in *db.Approach) *Approach {
	if in == nil {
		return nil
	}

	return &Approach{
		Approach: *in,
	}
}

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

type SiteUser struct {
	db.SiteUser
}

func NewSiteUser(in *db.SiteUser) *SiteUser {
	if in == nil {
		return nil
	}

	return &SiteUser{
		SiteUser: *in,
	}
}

func (u SiteUser) ToDB() db.SiteUser {
	return db.SiteUser{
		ID:             u.ID,
		TgID:           u.TgID,
		CreatedAt:      u.CreatedAt,
		LastActivityAt: u.LastActivityAt,
		StatisticID:    u.StatisticID,
		StatusID:       u.StatusID,
	}
}
