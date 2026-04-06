package rpc

import (
	"context"
	"net/http"
	"workout/pkg/db"
	"workout/pkg/workout"

	"github.com/vmkteam/embedlog"
	"github.com/vmkteam/zenrpc/v2"
)

type ExerciseService struct {
	zenrpc.Service
	embedlog.Logger

	tr db.TrainingRepo
}

func NewExerciseService(logger embedlog.Logger, dbo db.DB) *ExerciseService {
	return &ExerciseService{
		Logger: logger,
		tr:     db.NewTrainingRepo(dbo.DB),
	}
}

// CategoryList returns categories, optionally filtered by parent.
//
//zenrpc:parentId Parent category ID (null for root categories)
//zenrpc:return List of categories
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s ExerciseService) CategoryList(ctx context.Context, parentId *int) ([]Category, error) {
	if SiteUserFromContext(ctx) == nil {
		return nil, zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	enabled := db.StatusEnabled
	search := &db.CategorySearch{StatusID: &enabled, ParentCategoryID: parentId}
	if parentId == nil {
		search.With(`t."parentCategoryId" IS NULL`)
	}

	list, err := s.tr.CategoriesByFilters(ctx, search, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	return NewCategories(workout.NewCategories(list)), nil
}

// List returns exercises for a category (global + personal for the current user).
//
//zenrpc:categoryId Category ID
//zenrpc:return List of exercises
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s ExerciseService) List(ctx context.Context, categoryId int) ([]Exercise, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	enabled := db.StatusEnabled
	search := &db.ExerciseSearch{CategoryID: &categoryId, StatusID: &enabled}
	search.With(`(t."siteUserId" IS NULL OR t."siteUserId" = ?)`, user.ID)

	list, err := s.tr.ExercisesByFilters(ctx, search, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	return NewExercises(workout.NewExercises(list)), nil
}

// AddCategory creates a new category.
//
//zenrpc:title Category title
//zenrpc:parentCategoryId Parent category ID (null for root)
//zenrpc:return Created category ID
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s ExerciseService) AddCategory(ctx context.Context, title string, parentCategoryId *int) (int, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return 0, zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	cat, err := s.tr.AddCategory(ctx, &db.Category{
		Title:            title,
		ParentCategoryID: parentCategoryId,
		SiteUserID:       workout.Ptr(user.ID),
		StatusID:         db.StatusEnabled,
	})
	if err != nil {
		return 0, newInternalError(err)
	}
	return cat.ID, nil
}

// Add creates a new exercise.
//
//zenrpc:title Exercise title
//zenrpc:categoryId Category ID
//zenrpc:typeId Exercise type: 1=Strength, 2=Timed
//zenrpc:return Created exercise ID
//zenrpc:401 Unauthorized
//zenrpc:500 Internal Error
func (s ExerciseService) Add(ctx context.Context, title string, categoryId, typeId int) (int, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return 0, zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	ex, err := s.tr.AddExercise(ctx, &db.Exercise{
		Title:      title,
		CategoryID: categoryId,
		TypeID:     typeId,
		SiteUserID: workout.Ptr(user.ID),
		StatusID:   db.StatusEnabled,
	})
	if err != nil {
		return 0, newInternalError(err)
	}
	return ex.ID, nil
}

func (s ExerciseService) Search(ctx context.Context, title string) ([]Exercise, error) {
	user := SiteUserFromContext(ctx)
	if user == nil {
		return nil, zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}

	enabled := db.StatusEnabled
	search := &db.ExerciseSearch{StatusID: &enabled}
	search.With(`(t."siteUserId" IS NULL OR t."siteUserId" = ?)`, user.ID)
	search.With(`(lower_ru(t."title") LIKE lower_ru(?))`, "%"+title+"%")
	list, err := s.tr.ExercisesByFilters(ctx, search, db.PagerNoLimit)
	if err != nil {
		return nil, err
	}

	return NewExercises(workout.NewExercises(list)), nil
}
