package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"workout/pkg/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"given_name"`
}

func (a *App) googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.cfg.Google.ClientID,
		ClientSecret: a.cfg.Google.ClientSecret,
		RedirectURL:  a.cfg.Google.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// handleGoogleLogin redirects the user to Google's OAuth consent page.
func (a *App) handleGoogleLogin(c echo.Context) error {
	url := a.googleOAuthConfig().AuthCodeURL("state", oauth2.AccessTypeOnline)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// handleGoogleCallback handles the OAuth callback from Google.
func (a *App) handleGoogleCallback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing code")
	}

	oauthCfg := a.googleOAuthConfig()
	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		a.Error(ctx, "google oauth exchange failed", "err", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "oauth exchange failed")
	}

	userInfo, err := fetchGoogleUserInfo(ctx, oauthCfg.Client(ctx, token))
	if err != nil {
		a.Error(ctx, "failed to fetch google user info", "err", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user info")
	}

	ur := db.NewUserRepo(a.db.DB)

	su, err := ur.OneSiteUser(ctx, &db.SiteUserSearch{GoogleID: &userInfo.ID})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if su == nil {
		enabled := db.StatusEnabled
		su, err = ur.AddSiteUser(ctx, &db.SiteUser{
			GoogleID: &userInfo.ID,
			StatusID: enabled,
		})
		if err != nil || su == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
		}
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": su.ID,
		"iat": time.Now().Unix(),
	})
	jwtStr, err := jwtToken.SignedString([]byte(a.cfg.Google.JWTSecret))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	su.ApiKey = &jwtStr
	if _, err := ur.UpdateSiteUser(ctx, su, db.WithColumns(db.Columns.SiteUser.ApiKey)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if a.cfg.Google.FrontendURL != "" {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?token=%s", a.cfg.Google.FrontendURL, jwtStr))
	}

	return c.JSON(http.StatusOK, map[string]string{"token": jwtStr})
}

func fetchGoogleUserInfo(ctx context.Context, client *http.Client) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
