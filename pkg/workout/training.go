package workout

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/vmkteam/embedlog"
	"time"
	"workout/pkg/db"
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
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	var startDate *time.Time
	var endDate *time.Time

	if date != nil {
		startDate = date
		endDate = Ptr(date.AddDate(0, 0, 1))
	}

	trainings, err := tm.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{SiteUserID: &user.ID, StatusID: Ptr(db.StatusEnabled), StartedAtGrater: startDate, StartedAtLess: endDate},
		db.PagerNoLimit,
	)

	return NewTrainings(trainings), err
}

// ExerciseList returns unique exercises present in a training, in order of first appearance.
func (tm TrainingManager) ExerciseList(ctx context.Context, tgId, trainingId int) ([]db.Exercise, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return nil, err
	}
	if len(training.ApproachIDs) == 0 {
		return nil, nil
	}

	search := &db.ApproachSearch{IDs: training.ApproachIDs, StatusID: Ptr(db.StatusEnabled)}
	search.With("t.\"exerciseId\" IS NOT NULL")

	approaches, err := tm.tr.ApproachesByFilters(ctx, search, db.PagerNoLimit,
		db.WithRelations(db.Columns.Approach.Exercise),
	)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]bool)
	var result []db.Exercise
	for _, a := range approaches {
		if a.ExerciseID != nil && !seen[*a.ExerciseID] && a.Exercise != nil {
			seen[*a.ExerciseID] = true
			result = append(result, *a.Exercise)
		}
	}
	return result, nil
}

// ApproachList returns approaches for a specific exercise within a training.
func (tm TrainingManager) ApproachList(ctx context.Context, tgId, trainingId, exerciseId int) (Approaches, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return nil, err
	}

	approaches, err := tm.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{
			IDs:        training.ApproachIDs,
			ExerciseID: &exerciseId,
			StatusID:   Ptr(db.StatusEnabled),
		},
		db.PagerNoLimit,
		db.WithRelations(db.Columns.Approach.Exercise),
	)

	return NewApproaches(approaches), err
}

// NewTraining creates a new training session for the given date.
func (tm TrainingManager) NewTraining(ctx context.Context, tgId int, date time.Time) (*Training, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.New("user not found")
	}

	training, err := tm.tr.AddTraining(ctx, &db.Training{
		SiteUserID: user.ID,
		StartedAt:  date,
		StatusID:   db.StatusEnabled,
	})
	if err != nil {
		return nil, err
	}
	return NewTraining(training), nil
}

// AddApproach creates an approach with filled fields and appends it to the training.
func (tm TrainingManager) AddApproach(ctx context.Context, tgId, trainingId, exerciseId, reps int, weight float64) error {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return err
	}

	weightInt := int(weight)
	return tm.dbo.RunInLock(ctx, fmt.Sprintf("training#%v", training.ID), func(tx *pg.Tx) error {
		trainingRepo := tm.tr.WithTransaction(tx)

		approach, err := trainingRepo.AddApproach(ctx, &db.Approach{
			ExerciseID: &exerciseId,
			Reps:       &reps,
			Weight:     &weightInt,
			CreatedAt:  Ptr(time.Now()),
			StatusID:   db.StatusEnabled,
		})
		if err != nil {
			return err
		}

		training.ApproachIDs = append(training.ApproachIDs, approach.ID)
		updated, err := trainingRepo.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.ApproachIDs))
		if err != nil {
			return err
		} else if !updated {
			return errors.New("update training failed")
		}
		return nil
	})
}

// UpdateApproach updates reps and weight of an existing approach.
func (tm TrainingManager) UpdateApproach(ctx context.Context, tgId, approachId, reps int, weight float64) error {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	weightInt := int(weight)
	approach, err := tm.tr.ApproachByID(ctx, approachId)
	if err != nil {
		return err
	} else if approach == nil {
		return errors.New("approach not found")
	}

	approach.Reps = &reps
	approach.Weight = &weightInt
	updated, err := tm.tr.UpdateApproach(ctx, approach,
		db.WithColumns(db.Columns.Approach.Reps, db.Columns.Approach.Weight),
	)
	if err != nil {
		return err
	} else if !updated {
		return errors.New("update approach failed")
	}
	return nil
}

// DeleteApproach soft-deletes an approach and removes it from the training's list.
func (tm TrainingManager) DeleteApproach(ctx context.Context, tgId, trainingId, approachId int) error {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return err
	}

	return tm.dbo.RunInLock(ctx, fmt.Sprintf("training#%v", training.ID), func(tx *pg.Tx) error {
		trainingRepo := tm.tr.WithTransaction(tx)

		approach, err := trainingRepo.ApproachByID(ctx, approachId)
		if err != nil {
			return err
		} else if approach == nil {
			return errors.New("approach not found")
		}
		approach.StatusID = db.StatusDeleted
		updated, err := trainingRepo.UpdateApproach(ctx, approach, db.WithColumns(db.Columns.Approach.StatusID))
		if err != nil {
			return err
		} else if !updated {
			return errors.New("delete approach failed")
		}

		newIDs := make([]int, 0, len(training.ApproachIDs))
		for _, id := range training.ApproachIDs {
			if id != approachId {
				newIDs = append(newIDs, id)
			}
		}
		training.ApproachIDs = newIDs
		_, err = trainingRepo.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.ApproachIDs))
		return err
	})
}

// DeleteTraining soft-deletes a training session.
func (tm TrainingManager) DeleteTraining(ctx context.Context, tgId, trainingId int) error {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return err
	} else if training == nil || training.SiteUserID != user.ID {
		return errors.New("training not found")
	}

	training.StatusID = db.StatusDeleted
	updated, err := tm.tr.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.StatusID))
	if err != nil {
		return err
	} else if !updated {
		return errors.New("delete training failed")
	}
	return nil
}

func (tm TrainingManager) NewApproach(ctx context.Context, tgId, trainingId int) error {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return err
	}

	err = tm.dbo.RunInLock(ctx, fmt.Sprintf("training#%v", training.ID), func(tx *pg.Tx) error {
		trainingRepo := tm.tr.WithTransaction(tx)

		approach, err := trainingRepo.AddApproach(ctx, &db.Approach{
			CreatedAt: Ptr(time.Now()),
			StatusID:  db.StatusEnabled,
		})
		if err != nil {
			return err
		}

		ds := training.ApproachIDs
		ds = append(ds, approach.ID)
		training.ApproachIDs = ds

		updated, err := trainingRepo.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.ApproachIDs))
		if err != nil {
			return err
		} else if !updated {
			return errors.New("update training failed")
		}

		return nil
	})

	return err
}
