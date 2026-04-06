package rpc

import (
	"time"
	"workout/pkg/workout"
)

//go:generate colgen

type Category struct {
	ID               int    `json:"id,omitempty"`
	ParentCategoryID *int   `json:"parentCategoryId,omitempty"`
	Title            string `json:"title,omitempty"`
	SiteUserID       *int   `json:"siteUserId,omitempty"`
	StatusID         int    `json:"statusId,omitempty"`
}

func NewCategory(in *workout.Category) *Category {
	if in == nil {
		return nil
	}

	return &Category{
		ID:               in.ID,
		ParentCategoryID: in.ParentCategoryID,
		Title:            in.Title,
		SiteUserID:       in.SiteUserID,
		StatusID:         in.StatusID,
	}
}

type Approach struct {
	ID         int        `json:"id,omitempty"`
	ExerciseID *int       `json:"exerciseId,omitempty"`
	Reps       *int       `json:"reps,omitempty"`
	Weight     *int       `json:"weight,omitempty"`
	Duration   *int       `json:"duration,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	StatusID   int        `json:"statusId,omitempty"`
}

func NewApproach(in *workout.Approach) *Approach {
	if in == nil {
		return nil
	}

	return &Approach{
		ID:         in.ID,
		ExerciseID: in.ExerciseID,
		Reps:       in.Reps,
		Weight:     in.Weight,
		Duration:   in.Duration,
		CreatedAt:  in.CreatedAt,
		StatusID:   in.StatusID,
	}
}

type Training struct {
	ID            int        `json:"id,omitempty"`
	SiteUserID    int        `json:"siteUserId,omitempty"`
	ApproachIDs   []int      `json:"approachIds,omitempty"`
	StartedAt     time.Time  `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt,omitempty"`
	StatusID      int        `json:"statusId,omitempty"`
	Date          string     `json:"date"`
	ExerciseCount int        `json:"exerciseCount"`
}

func NewTraining(in *workout.Training) *Training {
	if in == nil {
		return nil
	}

	return &Training{
		ID:            in.ID,
		SiteUserID:    in.SiteUserID,
		ApproachIDs:   in.ApproachIDs,
		StartedAt:     in.StartedAt,
		EndedAt:       in.EndedAt,
		StatusID:      in.StatusID,
		Date:          in.Date,
		ExerciseCount: in.ExerciseCount,
	}
}
