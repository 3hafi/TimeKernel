package features

import (
	"testing"
	"time"

	"trajectory/config"
	"trajectory/store"
)

func TestSleepInference(t *testing.T) {
	from, _ := time.Parse("2006-01-02", "2026-01-01")
	to := from.Add(48 * time.Hour)
	logs := []store.LogEntry{{Timestamp: mustT("2026-01-01 23:30"), Tokens: "😴"}, {Timestamp: mustT("2026-01-02 07:00"), Tokens: "🌅"}}
	d := BuildDaily(from, to, nil, logs, config.Default())
	if d["2026-01-01"].SleepMinutes < 400 {
		t.Fatalf("sleep not inferred")
	}
}

func mustT(s string) time.Time { t, _ := time.Parse("2006-01-02 15:04", s); return t }
