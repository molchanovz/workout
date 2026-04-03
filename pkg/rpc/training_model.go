package rpc

import "workout/pkg/workout"

type Approach struct {
	workout.Approach
}

func NewApproach(in *workout.Approach) *Approach {
	if in == nil {
		return nil
	}

	return &Approach{
		Approach: *in,
	}
}

type Training struct {
	workout.Training
}

func NewTraining(in *workout.Training) *Training {
	if in == nil {
		return nil
	}

	return &Training{
		Training: *in,
	}
}
