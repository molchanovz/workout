package suggest

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestSuggest_Demo prints the training history and what the algorithm
// recommends today. Run with `go test -v ./pkg/workout/suggest/` to eyeball it.
func TestSuggest_Demo(t *testing.T) {
	today := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC) // Friday
	history := demoHistory(today)

	t.Log("\n" + formatHistory(today, history))

	cases := []struct {
		label string
		opts  Options
	}{
		{"фулбади", Options{Mode: ModeFullbody}},
		{"верх", Options{Mode: ModeUpper}},
		{"низ", Options{Mode: ModeLower}},
	}

	for _, c := range cases {
		s := Suggest(today, history, c.opts)
		t.Log("\n" + formatSuggestion(c.label, s))
	}
}

func demoHistory(today time.Time) []PastTraining {
	day := func(d int) time.Time { return today.AddDate(0, 0, -d) }
	upper := func(id int, name, sub string) Approach {
		return Approach{ExerciseID: id, ExerciseName: name, CategoryID: UpperCategoryID, CategoryTitle: sub}
	}
	lower := func(id int, name, sub string) Approach {
		return Approach{ExerciseID: id, ExerciseName: name, CategoryID: LowerCategoryID, CategoryTitle: sub}
	}
	benchDay := func(date time.Time) PastTraining {
		return PastTraining{Date: date, Approaches: []Approach{
			upper(1, "Жим лёжа", "Верх"),
			upper(1, "Жим лёжа", "Верх"),
			upper(1, "Жим лёжа", "Верх"),
			upper(2, "Жим гантелей на наклонной", "Верх"),
			upper(2, "Жим гантелей на наклонной", "Верх"),
			upper(6, "Подтягивания", "Верх"),
			upper(6, "Подтягивания", "Верх"),
			upper(3, "Разгибания на трицепс", "Верх"),
			upper(8, "Подъём на бицепс", "Верх"),
		}}
	}
	legDay := func(date time.Time) PastTraining {
		return PastTraining{Date: date, Approaches: []Approach{
			lower(4, "Приседания", "Низ"),
			lower(4, "Приседания", "Низ"),
			lower(4, "Приседания", "Низ"),
			lower(5, "Сгибания ног", "Низ"),
			lower(5, "Сгибания ног", "Низ"),
			lower(12, "Подъём на носки", "Низ"),
			upper(7, "Тяга штанги в наклоне", "Верх"),
			upper(7, "Тяга штанги в наклоне", "Верх"),
		}}
	}
	return []PastTraining{
		benchDay(day(28)),
		legDay(day(26)),
		benchDay(day(23)),
		benchDay(day(21)),
		legDay(day(19)),
		benchDay(day(16)),
		benchDay(day(14)),
		legDay(day(12)),
		benchDay(day(9)),
		benchDay(day(7)),
		legDay(day(5)),
		benchDay(day(4)),
		legDay(day(2)),
	}
}

func formatHistory(today time.Time, history []PastTraining) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== HISTORY (today = %s) ===\n", today.Format("Mon 2006-01-02"))

	sorted := make([]PastTraining, len(history))
	copy(sorted, history)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.After(sorted[j].Date) })

	for _, tr := range sorted {
		daysAgo := int(today.Sub(tr.Date).Hours() / 24)
		fmt.Fprintf(&b, "\n%s (%d days ago, %s)\n",
			tr.Date.Format("Mon 2006-01-02"), daysAgo, dominantCategory(tr))

		type key struct {
			id   int
			name string
			cat  string
		}
		counts := map[key]int{}
		var order []key
		for _, a := range tr.Approaches {
			k := key{a.ExerciseID, a.ExerciseName, a.CategoryTitle}
			if _, ok := counts[k]; !ok {
				order = append(order, k)
			}
			counts[k]++
		}
		for _, k := range order {
			fmt.Fprintf(&b, "    [%s] %-20s  %d sets\n", k.cat, k.name, counts[k])
		}
	}
	return b.String()
}

func dominantCategory(tr PastTraining) string {
	count := map[string]int{}
	for _, a := range tr.Approaches {
		count[a.CategoryTitle]++
	}
	var top string
	var n int
	for c, k := range count {
		if k > n {
			n, top = k, c
		}
	}
	return top
}

func formatSuggestion(label string, s SuggestedTraining) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== SUGGESTION: %s ===\n", label)
	if len(s.Exercises) == 0 {
		b.WriteString("(nothing matches the filter)\n")
		return b.String()
	}
	cats := make([]string, 0, len(s.Categories))
	for _, c := range s.Categories {
		cats = append(cats, c.Title)
	}
	fmt.Fprintf(&b, "categories: %s\n", strings.Join(cats, ", "))
	for i, e := range s.Exercises {
		fmt.Fprintf(&b, "  %d. [%s] %-25s (freq=%d in window)\n",
			i+1, e.CategoryTitle, e.ExerciseName, e.Frequency)
	}
	return b.String()
}
