package workout

import (
	"context"
	"time"
	"workout/pkg/db"

	"github.com/vmkteam/embedlog"
)

type Stats struct {
	MonthCount   int
	YearCount    int
	TotalTonnage float64
}

type UserStats struct {
	SiteUser *SiteUser
	Stats    Stats
}

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
func (m *StatsManager) UserStats(ctx context.Context, tgId int) (*Stats, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return &Stats{}, nil
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	monthTrainings, err := m.tr.TrainingsByFilters(ctx,
		&db.TrainingSearch{
			SiteUserID:      &user.ID,
			StartedAtGrater: &monthStart,
			StatusID:        Ptr(db.StatusEnabled),
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
			StatusID:        Ptr(db.StatusEnabled),
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

	return &Stats{
		MonthCount:   len(monthTrainings),
		YearCount:    len(yearTrainings),
		TotalTonnage: tonnage,
	}, nil
}

// Leaderboard returns users sorted by total tonnage.
func (m *StatsManager) Leaderboard(ctx context.Context) ([]UserStats, error) {
	users, err := m.ur.SiteUsersByFilters(ctx, &db.SiteUserSearch{StatusID: Ptr(db.StatusEnabled)}, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	var result []UserStats
	for i := range users {
		trainings, err := m.tr.TrainingsByFilters(ctx,
			&db.TrainingSearch{SiteUserID: &users[i].ID, StatusID: Ptr(db.StatusEnabled)},
			db.PagerNoLimit,
		)
		if err != nil {
			return nil, err
		}
		tonnage, err := m.calcTonnage(ctx, trainings)
		if err != nil {
			return nil, err
		}
		u := NewSiteUser(&users[i])
		result = append(result, UserStats{SiteUser: u, Stats: Stats{TotalTonnage: tonnage}})
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

func (m *StatsManager) calcTonnage(ctx context.Context, trainings []db.Training) (float64, error) {
	var allApproachIDs []int
	for _, t := range trainings {
		allApproachIDs = append(allApproachIDs, t.ApproachIDs...)
	}
	if len(allApproachIDs) == 0 {
		return 0, nil
	}

	approaches, err := m.tr.ApproachesByFilters(ctx,
		&db.ApproachSearch{IDs: allApproachIDs, StatusID: Ptr(db.StatusEnabled)},
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
