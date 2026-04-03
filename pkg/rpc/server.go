package rpc

import (
	"net/http"
	"workout/pkg/workout/training"

	"workout/pkg/db"

	"github.com/vmkteam/embedlog"
	zm "github.com/vmkteam/zenrpc-middleware"
	"github.com/vmkteam/zenrpc/v2"
)

var (
	ErrNotImplemented = zenrpc.NewStringError(http.StatusInternalServerError, "not implemented")
	ErrInternal       = zenrpc.NewStringError(http.StatusInternalServerError, "internal error")
)

var allowDebugFn = func() zm.AllowDebugFunc {
	return func(req *http.Request) bool {
		return req != nil && req.FormValue("__level") == "5"
	}
}

var namespaces = struct {
	Auth     string
	Training string
	Exercise string
}{
	"auth",
	"training",
	"exercise",
}

type Options struct {
	AppName         string
	DB              db.DB
	Logger          embedlog.Logger
	TrainingManager *training.Manager
}

//go:generate zenrpc

// New returns new zenrpc Server.
func New(opt Options, isDevel bool) zenrpc.Server {
	rpc := zenrpc.NewServer(zenrpc.Options{
		ExposeSMD: true,
		AllowCORS: true,
	})

	rpc.Use(
		zm.WithDevel(isDevel),
		zm.WithHeaders(),
		zm.WithSentry(zm.DefaultServerName),
		zm.WithNoCancelContext(),
		zm.WithMetrics(zm.DefaultServerName),
		zm.WithTiming(isDevel, allowDebugFn()),
		zm.WithSQLLogger(opt.DB.DB, isDevel, allowDebugFn(), allowDebugFn()),
	)

	rpc.Use(
		zm.WithSLog(opt.Logger.Print, zm.DefaultServerName, nil),
		zm.WithErrorSLog(opt.Logger.Print, zm.DefaultServerName, nil),
		authMiddleware(db.NewUserRepo(opt.DB.DB)),
	)

	// services
	rpc.RegisterAll(map[string]zenrpc.Invoker{
		namespaces.Training: NewTrainingService(opt.Logger, opt.TrainingManager),
		namespaces.Exercise: NewExerciseService(opt.Logger, opt.DB),
	})

	return rpc
}

//nolint:unused
func newInternalError(err error) *zenrpc.Error {
	return zenrpc.NewError(http.StatusInternalServerError, err)
}
