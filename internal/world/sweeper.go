package world

import "github.com/comprofix/nightcrossing/internal/config"

// Sweeper is one of a bonus level's two street sweepers (DESIGN §6.7). It
// crawls along the road lanes one at a time, turning off the field at
// each end into the next lane, the other way. Both wait
// config.SweeperHoldTicks off the field, then one works down from the top
// lane and the other up from the bottom lane. They cross in the middle,
// going the same way when they share a lane, so they never meet. Each is
// gone after its last lane. They sweep up any fruit in their path, and
// touching one is a slip.
type Sweeper struct {
	On  bool // still sweeping
	Row int
	X   int // its left end from the field's left edge, in SubPx units
	Dir int // the way it is going: +1 right, -1 left

	up   bool // working up from the bottom lane, rather than down from the top
	lane int  // how many lanes it has finished
	wait int  // ticks still to wait off the field before it sets off
}

// sweeperUnits is a sweeper's length in SubPx units.
const sweeperUnits = config.SweeperCells * cellUnits

// sweeperLaneTicks is how long a sweeper takes over one lane, from coming
// onto the field to coming onto the next.
const sweeperLaneTicks = (fieldWidth + sweeperUnits + config.SweeperSpeed - 1) / config.SweeperSpeed

// roadRows are the field's road rows, bottom first.
func (w *World) roadRows() []int {
	var rows []int
	for r := w.Rows - 1; r >= 0; r-- {
		if w.Lanes[r].Kind == config.Road {
			rows = append(rows, r)
		}
	}
	return rows
}

// laneRow is the row of the k'th lane sweeper s sweeps.
func (w *World) laneRow(s *Sweeper, k int) int {
	rows := w.roadRows()
	if s.up {
		return rows[k]
	}
	return rows[len(rows)-1-k]
}

// enter brings s onto the start of its next lane, from the side Dir sends
// it in from.
func (s *Sweeper) enter(row int) {
	s.Row, s.X = row, -sweeperUnits
	if s.Dir < 0 {
		s.X = fieldWidth
	}
}

// newSweepers starts the sweepers, if the level has them, waiting off the
// field: the first by the top lane on a random side, the second by the
// bottom lane, set to go the first one's way in each lane they share.
func (w *World) newSweepers() {
	n := len(w.roadRows())
	if !w.Level.Bonus.Sweeper || n == 0 {
		return
	}
	dir := 1
	if w.rng.IntN(2) == 0 {
		dir = -1
	}
	dir2 := dir
	if n%2 == 0 { // lane k from the top is lane n-1-k from the bottom: an odd number of turns apart
		dir2 = -dir
	}
	w.Sweepers[0] = Sweeper{On: true, Dir: dir, wait: config.SweeperHoldTicks}
	w.Sweepers[1] = Sweeper{On: true, Dir: dir2, up: true, wait: config.SweeperHoldTicks}
	for i := range w.Sweepers {
		s := &w.Sweepers[i]
		s.enter(w.laneRow(s, 0))
	}
}

// updateSweepers moves the sweepers on a tick, each into its next lane as
// it leaves the field, and sweeps up the fruit in their path.
func (w *World) updateSweepers() {
	for i := range w.Sweepers {
		s := &w.Sweepers[i]
		if !s.On {
			continue
		}
		if s.wait > 0 {
			s.wait--
			continue
		}
		s.X += s.Dir * config.SweeperSpeed
		if s.Dir > 0 && s.X >= fieldWidth || s.Dir < 0 && s.X+sweeperUnits <= 0 {
			if s.lane++; s.lane == len(w.roadRows()) {
				s.On = false
				continue
			}
			s.Dir = -s.Dir
			s.enter(w.laneRow(s, s.lane))
		}
		for j := range w.Picks {
			p := &w.Picks[j]
			if x := w.PickX(j); p.Row == s.Row && !p.Taken && !p.Gone && x < s.X+sweeperUnits && s.X < x+cellUnits {
				p.Gone = true
				w.emit(FruitSwept)
				w.checkLeft()
			}
		}
	}
}

// sweeperHits reports whether a sweeper is on row r over left–right, in
// SubPx units, trimmed at each end like a vehicle.
func (w *World) sweeperHits(r, left, right int) bool {
	m := config.HitMargin * config.SubPx
	for i := range w.Sweepers {
		if s := &w.Sweepers[i]; s.On && s.Row == r && s.X+m < right && left < s.X+sweeperUnits-m {
			return true
		}
	}
	return false
}

// sweptAt is the tick, from the start, at which the first sweeper to come
// reaches the cell at x on row r, or -1 if none ever does. Worked out from
// where the sweepers start, so it is only good before they set off.
func (w *World) sweptAt(r, x int) int {
	first := -1
	for i := range w.Sweepers {
		s := w.Sweepers[i]
		if !s.On {
			continue
		}
		for k := range len(w.roadRows()) {
			if w.laneRow(&s, k) != r {
				continue
			}
			dist := x // going right, from coming on at the left edge
			if s.Dir*(1-2*(k%2)) < 0 {
				dist = fieldWidth - x - cellUnits
			}
			if t := s.wait + k*sweeperLaneTicks + dist/config.SweeperSpeed; first < 0 || t < first {
				first = t
			}
		}
	}
	return first
}

// beaten reports whether a sweeper would sweep up fruit at x on row r
// before the toad could get there, hopping straight to it from the start
// at a fair pace: no fruit is laid there, so every fruit can be had
// (DESIGN §6.7).
func (w *World) beaten(r, x int) bool {
	t := w.sweptAt(r, x)
	hops := abs(r-config.StartRow) + abs(columnOf(x)-config.StartCol)
	return t >= 0 && t < config.SweeperFairTicks+hops*config.SweeperFairHopTicks
}

// checkLeft ends the bonus level once nothing is left to collect, every
// fruit collected or swept up; the whole set collected ends it from
// checkSet (DESIGN §6.7).
func (w *World) checkLeft() {
	if w.Cleared {
		return
	}
	for i := range w.Picks {
		if p := &w.Picks[i]; !p.Taken && !p.Gone {
			return
		}
	}
	w.endBonus(false)
}
