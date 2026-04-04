package rpc

import (
	"context"
	"encoding/json"
	"net/http"

	"workout/pkg/db"

	"github.com/vmkteam/zenrpc/v2"
)

type siteUserCtx string

const siteUserKey siteUserCtx = "rpc.siteUser"

const AuthKey = "Authorization2"

var ErrUnauthorized = zenrpc.NewStringError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

func authMiddleware(ur db.UserRepo) zenrpc.MiddlewareFunc {
	return func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			req, ok := zenrpc.RequestFromContext(ctx)
			if !ok {
				return h(ctx, method, params)
			}

			authHeader := req.Header.Get(AuthKey)
			if authHeader == "" {
				return zenrpc.NewResponseError(zenrpc.IDFromContext(ctx), ErrUnauthorized.Code, ErrUnauthorized.Message, ErrUnauthorized.Data)
			}

			user, err := ur.OneSiteUser(ctx, &db.SiteUserSearch{ApiKey: &authHeader})
			if err != nil || user == nil {
				return zenrpc.NewResponseError(zenrpc.IDFromContext(ctx), ErrUnauthorized.Code, ErrUnauthorized.Message, ErrUnauthorized.Data)
			}

			return h(context.WithValue(ctx, siteUserKey, user), method, params)
		}
	}
}

func SiteUserFromContext(ctx context.Context) *db.SiteUser {
	if u, ok := ctx.Value(siteUserKey).(*db.SiteUser); ok {
		return u
	}
	return nil
}
