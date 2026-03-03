package ingest

import (
	"bufio"
	"os"
	"strings"
	"time"

	"trajectory/config"
	"trajectory/store"
)

func ParseICS(path string, cfg config.Config) ([]store.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	lines := []string{}
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	lines = unfold(lines)

	var items []store.Event
	cur := map[string]string{}
	block := ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		switch line {
		case "BEGIN:VEVENT", "BEGIN:VTODO":
			block = strings.TrimPrefix(line, "BEGIN:")
			cur = map[string]string{}
			continue
		case "END:VEVENT", "END:VTODO":
			if block == "VEVENT" {
				e := eventFromMap(cur, path, cfg)
				if !e.Start.IsZero() && !e.End.IsZero() {
					items = append(items, e)
				}
			}
			if block == "VTODO" {
				t := taskFromMap(cur, path, cfg)
				if !t.Start.IsZero() && !t.End.IsZero() {
					items = append(items, t)
				}
			}
			block = ""
			continue
		}
		if block == "" {
			continue
		}
		if i := strings.Index(line, ":"); i > 0 {
			k := line[:i]
			v := line[i+1:]
			cur[k] = v
		}
	}
	return items, nil
}

func unfold(lines []string) []string {
	out := []string{}
	for _, l := range lines {
		if len(out) > 0 && (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) {
			out[len(out)-1] += strings.TrimLeft(l, " \t")
			continue
		}
		out = append(out, l)
	}
	return out
}

func parseICSDateWithTZ(key, value string) (time.Time, bool) {
	if len(value) == 8 {
		t, _ := time.Parse("20060102", value)
		return t, true
	}
	if strings.HasSuffix(value, "Z") {
		t, _ := time.Parse("20060102T150405Z", value)
		return t.Local(), false
	}
	if strings.Contains(key, "TZID=") {
		parts := strings.Split(key, ";")
		for _, p := range parts {
			if strings.HasPrefix(p, "TZID=") {
				locName := strings.TrimPrefix(p, "TZID=")
				if loc, err := time.LoadLocation(locName); err == nil {
					t, _ := time.ParseInLocation("20060102T150405", value, loc)
					return t.Local(), false
				}
			}
		}
	}
	t, _ := time.ParseInLocation("20060102T150405", value, time.Local)
	return t, false
}

func eventFromMap(m map[string]string, src string, cfg config.Config) store.Event {
	var dsKey, deKey, ds, de string
	for k, v := range m {
		if strings.HasPrefix(k, "DTSTART") {
			dsKey, ds = k, v
		}
		if strings.HasPrefix(k, "DTEND") {
			deKey, de = k, v
		}
	}
	st, ad := parseICSDateWithTZ(dsKey, ds)
	en, _ := parseICSDateWithTZ(deKey, de)
	title := m["SUMMARY"]
	category := categoryFor(title+" "+m["DESCRIPTION"], cfg)
	if cfg.PrivacyRedact {
		title = "[redacted]"
	}
	return store.Event{ExternalID: m["UID"], Start: st, End: en, Title: title, Description: m["DESCRIPTION"], Location: m["LOCATION"], AllDay: ad, Timezone: "local", Category: category, Source: src}
}

func taskFromMap(m map[string]string, src string, cfg config.Config) store.Event {
	var dsKey, deKey, ds, de string
	for k, v := range m {
		if strings.HasPrefix(k, "DTSTART") {
			dsKey, ds = k, v
		}
		if strings.HasPrefix(k, "DUE") {
			deKey, de = k, v
		}
	}
	start, ad := parseICSDateWithTZ(dsKey, ds)
	end, _ := parseICSDateWithTZ(deKey, de)
	if start.IsZero() && !end.IsZero() {
		start = end.Add(-30 * time.Minute)
	}
	if end.IsZero() && !start.IsZero() {
		end = start.Add(30 * time.Minute)
	}
	title := m["SUMMARY"]
	if title == "" {
		title = "Untitled task"
	}
	if cfg.PrivacyRedact {
		title = "[redacted]"
	}
	return store.Event{ExternalID: "todo-" + m["UID"], Start: start, End: end, Title: title, Description: m["DESCRIPTION"], Location: m["LOCATION"], AllDay: ad, Timezone: "local", Category: "task", Source: src}
}

func categoryFor(text string, cfg config.Config) string {
	lower := strings.ToLower(text)
	for k, c := range cfg.CalendarMap {
		if strings.Contains(lower, k) {
			return c
		}
	}
	return ""
}
