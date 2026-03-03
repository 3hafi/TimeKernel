package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"trajectory/features"
	"trajectory/insights"
	"trajectory/store"
)

type AIExport struct {
	ExportedAt    string             `json:"exported_at"`
	Range         string             `json:"range"`
	EventCount    int                `json:"event_count"`
	LogCount      int                `json:"log_count"`
	DailyFeatures []*features.Daily  `json:"daily_features"`
	Insights      []insights.Insight `json:"insights"`
	Events        []store.Event      `json:"events"`
	Logs          []store.LogEntry   `json:"logs"`
	PromptHint    string             `json:"prompt_hint"`
}

func BuildAIExport(from, to time.Time, events []store.Event, logs []store.LogEntry, days map[string]*features.Daily, ins []insights.Insight, format string) (string, error) {
	daily := make([]*features.Daily, 0, len(days))
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		daily = append(daily, days[k])
	}
	data := AIExport{
		ExportedAt:    time.Now().Format(time.RFC3339),
		Range:         fmt.Sprintf("%s..%s", from.Format("2006-01-02"), to.Format("2006-01-02")),
		EventCount:    len(events),
		LogCount:      len(logs),
		DailyFeatures: daily,
		Insights:      ins,
		Events:        events,
		Logs:          logs,
		PromptHint:    "You are analyzing my routines. Use the provided daily_features + events + logs to identify patterns, tradeoffs, and 3 practical changes for next week.",
	}
	if strings.ToLower(format) == "md" {
		return buildAIMarkdown(data), nil
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func buildAIMarkdown(d AIExport) string {
	var sb strings.Builder
	sb.WriteString("# Trajectory AI Export\n\n")
	sb.WriteString("## Metadata\n")
	sb.WriteString(fmt.Sprintf("- exported_at: %s\n- range: %s\n- event_count: %d\n- log_count: %d\n\n", d.ExportedAt, d.Range, d.EventCount, d.LogCount))
	sb.WriteString("## Prompt Hint\n")
	sb.WriteString(d.PromptHint + "\n\n")
	sb.WriteString("## Daily Features\n")
	for _, day := range d.DailyFeatures {
		sb.WriteString(fmt.Sprintf("- %s sleep=%dm meetings=%dm doom=%dm gaming=%dm focus_windows=%d stress=%d\n",
			day.Date, day.SleepMinutes, day.MeetingMinutes, day.DoomscrollMinutes, day.GamingMinutes, day.FocusWindows90, day.StressLogs))
	}
	sb.WriteString("\n## Insights\n")
	for _, i := range d.Insights {
		sb.WriteString(fmt.Sprintf("- %s (%s, n=%d): %s\n", i.Title, i.Confidence, i.SampleN, i.Evidence))
	}
	sb.WriteString("\n## Events and Tasks\n")
	for _, e := range d.Events {
		sb.WriteString(fmt.Sprintf("- %s -> %s | %s | category=%s | source=%s\n", e.Start.Format(time.RFC3339), e.End.Format(time.RFC3339), e.Title, e.Category, e.Source))
	}
	sb.WriteString("\n## Logs\n")
	for _, l := range d.Logs {
		sb.WriteString(fmt.Sprintf("- %s | %s | %s\n", l.Timestamp.Format(time.RFC3339), l.Tokens, l.Note))
	}
	return sb.String()
}
