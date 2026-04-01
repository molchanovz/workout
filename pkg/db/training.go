package db

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type TrainingRepo struct {
	db      orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// NewTrainingRepo returns new repository
func NewTrainingRepo(db orm.DB) TrainingRepo {
	return TrainingRepo{
		db: db,
		filters: map[string][]Filter{
			Tables.Approach.Name: {StatusFilter},
			Tables.Category.Name: {StatusFilter},
			Tables.Exercise.Name: {StatusFilter},
			Tables.Training.Name: {StatusFilter},
		},
		sort: map[string][]SortField{
			Tables.Approach.Name: {{Column: Columns.Approach.CreatedAt, Direction: SortDesc}},
			Tables.Category.Name: {{Column: Columns.Category.Title, Direction: SortAsc}},
			Tables.Exercise.Name: {{Column: Columns.Exercise.Title, Direction: SortAsc}},
			Tables.Training.Name: {{Column: Columns.Training.ID, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.Approach.Name: {TableColumns, Columns.Approach.Exercise},
			Tables.Category.Name: {TableColumns, Columns.Category.ParentCategory, Columns.Category.SiteUser},
			Tables.Exercise.Name: {TableColumns, Columns.Exercise.Category, Columns.Exercise.SiteUser},
			Tables.Training.Name: {TableColumns, Columns.Training.SiteUser},
		},
	}
}

// WithTransaction is a function that wraps TrainingRepo with pg.Tx transaction.
func (tr TrainingRepo) WithTransaction(tx *pg.Tx) TrainingRepo {
	tr.db = tx
	return tr
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func (tr TrainingRepo) WithEnabledOnly() TrainingRepo {
	f := make(map[string][]Filter, len(tr.filters))
	for i := range tr.filters {
		f[i] = make([]Filter, len(tr.filters[i]))
		copy(f[i], tr.filters[i])
		f[i] = append(f[i], StatusEnabledFilter)
	}
	tr.filters = f

	return tr
}

/*** Approach ***/

// FullApproach returns full joins with all columns
func (tr TrainingRepo) FullApproach() OpFunc {
	return WithColumns(tr.join[Tables.Approach.Name]...)
}

// DefaultApproachSort returns default sort.
func (tr TrainingRepo) DefaultApproachSort() OpFunc {
	return WithSort(tr.sort[Tables.Approach.Name]...)
}

// ApproachByID is a function that returns Approach by ID(s) or nil.
func (tr TrainingRepo) ApproachByID(ctx context.Context, id int, ops ...OpFunc) (*Approach, error) {
	return tr.OneApproach(ctx, &ApproachSearch{ID: &id}, ops...)
}

// OneApproach is a function that returns one Approach by filters. It could return pg.ErrMultiRows.
func (tr TrainingRepo) OneApproach(ctx context.Context, search *ApproachSearch, ops ...OpFunc) (*Approach, error) {
	obj := &Approach{}
	err := buildQuery(ctx, tr.db, obj, search, tr.filters[Tables.Approach.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// ApproachesByFilters returns Approach list.
func (tr TrainingRepo) ApproachesByFilters(ctx context.Context, search *ApproachSearch, pager Pager, ops ...OpFunc) (approaches []Approach, err error) {
	err = buildQuery(ctx, tr.db, &approaches, search, tr.filters[Tables.Approach.Name], pager, ops...).Select()
	return
}

// CountApproaches returns count
func (tr TrainingRepo) CountApproaches(ctx context.Context, search *ApproachSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, tr.db, &Approach{}, search, tr.filters[Tables.Approach.Name], PagerOne, ops...).Count()
}

// AddApproach adds Approach to DB.
func (tr TrainingRepo) AddApproach(ctx context.Context, approach *Approach, ops ...OpFunc) (*Approach, error) {
	q := tr.db.ModelContext(ctx, approach)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Approach.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return approach, err
}

// UpdateApproach updates Approach in DB.
func (tr TrainingRepo) UpdateApproach(ctx context.Context, approach *Approach, ops ...OpFunc) (bool, error) {
	q := tr.db.ModelContext(ctx, approach).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Approach.ID, Columns.Approach.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteApproach set statusId to deleted in DB.
func (tr TrainingRepo) DeleteApproach(ctx context.Context, id int) (deleted bool, err error) {
	approach := &Approach{ID: id, StatusID: StatusDeleted}

	return tr.UpdateApproach(ctx, approach, WithColumns(Columns.Approach.StatusID))
}

/*** Category ***/

// FullCategory returns full joins with all columns
func (tr TrainingRepo) FullCategory() OpFunc {
	return WithColumns(tr.join[Tables.Category.Name]...)
}

// DefaultCategorySort returns default sort.
func (tr TrainingRepo) DefaultCategorySort() OpFunc {
	return WithSort(tr.sort[Tables.Category.Name]...)
}

// CategoryByID is a function that returns Category by ID(s) or nil.
func (tr TrainingRepo) CategoryByID(ctx context.Context, id int, ops ...OpFunc) (*Category, error) {
	return tr.OneCategory(ctx, &CategorySearch{ID: &id}, ops...)
}

// OneCategory is a function that returns one Category by filters. It could return pg.ErrMultiRows.
func (tr TrainingRepo) OneCategory(ctx context.Context, search *CategorySearch, ops ...OpFunc) (*Category, error) {
	obj := &Category{}
	err := buildQuery(ctx, tr.db, obj, search, tr.filters[Tables.Category.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// CategoriesByFilters returns Category list.
func (tr TrainingRepo) CategoriesByFilters(ctx context.Context, search *CategorySearch, pager Pager, ops ...OpFunc) (categories []Category, err error) {
	err = buildQuery(ctx, tr.db, &categories, search, tr.filters[Tables.Category.Name], pager, ops...).Select()
	return
}

// CountCategories returns count
func (tr TrainingRepo) CountCategories(ctx context.Context, search *CategorySearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, tr.db, &Category{}, search, tr.filters[Tables.Category.Name], PagerOne, ops...).Count()
}

// AddCategory adds Category to DB.
func (tr TrainingRepo) AddCategory(ctx context.Context, category *Category, ops ...OpFunc) (*Category, error) {
	q := tr.db.ModelContext(ctx, category)
	applyOps(q, ops...)
	_, err := q.Insert()

	return category, err
}

// UpdateCategory updates Category in DB.
func (tr TrainingRepo) UpdateCategory(ctx context.Context, category *Category, ops ...OpFunc) (bool, error) {
	q := tr.db.ModelContext(ctx, category).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Category.ID)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteCategory set statusId to deleted in DB.
func (tr TrainingRepo) DeleteCategory(ctx context.Context, id int) (deleted bool, err error) {
	category := &Category{ID: id, StatusID: StatusDeleted}

	return tr.UpdateCategory(ctx, category, WithColumns(Columns.Category.StatusID))
}

/*** Exercise ***/

// FullExercise returns full joins with all columns
func (tr TrainingRepo) FullExercise() OpFunc {
	return WithColumns(tr.join[Tables.Exercise.Name]...)
}

// DefaultExerciseSort returns default sort.
func (tr TrainingRepo) DefaultExerciseSort() OpFunc {
	return WithSort(tr.sort[Tables.Exercise.Name]...)
}

// ExerciseByID is a function that returns Exercise by ID(s) or nil.
func (tr TrainingRepo) ExerciseByID(ctx context.Context, id int, ops ...OpFunc) (*Exercise, error) {
	return tr.OneExercise(ctx, &ExerciseSearch{ID: &id}, ops...)
}

// OneExercise is a function that returns one Exercise by filters. It could return pg.ErrMultiRows.
func (tr TrainingRepo) OneExercise(ctx context.Context, search *ExerciseSearch, ops ...OpFunc) (*Exercise, error) {
	obj := &Exercise{}
	err := buildQuery(ctx, tr.db, obj, search, tr.filters[Tables.Exercise.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// ExercisesByFilters returns Exercise list.
func (tr TrainingRepo) ExercisesByFilters(ctx context.Context, search *ExerciseSearch, pager Pager, ops ...OpFunc) (exercises []Exercise, err error) {
	err = buildQuery(ctx, tr.db, &exercises, search, tr.filters[Tables.Exercise.Name], pager, ops...).Select()
	return
}

// CountExercises returns count
func (tr TrainingRepo) CountExercises(ctx context.Context, search *ExerciseSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, tr.db, &Exercise{}, search, tr.filters[Tables.Exercise.Name], PagerOne, ops...).Count()
}

// AddExercise adds Exercise to DB.
func (tr TrainingRepo) AddExercise(ctx context.Context, exercise *Exercise, ops ...OpFunc) (*Exercise, error) {
	q := tr.db.ModelContext(ctx, exercise)
	applyOps(q, ops...)
	_, err := q.Insert()

	return exercise, err
}

// UpdateExercise updates Exercise in DB.
func (tr TrainingRepo) UpdateExercise(ctx context.Context, exercise *Exercise, ops ...OpFunc) (bool, error) {
	q := tr.db.ModelContext(ctx, exercise).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Exercise.ID)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteExercise set statusId to deleted in DB.
func (tr TrainingRepo) DeleteExercise(ctx context.Context, id int) (deleted bool, err error) {
	exercise := &Exercise{ID: id, StatusID: StatusDeleted}

	return tr.UpdateExercise(ctx, exercise, WithColumns(Columns.Exercise.StatusID))
}

/*** Training ***/

// FullTraining returns full joins with all columns
func (tr TrainingRepo) FullTraining() OpFunc {
	return WithColumns(tr.join[Tables.Training.Name]...)
}

// DefaultTrainingSort returns default sort.
func (tr TrainingRepo) DefaultTrainingSort() OpFunc {
	return WithSort(tr.sort[Tables.Training.Name]...)
}

// TrainingByID is a function that returns Training by ID(s) or nil.
func (tr TrainingRepo) TrainingByID(ctx context.Context, id int, ops ...OpFunc) (*Training, error) {
	return tr.OneTraining(ctx, &TrainingSearch{ID: &id}, ops...)
}

// OneTraining is a function that returns one Training by filters. It could return pg.ErrMultiRows.
func (tr TrainingRepo) OneTraining(ctx context.Context, search *TrainingSearch, ops ...OpFunc) (*Training, error) {
	obj := &Training{}
	err := buildQuery(ctx, tr.db, obj, search, tr.filters[Tables.Training.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// TrainingsByFilters returns Training list.
func (tr TrainingRepo) TrainingsByFilters(ctx context.Context, search *TrainingSearch, pager Pager, ops ...OpFunc) (trainings []Training, err error) {
	err = buildQuery(ctx, tr.db, &trainings, search, tr.filters[Tables.Training.Name], pager, ops...).Select()
	return
}

// CountTrainings returns count
func (tr TrainingRepo) CountTrainings(ctx context.Context, search *TrainingSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, tr.db, &Training{}, search, tr.filters[Tables.Training.Name], PagerOne, ops...).Count()
}

// AddTraining adds Training to DB.
func (tr TrainingRepo) AddTraining(ctx context.Context, training *Training, ops ...OpFunc) (*Training, error) {
	q := tr.db.ModelContext(ctx, training)
	applyOps(q, ops...)
	_, err := q.Insert()

	return training, err
}

// UpdateTraining updates Training in DB.
func (tr TrainingRepo) UpdateTraining(ctx context.Context, training *Training, ops ...OpFunc) (bool, error) {
	q := tr.db.ModelContext(ctx, training).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Training.ID)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteTraining set statusId to deleted in DB.
func (tr TrainingRepo) DeleteTraining(ctx context.Context, id int) (deleted bool, err error) {
	training := &Training{ID: id, StatusID: StatusDeleted}

	return tr.UpdateTraining(ctx, training, WithColumns(Columns.Training.StatusID))
}
