package rpc

import "workout/pkg/workout"

type Exercise struct {
	ID         int    `json:"id,omitempty"`
	Title      string `json:"title,omitempty"`
	CategoryID int    `json:"categoryId,omitempty"`
	SiteUserID *int   `json:"siteUserId,omitempty"`
	TypeID     int    `json:"typeId,omitempty"`
	StatusID   int    `json:"statusId,omitempty"`
}

func NewExercise(in *workout.Exercise) *Exercise {
	if in == nil {
		return nil
	}

	return &Exercise{
		ID:         in.ID,
		Title:      in.Title,
		CategoryID: in.CategoryID,
		SiteUserID: in.SiteUserID,
		TypeID:     in.TypeID,
		StatusID:   in.StatusID,
	}
}
