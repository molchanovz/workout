package training

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"workout/pkg/db"
	"workout/pkg/workout"
	"workout/pkg/workout/suggest"

	"github.com/go-pg/pg/v10"
)

// suggestHistoryWindow is how far back we look for patterns.
const suggestHistoryWindow = 30 * 24 * time.Hour

// Suggest builds a workout suggestion for the user based on recent history.
func (tm Manager) Suggest(ctx context.Context, userID int, mode suggest.Mode) (*suggest.SuggestedTraining, error) {
	now := time.Now()
	from := now.Add(-suggestHistoryWindow)

	trainings, err := tm.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{
			SiteUserID:      &userID,
			StatusID:        workout.Ptr(db.StatusEnabled),
			StartedAtGrater: &from,
			StartedAtLess:   &now,
		},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	history, err := tm.buildSuggestHistory(ctx, trainings)
	if err != nil {
		return nil, err
	}

	out := suggest.Suggest(now, history, suggest.Options{Mode: mode})
	return &out, nil
}

// buildSuggestHistory converts DB trainings into suggest.PastTraining,
// resolving each exercise's root category title in batch.
func (tm Manager) buildSuggestHistory(ctx context.Context, trainings []db.Training) ([]suggest.PastTraining, error) {
	if len(trainings) == 0 {
		return nil, nil
	}

	var allApproachIDs []int
	for _, t := range trainings {
		allApproachIDs = append(allApproachIDs, t.ApproachIDs...)
	}
	if len(allApproachIDs) == 0 {
		return nil, nil
	}

	approaches, err := tm.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	approachByID := make(map[int]db.Approach, len(approaches))
	exerciseSet := map[int]bool{}
	for _, a := range approaches {
		approachByID[a.ID] = a
		if a.ExerciseID != nil {
			exerciseSet[*a.ExerciseID] = true
		}
	}

	exInfo, err := tm.loadExerciseInfo(ctx, exerciseSet)
	if err != nil {
		return nil, err
	}

	out := make([]suggest.PastTraining, 0, len(trainings))
	for _, t := range trainings {
		pt := suggest.PastTraining{Date: t.StartedAt}
		for _, aid := range t.ApproachIDs {
			a, ok := approachByID[aid]
			if !ok || a.ExerciseID == nil {
				continue
			}
			info := exInfo[*a.ExerciseID]
			pt.Approaches = append(pt.Approaches, suggest.Approach{
				ExerciseID:    *a.ExerciseID,
				ExerciseName:  info.Name,
				CategoryID:    info.CategoryID,
				CategoryTitle: info.CategoryTitle,
			})
		}
		if len(pt.Approaches) > 0 {
			out = append(out, pt)
		}
	}
	return out, nil
}

type exerciseInfo struct {
	Name          string
	CategoryID    int    // root category ID (parent if nested)
	CategoryTitle string // root category title
}

func (tm Manager) loadExerciseInfo(ctx context.Context, ids map[int]bool) (map[int]exerciseInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	exerciseIDs := make([]int, 0, len(ids))
	for id := range ids {
		exerciseIDs = append(exerciseIDs, id)
	}

	exercises, err := tm.tr.ExercisesByFilters(ctx,
		&db.ExerciseSearch{IDs: exerciseIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
		db.WithRelations(db.Columns.Exercise.Category),
	)
	if err != nil {
		return nil, err
	}

	parentSet := map[int]bool{}
	for _, ex := range exercises {
		if ex.Category != nil && ex.Category.ParentCategoryID != nil {
			parentSet[*ex.Category.ParentCategoryID] = true
		}
	}
	parentTitle := map[int]string{}
	if len(parentSet) > 0 {
		parentIDs := make([]int, 0, len(parentSet))
		for id := range parentSet {
			parentIDs = append(parentIDs, id)
		}
		parents, err := tm.tr.CategoriesByFilters(ctx,
			&db.CategorySearch{IDs: parentIDs, StatusID: workout.Ptr(db.StatusEnabled)},
			db.PagerNoLimit,
		)
		if err != nil {
			return nil, err
		}
		for _, c := range parents {
			parentTitle[c.ID] = c.Title
		}
	}

	out := make(map[int]exerciseInfo, len(exercises))
	for _, ex := range exercises {
		info := exerciseInfo{Name: ex.Title}
		if ex.Category != nil {
			info.CategoryID = ex.Category.ID
			info.CategoryTitle = ex.Category.Title
			if ex.Category.ParentCategoryID != nil {
				info.CategoryID = *ex.Category.ParentCategoryID
				if title, ok := parentTitle[*ex.Category.ParentCategoryID]; ok {
					info.CategoryTitle = title
				}
			}
		}
		out[ex.ID] = info
	}
	return out, nil
}

// ApplySuggestion ensures each confirmed exercise is present in the training.
//   - exercises already in the training are skipped;
//   - exercises with history get their approaches prefilled from the user's
//     most recent session of that exercise (same sets × reps × weight / duration);
//   - exercises with no history (first time) get a single empty approach.
//
// Reps/weight/duration can still be edited afterwards via UpdateApproach.
func (tm Manager) ApplySuggestion(ctx context.Context, userID, trainingID int, exerciseIDs []int) error {
	if len(exerciseIDs) == 0 {
		return nil
	}

	training, err := tm.tr.OneTraining(ctx, &db.TrainingSearch{
		ID:         &trainingID,
		SiteUserID: &userID,
		StatusID:   workout.Ptr(db.StatusEnabled),
	})
	if err != nil {
		return err
	}
	if training == nil {
		return errors.New("training not found")
	}

	existing := map[int]bool{}
	if len(training.ApproachIDs) > 0 {
		approaches, err := tm.tr.ApproachesByFilters(ctx,
			&db.ApproachSearch{IDs: training.ApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
			db.PagerNoLimit,
		)
		if err != nil {
			return err
		}
		for _, a := range approaches {
			if a.ExerciseID != nil {
				existing[*a.ExerciseID] = true
			}
		}
	}

	templates, err := tm.lastApproachesByExercise(ctx, userID, training.ID, exerciseIDs)
	if err != nil {
		return err
	}

	return tm.dbo.RunInLock(ctx, fmt.Sprintf("training#%v", training.ID), func(tx *pg.Tx) error {
		repo := tm.tr.WithTransaction(tx)

		added := false
		for _, exID := range exerciseIDs {
			if exID <= 0 || existing[exID] {
				continue
			}
			existing[exID] = true

			tmpl := templates[exID]
			if len(tmpl) == 0 {
				// First time doing this exercise — placeholder only.
				tmpl = []db.Approach{{}}
			}

			for _, t := range tmpl {
				id := exID
				a := &db.Approach{
					ExerciseID: &id,
					Reps:       copyIntPtr(t.Reps),
					Weight:     copyIntPtr(t.Weight),
					Duration:   copyIntPtr(t.Duration),
					CreatedAt:  workout.Ptr(time.Now()),
					StatusID:   db.StatusEnabled,
				}
				created, err := repo.AddApproach(ctx, a)
				if err != nil {
					return err
				}
				training.ApproachIDs = append(training.ApproachIDs, created.ID)
				added = true
			}
		}

		if !added {
			return nil
		}

		ok, err := repo.UpdateTraining(ctx, training, db.WithColumns(db.Columns.Training.ApproachIDs))
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("update training failed")
		}
		return nil
	})
}

// lastApproachesByExercise returns, for each wanted exercise, the group of
// approaches from the user's most recent past training that included it.
// The current training is excluded. Exercises never done before are absent.
func (tm Manager) lastApproachesByExercise(ctx context.Context, userID, currentTrainingID int, wanted []int) (map[int][]db.Approach, error) {
	if len(wanted) == 0 {
		return nil, nil
	}
	now := time.Now()
	from := now.Add(-suggestHistoryWindow)

	trainings, err := tm.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{
			SiteUserID:      &userID,
			StatusID:        workout.Ptr(db.StatusEnabled),
			StartedAtGrater: &from,
			StartedAtLess:   &now,
		},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}
	if len(trainings) == 0 {
		return nil, nil
	}

	wantedSet := map[int]bool{}
	for _, id := range wanted {
		wantedSet[id] = true
	}

	var allApproachIDs []int
	for _, t := range trainings {
		if t.ID == currentTrainingID {
			continue
		}
		allApproachIDs = append(allApproachIDs, t.ApproachIDs...)
	}
	if len(allApproachIDs) == 0 {
		return nil, nil
	}

	approaches, err := tm.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}
	approachByID := make(map[int]db.Approach, len(approaches))
	for _, a := range approaches {
		approachByID[a.ID] = a
	}

	sorted := make([]db.Training, 0, len(trainings))
	for _, t := range trainings {
		if t.ID != currentTrainingID {
			sorted = append(sorted, t)
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartedAt.After(sorted[j].StartedAt) })

	out := map[int][]db.Approach{}
	for _, t := range sorted {
		perEx := map[int][]db.Approach{}
		for _, aid := range t.ApproachIDs {
			a, ok := approachByID[aid]
			if !ok || a.ExerciseID == nil {
				continue
			}
			exID := *a.ExerciseID
			if !wantedSet[exID] {
				continue
			}
			if _, alreadyCaptured := out[exID]; alreadyCaptured {
				continue
			}
			perEx[exID] = append(perEx[exID], a)
		}
		for exID, group := range perEx {
			out[exID] = group
		}
	}
	return out, nil
}

func copyIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
