package training

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/vmkteam/embedlog"
	"time"
	"workout/pkg/db"
	"workout/pkg/workout"
)

const (
	ExerciseTypeStrength = 1
	ExerciseTypeTimed    = 2
)

type Manager struct {
	embedlog.Logger
	dbo db.DB
	tr  db.TrainingRepo
	ur  db.UserRepo
}

func NewTrainingManager(dbo db.DB, log embedlog.Logger) *Manager {
	return &Manager{
		Logger: log,
		dbo:    dbo,
		tr:     db.NewTrainingRepo(dbo.DB),
		ur:     db.NewUserRepo(dbo.DB),
	}
}

// TrainingList get trainings by date
func (tm Manager) TrainingList(ctx context.Context, tgId int, date *time.Time) (workout.Trainings, error) {
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
		endDate = workout.Ptr(date.AddDate(0, 0, 1))
	}

	trainings, err := tm.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{SiteUserID: &user.ID, StatusID: workout.Ptr(db.StatusEnabled), StartedAtGrater: startDate, StartedAtLess: endDate},
		db.PagerNoLimit,
	)

	return workout.NewTrainings(trainings), err
}

// TrainingListByRange returns trainings for a date range (both from/to can be nil for all).
func (tm Manager) TrainingListByRange(ctx context.Context, tgId int, from, to *time.Time) (workout.Trainings, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	trainings, err := tm.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{SiteUserID: &user.ID, StatusID: workout.Ptr(db.StatusEnabled), StartedAtGrater: from, StartedAtLess: to},
		db.PagerNoLimit,
	)
	return workout.NewTrainings(trainings), err
}

// TrainingByID returns a single training by id.
func (tm Manager) TrainingByID(ctx context.Context, tgId, trainingId int) (*workout.Training, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}

	t, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return nil, err
	}
	if t == nil || t.SiteUserID != user.ID {
		return nil, nil
	}
	return workout.NewTraining(t), nil
}

// ExerciseList returns unique exercises present in a training, in order of first appearance.
func (tm Manager) ExerciseList(ctx context.Context, tgId, trainingId int) ([]db.Exercise, error) {
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

	search := &db.ApproachSearch{IDs: training.ApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)}
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
func (tm Manager) ApproachList(ctx context.Context, tgId, trainingId, exerciseId int) (workout.Approaches, error) {
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

	approaches, err := tm.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{
			IDs:        training.ApproachIDs,
			ExerciseID: &exerciseId,
			StatusID:   workout.Ptr(db.StatusEnabled),
		},
		db.PagerNoLimit,
		db.WithRelations(db.Columns.Approach.Exercise),
	)

	return workout.NewApproaches(approaches), err
}

// NewTraining creates a new training session for the given date.
func (tm Manager) NewTraining(ctx context.Context, tgId int, date time.Time) (*workout.Training, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.New("user not found")
	}

	return tm.newTraining(ctx, user.ID, date)
}

// NewTrainingForUser creates a new training session for a user identified by siteUserID.
func (tm Manager) NewTrainingForUser(ctx context.Context, siteUserID int, date time.Time) (*workout.Training, error) {
	return tm.newTraining(ctx, siteUserID, date)
}

func (tm Manager) newTraining(ctx context.Context, siteUserID int, date time.Time) (*workout.Training, error) {
	fmt.Println("OK")
	training, err := tm.tr.AddTraining(ctx, &db.Training{
		SiteUserID: siteUserID,
		StartedAt:  date,
		StatusID:   db.StatusEnabled,
	})
	if err != nil {
		return nil, err
	}
	return workout.NewTraining(training), nil
}

// AddApproach creates an approach with filled fields and appends it to the training.
func (tm Manager) AddApproach(ctx context.Context, tgId, trainingId, exerciseId, reps int, weight float64) (int, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return 0, err
	} else if user == nil {
		return 0, errors.New("user not found")
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return 0, err
	}

	weightInt := int(weight)
	var approachID int
	err = tm.dbo.RunInLock(ctx, fmt.Sprintf("training#%v", training.ID), func(tx *pg.Tx) error {
		trainingRepo := tm.tr.WithTransaction(tx)

		approach, err := trainingRepo.AddApproach(ctx, &db.Approach{
			ExerciseID: &exerciseId,
			Reps:       &reps,
			Weight:     &weightInt,
			CreatedAt:  workout.Ptr(time.Now()),
			StatusID:   db.StatusEnabled,
		})
		if err != nil {
			return err
		}
		approachID = approach.ID

		training.ApproachIDs = append(training.ApproachIDs, approach.ID)
		updated, err := trainingRepo.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.ApproachIDs))
		if err != nil {
			return err
		} else if !updated {
			return errors.New("update training failed")
		}
		return nil
	})
	return approachID, err
}

// UpdateApproach updates reps and weight of an existing approach.
func (tm Manager) UpdateApproach(ctx context.Context, tgId, approachId, reps int, weight float64) error {
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
func (tm Manager) DeleteApproach(ctx context.Context, tgId, trainingId, approachId int) error {
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
func (tm Manager) DeleteTraining(ctx context.Context, tgId, trainingId int) error {
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

	return tm.dbo.RunInLock(ctx, fmt.Sprintf("delete-training-%d", trainingId), func(tx *pg.Tx) error {
		tr := tm.tr.WithTransaction(tx)

		training.StatusID = db.StatusDeleted
		updated, err := tr.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.StatusID))
		if err != nil {
			return err
		} else if !updated {
			return errors.New("delete training failed")
		}

		for _, id := range training.ApproachIDs {
			if _, err := tr.DeleteApproach(ctx, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// ExerciseType returns the TypeID of the given exercise.
func (tm Manager) ExerciseType(ctx context.Context, tgId, exerciseId int) (int, error) {
	exercise, err := tm.tr.ExerciseByID(ctx, exerciseId)
	if err != nil {
		return ExerciseTypeStrength, err
	} else if exercise == nil {
		return ExerciseTypeStrength, nil
	}
	return exercise.TypeID, nil
}

// AddTimedApproach creates an approach with Duration set and appends it to the training.
func (tm Manager) AddTimedApproach(ctx context.Context, tgId, trainingId, exerciseId, duration int) (int, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return 0, err
	} else if user == nil {
		return 0, errors.New("user not found")
	}

	training, err := tm.tr.TrainingByID(ctx, trainingId)
	if err != nil {
		return 0, err
	}

	var approachID int
	err = tm.dbo.RunInLock(ctx, fmt.Sprintf("training#%v", training.ID), func(tx *pg.Tx) error {
		trainingRepo := tm.tr.WithTransaction(tx)

		approach, err := trainingRepo.AddApproach(ctx, &db.Approach{
			ExerciseID: &exerciseId,
			Duration:   &duration,
			CreatedAt:  workout.Ptr(time.Now()),
			StatusID:   db.StatusEnabled,
		})
		if err != nil {
			return err
		}
		approachID = approach.ID

		training.ApproachIDs = append(training.ApproachIDs, approach.ID)
		updated, err := trainingRepo.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.ApproachIDs))
		if err != nil {
			return err
		} else if !updated {
			return errors.New("update training failed")
		}
		return nil
	})
	return approachID, err
}

// UpdateTimedApproach updates only Duration of an existing approach.
func (tm Manager) UpdateTimedApproach(ctx context.Context, tgId, approachId, duration int) error {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return err
	} else if user == nil {
		return errors.New("user not found")
	}

	approach, err := tm.tr.ApproachByID(ctx, approachId)
	if err != nil {
		return err
	} else if approach == nil {
		return errors.New("approach not found")
	}

	approach.Duration = &duration
	updated, err := tm.tr.UpdateApproach(ctx, approach,
		db.WithColumns(db.Columns.Approach.Duration),
	)
	if err != nil {
		return err
	} else if !updated {
		return errors.New("update approach failed")
	}
	return nil
}

func (tm Manager) NewApproach(ctx context.Context, tgId, trainingId int) error {
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
			CreatedAt: workout.Ptr(time.Now()),
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
