package simulate

import (
	"fmt"
	"sort"
	"time"

	"trajectory/features"
)

type ForecastDay struct {
	Date                                  string
	SleepMin, FocusWindows, DoomscrollMin int
	Range                                 string
	Warnings                              []string
}

type WhatIfVariant struct {
	Name                                      string
	PredBedtime                               string
	PredSleepMin, PredFocusWindows, LoadScore int
	Feasible                                  bool
	Warnings                                  []string
	Schedule                                  []string
}

func Trajectory(days map[string]*features.Daily, n int) []ForecastDay {
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil
	}
	last := days[keys[len(keys)-1]]
	avgSleep, avgFocus, avgDoom := mean(days, func(d *features.Daily) int { return d.SleepMinutes }), mean(days, func(d *features.Daily) int { return d.FocusWindows90 }), mean(days, func(d *features.Daily) int { return d.DoomscrollMinutes })
	start, _ := time.Parse("2006-01-02", last.Date)
	out := []ForecastDay{}
	for i := 1; i <= n; i++ {
		d := start.AddDate(0, 0, i)
		f := ForecastDay{Date: d.Format("2006-01-02"), SleepMin: avgSleep, FocusWindows: avgFocus, DoomscrollMin: avgDoom, Range: fmt.Sprintf("sleep %d-%d min", avgSleep-45, avgSleep+30)}
		if avgDoom > 90 {
			f.Warnings = append(f.Warnings, "Doomscroll trend likely pushes bedtime later and cuts focus")
		}
		if avgSleep > 0 && avgSleep < 420 {
			f.Warnings = append(f.Warnings, "Sleep debt compounds (~"+fmt.Sprint((420-avgSleep)*7/60)+"h/week deficit)")
		}
		if avgFocus < 2 {
			f.Warnings = append(f.Warnings, "Low focus-window trajectory")
		}
		out = append(out, f)
	}
	return out
}

func WhatIf(base *features.Daily, alloc map[string]time.Duration, freeMinutes int) []WhatIfVariant {
	if base == nil {
		base = &features.Daily{SleepMinutes: 420, FocusWindows90: 2}
	}

	variants := []struct {
		name string
		m    float64
	}{{"conservative", 0.8}, {"balanced", 1.0}, {"aggressive", 1.2}}
	out := []WhatIfVariant{}
	for _, v := range variants {
		total := 0
		for _, d := range alloc {
			total += int(float64(d.Minutes()) * v.m)
		}
		feasible := total <= freeMinutes
		doom := int(float64(alloc["doomscroll"].Minutes()) * v.m)
		study := int(float64(alloc["study"].Minutes()) * v.m)
		exercise := int(float64(alloc["exercise"].Minutes()) * v.m)
		bedShift := doom/20 - exercise/30
		bed := time.Date(2025, 1, 1, 23, 0, 0, 0, time.Local).Add(time.Duration(bedShift) * time.Minute)
		sleep := max(300, base.SleepMinutes-bedShift-doom/10)
		focus := max(0, base.FocusWindows90+study/90+exercise/60-doom/60)
		load := total/60 + doom/30
		w := []string{}
		if !feasible {
			w = append(w, fmt.Sprintf("Infeasible: allocated %dh vs ~%dh free", total/60, freeMinutes/60))
		}
		if doom > 90 {
			w = append(w, "High doomscroll allocation likely delays sleep")
		}
		out = append(out, WhatIfVariant{Name: v.name, PredBedtime: bed.Format("15:04"), PredSleepMin: sleep, PredFocusWindows: focus, LoadScore: load, Feasible: feasible, Warnings: w,
			Schedule: []string{fmt.Sprintf("Deep work: %dm in top free windows", study), fmt.Sprintf("Exercise: %dm", exercise), fmt.Sprintf("Doomscroll cap: %dm", doom)}})
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mean(days map[string]*features.Daily, f func(*features.Daily) int) int {
	if len(days) == 0 {
		return 0
	}
	s := 0
	n := 0
	for _, d := range days {
		v := f(d)
		if v == 0 {
			continue
		}
		s += v
		n++
	}
	if n == 0 {
		return 0
	}
	return s / n
}
