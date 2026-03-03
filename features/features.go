package features

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"trajectory/config"
	"trajectory/store"
)

type Daily struct {
	Date                                              string
	SleepMinutes, MeetingMinutes, ScheduledMinutes    int
	Events, BackToBack, ShortGaps, FocusWindows90     int
	DoomscrollMinutes, GamingMinutes, ExerciseMinutes int
	HygieneCount, DeepWorkLogs, StressLogs            int
	BedtimeShiftMin                                   int
}

func BuildDaily(from, to time.Time, events []store.Event, logs []store.LogEntry, cfg config.Config) map[string]*Daily {
	out := map[string]*Daily{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		out[k] = &Daily{Date: k}
	}
	for _, e := range events {
		k := e.Start.Format("2006-01-02")
		d, ok := out[k]
		if !ok {
			continue
		}
		mins := int(e.End.Sub(e.Start).Minutes())
		d.ScheduledMinutes += mins
		d.Events++
		if strings.Contains(strings.ToLower(e.Category), "meeting") {
			d.MeetingMinutes += mins
		}
	}
	for _, l := range logs {
		day := logicalDay(l.Timestamp, cfg.DayBoundaryHour).Format("2006-01-02")
		d, ok := out[day]
		if !ok {
			continue
		}
		t := l.Tokens + " " + l.Note
		if strings.Contains(t, "📱") {
			d.DoomscrollMinutes += extractMinutes(t, 30)
		}
		if strings.Contains(t, "🎮") {
			d.GamingMinutes += extractMinutes(t, 45)
		}
		if strings.Contains(t, "🏃") || strings.Contains(t, "🏋") {
			d.ExerciseMinutes += extractMinutes(t, 45)
		}
		if strings.Contains(t, "🚿") {
			d.HygieneCount++
		}
		if strings.Contains(t, "🧠") || strings.Contains(t, "📚") {
			d.DeepWorkLogs++
		}
		if strings.Contains(t, "😡") || strings.Contains(t, "🤯") {
			d.StressLogs++
		}
	}
	inferSleep(out, logs, cfg)
	computeFragmentation(out, events, cfg)
	return out
}

func logicalDay(ts time.Time, boundary int) time.Time {
	if ts.Hour() < boundary {
		return ts.AddDate(0, 0, -1)
	}
	return ts
}

func inferSleep(out map[string]*Daily, logs []store.LogEntry, cfg config.Config) {
	var sleep, wake []time.Time
	for _, l := range logs {
		if strings.Contains(l.Tokens, "😴") || strings.Contains(l.Tokens, "💤") || strings.Contains(l.Tokens, "zzz") {
			sleep = append(sleep, l.Timestamp)
		}
		if strings.Contains(l.Tokens, "🌅") {
			wake = append(wake, l.Timestamp)
		}
	}
	sort.Slice(sleep, func(i, j int) bool { return sleep[i].Before(sleep[j]) })
	sort.Slice(wake, func(i, j int) bool { return wake[i].Before(wake[j]) })
	wi := 0
	for _, st := range sleep {
		for wi < len(wake) && !wake[wi].After(st) {
			wi++
		}
		if wi >= len(wake) {
			break
		}
		w := wake[wi]
		if w.Sub(st) > 12*time.Hour || w.Sub(st) < 2*time.Hour {
			continue
		}
		k := logicalDay(st, cfg.DayBoundaryHour).Format("2006-01-02")
		if d := out[k]; d != nil {
			d.SleepMinutes = int(w.Sub(st).Minutes())
			d.BedtimeShiftMin = st.Hour()*60 + st.Minute() - 23*60
		}
	}
}

func computeFragmentation(out map[string]*Daily, events []store.Event, cfg config.Config) {
	dayEvents := map[string][]store.Event{}
	for _, e := range events {
		k := e.Start.Format("2006-01-02")
		dayEvents[k] = append(dayEvents[k], e)
	}
	for k, arr := range dayEvents {
		d := out[k]
		if d == nil {
			continue
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].Start.Before(arr[j].Start) })
		workStart := time.Date(arr[0].Start.Year(), arr[0].Start.Month(), arr[0].Start.Day(), cfg.WorkdayStart, 0, 0, 0, arr[0].Start.Location())
		workEnd := time.Date(arr[0].Start.Year(), arr[0].Start.Month(), arr[0].Start.Day(), cfg.WorkdayEnd, 0, 0, 0, arr[0].Start.Location())
		prev := workStart
		for _, e := range arr {
			gap := e.Start.Sub(prev)
			if gap <= 15*time.Minute {
				d.BackToBack++
			}
			if gap > 0 && gap < 30*time.Minute {
				d.ShortGaps++
			}
			if gap >= 90*time.Minute && prev.After(workStart.Add(-1*time.Minute)) {
				d.FocusWindows90++
			}
			if e.End.After(prev) {
				prev = e.End
			}
		}
		if workEnd.Sub(prev) >= 90*time.Minute {
			d.FocusWindows90++
		}
	}
}

func extractMinutes(s string, fallback int) int {
	for i := 1; i <= 180; i++ {
		needle := " " + strconv.Itoa(i) + "m"
		if strings.Contains(s, needle) || strings.HasSuffix(strings.TrimSpace(s), strconv.Itoa(i)+"m") {
			return i
		}
	}
	return fallback
}

func Median(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	sort.Ints(vals)
	m := len(vals) / 2
	if len(vals)%2 == 1 {
		return vals[m]
	}
	return int(math.Round(float64(vals[m-1]+vals[m]) / 2))
}
