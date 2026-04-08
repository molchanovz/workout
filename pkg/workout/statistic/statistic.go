package statistic

import (
	"context"
	"math"
	"sort"
	"time"

	"workout/pkg/db"
	"workout/pkg/workout"

	"github.com/vmkteam/embedlog"
)

type StatsManager struct {
	embedlog.Logger
	dbo db.DB
	tr  db.TrainingRepo
	ur  db.UserRepo
}

func NewStatsManager(dbo db.DB, log embedlog.Logger) *StatsManager {
	return &StatsManager{
		Logger: log,
		dbo:    dbo,
		tr:     db.NewTrainingRepo(dbo.DB),
		ur:     db.NewUserRepo(dbo.DB),
	}
}

// UserStats returns statistics for the given user.
func (m *StatsManager) UserStats(ctx context.Context, tgId int) (*workout.Stats, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return &workout.Stats{}, nil
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	monthTrainings, err := m.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{
			SiteUserID:      &user.ID,
			StartedAtGrater: &monthStart,
			StatusID:        workout.Ptr(db.StatusEnabled),
		},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	yearTrainings, err := m.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{
			SiteUserID:      &user.ID,
			StartedAtGrater: &yearStart,
			StatusID:        workout.Ptr(db.StatusEnabled),
		},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	tonnage, err := m.calcTonnage(ctx, yearTrainings)
	if err != nil {
		return nil, err
	}

	return &workout.Stats{
		MonthCount:   len(monthTrainings),
		YearCount:    len(yearTrainings),
		TotalTonnage: tonnage,
	}, nil
}

// Leaderboard returns users sorted by total tonnage.
func (m *StatsManager) Leaderboard(ctx context.Context) ([]workout.UserStats, error) {
	users, err := m.ur.SiteUsersByFilters(ctx, &db.SiteUserSearch{StatusID: workout.Ptr(db.StatusEnabled)}, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	var result []workout.UserStats
	for i := range users {
		trainings, err := m.tr.TrainingsByFilters(ctx,
			&db.TrainingSearch{SiteUserID: &users[i].ID, StatusID: workout.Ptr(db.StatusEnabled)},
			db.PagerNoLimit,
		)
		if err != nil {
			return nil, err
		}
		tonnage, err := m.calcTonnage(ctx, trainings)
		if err != nil {
			return nil, err
		}
		u := workout.NewSiteUser(&users[i])
		result = append(result, workout.UserStats{SiteUser: u, Stats: workout.Stats{TotalTonnage: tonnage}})
	}

	// sort by tonnage descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Stats.TotalTonnage > result[i].Stats.TotalTonnage {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// PersonalRecords returns per-exercise personal records for the given user.
func (m *StatsManager) PersonalRecords(ctx context.Context, userID int) ([]workout.PREntry, error) {
	trainings, err := m.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	var allApproachIDs []int
	for _, t := range trainings {
		allApproachIDs = append(allApproachIDs, t.ApproachIDs...)
	}
	if len(allApproachIDs) == 0 {
		return nil, nil
	}

	approaches, err := m.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
		m.tr.FullApproach(),
	)
	if err != nil {
		return nil, err
	}

	type best struct {
		weight     int
		reps       int
		achievedAt time.Time
		title      string
	}

	bests := make(map[int]*best)
	for _, a := range approaches {
		if a.Reps == nil || a.Weight == nil || a.ExerciseID == nil {
			continue
		}
		eid := *a.ExerciseID
		b, ok := bests[eid]
		if !ok || *a.Weight > b.weight || (*a.Weight == b.weight && *a.Reps > b.reps) {
			title := ""
			if a.Exercise != nil {
				title = a.Exercise.Title
			}
			achievedAt := time.Time{}
			if a.CreatedAt != nil {
				achievedAt = *a.CreatedAt
			}
			bests[eid] = &best{weight: *a.Weight, reps: *a.Reps, achievedAt: achievedAt, title: title}
		}
	}

	result := make([]workout.PREntry, 0, len(bests))
	for eid, b := range bests {
		est1rm := math.Round(float64(b.weight)*(1.0+float64(b.reps)/30.0)*10) / 10
		result = append(result, workout.PREntry{
			ExerciseID:    eid,
			ExerciseTitle: b.title,
			MaxWeight:     b.weight,
			Reps:          b.reps,
			Est1RM:        est1rm,
			AchievedAt:    b.achievedAt,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ExerciseTitle < result[j].ExerciseTitle
	})

	return result, nil
}

// WeeklyVolume returns tonnage per ISO week for the last `weeks` weeks.
func (m *StatsManager) WeeklyVolume(ctx context.Context, userID, weeks int) ([]workout.WeekVolume, error) {
	if weeks <= 0 {
		weeks = 12
	}

	now := time.Now().UTC()
	currentWeekMon := workout.StartOfISOWeek(now)
	from := currentWeekMon.AddDate(0, 0, -(weeks-1)*7)

	trainings, err := m.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{
			SiteUserID:      &userID,
			StartedAtGrater: &from,
			StatusID:        workout.Ptr(db.StatusEnabled),
		},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	approachWeek := make(map[int]string) // approachID → week key
	var allApproachIDs []int
	for _, t := range trainings {
		weekKey := workout.StartOfISOWeek(t.StartedAt).Format("2006-01-02")
		for _, aid := range t.ApproachIDs {
			allApproachIDs = append(allApproachIDs, aid)
			approachWeek[aid] = weekKey
		}
	}

	weekVolume := make(map[string]float64)
	if len(allApproachIDs) > 0 {
		approaches, err := m.tr.ApproachesByFilters(ctx,
			&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
			db.PagerNoLimit,
		)
		if err != nil {
			return nil, err
		}
		for _, a := range approaches {
			if a.Reps == nil || a.Weight == nil {
				continue
			}
			weekKey, ok := approachWeek[a.ID]
			if !ok {
				continue
			}
			weekVolume[weekKey] += float64(*a.Reps) * float64(*a.Weight)
		}
	}

	result := make([]workout.WeekVolume, weeks)
	for i := 0; i < weeks; i++ {
		weekMon := currentWeekMon.AddDate(0, 0, -(weeks-1-i)*7)
		key := weekMon.Format("2006-01-02")
		result[i] = workout.WeekVolume{Week: key, Volume: weekVolume[key]}
	}

	return result, nil
}

// TrainingStreak returns consecutive-week streak and period counts for the given user.
func (m *StatsManager) TrainingStreak(ctx context.Context, userID int) (*workout.StreakStats, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	allTrainings, err := m.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{SiteUserID: &userID, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return nil, err
	}

	monthCount, yearCount := 0, 0
	trainingWeeks := make(map[string]bool)
	for _, t := range allTrainings {
		if !t.StartedAt.Before(yearStart) {
			yearCount++
			if !t.StartedAt.Before(monthStart) {
				monthCount++
			}
		}
		trainingWeeks[workout.StartOfISOWeek(t.StartedAt).Format("2006-01-02")] = true
	}

	streak := 0
	cur := workout.StartOfISOWeek(now)
	for trainingWeeks[cur.Format("2006-01-02")] {
		streak++
		cur = cur.AddDate(0, 0, -7)
	}

	return &workout.StreakStats{
		CurrentStreak: streak,
		MonthCount:    monthCount,
		YearCount:     yearCount,
	}, nil
}

func (m *StatsManager) calcTonnage(ctx context.Context, trainings []db.Training) (float64, error) {
	var allApproachIDs []int
	for _, t := range trainings {
		allApproachIDs = append(allApproachIDs, t.ApproachIDs...)
	}
	if len(allApproachIDs) == 0 {
		return 0, nil
	}

	approaches, err := m.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: workout.Ptr(db.StatusEnabled)},
		db.PagerNoLimit,
	)
	if err != nil {
		return 0, err
	}

	var tonnage float64
	for _, a := range approaches {
		if a.Reps != nil && a.Weight != nil {
			tonnage += float64(*a.Reps) * float64(*a.Weight)
		}
	}
	return tonnage, nil
}
