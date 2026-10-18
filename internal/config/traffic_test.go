package config

import (
	"slices"
	"testing"
)

// mixedLanes counts the road lanes of level lv that carry more than one
// kind of vehicle.
func mixedLanes(lv Level) int {
	n := 0
	for _, l := range lv.Lanes {
		if l.Kind != Road {
			continue
		}
		var kinds []string
		for _, s := range l.Pattern {
			if id := s.Entity.ID; !slices.Contains(kinds, id) {
				kinds = append(kinds, id)
			}
		}
		if len(kinds) > 1 && !onlyCars(kinds) {
			n++
		}
	}
	return n
}

// onlyCars reports whether kinds are all cars, which differ only in
// colour: Stroll's car lane is one kind of vehicle.
func onlyCars(kinds []string) bool {
	for _, k := range kinds {
		if k != CarRed.ID && k != CarBlue.ID && k != CarYellow.ID {
			return false
		}
	}
	return true
}

// TestMixedTrafficRamp: no mixed lanes in Stroll; in every later mode,
// none on levels 1–3, one on levels 4–5, two on 6–7 and all four from 8,
// through the loops (DESIGN §5.8, D65).
func TestMixedTrafficRamp(t *testing.T) {
	want := []int{0, 0, 0, 1, 1, 2, 2, 4}
	for _, mode := range Modes {
		for n := 1; n <= 30; n++ {
			got := mixedLanes(mode.Levels.Level(n))
			w := 4
			if n <= len(want) {
				w = want[n-1]
			}
			if mode.ID == Stroll.ID {
				w = 0
			}
			if got != w {
				t.Errorf("%s level %d: %d mixed lanes, want %d", mode.Name, n, got, w)
			}
		}
	}
}

// TestMixedTrafficKeepsTheRiver: the mixed roads change only the road;
// every other row of a later mode's level is Stroll's.
func TestMixedTrafficKeepsTheRiver(t *testing.T) {
	for n := 1; n <= 12; n++ {
		s, lt := Stroll.Levels.Level(n), LastTrain.Levels.Level(n)
		for r := range Rows {
			if s.Lanes[r].Kind == Road {
				if s.Lanes[r].Speed != lt.Lanes[r].Speed || s.Lanes[r].Dir != lt.Lanes[r].Dir {
					t.Errorf("level %d row %d: a mixed lane at speed %d dir %d, want Stroll's %d dir %d",
						n, r, lt.Lanes[r].Speed, lt.Lanes[r].Dir, s.Lanes[r].Speed, s.Lanes[r].Dir)
				}
				continue
			}
			if !slices.Equal(s.Lanes[r].Pattern, lt.Lanes[r].Pattern) {
				t.Errorf("level %d row %d differs from Stroll's", n, r)
			}
		}
	}
}

// TestMixedLanes: every mixed lane carries cars, bikes and vans, and big
// trucks run only on row 8 (D65).
func TestMixedLanes(t *testing.T) {
	for r, l := range mixedRoad3 {
		if l.Kind != Road {
			continue
		}
		has := map[string]bool{}
		for _, s := range l.Pattern {
			has[s.Entity.ID] = true
		}
		if !has[Bike.ID] || !has[Van.ID] || !(has[CarRed.ID] || has[CarBlue.ID] || has[CarYellow.ID]) {
			t.Errorf("row %d: want cars, bikes and vans, got %v", r, has)
		}
		if has[Truck.ID] != (r == 8) {
			t.Errorf("row %d: trucks %v, want them on row 8 only", r, has[Truck.ID])
		}
	}
}

// TestMixedTrafficHint: each later mode's level 4 shows the mixed traffic
// hint (D65).
func TestMixedTrafficHint(t *testing.T) {
	for _, mode := range Modes {
		if mode.ID == Stroll.ID {
			continue
		}
		if h := mode.Levels.Level(MixedFrom).Hint; h != MixedHint {
			t.Errorf("%s level %d: hint %q, want %q", mode.Name, MixedFrom, h, MixedHint)
		}
	}
}

// TestCoverTicks: the longest cover is a truck in Stroll's level 1 heavy
// lane, and a bike covers a cell for less time than a truck (D65).
func TestCoverTicks(t *testing.T) {
	if got, want := CoverMaxTicks, ((72-8+24-6)*SubPx+9)/10; got != want {
		t.Errorf("CoverMaxTicks = %d, want %d", got, want)
	}
	if CoverTicks(Bike.Length, 28) >= CoverTicks(Truck.Length, 28) {
		t.Error("a bike covers a cell as long as a truck")
	}
}
