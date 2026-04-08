package rpc

import (
	"context"

	"workout/pkg/workout"
	"workout/pkg/workout/statistic"

	"github.com/vmkteam/embedlog"
	"github.com/vmkteam/zenrpc/v2"
)

type StatsService struct {
	zenrpc.Service
	embedlog.Logger
	sm *statistic.StatsManager
}

func NewStatsService(logger embedlog.Logger, sm *statistic.StatsManager) *StatsService {
	return &StatsService{
		Logger: logger,
		sm:     sm,
	}
}

// PersonalRecords returns per-exercise personal records for the current user.
//
//zenrpc:return Personal records per exercise
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s StatsService) PersonalRecords(ctx context.Context) ([]workout.PREntry, error) { //zenrpc
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	records, err := s.sm.PersonalRecords(ctx, user.ID)
	if err != nil {
		return nil, newInternalError(err)
	}
	return records, nil
}

// WeeklyVolume returns tonnage per ISO week for the last N weeks.
//
//zenrpc:weeks Number of weeks to return (default 12)
//zenrpc:return Weekly volume data
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s StatsService) WeeklyVolume(ctx context.Context, weeks int) ([]workout.WeekVolume, error) { //zenrpc
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	if weeks <= 0 {
		weeks = 12
	}

	volumes, err := s.sm.WeeklyVolume(ctx, user.ID, weeks)
	if err != nil {
		return nil, newInternalError(err)
	}
	return volumes, nil
}

// Streak returns training streak and period counts for the current user.
//
//zenrpc:return Streak statistics
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s StatsService) Streak(ctx context.Context) (*workout.StreakStats, error) { //zenrpc
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	stats, err := s.sm.TrainingStreak(ctx, user.ID)
	if err != nil {
		return nil, newInternalError(err)
	}
	return stats, nil
}
