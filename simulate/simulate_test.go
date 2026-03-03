package simulate

import (
	"testing"
	"time"

	"trajectory/features"
)

func TestWhatIfFeasibility(t *testing.T) {
	alloc := map[string]time.Duration{"study": 4 * time.Hour, "doomscroll": 2 * time.Hour}
	v := WhatIf(&features.Daily{}, alloc, 240)
	if v[2].Feasible {
		t.Fatal("aggressive should be infeasible")
	}
}
