package report

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"trajectory/features"
	"trajectory/insights"
	"trajectory/simulate"
)

func BuildMarkdown(from, to time.Time, days map[string]*features.Daily, ins []insights.Insight) string {
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	missingSleep := 0
	totalDoom := 0
	for _, k := range keys {
		d := days[k]
		if d.SleepMinutes == 0 {
			missingSleep++
		}
		totalDoom += d.DoomscrollMinutes
	}
	avgDoom := 0
	if len(keys) > 0 {
		avgDoom = totalDoom / len(keys)
	}

	var sb strings.Builder
	sb.WriteString("# Trajectory Report\n\n")
	sb.WriteString(fmt.Sprintf("Range: %s to %s\n\n", from.Format("2006-01-02"), to.Format("2006-01-02")))
	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("Days analyzed: %d\n", len(keys)))
	sb.WriteString(fmt.Sprintf("Data quality: %d days without inferred sleep.\n\n", missingSleep))
	sb.WriteString("## Metrics\n")
	for _, k := range keys {
		d := days[k]
		sb.WriteString(fmt.Sprintf("- %s: sleep=%dm meeting=%dm doom=%dm gaming=%dm focus_windows=%d\n", k, d.SleepMinutes, d.MeetingMinutes, d.DoomscrollMinutes, d.GamingMinutes, d.FocusWindows90))
	}
	sb.WriteString("\n## Insights\n")
	for _, i := range ins {
		sb.WriteString(fmt.Sprintf("- **%s** (%s, n=%d): %s\n  - Experiment: %s\n", i.Title, i.Confidence, i.SampleN, i.Evidence, i.Experiment))
	}
	sb.WriteString("\n## Reality Engine\n")
	sb.WriteString("- Opportunity cost: each added 60m doomscroll displaces one 60m recovery/deep-work block.\n")
	sb.WriteString("- Constraints: day capped at 24h; sleep + scheduled + allocations checked for feasibility.\n")
	sb.WriteString("- Compounding: nightly sleep delta ×7 shown in trajectory warnings.\n")
	sb.WriteString("- Identity-to-outcomes: behavior-outcome statements are evidence-tied, non-moralized.\n")
	sb.WriteString("\n## Top Levers\n")
	if avgDoom > 60 {
		sb.WriteString("- Cap doomscroll to <60m to reclaim focus/sleep capacity.\n")
	}
	sb.WriteString("- Protect first 90m focus window before meetings.\n")
	sb.WriteString("- Add 30m exercise on low-sleep days to reduce bedtime drift.\n")
	return sb.String()
}

func BuildTrajectoryMarkdown(days []simulate.ForecastDay) string {
	var sb strings.Builder
	sb.WriteString("# Trajectory Forecast\n\n")
	for _, d := range days {
		sb.WriteString(fmt.Sprintf("- %s: sleep=%dm, focus_windows=%d, doom=%dm (%s)\n", d.Date, d.SleepMin, d.FocusWindows, d.DoomscrollMin, d.Range))
		for _, w := range d.Warnings {
			sb.WriteString("  - ⚠️ " + w + "\n")
		}
	}
	return sb.String()
}

func BuildWhatIfMarkdown(date string, vars []simulate.WhatIfVariant) string {
	var sb strings.Builder
	sb.WriteString("# What-if Plan " + date + "\n\n")
	for _, v := range vars {
		sb.WriteString(fmt.Sprintf("## %s\n- feasible: %v\n- predicted bedtime: %s\n- predicted sleep: %d min\n- predicted focus windows: %d\n- load score: %d\n", strings.Title(v.Name), v.Feasible, v.PredBedtime, v.PredSleepMin, v.PredFocusWindows, v.LoadScore))
		for _, s := range v.Schedule {
			sb.WriteString("- " + s + "\n")
		}
		for _, w := range v.Warnings {
			sb.WriteString("- ⚠️ " + w + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func Write(path, content, format string) error {
	if format == "html" {
		content = "<html><body><pre>" + content + "</pre></body></html>"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
