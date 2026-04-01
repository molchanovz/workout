package workout

import (
	"context"
	"github.com/vmkteam/embedlog"
	"time"
	"workout/pkg/db"
)

type SiteUserManager struct {
	embedlog.Logger
	dbo      db.DB
	userRepo db.UserRepo
}

func NewSiteUserManager(dbo db.DB, log embedlog.Logger) *SiteUserManager {
	return &SiteUserManager{
		Logger:   log,
		dbo:      dbo,
		userRepo: db.NewUserRepo(dbo.DB),
	}
}

func (m SiteUserManager) RegisterUser(ctx context.Context, tgId int) (bool, error) {
	user, err := m.userRepo.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgId})
	if err != nil {
		return false, err
	}

	if user != nil {
		return false, nil
	}

	_, err = m.userRepo.AddSiteUser(ctx, &db.SiteUser{
		TgID:      tgId,
		CreatedAt: time.Now(),
		StatusID:  db.StatusEnabled,
	})

	return true, err
}
