package rpc

import (
	"context"
	"net/http"
	"time"

	"workout/pkg/db"
	"workout/pkg/workout"
	"workout/pkg/workout/training"

	"github.com/vmkteam/embedlog"
	"github.com/vmkteam/zenrpc/v2"
)

var (
	errNotFound     = zenrpc.NewStringError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	errUnauthorized = zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
)

type TrainingService struct {
	zenrpc.Service
	embedlog.Logger

	tm *training.Manager
}

func NewTrainingService(logger embedlog.Logger, tm *training.Manager) *TrainingService {
	return &TrainingService{
		Logger: logger,
		tm:     tm,
	}
}

// ExerciseWithApproaches holds an exercise and its approaches in a training.
type ExerciseWithApproaches struct {
	Exercise   db.Exercise        `json:"exercise"`
	Approaches workout.Approaches `json:"approaches"`
}

// TrainingDetail is a full training with exercises and approaches.
type TrainingDetail struct {
	workout.Training
	Exercises []ExerciseWithApproaches `json:"exercises"`
}

// List returns trainings for a given date or date range.
//
//zenrpc:date Filter by specific date (YYYY-MM-DD)
//zenrpc:from Range start date (YYYY-MM-DD)
//zenrpc:to Range end date (YYYY-MM-DD)
//zenrpc:return List of trainings
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) List(ctx context.Context, date, from, to *string) (workout.Trainings, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	var fromT, toT *time.Time
	if date != nil {
		t, err := time.Parse("2006-01-02", *date)
		if err != nil {
			return nil, zenrpc.NewStringError(http.StatusBadRequest, "invalid date format")
		}
		fromT = &t
		end := t.AddDate(0, 0, 1)
		toT = &end
	} else {
		if from != nil {
			t, err := time.Parse("2006-01-02", *from)
			if err != nil {
				return nil, zenrpc.NewStringError(http.StatusBadRequest, "invalid from date format")
			}
			fromT = &t
		}
		if to != nil {
			t, err := time.Parse("2006-01-02", *to)
			if err != nil {
				return nil, zenrpc.NewStringError(http.StatusBadRequest, "invalid to date format")
			}
			toT = &t
		}
	}

	return s.tm.TrainingListByRange(ctx, user.ID, fromT, toT)
}

// Get returns a full training with exercises and approaches.
//
//zenrpc:id Training ID
//zenrpc:return Full training details
//zenrpc:401 Unauthorized
//zenrpc:404 Not Found
//zenrpc:500 Internal Error
func (s TrainingService) Get(ctx context.Context, id int) (*TrainingDetail, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	t, err := s.tm.TrainingByID(ctx, user.ID, id)
	if err != nil {
		return nil, newInternalError(err)
	}
	if t == nil {
		return nil, errNotFound
	}

	exercises, err := s.tm.ExerciseList(ctx, user.ID, id)
	if err != nil {
		return nil, newInternalError(err)
	}

	detail := &TrainingDetail{Training: *t}
	for _, ex := range exercises {
		approaches, err := s.tm.ApproachList(ctx, user.ID, id, ex.ID)
		if err != nil {
			return nil, newInternalError(err)
		}
		detail.Exercises = append(detail.Exercises, ExerciseWithApproaches{
			Exercise:   ex,
			Approaches: approaches,
		})
	}

	return detail, nil
}

// New creates a new training for the given date.
//
//zenrpc:date Training date (YYYY-MM-DD)
//zenrpc:return Created training
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) New(ctx context.Context, date string) (*workout.Training, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, zenrpc.NewStringError(http.StatusBadRequest, "invalid date format")
	}

	return s.tm.NewTrainingForUser(ctx, user.ID, t)
}

// Delete soft-deletes a training and all its approaches.
//
//zenrpc:id Training ID
//zenrpc:return Success
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) Delete(ctx context.Context, id int) (bool, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return false, errUnauthorized
	}

	if err := s.tm.DeleteTraining(ctx, user.ID, id); err != nil {
		return false, newInternalError(err)
	}
	return true, nil
}

// ExerciseList returns unique exercises in a training.
//
//zenrpc:trainingId Training ID
//zenrpc:return List of exercises
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) ExerciseList(ctx context.Context, trainingId int) ([]db.Exercise, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	return s.tm.ExerciseList(ctx, user.ID, trainingId)
}

// ApproachList returns approaches for an exercise in a training.
//
//zenrpc:trainingId Training ID
//zenrpc:exerciseId Exercise ID
//zenrpc:return List of approaches
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) ApproachList(ctx context.Context, trainingId, exerciseId int) (workout.Approaches, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, errUnauthorized
	}

	return s.tm.ApproachList(ctx, user.ID, trainingId, exerciseId)
}

// AddApproach adds a strength approach (reps + weight) to a training.
//
//zenrpc:trainingId Training ID
//zenrpc:exerciseId Exercise ID
//zenrpc:reps Number of repetitions
//zenrpc:weight Weight in kg
//zenrpc:return Created approach ID
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) AddApproach(ctx context.Context, trainingId, exerciseId, reps int, weight float64) (int, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return 0, errUnauthorized
	}

	id, err := s.tm.AddApproach(ctx, user.ID, trainingId, exerciseId, reps, weight)
	if err != nil {
		return 0, newInternalError(err)
	}
	return id, nil
}

// AddTimedApproach adds a timed approach (duration in seconds) to a training.
//
//zenrpc:trainingId Training ID
//zenrpc:exerciseId Exercise ID
//zenrpc:duration Duration in seconds
//zenrpc:return Created approach ID
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) AddTimedApproach(ctx context.Context, trainingId, exerciseId, duration int) (int, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return 0, errUnauthorized
	}

	id, err := s.tm.AddTimedApproach(ctx, user.ID, trainingId, exerciseId, duration)
	if err != nil {
		return 0, newInternalError(err)
	}
	return id, nil
}

// UpdateApproach updates reps and weight of a strength approach.
//
//zenrpc:approachId Approach ID
//zenrpc:reps Number of repetitions
//zenrpc:weight Weight in kg
//zenrpc:return Success
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) UpdateApproach(ctx context.Context, approachId, reps int, weight float64) (bool, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return false, errUnauthorized
	}

	if err := s.tm.UpdateApproach(ctx, user.ID, approachId, reps, weight); err != nil {
		return false, newInternalError(err)
	}
	return true, nil
}

// UpdateTimedApproach updates duration of a timed approach.
//
//zenrpc:approachId Approach ID
//zenrpc:duration Duration in seconds
//zenrpc:return Success
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) UpdateTimedApproach(ctx context.Context, approachId, duration int) (bool, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return false, errUnauthorized
	}

	if err := s.tm.UpdateTimedApproach(ctx, user.ID, approachId, duration); err != nil {
		return false, newInternalError(err)
	}
	return true, nil
}

// DeleteApproach soft-deletes an approach from a training.
//
//zenrpc:trainingId Training ID
//zenrpc:approachId Approach ID
//zenrpc:return Success
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s TrainingService) DeleteApproach(ctx context.Context, trainingId, approachId int) (bool, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return false, errUnauthorized
	}

	if err := s.tm.DeleteApproach(ctx, user.ID, trainingId, approachId); err != nil {
		return false, newInternalError(err)
	}
	return true, nil
}
