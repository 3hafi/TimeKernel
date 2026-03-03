package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trajectory/config"
	"trajectory/features"
	"trajectory/ingest"
	"trajectory/insights"
	"trajectory/report"
	"trajectory/simulate"
	"trajectory/store"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		usage()
		return nil
	}
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".trajectory")
	_ = os.MkdirAll(base, 0o755)
	dbPath := filepath.Join(base, "trajectory.db")
	cfgPath := filepath.Join(base, "config.json")

	switch args[1] {
	case "config":
		if len(args) > 2 && args[2] == "init" {
			return config.Save(cfgPath, config.Default())
		}
		return errors.New("usage: trajectory config init")
	case "import-ics":
		if len(args) < 3 {
			return errors.New("usage: trajectory import-ics <file.ics>")
		}
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		cfg := loadCfg(cfgPath)
		events, err := ingest.ParseICS(args[2], cfg)
		if err != nil {
			return err
		}
		for _, e := range events {
			if err := st.UpsertEvent(e); err != nil {
				return err
			}
		}
		h, _ := ingest.HashFile(args[2])
		_ = st.MarkImport(args[2], h)
		fmt.Printf("Imported %d ICS events\n", len(events))
		return nil
	case "import-logs":
		if len(args) < 3 {
			return errors.New("usage: trajectory import-logs <file.txt|file.csv>")
		}
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		logs, err := ingest.ParseLogs(args[2])
		if err != nil {
			return err
		}
		for _, l := range logs {
			if err := st.AddLog(l); err != nil {
				return err
			}
		}
		h, _ := ingest.HashFile(args[2])
		_ = st.MarkImport(args[2], h)
		fmt.Printf("Imported %d logs\n", len(logs))
		return nil
	case "log":
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		ts := time.Now()
		parts := args[2:]
		if len(parts) >= 2 && parts[0] == "-t" {
			t, err := time.Parse("2006-01-02 15:04", parts[1])
			if err != nil {
				return err
			}
			ts = t
			parts = parts[2:]
		}
		if len(parts) == 0 {
			return errors.New("usage: trajectory log [-t 'YYYY-MM-DD HH:MM'] '<tokens>'")
		}
		tokens := strings.Trim(strings.Join(parts, " "), "\"")
		if err := st.AddLog(store.LogEntry{Timestamp: ts, Tokens: tokens, Source: "manual"}); err != nil {
			return err
		}
		fmt.Println("logged", ts.Format(time.RFC3339), tokens)
		return nil
	case "report":
		return reportCmd(dbPath, cfgPath, args[2:])
	case "trajectory":
		return trajCmd(dbPath, cfgPath, args[2:])
	case "whatif":
		return whatifCmd(dbPath, cfgPath, args[2:])
	case "doctor":
		return doctorCmd(dbPath, cfgPath)
	default:
		usage()
		return nil
	}
}

func reportCmd(dbPath, cfgPath string, args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fromS := fs.String("from", time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "")
	toS := fs.String("to", time.Now().Format("2006-01-02"), "")
	format := fs.String("format", "md", "")
	out := fs.String("out", "report.md", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	from, _ := time.Parse("2006-01-02", *fromS)
	to, _ := time.Parse("2006-01-02", *toS)
	to = to.Add(23*time.Hour + 59*time.Minute)
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	cfg := loadCfg(cfgPath)
	e, err := st.EventsBetween(from, to)
	if err != nil {
		return err
	}
	l, err := st.LogsBetween(from, to)
	if err != nil {
		return err
	}
	days := features.BuildDaily(from, to, e, l, cfg)
	ins := insights.Generate(days)
	return report.Write(*out, report.BuildMarkdown(from, to, days, ins), *format)
}

func trajCmd(dbPath, cfgPath string, args []string) error {
	fs := flag.NewFlagSet("trajectory", flag.ContinueOnError)
	daysN := fs.Int("days", 14, "")
	out := fs.String("out", "trajectory.md", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	to := time.Now()
	from := to.AddDate(0, 0, -21)
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	cfg := loadCfg(cfgPath)
	e, err := st.EventsBetween(from, to)
	if err != nil {
		return err
	}
	l, err := st.LogsBetween(from, to)
	if err != nil {
		return err
	}
	d := features.BuildDaily(from, to, e, l, cfg)
	return report.Write(*out, report.BuildTrajectoryMarkdown(simulate.Trajectory(d, *daysN)), "md")
}

func whatifCmd(dbPath, cfgPath string, args []string) error {
	fs := flag.NewFlagSet("whatif", flag.ContinueOnError)
	date := fs.String("date", time.Now().AddDate(0, 0, 1).Format("2006-01-02"), "")
	allocS := fs.String("alloc", "study=2h,doomscroll=30m,gaming=1h", "")
	out := fs.String("out", "whatif.md", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	alloc := ingest.ParseAlloc(*allocS)
	dt, _ := time.Parse("2006-01-02", *date)
	dayFrom, dayTo := dt, dt.Add(23*time.Hour+59*time.Minute)
	histFrom := dt.AddDate(0, 0, -21)

	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	cfg := loadCfg(cfgPath)
	allEvents, err := st.EventsBetween(histFrom, dayTo)
	if err != nil {
		return err
	}
	logs, err := st.LogsBetween(histFrom, dayTo)
	if err != nil {
		return err
	}
	days := features.BuildDaily(histFrom, dayTo, allEvents, logs, cfg)
	dayEvents, _ := st.EventsBetween(dayFrom, dayTo)
	free := 24 * 60
	for _, ev := range dayEvents {
		free -= int(ev.End.Sub(ev.Start).Minutes())
	}
	variants := simulate.WhatIf(days[dt.Format("2006-01-02")], alloc, free)
	return report.Write(*out, report.BuildWhatIfMarkdown(*date, variants), "md")
}

func doctorCmd(dbPath, cfgPath string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	cfg := loadCfg(cfgPath)
	to := time.Now()
	from := to.AddDate(0, 0, -14)
	e, err := st.EventsBetween(from, to)
	if err != nil {
		return err
	}
	l, err := st.LogsBetween(from, to)
	if err != nil {
		return err
	}
	d := features.BuildDaily(from, to, e, l, cfg)
	missingSleep := 0
	for _, x := range d {
		if x.SleepMinutes == 0 {
			missingSleep++
		}
	}
	fmt.Printf("Data quality: %d days, %d without inferred sleep.\n", len(d), missingSleep)
	fmt.Println("Suggestions: log 😴 at bedtime, 🌅 at wake, and duration tokens e.g. 📱 doomscroll 45m")
	return nil
}

func usage() {
	fmt.Println("trajectory commands: config init | import-ics <file> | import-logs <file> | log [-t 'YYYY-MM-DD HH:MM'] '<tokens>' | report --from --to [--format md|html] | trajectory --days 14 | whatif --date --alloc | doctor")
}

func loadCfg(path string) config.Config {
	c, err := config.Load(path)
	if err != nil {
		c = config.Default()
		_ = config.Save(path, c)
	}
	return c
}
