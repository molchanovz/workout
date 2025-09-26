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
			Tables.BotUser.Name: {StatusFilter},
		},
		sort: map[string][]SortField{
			Tables.BotUser.Name: {{Column: Columns.BotUser.CreatedAt, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.BotUser.Name: {TableColumns},
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

/*** BotUser ***/

// FullBotUser returns full joins with all columns
func (ur UserRepo) FullBotUser() OpFunc {
	return WithColumns(ur.join[Tables.BotUser.Name]...)
}

// DefaultBotUserSort returns default sort.
func (ur UserRepo) DefaultBotUserSort() OpFunc {
	return WithSort(ur.sort[Tables.BotUser.Name]...)
}

// BotUserByID is a function that returns BotUser by ID(s) or nil.
func (ur UserRepo) BotUserByID(ctx context.Context, id int, ops ...OpFunc) (*BotUser, error) {
	return ur.OneBotUser(ctx, &BotUserSearch{ID: &id}, ops...)
}

// OneBotUser is a function that returns one BotUser by filters. It could return pg.ErrMultiRows.
func (ur UserRepo) OneBotUser(ctx context.Context, search *BotUserSearch, ops ...OpFunc) (*BotUser, error) {
	obj := &BotUser{}
	err := buildQuery(ctx, ur.db, obj, search, ur.filters[Tables.BotUser.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// BotUsersByFilters returns BotUser list.
func (ur UserRepo) BotUsersByFilters(ctx context.Context, search *BotUserSearch, pager Pager, ops ...OpFunc) (botUsers []BotUser, err error) {
	err = buildQuery(ctx, ur.db, &botUsers, search, ur.filters[Tables.BotUser.Name], pager, ops...).Select()
	return
}

// CountBotUsers returns count
func (ur UserRepo) CountBotUsers(ctx context.Context, search *BotUserSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, ur.db, &BotUser{}, search, ur.filters[Tables.BotUser.Name], PagerOne, ops...).Count()
}

// AddBotUser adds BotUser to DB.
func (ur UserRepo) AddBotUser(ctx context.Context, botUser *BotUser, ops ...OpFunc) (*BotUser, error) {
	q := ur.db.ModelContext(ctx, botUser)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.BotUser.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return botUser, err
}

// UpdateBotUser updates BotUser in DB.
func (ur UserRepo) UpdateBotUser(ctx context.Context, botUser *BotUser, ops ...OpFunc) (bool, error) {
	q := ur.db.ModelContext(ctx, botUser).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.BotUser.ID, Columns.BotUser.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteBotUser set statusId to deleted in DB.
func (ur UserRepo) DeleteBotUser(ctx context.Context, id int) (deleted bool, err error) {
	botUser := &BotUser{ID: id, StatusID: StatusDeleted}

	return ur.UpdateBotUser(ctx, botUser, WithColumns(Columns.BotUser.StatusID))
}
