package ingest

import (
	"os"
	"testing"

	"trajectory/config"
)

func TestParseICS(t *testing.T) {
	data := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nUID:1\nDTSTART;TZID=America/New_York:20260101T090000\nDTEND;TZID=America/New_York:20260101T100000\nSUMMARY:Standup meeting\nDESCRIPTION:team\n sync\nEND:VEVENT\nBEGIN:VTODO\nUID:t1\nDUE:20260101T140000\nSUMMARY:Finish draft\nEND:VTODO\nEND:VCALENDAR\n"
	f, _ := os.CreateTemp("", "t*.ics")
	defer os.Remove(f.Name())
	_, _ = f.WriteString(data)
	_ = f.Close()
	e, err := ParseICS(f.Name(), config.Default())
	if err != nil || len(e) != 2 {
		t.Fatalf("parse err %v len %d", err, len(e))
	}
	if e[0].Category != "meeting" {
		t.Fatalf("want meeting got %s", e[0].Category)
	}
	if e[1].Category != "task" {
		t.Fatalf("want task got %s", e[1].Category)
	}
}

func TestParseLogs(t *testing.T) {
	f, _ := os.CreateTemp("", "l*.txt")
	defer os.Remove(f.Name())
	_, _ = f.WriteString("2026-01-01 22:30 😴 📱 doomscroll 20m [late]\n")
	_ = f.Close()
	l, err := ParseLogs(f.Name())
	if err != nil || len(l) != 1 {
		t.Fatalf("err %v len %d", err, len(l))
	}
	if l[0].Note != "late" {
		t.Fatalf("expected note parsing")
	}
}
