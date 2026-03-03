package insights

import (
	"fmt"
	"sort"

	"trajectory/features"
)

type Insight struct {
	Title, Evidence, Confidence, Experiment string
	SampleN                                 int
}

func Generate(days map[string]*features.Daily) []Insight {
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var arr []*features.Daily
	for _, k := range keys {
		arr = append(arr, days[k])
	}
	ins := []Insight{}
	add := func(title, evidence string, n int) {
		ins = append(ins, Insight{Title: title, Evidence: evidence, SampleN: n, Confidence: conf(n), Experiment: exp(title)})
	}

	withPhone, noPhone := split(arr, func(d *features.Daily) bool { return d.DoomscrollMinutes > 90 }, func(d *features.Daily) int { return d.SleepMinutes })
	if len(withPhone) > 1 && len(noPhone) > 1 {
		add("High doomscroll nights reduce sleep", fmt.Sprintf("Median sleep %d vs %d min when doomscroll >90m (Δ %d).", features.Median(withPhone), features.Median(noPhone), features.Median(noPhone)-features.Median(withPhone)), len(withPhone)+len(noPhone))
	}
	withMeet, noMeet := split(arr, func(d *features.Daily) bool { return d.MeetingMinutes > 240 }, func(d *features.Daily) int { return d.FocusWindows90 })
	if len(withMeet) > 1 && len(noMeet) > 1 {
		add("Heavy meeting days cut focus windows", fmt.Sprintf("Median focus windows %d vs %d when meetings >4h.", features.Median(withMeet), features.Median(noMeet)), len(withMeet)+len(noMeet))
	}
	withGym, noGym := split(arr, func(d *features.Daily) bool { return d.ExerciseMinutes >= 30 }, func(d *features.Daily) int { return d.BedtimeShiftMin })
	if len(withGym) > 1 && len(noGym) > 1 {
		add("Exercise correlates with earlier sleep", fmt.Sprintf("Bedtime shift %d vs %d min from 23:00 baseline.", features.Median(withGym), features.Median(noGym)), len(withGym)+len(noGym))
	}
	withShower, noShower := split(arr, func(d *features.Daily) bool { return d.HygieneCount > 0 }, func(d *features.Daily) int { return d.StressLogs })
	if len(withShower) > 1 && len(noShower) > 1 {
		add("Hygiene consistency tracks lower stress signals", fmt.Sprintf("Stress logs median %d vs %d on shower vs no-shower days.", features.Median(withShower), features.Median(noShower)), len(withShower)+len(noShower))
	}
	withGame, noGame := split(arr, func(d *features.Daily) bool { return d.GamingMinutes > 60 }, func(d *features.Daily) int { return d.SleepMinutes })
	if len(withGame) > 1 && len(noGame) > 1 {
		add("Long gaming sessions trade off sleep", fmt.Sprintf("Sleep median %d vs %d min on gaming>60m days.", features.Median(withGame), features.Median(noGame)), len(withGame)+len(noGame))
	}
	withB2b, noB2b := split(arr, func(d *features.Daily) bool { return d.BackToBack >= 3 }, func(d *features.Daily) int { return d.StressLogs })
	if len(withB2b) > 1 && len(noB2b) > 1 {
		add("Back-to-back calendar compression increases load", fmt.Sprintf("Stress logs median %d vs %d with >=3 back-to-back transitions.", features.Median(withB2b), features.Median(noB2b)), len(withB2b)+len(noB2b))
	}
	withShortGap, noShortGap := split(arr, func(d *features.Daily) bool { return d.ShortGaps >= 2 }, func(d *features.Daily) int { return d.DeepWorkLogs })
	if len(withShortGap) > 1 && len(noShortGap) > 1 {
		add("Fragmented gaps suppress deep work", fmt.Sprintf("Deep-work logs median %d vs %d when short gaps >=2.", features.Median(withShortGap), features.Median(noShortGap)), len(withShortGap)+len(noShortGap))
	}
	lateSleep, normalSleep := split(arr, func(d *features.Daily) bool { return d.BedtimeShiftMin > 60 }, func(d *features.Daily) int { return d.DeepWorkLogs })
	if len(lateSleep) > 1 && len(normalSleep) > 1 {
		add("Late bedtime harms next-day focus", fmt.Sprintf("Deep-work logs median %d vs %d when bedtime >00:00.", features.Median(lateSleep), features.Median(normalSleep)), len(lateSleep)+len(normalSleep))
	}
	stressDays, calmDays := split(arr, func(d *features.Daily) bool { return d.StressLogs > 0 }, func(d *features.Daily) int { return d.DoomscrollMinutes })
	if len(stressDays) > 1 && len(calmDays) > 1 {
		add("Stress and doomscroll reinforce", fmt.Sprintf("Doomscroll median %d vs %d min on stress-signal days.", features.Median(stressDays), features.Median(calmDays)), len(stressDays)+len(calmDays))
	}
	lowSleep, okSleep := split(arr, func(d *features.Daily) bool { return d.SleepMinutes > 0 && d.SleepMinutes < 420 }, func(d *features.Daily) int { return d.MeetingMinutes })
	if len(lowSleep) > 1 && len(okSleep) > 1 {
		add("Short sleep precedes heavier meeting burden", fmt.Sprintf("Meeting load median %d vs %d min after <7h sleep nights.", features.Median(lowSleep), features.Median(okSleep)), len(lowSleep)+len(okSleep))
	}

	if len(ins) > 10 {
		ins = ins[:10]
	}
	return ins
}

func split(arr []*features.Daily, pred func(*features.Daily) bool, val func(*features.Daily) int) ([]int, []int) {
	a, b := []int{}, []int{}
	for _, d := range arr {
		if pred(d) {
			a = append(a, val(d))
		} else {
			b = append(b, val(d))
		}
	}
	return a, b
}

func conf(n int) string {
	if n >= 12 {
		return "high"
	}
	if n >= 6 {
		return "med"
	}
	return "low"
}
func exp(title string) string {
	return "3-day experiment: " + title + ". Track sleep/focus/stress deltas."
}
