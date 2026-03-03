package store

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Store struct{ Path string }

type Event struct {
	ExternalID, Title, Description, Location, Timezone, Category, Source string
	Start, End                                                           time.Time
	AllDay                                                               bool
}

type LogEntry struct {
	Timestamp            time.Time
	Tokens, Note, Source string
}

func Open(path string) (*Store, error) {
	s := &Store{Path: path}
	return s, s.Migrate()
}

func (s *Store) exec(sql string) error {
	cmd := exec.Command("sqlite3", s.Path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite3 exec: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Store) query(sql string) (string, error) {
	cmd := exec.Command("sqlite3", "-csv", s.Path, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sqlite3 query: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func esc(v string) string { return strings.ReplaceAll(v, "'", "''") }

func (s *Store) Migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS events(
 id INTEGER PRIMARY KEY,
 external_id TEXT UNIQUE,
 start_ts TEXT NOT NULL,
 end_ts TEXT NOT NULL,
 title TEXT, description TEXT, location TEXT,
 all_day INTEGER DEFAULT 0,
 timezone TEXT,
 category TEXT,
 source TEXT,
 created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS logs(
 id INTEGER PRIMARY KEY,
 ts TEXT NOT NULL,
 tokens TEXT NOT NULL,
 note TEXT,
 source TEXT,
 unique_key TEXT UNIQUE,
 created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS imports(
 id INTEGER PRIMARY KEY,
 source TEXT,
 source_hash TEXT UNIQUE,
 imported_at TEXT DEFAULT CURRENT_TIMESTAMP
);`
	return s.exec(schema)
}

func hash(vals ...string) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%v", vals)))
	return hex.EncodeToString(h[:])
}

func (s *Store) MarkImport(source, sourceHash string) error {
	q := fmt.Sprintf(`INSERT OR IGNORE INTO imports(source,source_hash) VALUES('%s','%s');`, esc(source), esc(sourceHash))
	return s.exec(q)
}

func (s *Store) UpsertEvent(e Event) error {
	if e.ExternalID == "" {
		e.ExternalID = hash(e.Start.String(), e.End.String(), e.Title, e.Source)
	}
	q := fmt.Sprintf(`INSERT OR IGNORE INTO events(external_id,start_ts,end_ts,title,description,location,all_day,timezone,category,source)
VALUES('%s','%s','%s','%s','%s','%s',%d,'%s','%s','%s');`,
		esc(e.ExternalID), e.Start.Format(time.RFC3339), e.End.Format(time.RFC3339), esc(e.Title), esc(e.Description), esc(e.Location), btoi(e.AllDay), esc(e.Timezone), esc(e.Category), esc(e.Source))
	return s.exec(q)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) AddLog(l LogEntry) error {
	uk := hash(l.Timestamp.Format(time.RFC3339), l.Tokens, l.Note, l.Source)
	q := fmt.Sprintf(`INSERT OR IGNORE INTO logs(ts,tokens,note,source,unique_key) VALUES('%s','%s','%s','%s','%s');`, l.Timestamp.Format(time.RFC3339), esc(l.Tokens), esc(l.Note), esc(l.Source), uk)
	return s.exec(q)
}

func (s *Store) EventsBetween(from, to time.Time) ([]Event, error) {
	q := fmt.Sprintf(`SELECT external_id,start_ts,end_ts,title,description,location,all_day,timezone,category,source FROM events WHERE start_ts < '%s' AND end_ts > '%s' ORDER BY start_ts;`, to.Format(time.RFC3339), from.Format(time.RFC3339))
	res, err := s.query(q)
	if err != nil {
		return nil, err
	}
	rows, err := parseCSV(res)
	if err != nil {
		return nil, err
	}
	out := []Event{}
	for _, p := range rows {
		if len(p) < 10 {
			continue
		}
		st, _ := time.Parse(time.RFC3339, p[1])
		en, _ := time.Parse(time.RFC3339, p[2])
		out = append(out, Event{ExternalID: p[0], Start: st, End: en, Title: p[3], Description: p[4], Location: p[5], AllDay: p[6] == "1", Timezone: p[7], Category: p[8], Source: p[9]})
	}
	return out, nil
}

func (s *Store) LogsBetween(from, to time.Time) ([]LogEntry, error) {
	q := fmt.Sprintf(`SELECT ts,tokens,note,source FROM logs WHERE ts >= '%s' AND ts <= '%s' ORDER BY ts;`, from.Format(time.RFC3339), to.Format(time.RFC3339))
	res, err := s.query(q)
	if err != nil {
		return nil, err
	}
	rows, err := parseCSV(res)
	if err != nil {
		return nil, err
	}
	out := []LogEntry{}
	for _, p := range rows {
		if len(p) < 4 {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, p[0])
		out = append(out, LogEntry{Timestamp: ts, Tokens: p[1], Note: p[2], Source: p[3]})
	}
	return out, nil
}

func parseCSV(raw string) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(strings.TrimSpace(raw)))
	recs, err := r.ReadAll()
	if err != nil {
		if strings.TrimSpace(raw) == "" {
			return nil, nil
		}
		return nil, err
	}
	return recs, nil
}
