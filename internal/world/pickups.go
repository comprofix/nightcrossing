package world

import (
	"math"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Flyer is the firefly, or by day the dragonfly, on its visit to the river
// (DESIGN §6.6, D58). It flies in, lands on platforms and flits between
// them, and flies away when its visit is up. It can only be caught when it
// has landed.
type Flyer struct {
	Visiting bool // on the field: flying in, about, or away
	Day      bool // a dragonfly, come in by day
	X, Y     int  // its cell's top-left from the field's, in SubPx units, as the hero's
	Landed   bool // settled on a platform
	Leaving  bool // flying away, its visit over
	Facing   int  // the way it last flew: -1 left, +1 right

	row    int // the river row it is landed on or flying to
	host   int // the platform it is landed on or flying to, in that row's lane
	offset int // from the platform's left end to its cell, in SubPx units
	due    int // ticks until a firefly rolled for comes in; 0 for none
	visit  int // ticks left of its visit
	sit    int // ticks left landed
}

// rollFirefly decides, at the level's start and after each nook is
// filled, whether a firefly comes in this crossing: one roll, so dying
// can't fish for one (D58). One still about, or due, carries on instead.
// Levels without fireflies make no roll, so their random choices are as
// before.
func (w *World) rollFirefly() {
	f := &w.Flyer
	if f.Visiting || f.due > 0 || w.Level.FireflyPct <= 0 || w.rng.IntN(100) >= w.Level.FireflyPct {
		return
	}
	f.due = between(w, config.FireflyComeMinTicks, config.FireflyComeMaxTicks)
}

// between is a seeded number from lo to hi.
func between(w *World, lo, hi int) int { return lo + w.rng.IntN(hi-lo+1) }

// updateFirefly runs the firefly's visit: it comes in when it is due, sits
// out its time on each platform, lifts off early when its spot stops being
// safe, flies to its next spot, and flies away when its visit is up (D58).
// It goes on through deaths and nooks filled.
func (w *World) updateFirefly() {
	f := &w.Flyer
	if f.due > 0 {
		if f.due--; f.due == 0 {
			w.flyIn()
		}
		return
	}
	if !f.Visiting {
		return
	}
	if !f.Leaving {
		if f.visit--; f.visit <= 0 {
			w.flyAway()
		}
	}
	switch {
	case f.Leaving:
		out := -2 * cellUnits
		if f.Facing > 0 {
			out = fieldWidth + cellUnits
		}
		if f.flyTo(out, f.Y) {
			f.Visiting = false
			w.emit(FireflyGone)
		}
	case f.Landed:
		e := w.Lanes[f.row].Entities[f.host]
		f.X = e.X() + f.offset
		if f.sit--; f.sit <= 0 || !w.perch(f.row, e, f.offset, 1) {
			w.flit()
		}
	default:
		e := w.Lanes[f.row].Entities[f.host]
		if !w.perch(f.row, e, f.offset, 1) {
			w.flit()
			return
		}
		f.X += w.Lanes[f.row].Step() // it flies with its spot as it goes, so it always gets there
		if f.flyTo(e.X()+f.offset, f.row*cellUnits) {
			f.Landed, f.sit = true, between(w, config.FireflySitMinTicks, config.FireflySitMaxTicks)
		}
	}
}

// flyIn brings the firefly in from either side of the field, flying to a
// spot on the river; a dragonfly if it is day (D58).
func (w *World) flyIn() {
	f := &w.Flyer
	f.X, f.Facing = -cellUnits, 1
	if w.rng.IntN(2) == 0 {
		f.X, f.Facing = fieldWidth, -1
	}
	rows := w.riverRows()
	w.rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
	if !w.pickSpot(rows) {
		return // nowhere to land just now: it doesn't come
	}
	f.Visiting, f.Landed, f.Leaving = true, false, false
	f.Day = w.daytime()
	f.visit = between(w, config.FireflyVisitMinTicks, config.FireflyVisitMaxTicks)
	f.Y = f.row * cellUnits
}

// flit lifts the firefly off and sends it to a new spot: usually a row up
// or down, sometimes along its row. With nowhere to go, it flies away.
func (w *World) flit() {
	f := &w.Flyer
	f.Landed = false
	near := []int{f.row - 1, f.row + 1}
	if w.rng.IntN(2) == 0 {
		near[0], near[1] = near[1], near[0]
	}
	rows := append(near, f.row)
	if w.rng.IntN(100) < config.FireflySameRowPct {
		rows = append([]int{f.row}, near...)
	}
	if !w.pickSpot(rows) && !w.pickSpot(w.riverRows()) {
		w.flyAway()
	}
}

// flyAway ends the visit: the firefly flies off the field by the nearer
// side.
func (w *World) flyAway() {
	f := &w.Flyer
	f.Leaving, f.Landed = true, false
	f.Facing = 1
	if f.X+cellUnits/2 < fieldWidth/2 {
		f.Facing = -1
	}
}

// riverRows are the field's river rows, top to bottom.
func (w *World) riverRows() []int {
	var rows []int
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.River {
			rows = append(rows, r)
		}
	}
	return rows
}

// pickSpot chooses where the firefly flies to next: a seeded spot it can
// land on, well in from the field's edges and within its reach, in the
// first of rows that has one; or, if none is in reach, the nearest in
// rows. It reports whether it found one.
func (w *World) pickSpot(rows []int) bool {
	f := &w.Flyer
	type spot struct{ row, host, offset, dist int }
	var nearest *spot
	for _, r := range rows {
		if r < 0 || r >= len(w.Lanes) || w.Lanes[r].Kind != config.River {
			continue
		}
		var spots []spot
		for i, e := range w.Lanes[r].Entities {
			for c := range e.Kind.Length / config.Cell {
				if !w.perch(r, e, c*cellUnits, config.FireflyLandMargin) {
					continue
				}
				s := spot{r, i, c * cellUnits, abs(e.X() + c*cellUnits - f.X)}
				if s.dist <= config.FireflyReach*cellUnits {
					spots = append(spots, s)
				}
				if nearest == nil || s.dist < nearest.dist {
					nearest = &s
				}
			}
		}
		if len(spots) > 0 {
			nearest = &spots[w.rng.IntN(len(spots))]
			break
		}
	}
	if nearest == nil {
		return false
	}
	f.row, f.host, f.offset = nearest.row, nearest.host, nearest.offset
	return true
}

// perch reports whether the firefly can be on platform e of row r, at
// offset from its left end: a platform it rides on this level, afloat and
// not flashing (a ship may flash, until it breaks up), never a log with a
// heron, and at least margin cells in from either edge of the field, so
// it isn't carried off (D58, D60).
func (w *World) perch(r int, e Entity, offset, margin int) bool {
	if !w.Level.FireflyOn[config.HostOf(e.Kind)] || e.Heron {
		return false
	}
	if st := e.Stage(); st != Afloat && !(e.Sinks && st == Warning) {
		return false
	}
	x := e.X() + offset
	return x >= margin*cellUnits && x+cellUnits <= fieldWidth-margin*cellUnits
}

// flyTo moves the firefly toward x, y at its speed, and reports whether
// it is there.
func (f *Flyer) flyTo(x, y int) bool {
	dx, dy := x-f.X, y-f.Y
	if dx != 0 {
		f.Facing = 1
		if dx < 0 {
			f.Facing = -1
		}
	}
	d := int(math.Sqrt(float64(dx*dx + dy*dy)))
	if d <= config.FireflySpeed {
		f.X, f.Y = x, y
		return true
	}
	f.X += dx * config.FireflySpeed / d
	f.Y += dy * config.FireflySpeed / d
	return false
}

// daytime reports whether it is day on a level with the day and night
// cycle; every other level is night (D39).
func (w *World) daytime() bool {
	return w.Level.DayCycle && config.DayCycle[w.filled()] == config.Day
}

// filled is how many nooks are filled.
func (w *World) filled() int {
	n := 0
	for _, f := range w.Filled {
		if f {
			n++
		}
	}
	return n
}

// catchFirefly catches the firefly if it has landed on row r over
// left–right, in SubPx units, where the hero is down: a life at once, or
// points with lives at the cap (D57, D58). Its box is its cell, trimmed
// like a heron's.
func (w *World) catchFirefly(r, left, right int) {
	f := &w.Flyer
	m := config.HitMargin * config.SubPx
	if !f.Visiting || !f.Landed || f.row != r || f.X+m >= right || left >= f.X+cellUnits-m {
		return
	}
	f.Visiting, f.Landed = false, false
	w.Stats.Fireflies++
	w.emit(FireflyCaught)
	if w.Session.Lives >= config.MaxLives {
		w.score(config.PointsFireflyAtCap)
		w.Stats.AtCap += config.PointsFireflyAtCap
		w.Gained += config.PointsFireflyAtCap
		return
	}
	w.gainLives(1)
}

// newFruit starts the level's fruit, if it has any, waiting to show
// (D59).
func (w *World) newFruit() {
	w.FruitNook = -1
	if w.Level.FruitPts > 0 {
		w.fruitTicks = w.fruitWait()
	}
}

// fruitWait is how long fruit stays away before it shows again.
func (w *World) fruitWait() int {
	return config.FruitWaitMinTicks + w.rng.IntN(config.FruitWaitMaxTicks-config.FruitWaitMinTicks+1)
}

// updateFruit shows fruit in an empty nook when its wait is up, and takes
// it away when its time is (D59). The clock runs on through deaths and
// nooks filled.
func (w *World) updateFruit() {
	if w.Level.FruitPts <= 0 {
		return
	}
	if w.fruitTicks--; w.fruitTicks > 0 {
		return
	}
	if w.FruitNook >= 0 {
		w.FruitNook, w.fruitTicks = -1, w.fruitWait()
		return
	}
	var empty []int
	for n, f := range w.Filled {
		if !f {
			empty = append(empty, n)
		}
	}
	if len(empty) == 0 {
		return
	}
	w.FruitNook, w.fruitTicks = empty[w.rng.IntN(len(empty))], config.FruitShowTicks
}

// takeFruit takes the fruit in nook n, if it is there, and is its points,
// before the combo; then the fruit waits to show again (D59).
func (w *World) takeFruit(n int) int {
	if n != w.FruitNook {
		return 0
	}
	w.FruitNook, w.fruitTicks = -1, w.fruitWait()
	w.Stats.Fruit += w.Level.FruitPts
	w.emit(FruitTaken)
	return w.Level.FruitPts
}
