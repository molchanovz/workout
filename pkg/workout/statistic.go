package workout

import (
	"time"
)

type Stats struct {
	MonthCount   int
	YearCount    int
	TotalTonnage float64
}

type UserStats struct {
	SiteUser *SiteUser
	Stats    Stats
}

// PREntry holds a personal record for a single exercise.
type PREntry struct {
	ExerciseID    int       `json:"exerciseId"`
	ExerciseTitle string    `json:"exerciseTitle"`
	MaxWeight     int       `json:"maxWeight"`
	Reps          int       `json:"reps"`
	Est1RM        float64   `json:"est1rm"`
	AchievedAt    time.Time `json:"achievedAt"`
}

// WeekVolume holds total tonnage for an ISO week.
type WeekVolume struct {
	Week   string  `json:"week"`   // Monday date "YYYY-MM-DD"
	Volume float64 `json:"volume"` // Σ(reps × weight)
}

// StreakStats holds consecutive-week streak and period training counts.
type StreakStats struct {
	CurrentStreak int `json:"currentStreak"`
	MonthCount    int `json:"monthCount"`
	YearCount     int `json:"yearCount"`
}

// StartOfISOWeek returns the Monday that starts the ISO week containing t.
func StartOfISOWeek(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // Sunday → 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-wd+1, 0, 0, 0, 0, time.UTC)
}
