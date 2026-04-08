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
func (tm Manager) TrainingListByRange(ctx context.Context, userID int, from, to *time.Time) (workout.Trainings, error) {
	trainings, err := tm.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled), StartedAtGrater: from, StartedAtLess: to},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	result := workout.NewTrainings(trainings)
	if err = tm.fillExerciseCounts(ctx, result); err != nil {
		return nil, err
	}
	if err = tm.fillCategoryTitles(ctx, result); err != nil {
		return nil, err
	}

	result.SetName()
	return result, nil
}

// fillExerciseCounts populates ExerciseCount for each training using a single batch approach query.
func (tm Manager) fillExerciseCounts(ctx context.Context, trainings workout.Trainings) error {
	// build approachID -> training index map
	approachToIdx := make(map[int]int)
	var allApproachIDs []int
	for i, t := range trainings {
		for _, aid := range t.ApproachIDs {
			approachToIdx[aid] = i
			allApproachIDs = append(allApproachIDs, aid)
		}
	}
	if len(allApproachIDs) == 0 {
		return nil
	}

	approaches, err := tm.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return err
	}

	// count unique exercise IDs per training
	seenExercises := make(map[int]map[int]bool)
	for _, a := range approaches {
		if a.ExerciseID == nil {
			continue
		}
		idx, ok := approachToIdx[a.ID]
		if !ok {
			continue
		}
		if seenExercises[idx] == nil {
			seenExercises[idx] = make(map[int]bool)
		}
		seenExercises[idx][*a.ExerciseID] = true
	}

	for idx, exercises := range seenExercises {
		trainings[idx].ExerciseCount = len(exercises)
	}
	return nil
}

// fillCategoryTitles populates CategoryTitle for each training by finding the dominant root category.
func (tm Manager) fillCategoryTitles(ctx context.Context, trainings workout.Trainings) error {
	approachToIdx := make(map[int]int)
	var allApproachIDs []int
	for i, t := range trainings {
		for _, aid := range t.ApproachIDs {
			approachToIdx[aid] = i
			allApproachIDs = append(allApproachIDs, aid)
		}
	}
	if len(allApproachIDs) == 0 {
		return nil
	}

	approaches, err := tm.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return err
	}

	trainingExercises := make(map[int]map[int]bool)
	exerciseSet := make(map[int]bool)
	for _, a := range approaches {
		if a.ExerciseID == nil {
			continue
		}
		idx, ok := approachToIdx[a.ID]
		if !ok {
			continue
		}
		if trainingExercises[idx] == nil {
			trainingExercises[idx] = make(map[int]bool)
		}
		trainingExercises[idx][*a.ExerciseID] = true
		exerciseSet[*a.ExerciseID] = true
	}
	if len(exerciseSet) == 0 {
		return nil
	}

	exerciseIDs := make([]int, 0, len(exerciseSet))
	for id := range exerciseSet {
		exerciseIDs = append(exerciseIDs, id)
	}

	exercises, err := tm.tr.ExercisesByFilters(ctx,
		&db.ExerciseSearch{IDs: exerciseIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
		db.WithRelations(db.Columns.Exercise.Category),
	)
	if err != nil {
		return err
	}

	parentCategorySet := make(map[int]bool)
	for _, ex := range exercises {
		if ex.Category != nil && ex.Category.ParentCategoryID != nil {
			parentCategorySet[*ex.Category.ParentCategoryID] = true
		}
	}

	parentCategoryTitle := make(map[int]string)
	if len(parentCategorySet) > 0 {
		parentIDs := make([]int, 0, len(parentCategorySet))
		for id := range parentCategorySet {
			parentIDs = append(parentIDs, id)
		}
		parentCategories, err := tm.tr.CategoriesByFilters(ctx,
			&db.CategorySearch{IDs: parentIDs, StatusID: workout.Ptr(db.StatusEnabled)},
			db.PagerNoLimit,
		)
		if err != nil {
			return err
		}
		for _, c := range parentCategories {
			parentCategoryTitle[c.ID] = c.Title
		}
	}

	exerciseCategoryTitle := make(map[int]string)
	for _, ex := range exercises {
		if ex.Category == nil {
			continue
		}
		title := ex.Category.Title
		if ex.Category.ParentCategoryID != nil {
			if parentTitle, ok := parentCategoryTitle[*ex.Category.ParentCategoryID]; ok {
				title = parentTitle
			}
		}
		exerciseCategoryTitle[ex.ID] = title
	}

	for idx, exIDs := range trainingExercises {
		categoryCount := make(map[string]int)
		for exID := range exIDs {
			if title, ok := exerciseCategoryTitle[exID]; ok {
				categoryCount[title]++
			}
		}
		var maxTitle string
		var maxCount int
		for title, count := range categoryCount {
			if count > maxCount {
				maxCount = count
				maxTitle = title
			}
		}
		trainings[idx].CategoryTitle = maxTitle
	}
	return nil
}

// TrainingByID returns a single training by id.
func (tm Manager) TrainingByID(ctx context.Context, userID, trainingId int) (*workout.Training, error) {
	t, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
	if err != nil {
		return nil, err
	}

	return workout.NewTraining(t), nil
}

// ExerciseList returns unique exercises present in a training, in order of first appearance.
// TODO поменять для бота
func (tm Manager) ExerciseList(ctx context.Context, userID, trainingId int) ([]workout.Exercise, error) {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
	if err != nil {
		return nil, err
	} else if len(training.ApproachIDs) == 0 {
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
	return workout.NewExercises(result), nil
}

// ApproachList returns approaches for a specific exercise within a training.
// TODO поменять для бота
func (tm Manager) ApproachList(ctx context.Context, userID, trainingId, exerciseId int) (workout.Approaches, error) {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
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
func (tm Manager) NewTraining(ctx context.Context, userID int, date time.Time) (*workout.Training, error) {
	user, err := tm.ur.OneSiteUser(ctx, &db.SiteUserSearch{ID: &userID})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.New("user not found")
	}

	return tm.newTraining(ctx, user.ID, date)
}

// NewTrainingForUser creates a new training session for a user identified by siteUserID.
func (tm Manager) NewTrainingForUser(ctx context.Context, userID int, date time.Time) (*workout.Training, error) {
	return tm.newTraining(ctx, userID, date)
}

func (tm Manager) newTraining(ctx context.Context, userID int, date time.Time) (*workout.Training, error) {
	training, err := tm.tr.AddTraining(ctx, &db.Training{
		SiteUserID: userID,
		StartedAt:  date,
		StatusID:   db.StatusEnabled,
	})
	if err != nil {
		return nil, err
	}
	return workout.NewTraining(training), nil
}

// AddApproach creates an approach with filled fields and appends it to the training.
// TODO fix for bot
func (tm Manager) AddApproach(ctx context.Context, userID, trainingId, exerciseId, reps int, weight float64) (int, error) {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
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
func (tm Manager) UpdateApproach(ctx context.Context, approachId, reps int, weight float64) error {
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
// TODO fix for bot
func (tm Manager) DeleteApproach(ctx context.Context, userID, trainingId, approachId int) error {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
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
// TODO fix for bot
func (tm Manager) DeleteTraining(ctx context.Context, userID, trainingId int) error {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
	if err != nil {
		return err
	} else if training == nil {
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
// TODO fix for bot
func (tm Manager) AddTimedApproach(ctx context.Context, userID, trainingId, exerciseId, duration int) (int, error) {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
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
// TODO fix for bot
func (tm Manager) UpdateTimedApproach(ctx context.Context, tgId, approachId, duration int) error {
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

func (tm Manager) NewApproach(ctx context.Context, userID, trainingId int) error {
	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{ID: &trainingId, SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)})
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
