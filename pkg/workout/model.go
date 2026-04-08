package workout

import (
	"workout/pkg/db"
)

//go:generate colgen

type Category struct {
	db.Category
}

func NewCategory(in *db.Category) *Category {
	if in == nil {
		return nil
	}

	return &Category{
		Category: *in,
	}
}

type Exercise struct {
	db.Exercise
}

func NewExercise(in *db.Exercise) *Exercise {
	if in == nil {
		return nil
	}

	return &Exercise{
		Exercise: *in,
	}
}

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
	Date          string  `json:"date"`
	ExerciseCount int     `json:"exerciseCount"`
	CategoryTitle string  `json:"-"`
	Name          *string `json:"-"`
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
