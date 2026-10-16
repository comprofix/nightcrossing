package config

import (
	"fmt"
	"testing"
)

// The "can it be crossed" check (DESIGN §5.7, D21). For every level of
// every mode, through the loops to the speed ceiling, it works out from
// the level data alone, with no simulation:
//
//   - Road: each gap stays open over the hero's cell for CrossOpenTicks.
//   - River: from the row below, a platform that will hold the hero comes
//     within reach before the row below carries it off the field. This is
//     the worst case: diving pads and sinking ships count as water, and
//     each log loses a cell to a heron (D34).
//   - A hero on a sinking ship can reach a platform in the row above or
//     below before the ship's stern goes under (D33).
//   - Rail: the quiet between trains is long enough to hop on and off
//     (D17, D37).
//
// A failure names the mode, the level and the row.
func TestEveryLevelCanBeCrossed(t *testing.T) {
	for _, mode := range append(append([]Mode(nil), Modes...), ModesInProgress...) {
		name, m := mode.Name, mode.Levels
		last := len(m.Levels) + 1
		for ; ; last++ {
			if _, s := m.Def(last); s >= m.CeilingPct {
				break
			}
		}
		last += LoopLevels // one whole round at the ceiling
		for n := 1; n <= last; n++ {
			for _, err := range crossingProblems(m.Level(n)) {
				t.Errorf("%s level %d: %v", name, n, err)
			}
		}
	}
}

// crossingProblems lists what stops level lv from being crossed.
func crossingProblems(lv Level) []error {
	var errs []error
	const sub = SubPx
	for r, l := range lv.Lanes {
		switch l.Kind {
		case Road:
			for _, s := range l.Pattern {
				free := (s.Gap + 2*HitMargin) - (Cell - 2*HeroMargin)
				if open := free * sub / l.Speed; open < CrossOpenTicks {
					errs = append(errs, fmt.Errorf("row %d: a gap of %dpx behind a %s is open for %d ticks, want %d", r, s.Gap, s.Entity.ID, open, CrossOpenTicks))
				}
			}
		case Rail:
			if lv.Trains && lv.Train.QuietMin < CrossOpenTicks {
				errs = append(errs, fmt.Errorf("row %d: %d quiet ticks between trains, want %d", r, lv.Train.QuietMin, CrossOpenTicks))
			}
		case River:
			below := lv.Lanes[r+1]
			wait, ok := boardTicks(lv, r, below)
			if !ok {
				errs = append(errs, fmt.Errorf("row %d: no platform that always holds the hero", r))
				continue
			}
			if below.Kind == River {
				if ride := (fieldUnits - Cell*sub) / below.Speed; wait > ride {
					errs = append(errs, fmt.Errorf("row %d: up to %d ticks to wait for a platform, but row %d carries the hero off in %d", r, wait, r+1, ride))
				}
			}
			for _, s := range l.Pattern {
				if !s.Sinks {
					continue
				}
				// Run back to the stern, then hop off before it goes.
				left := SinkTicks(s.Entity) - (Pieces(s.Entity)-1)*HopTicks - HopTicks
				best := -1
				for _, adj := range []int{r - 1, r + 1} {
					a := lv.Lanes[adj]
					if a.Kind != River {
						best = 0 // a safe row, or the bank's nooks, is right there
						break
					}
					if w, ok := boardTicksFrom(a, l); ok && (best < 0 || w < best) {
						best = w
					}
				}
				if best < 0 || best > left {
					errs = append(errs, fmt.Errorf("row %d: a hero on a sinking %s may wait %d ticks for a way off, but has %d", r, s.Entity.ID, best, left))
				}
			}
		}
	}
	return errs
}

const fieldUnits = Cols * Cell * SubPx

// boardTicks is the longest a hero in the row below row r may wait for a
// platform in row r to come under it, and whether one ever does.
func boardTicks(lv Level, r int, below Lane) (int, bool) {
	return boardTicksFrom(lv.Lanes[r], below)
}

// boardTicksFrom is the longest a hero riding in lane from (or standing,
// if from is not a river) may wait for a platform in river lane to that
// will hold it, and whether one ever comes.
func boardTicksFrom(to, from Lane) (int, bool) {
	v := to.Dir * to.Speed
	if from.Kind == River {
		v -= from.Dir * from.Speed
	}
	if v < 0 {
		v = -v
	}
	water, ok := widestWater(to)
	if !ok || v == 0 {
		return 0, false
	}
	return (water + Cell) * SubPx / v, true
}

// widestWater is the widest stretch, in pixels, along river lane l with no
// platform that will hold the hero, in the worst case: diving pads and
// sinking ships are under, and each log has a heron on an end cell. It
// reports false if no platform always holds.
func widestWater(l Lane) (int, bool) {
	n := len(l.Pattern)
	steady := func(s Spawn) int { // how much of it always holds
		switch {
		case s.Dives || s.Sinks:
			return 0
		case s.Entity == LogShort || s.Entity == LogLong:
			return s.Entity.Length - Cell
		}
		return s.Entity.Length
	}
	widest, run, any := 0, 0, false
	for i := range 2 * n { // twice round, so a stretch across the pattern's end counts
		s := l.Pattern[i%n]
		if h := steady(s); h >= Cell {
			any = true
			widest = max(widest, run)
			run = s.Entity.Length - h + s.Gap
			continue
		}
		run += s.Entity.Length + s.Gap
	}
	return max(widest, run), any
}
