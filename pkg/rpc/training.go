package rpc

import (
	"github.com/vmkteam/embedlog"
	"github.com/vmkteam/zenrpc/v2"
	"workout/pkg/workout/training"
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
