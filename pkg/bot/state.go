package bot

import (
	"sync"
	"time"
)

type Step string

const (
	StepSelectCategory    Step = "select_category"
	StepSelectSubcategory Step = "select_subcategory"
	StepSelectExercise    Step = "select_exercise"
	StepEnterReps         Step = "enter_reps"
	StepEnterWeight       Step = "enter_weight"
	StepEnterCategoryName Step = "enter_category_name"
	StepEnterExerciseName Step = "enter_exercise_name"
)

type UserState struct {
	Step       Step
	TrainingID int
	Date       string
	CategoryID int
	ExerciseID int
	ApproachID int
	Reps       int
	ExpiresAt  time.Time
}

type StateStore struct {
	mu    sync.Mutex
	store map[int64]*UserState
}

func NewStateStore() *StateStore {
	return &StateStore{store: make(map[int64]*UserState)}
}

func (s *StateStore) Set(tgID int64, state *UserState) {
	state.ExpiresAt = time.Now().Add(30 * time.Minute)
	s.mu.Lock()
	s.store[tgID] = state
	s.mu.Unlock()
}

func (s *StateStore) Get(tgID int64) *UserState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store[tgID]
	if st == nil || time.Now().After(st.ExpiresAt) {
		delete(s.store, tgID)
		return nil
	}
	return st
}

func (s *StateStore) Delete(tgID int64) {
	s.mu.Lock()
	delete(s.store, tgID)
	s.mu.Unlock()
}
