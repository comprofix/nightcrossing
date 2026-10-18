package config

import (
	"strings"
	"testing"
)

// TestBonusLayouts checks each mode's bonus level: parked cars only on
// road rows with no traffic, each row as wide as the field, each car's
// letter repeated for exactly the cells it takes, and the start cell clear
// (D61).
func TestBonusLayouts(t *testing.T) {
	for _, m := range Modes {
		b := m.Bonus
		if b == nil {
			continue
		}
		if b.Fireflies < b.AtOnce || b.AtOnce <= 0 || b.Secs <= 0 || b.OnRoad < 0 || b.OnRoad > b.Fireflies {
			t.Errorf("%s: %d fireflies, %d on the road, %d at once, %d s", m.ID, b.Fireflies, b.OnRoad, b.AtOnce, b.Secs)
		}
		for r, s := range b.Parked {
			if s == "" {
				continue
			}
			lane := b.Layout.Lanes[r]
			if lane.Kind != Road || len(lane.Pattern) > 0 {
				t.Errorf("%s: parked cars on row %d, which isn't a road without traffic", m.ID, r)
			}
			if len(s) != Cols {
				t.Errorf("%s: row %d is %d cells, want %d", m.ID, r, len(s), Cols)
			}
			for c := 0; c < len(s); {
				if s[c] == '.' {
					c++
					continue
				}
				e, ok := ParkedCars[s[c]]
				if !ok {
					t.Fatalf("%s: row %d, column %d: %q is no car", m.ID, r, c, s[c])
				}
				n := ParkedCells(e)
				if c+n > len(s) || s[c:c+n] != strings.Repeat(s[c:c+1], n) {
					t.Errorf("%s: row %d, column %d: a %s takes %d cells", m.ID, r, c, e.ID, n)
				}
				c += n
			}
		}
		if s := b.Parked[StartRow-1]; s != "" && s[StartCol] != '.' {
			t.Errorf("%s: a parked car right above the start", m.ID)
		}
		lv, ok := BonusLevel(m)
		if !ok || lv.Bonus != b || lv.TimerTicks != b.Secs*TPS || lv.FireflyPct != 0 || lv.FruitPts != 0 {
			t.Errorf("%s: bonus level %+v", m.ID, lv)
		}
		if lv.FireflyOn[HostPads] || !lv.FireflyOn[HostLogs] {
			t.Errorf("%s: fireflies on %v, want logs and never lily pads", m.ID, lv.FireflyOn)
		}
	}
	if Stroll.Bonus != FireflyNight {
		t.Error("Stroll's bonus level isn't firefly night")
	}
}
