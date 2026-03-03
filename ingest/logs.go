package ingest

import (
	"bufio"
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trajectory/store"
)

func ParseLogs(path string) ([]store.LogEntry, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".csv" {
		return parseCSV(path)
	}
	return parseText(path)
}

func HashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:]), nil
}

func parseText(path string) ([]store.LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	var out []store.LogEntry
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if len(line) < 17 {
			continue
		}
		tsRaw := line[:16]
		t, err := time.ParseInLocation("2006-01-02 15:04", tsRaw, time.Local)
		if err != nil {
			continue
		}
		rest := strings.TrimSpace(line[16:])
		tokens, note := splitTokensAndNote(rest)
		if tokens == "" {
			continue
		}
		out = append(out, store.LogEntry{Timestamp: t, Tokens: tokens, Note: note, Source: path})
	}
	return out, s.Err()
}

func splitTokensAndNote(rest string) (string, string) {
	// Optional note syntax: [note text]
	if i := strings.LastIndex(rest, "["); i >= 0 && strings.HasSuffix(rest, "]") {
		note := strings.TrimSpace(rest[i+1 : len(rest)-1])
		tokens := strings.TrimSpace(rest[:i])
		return tokens, note
	}
	return rest, ""
}

func parseCSV(path string) ([]store.LogEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	recs, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []store.LogEntry
	for i, rec := range recs {
		if len(rec) < 2 {
			continue
		}
		if i == 0 && strings.Contains(strings.ToLower(rec[0]), "timestamp") {
			continue
		}
		t, err := parseFlexibleTimestamp(rec[0])
		if err != nil {
			continue
		}
		note := ""
		if len(rec) > 2 {
			note = rec[2]
		}
		out = append(out, store.LogEntry{Timestamp: t, Tokens: strings.TrimSpace(rec[1]), Note: strings.TrimSpace(note), Source: path})
	}
	return out, nil
}

func parseFlexibleTimestamp(v string) (time.Time, error) {
	layouts := []string{"2006-01-02 15:04", time.RFC3339, "2006-01-02T15:04"}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, strings.TrimSpace(v), time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp: %s", v)
}

func ParseAlloc(spec string) map[string]time.Duration {
	out := map[string]time.Duration{}
	for _, p := range strings.Split(spec, ",") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		d := parseDur(kv[1])
		out[strings.ToLower(kv[0])] = d
	}
	return out
}

func parseDur(v string) time.Duration {
	v = strings.TrimSpace(v)
	if strings.HasSuffix(v, "h") {
		d, _ := time.ParseDuration(strings.TrimSuffix(v, "h") + "h")
		return d
	}
	if strings.HasSuffix(v, "m") {
		d, _ := time.ParseDuration(strings.TrimSuffix(v, "m") + "m")
		return d
	}
	d, _ := time.ParseDuration(v)
	return d
}
