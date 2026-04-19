// Package suggest picks exercises for "today's training" based on the user's
// recent history.
//
// The algorithm:
//   - ranks exercises by approach count inside a lookback window
//     (default: last 30 days);
//   - excludes exercises the user did in their most recent past training
//     (to avoid proposing what was just done);
//   - narrows to upper (root category 1) or lower (root category 2) when
//     the caller asks for it.
//
// Only exercises are returned — reps/weight/sets are filled in separately
// after the user applies the suggestion.
//
// No database dependency: operates on plain structs so the training
// manager can feed whatever history it has.
package suggest

import (
	"sort"
	"time"
)

// Mode is the body-part mode picked on the frontend.
type Mode int

const (
	ModeFullbody Mode = iota
	ModeUpper
	ModeLower
)

// Root category IDs in the workout DB.
const (
	UpperCategoryID = 1
	LowerCategoryID = 2
)

// Approach is one logged set of a single exercise.
// CategoryID / CategoryTitle refer to the *root* category (parent if nested).
type Approach struct {
	ExerciseID    int
	ExerciseName  string
	CategoryID    int
	CategoryTitle string
}

// PastTraining is one completed training session.
type PastTraining struct {
	Date       time.Time
	Approaches []Approach
}

// SuggestedExercise is one item in the suggested plan.
type SuggestedExercise struct {
	ExerciseID    int    `json:"exerciseId"`
	ExerciseName  string `json:"name"`
	CategoryID    int    `json:"categoryId"`
	CategoryTitle string `json:"categoryTitle"`
	// Frequency — total approaches of this exercise inside the window.
	Frequency int `json:"frequency"`
}

// Category is a root muscle-group bucket present in the suggestion.
type Category struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// SuggestedTraining is the output of Suggest.
type SuggestedTraining struct {
	For        time.Time           `json:"for"`
	Categories []Category          `json:"categories"`
	Exercises  []SuggestedExercise `json:"exercises"`
}

// Options tunes the suggestion.
type Options struct {
	// Mode — fullbody / upper / lower.
	Mode Mode
	// Window is the lookback; defaults to 30 days.
	Window time.Duration
	// IncludeLastTraining — by default (false) exercises from the most
	// recent past training are excluded.
	IncludeLastTraining bool
}

func (o Options) withDefaults() Options {
	if o.Window <= 0 {
		o.Window = 30 * 24 * time.Hour
	}
	return o
}

type exerciseStat struct {
	id            int
	name          string
	categoryID    int
	categoryTitle string
	lastSeen      time.Time
	approachCount int
}

// Suggest picks a training for `today` based on `history`.
// Returns an empty SuggestedTraining if history is empty.
func Suggest(today time.Time, history []PastTraining, opts Options) SuggestedTraining {
	opts = opts.withDefaults()
	out := SuggestedTraining{For: today}
	if len(history) == 0 {
		return out
	}

	sorted := make([]PastTraining, len(history))
	copy(sorted, history)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.After(sorted[j].Date) })

	// IDs of exercises done in the most recent past training.
	lastTrainingIDs := map[int]bool{}
	for i := range sorted {
		if sorted[i].Date.Before(today) {
			for _, a := range sorted[i].Approaches {
				lastTrainingIDs[a.ExerciseID] = true
			}
			break
		}
	}

	allowedID := allowedCategoryID(opts.Mode)
	cutoff := today.Add(-opts.Window)

	collect := func(excludeLast bool) map[int]*exerciseStat {
		stats := map[int]*exerciseStat{}
		for _, t := range sorted {
			if t.Date.Before(cutoff) || t.Date.After(today) {
				continue
			}
			for _, a := range t.Approaches {
				if excludeLast && lastTrainingIDs[a.ExerciseID] {
					continue
				}
				if allowedID != 0 && a.CategoryID != allowedID {
					continue
				}
				s, ok := stats[a.ExerciseID]
				if !ok {
					s = &exerciseStat{
						id:            a.ExerciseID,
						name:          a.ExerciseName,
						categoryID:    a.CategoryID,
						categoryTitle: a.CategoryTitle,
						lastSeen:      t.Date,
					}
					stats[a.ExerciseID] = s
				}
				s.approachCount++
				if t.Date.After(s.lastSeen) {
					s.lastSeen = t.Date
				}
			}
		}
		return stats
	}

	excludeLast := !opts.IncludeLastTraining
	stats := collect(excludeLast)
	// Fallback: if the filter + last-training exclusion removed everything
	// (e.g. user picked "lower" right after a legs day), retry without
	// the exclusion so the user still gets something.
	if len(stats) == 0 && excludeLast {
		stats = collect(false)
	}
	if len(stats) == 0 {
		return out
	}

	// Rank: frequency desc, then recency desc, then name asc.
	list := make([]*exerciseStat, 0, len(stats))
	for _, s := range stats {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].approachCount != list[j].approachCount {
			return list[i].approachCount > list[j].approachCount
		}
		if !list[i].lastSeen.Equal(list[j].lastSeen) {
			return list[i].lastSeen.After(list[j].lastSeen)
		}
		return list[i].name < list[j].name
	})

	seenCat := map[int]bool{}
	for _, s := range list {
		if !seenCat[s.categoryID] {
			seenCat[s.categoryID] = true
			out.Categories = append(out.Categories, Category{ID: s.categoryID, Title: s.categoryTitle})
		}
		out.Exercises = append(out.Exercises, SuggestedExercise{
			ExerciseID:    s.id,
			ExerciseName:  s.name,
			CategoryID:    s.categoryID,
			CategoryTitle: s.categoryTitle,
			Frequency:     s.approachCount,
		})
	}
	return out
}

func allowedCategoryID(m Mode) int {
	switch m {
	case ModeUpper:
		return UpperCategoryID
	case ModeLower:
		return LowerCategoryID
	default:
		return 0
	}
}
