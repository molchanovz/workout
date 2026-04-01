package db

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type UserRepo struct {
	db      orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// NewUserRepo returns new repository
func NewUserRepo(db orm.DB) UserRepo {
	return UserRepo{
		db: db,
		filters: map[string][]Filter{
			Tables.SiteUser.Name: {StatusFilter},
		},
		sort: map[string][]SortField{
			Tables.SiteUser.Name: {{Column: Columns.SiteUser.CreatedAt, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.SiteUser.Name: {TableColumns},
		},
	}
}

// WithTransaction is a function that wraps UserRepo with pg.Tx transaction.
func (ur UserRepo) WithTransaction(tx *pg.Tx) UserRepo {
	ur.db = tx
	return ur
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func (ur UserRepo) WithEnabledOnly() UserRepo {
	f := make(map[string][]Filter, len(ur.filters))
	for i := range ur.filters {
		f[i] = make([]Filter, len(ur.filters[i]))
		copy(f[i], ur.filters[i])
		f[i] = append(f[i], StatusEnabledFilter)
	}
	ur.filters = f

	return ur
}

/*** SiteUser ***/

// FullSiteUser returns full joins with all columns
func (ur UserRepo) FullSiteUser() OpFunc {
	return WithColumns(ur.join[Tables.SiteUser.Name]...)
}

// DefaultSiteUserSort returns default sort.
func (ur UserRepo) DefaultSiteUserSort() OpFunc {
	return WithSort(ur.sort[Tables.SiteUser.Name]...)
}

// SiteUserByID is a function that returns SiteUser by ID(s) or nil.
func (ur UserRepo) SiteUserByID(ctx context.Context, id int, ops ...OpFunc) (*SiteUser, error) {
	return ur.OneSiteUser(ctx, &SiteUserSearch{ID: &id}, ops...)
}

// OneSiteUser is a function that returns one SiteUser by filters. It could return pg.ErrMultiRows.
func (ur UserRepo) OneSiteUser(ctx context.Context, search *SiteUserSearch, ops ...OpFunc) (*SiteUser, error) {
	obj := &SiteUser{}
	err := buildQuery(ctx, ur.db, obj, search, ur.filters[Tables.SiteUser.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// SiteUsersByFilters returns SiteUser list.
func (ur UserRepo) SiteUsersByFilters(ctx context.Context, search *SiteUserSearch, pager Pager, ops ...OpFunc) (siteUsers []SiteUser, err error) {
	err = buildQuery(ctx, ur.db, &siteUsers, search, ur.filters[Tables.SiteUser.Name], pager, ops...).Select()
	return
}

// CountSiteUsers returns count
func (ur UserRepo) CountSiteUsers(ctx context.Context, search *SiteUserSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, ur.db, &SiteUser{}, search, ur.filters[Tables.SiteUser.Name], PagerOne, ops...).Count()
}

// AddSiteUser adds SiteUser to DB.
func (ur UserRepo) AddSiteUser(ctx context.Context, siteUser *SiteUser, ops ...OpFunc) (*SiteUser, error) {
	q := ur.db.ModelContext(ctx, siteUser)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.SiteUser.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return siteUser, err
}

// UpdateSiteUser updates SiteUser in DB.
func (ur UserRepo) UpdateSiteUser(ctx context.Context, siteUser *SiteUser, ops ...OpFunc) (bool, error) {
	q := ur.db.ModelContext(ctx, siteUser).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.SiteUser.ID, Columns.SiteUser.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteSiteUser set statusId to deleted in DB.
func (ur UserRepo) DeleteSiteUser(ctx context.Context, id int) (deleted bool, err error) {
	siteUser := &SiteUser{ID: id, StatusID: StatusDeleted}

	return ur.UpdateSiteUser(ctx, siteUser, WithColumns(Columns.SiteUser.StatusID))
}
