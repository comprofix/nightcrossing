package config

import (
	"fmt"
	"slices"
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
//     each log loses a cell to a heron (D34), or two where herons strike
//     (D67); where water rats swim, each steady ship loses its front cell
//     to one, and a rat never swims for a log with a heron (D69).
//   - Stalking herons (D67): every log keeps a cell out of a heron's reach.
//   - The snake (D70): the verge keeps VergeFreeMin cells free of its
//     stretch.
//   - A hero on a sinking ship can reach a platform in the row above or
//     below before the ship's stern goes under (D33).
//   - Mixed traffic (D65): each gap behind a truck is at least
//     TruckGapMin, and no vehicle covers the hero's cell for longer than
//     CoverMaxTicks.
//   - Rail: the quiet between trains is long enough to hop on and off
//     (D17, D37).
//   - Currents (D46): each is weaker than its row's platforms; a hero
//     riding a current row moves at the platform's speed or, with the
//     current, faster (it can hop back against the slide, but not slow
//     the platform), and waits for the row above at the worse of the two;
//     and sliding one cell takes at least CrossOpenTicks, so a hero who
//     lands a cell in from a platform's end has time to hop.
//   - Wind (D47), in dry storms: each push can carry a riding hero a cell
//     nearer the edge, so the ride is shortened by a cell a beat; and
//     every river row keeps a platform at least two cells long that always
//     holds, so a push can be taken on it.
//   - Floods (D48), in wet storms: the river and its currents run
//     FloodSurgePct faster, so the ride is shorter; and a hero riding the
//     row above the verge, at its slowest, gets past the longest flooded
//     stretch to a dry cell before that row, at its fastest, carries it
//     off.
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
				if open := GapOpenTicks(s.Gap, l.Speed); open < CrossOpenTicks {
					errs = append(errs, fmt.Errorf("row %d: a gap of %dpx behind a %s is open for %d ticks, want %d", r, s.Gap, s.Entity.ID, open, CrossOpenTicks))
				}
				if s.Entity == Truck && s.Gap < TruckGapMin {
					errs = append(errs, fmt.Errorf("row %d: a gap of %dpx behind a truck, want %d", r, s.Gap, TruckGapMin))
				}
				if c := CoverTicks(s.Entity.Length, l.Speed); c > CoverMaxTicks {
					errs = append(errs, fmt.Errorf("row %d: a %s covers the hero's cell for %d ticks, want at most %d", r, s.Entity.ID, c, CoverMaxTicks))
				}
			}
		case Safe:
			if r == VergeRow && lv.Snake && Cols-SnakeStretch < VergeFreeMin {
				errs = append(errs, fmt.Errorf("row %d: %d verge cells free of the snake's stretch, want %d", r, Cols-SnakeStretch, VergeFreeMin))
			}
		case Rail:
			if lv.Trains && lv.Train.QuietMin < CrossOpenTicks {
				errs = append(errs, fmt.Errorf("row %d: %d quiet ticks between trains, want %d", r, lv.Train.QuietMin, CrossOpenTicks))
			}
		case River:
			if c := l.Current; c != 0 {
				if abs(c) >= l.Speed {
					errs = append(errs, fmt.Errorf("row %d: a current of %d is not weaker than the platforms' %d", r, c, l.Speed))
				}
				if t := Cell * sub / abs(c); t < CrossOpenTicks {
					errs = append(errs, fmt.Errorf("row %d: the current slides the hero a cell in %d ticks, want %d", r, t, CrossOpenTicks))
				}
			}
			if lv.Events[StormFront].Beat > 0 && !holdsAPush(l, bitesOf(lv)) {
				errs = append(errs, fmt.Errorf("row %d: no platform two cells long that always holds, to take the wind's push", r))
			}
			below := lv.Lanes[r+1]
			wait, ok := boardTicks(lv, r, below)
			if !ok {
				errs = append(errs, fmt.Errorf("row %d: no platform that always holds the hero", r))
				continue
			}
			if below.Kind == River {
				if ride := (fieldUnits - Cell*sub) / rideSpeed(lv, below); wait > ride {
					errs = append(errs, fmt.Errorf("row %d: up to %d ticks to wait for a platform, but row %d carries the hero off in %d", r, wait, r+1, ride))
				}
			}
			if below.Kind == Safe && lv.Events[StormFront].Size > 0 {
				slowest := l.Speed - max(-l.Current, 0)
				past := (FloodStretchMax + 1) * Cell * sub / slowest
				if ride := (fieldUnits - Cell*sub) / rideSpeed(lv, l); past > ride {
					errs = append(errs, fmt.Errorf("row %d: %d ticks to ride past a flood to a dry verge cell, but the row carries the hero off in %d", r, past, ride))
				}
			}
			for _, s := range l.Pattern {
				if lv.Stalk && slices.Contains(Logs(), s.Entity) && s.Entity.Length/Cell <= bitesOf(lv).log {
					errs = append(errs, fmt.Errorf("row %d: a %s has no cell out of a stalking heron's reach", r, s.Entity.ID))
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
					if w, ok := boardTicksFrom(a, l, bitesOf(lv)); ok && (best < 0 || w < best) {
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
	return boardTicksFrom(lv.Lanes[r], below, bitesOf(lv))
}

// boardTicksFrom is the longest a hero riding in lane from (or standing,
// if from is not a river) may wait for a platform in river lane to that
// will hold it, and whether one ever comes.
// On a row with a current the hero may move at its platform's speed or
// slide along with the current, so the slower of the two meetings counts.
func boardTicksFrom(to, from Lane, b bites) (int, bool) {
	water, ok := widestWater(to, b)
	if !ok {
		return 0, false
	}
	vels := []int{0} // standing on land
	if from.Kind == River {
		vels = []int{from.Dir * from.Speed, from.Dir * (from.Speed + from.Current)}
	}
	worst := 0
	for _, hv := range vels {
		v := abs(to.Dir*to.Speed - hv)
		if v == 0 {
			return 0, false
		}
		worst = max(worst, (water+Cell)*SubPx/v)
	}
	return worst, true
}

// rideSpeed is the fastest a hero riding river lane l can be carried
// toward the edge, in SubPx units a tick: its platforms, with the current
// if it runs that way, and then a dry storm's push each beat or a wet
// storm's flood surge, whichever is worse; the two never come together
// (D46–D48, D51).
func rideSpeed(lv Level, l Lane) int {
	v := l.Speed + max(l.Current, 0)
	worst := v
	if s := lv.Events[StormFront]; s.Beat > 0 {
		worst = max(worst, v+Cell*SubPx/s.Beat)
	}
	if lv.Events[StormFront].Size > 0 {
		worst = max(worst, v*(100+FloodSurgePct)/100)
	}
	return worst
}

// bites is how many cells of a platform that always holds a predator can
// make deadly at once: of a log, and of a ship that never sinks.
type bites struct{ log, ship int }

// bitesOf is the bites on level lv: a heron's cell on a log, and the one
// it jabs where herons strike (D34, D67); and where water rats swim, the
// front cell of a ship. A rat never swims for a log with a heron on it, so
// a log loses no more to one (D69).
func bitesOf(lv Level) bites {
	b := bites{log: 1}
	if lv.Strike {
		b.log = 2
	}
	if lv.Rats {
		b.ship = 1
	}
	return b
}

// holdsAPush reports whether river lane l has a platform at least two
// cells long that always holds the hero: not one that dives or sinks,
// still two cells with b's cells of it deadly (D34, D47, D67, D69).
func holdsAPush(l Lane, b bites) bool {
	for _, s := range l.Pattern {
		if !s.Dives && !s.Sinks && steadyLength(s, b) >= 2*Cell {
			return true
		}
	}
	return false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// steadyLength is how much of platform s always holds, in pixels, with
// b's cells of it deadly: none of a platform that dives or sinks.
func steadyLength(s Spawn, b bites) int {
	switch {
	case s.Dives || s.Sinks:
		return 0
	case s.Entity == LogShort || s.Entity == LogLong:
		return s.Entity.Length - b.log*Cell
	case s.Entity == Barge || s.Entity == Ferry:
		return s.Entity.Length - b.ship*Cell
	}
	return s.Entity.Length
}

// widestWater is the widest stretch, in pixels, along river lane l with no
// platform that will hold the hero, in the worst case: diving pads and
// sinking ships are under, each log has a heron on an end cell, and where
// water rats swim, each steady ship has one at its front (b). It reports
// false if no platform always holds.
func widestWater(l Lane, b bites) (int, bool) {
	n := len(l.Pattern)
	widest, run, any := 0, 0, false
	for i := range 2 * n { // twice round, so a stretch across the pattern's end counts
		s := l.Pattern[i%n]
		if h := steadyLength(s, b); h >= Cell {
			any = true
			widest = max(widest, run)
			run = s.Entity.Length - h + s.Gap
			continue
		}
		run += s.Entity.Length + s.Gap
	}
	return max(widest, run), any
}
