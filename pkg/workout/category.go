package workout

import (
	"context"
	"errors"

	"workout/pkg/db"

	"github.com/vmkteam/embedlog"
)

type CategoryManager struct {
	embedlog.Logger
	dbo db.DB
	tr  db.TrainingRepo
	ur  db.UserRepo
}

func NewCategoryManager(dbo db.DB, log embedlog.Logger) *CategoryManager {
	return &CategoryManager{
		Logger: log,
		dbo:    dbo,
		tr:     db.NewTrainingRepo(dbo.DB),
		ur:     db.NewUserRepo(dbo.DB),
	}
}

// RootCategories returns categories without a parent (global + user's own).
func (m *CategoryManager) RootCategories(ctx context.Context, tgId int) ([]db.Category, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	}

	search := &db.CategorySearch{StatusID: Ptr(db.StatusEnabled)}
	search.With("t.\"parentCategoryId\" IS NULL")
	if user != nil {
		search.With("(t.\"siteUserId\" IS NULL OR t.\"siteUserId\" = ?)", user.ID)
	} else {
		search.With("t.\"siteUserId\" IS NULL")
	}

	return m.tr.CategoriesByFilters(ctx, search, db.PagerNoLimit)
}

// SubCategories returns subcategories of the given parent.
func (m *CategoryManager) SubCategories(ctx context.Context, tgId int, parentID int) ([]db.Category, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	}

	search := &db.CategorySearch{
		StatusID:         Ptr(db.StatusEnabled),
		ParentCategoryID: &parentID,
	}
	if user != nil {
		search.With("(t.\"siteUserId\" IS NULL OR t.\"siteUserId\" = ?)", user.ID)
	} else {
		search.With("t.\"siteUserId\" IS NULL")
	}

	return m.tr.CategoriesByFilters(ctx, search, db.PagerNoLimit)
}

// ExercisesByCategory returns exercises in the given category (global + user's own).
func (m *CategoryManager) ExercisesByCategory(ctx context.Context, tgId int, categoryID int) ([]db.Exercise, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	}

	search := &db.ExerciseSearch{
		StatusID:   Ptr(db.StatusEnabled),
		CategoryID: &categoryID,
	}
	if user != nil {
		search.With("(t.\"siteUserId\" IS NULL OR t.\"siteUserId\" = ?)", user.ID)
	} else {
		search.With("t.\"siteUserId\" IS NULL")
	}

	return m.tr.ExercisesByFilters(ctx, search, db.PagerNoLimit)
}

// AddCategory creates a new category for the user.
func (m *CategoryManager) AddCategory(ctx context.Context, tgId int, title string, parentID *int) (*db.Category, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.New("user not found")
	}

	return m.tr.AddCategory(ctx, &db.Category{
		Title:            title,
		ParentCategoryID: parentID,
		SiteUserID:       &user.ID,
		StatusID:         db.StatusEnabled,
	})
}

// AddExercise creates a new exercise for the user.
func (m *CategoryManager) AddExercise(ctx context.Context, tgId int, title string, categoryID, typeID int) (*db.Exercise, error) {
	user, err := m.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.New("user not found")
	}

	return m.tr.AddExercise(ctx, &db.Exercise{
		Title:      title,
		CategoryID: categoryID,
		TypeID:     typeID,
		SiteUserID: &user.ID,
		StatusID:   db.StatusEnabled,
	})
}
