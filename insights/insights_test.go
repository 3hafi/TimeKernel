package insights

import (
	"fmt"
	"testing"
	"trajectory/features"
)

func TestGenerate(t *testing.T) {
	d := map[string]*features.Daily{}
	for i := 1; i <= 14; i++ {
		k := fmt.Sprintf("2026-01-%02d", i)
		d[k] = &features.Daily{Date: k, SleepMinutes: 420, MeetingMinutes: 200, FocusWindows90: 2, DoomscrollMinutes: 30, ExerciseMinutes: 10, HygieneCount: 1}
		if i%2 == 0 {
			d[k].DoomscrollMinutes = 120
			d[k].SleepMinutes = 360
			d[k].MeetingMinutes = 300
		}
	}
	ins := Generate(d)
	if len(ins) == 0 {
		t.Fatal("want insights")
	}
}
