package config

import "testing"

// TestPredatorsRamp: no stalking herons or foxes in Stroll; in every later
// mode, both from PredatorsFrom, the strike from StrikeFrom, through the
// loops; the heron's chance rising from HeronPctFirst to HeronPctTop
// (DESIGN §6.8, D66–D68).
func TestPredatorsRamp(t *testing.T) {
	for _, mode := range Modes {
		for n := 1; n <= 30; n++ {
			lv := mode.Levels.Level(n)
			on := mode.ID != Stroll.ID && n >= PredatorsFrom
			if lv.Stalk != on || lv.Fox != on || lv.Strike != (on && n >= StrikeFrom) {
				t.Errorf("%s level %d: stalk %v, fox %v, strike %v", mode.Name, n, lv.Stalk, lv.Fox, lv.Strike)
			}
			if on && lv.HeronPct != StalkingHeronPct(n) {
				t.Errorf("%s level %d: heron chance %d%%, want %d%%", mode.Name, n, lv.HeronPct, StalkingHeronPct(n))
			}
		}
	}
	for n, want := range map[int]int{3: 15, 4: 19, 8: 35, 20: 35} {
		if got := StalkingHeronPct(n); got != want {
			t.Errorf("level %d: heron chance %d%%, want %d%%", n, got, want)
		}
	}
}

// TestPredatorHints: each later mode shows the stalking hint on level 3
// and the strike hint on level 6, and no longer Stroll's standing heron
// hint (D66).
func TestPredatorHints(t *testing.T) {
	stroll := Stroll.Levels.Level(StrikeFrom).Hint
	for _, mode := range Modes {
		for n := 1; n <= 12; n++ {
			lv := mode.Levels.Level(n)
			want := ""
			if mode.ID != Stroll.ID {
				want = map[int]string{PredatorsFrom: StalkHint, StrikeFrom: StrikeHint}[n]
				if lv.Hint == stroll {
					t.Errorf("%s level %d: Stroll's heron hint", mode.Name, n)
				}
			}
			if lv.PreyHint != want {
				t.Errorf("%s level %d: predator hint %q, want %q", mode.Name, n, lv.PreyHint, want)
			}
		}
	}
}
