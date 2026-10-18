package world

import (
	"github.com/comprofix/nightcrossing/internal/config"
)

// A bonus level (DESIGN §6.7, D61) is played by the same world, with these
// differences:
//   - The road's cars stand parked, and a hop into one is blocked.
//   - A slip, into the water or a car, costs time but no life: the toad
//     starts again at the start row, keeping what it has collected.
//   - The clock is the whole level's, and never stops. When it runs out,
//     the level ends.
//   - A nook pays its points and sends the toad back, but never fills.
//   - Fireflies, the Swarm, come to road gaps and logs, a share to each,
//     staying where they come until caught; each one caught
//     brings the next a little later, until all have come. Catching every
//     one ends the level early, and the seconds left pay a time bonus.
//   - Or a set of fruit lies about from the start, on the ground, logs or
//     the carriages of a goods yard (yard.go), with the same ending.
//   - Its points never give a life.

// Bonus reports whether w is a bonus level.
func (w *World) Bonus() bool { return w.Level.Bonus != nil }

// newBonus sets up the bonus level: where the toad can get to past the
// parked cars, and the first fireflies, landed where they will be caught.
func (w *World) newBonus() {
	w.reach = w.reachable()
	for _, r := range w.Level.Bonus.Yard {
		w.Yard = append(w.Yard, newRake(r))
	}
	w.layFruit()
	w.Swarm = make([]Flyer, w.Level.Bonus.AtOnce)
	for i := range w.Swarm {
		f := &w.Swarm[i]
		w.sent++ // whether it lands now or comes in as soon as there is room
		if !w.swarmSpot(i) {
			f.due = 1 // nowhere just now: it comes in as soon as there is
			continue
		}
		f.Visiting, f.Landed = true, true
		f.X, f.Y, f.Facing = w.spotX(f), f.row*cellUnits, 1
	}
}

// Parked reports whether a parked car takes cell col of row r.
func (w *World) Parked(col, r int) bool {
	b := w.Level.Bonus
	if b == nil || col < 0 || col >= len(b.Parked[r]) {
		return false
	}
	_, ok := config.ParkedCars[b.Parked[r][col]]
	return ok
}

// land reports whether row r is land: not the river or the far bank.
func (w *World) land(r int) bool {
	k := w.Lanes[r].Kind
	return k != config.River && k != config.Bank
}

// reachable is which land cells the toad can get to from where it starts,
// hopping round the parked cars.
func (w *World) reachable() [config.Rows][config.Cols]bool {
	var seen [config.Rows][config.Cols]bool
	type cell struct{ col, row int }
	queue := []cell{{config.StartCol, config.StartRow}}
	seen[config.StartRow][config.StartCol] = true
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range [...]cell{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			n := cell{c.col + d.col, c.row + d.row}
			if n.row < 0 || n.row >= w.Rows || n.col < 0 || n.col >= w.Cols || seen[n.row][n.col] ||
				!w.land(n.row) || w.Parked(n.col, n.row) {
				continue
			}
			seen[n.row][n.col] = true
			queue = append(queue, n)
		}
	}
	return seen
}

// Reachable reports whether the toad can get to land cell col of row r
// past the parked cars.
func (w *World) Reachable(col, r int) bool { return w.reach[r][col] }

// spotX is where firefly f's spot is across the field, in SubPx units:
// its cell on a log, which moves, or in a road gap, which doesn't.
func (w *World) spotX(f *Flyer) int {
	if f.host < 0 {
		return f.offset
	}
	return w.Lanes[f.row].Entities[f.host].X() + f.offset
}

// swarmSide chooses which side a firefly coming in goes to, the road or
// the river, as if drawn from a deck of those still to come to each: it
// can go to either, the sides stay mixed all through, and each gets its
// share of the set (D61).
func (w *World) swarmSide() (road bool) {
	b := w.Level.Bonus
	roadLeft, riverLeft := b.OnRoad-w.cameRoad, b.Fireflies-b.OnRoad-w.cameRiver
	return w.rng.IntN(max(roadLeft+riverLeft, 1)) < roadLeft
}

// bonusSpot chooses a spot for firefly i to land on the road or the
// river: a road gap the toad can reach, or a cell of a log well in from
// the field's edges. It never lands within config.BonusClearCells of the
// toad, so it is never a free catch, nor next to another firefly or where
// one is going, even corner to corner. It reports whether it found one.
func (w *World) bonusSpot(i int, road bool) bool {
	type spot struct{ row, host, offset int }
	p := &w.Player
	near := func(r, x int) bool {
		return abs(r-p.Row) <= config.BonusClearCells && abs(columnOf(x)-columnOf(p.X)) <= config.BonusClearCells
	}
	taken := func(r, x int) bool {
		for j := range w.Swarm {
			o := &w.Swarm[j]
			if j != i && o.Visiting && abs(o.row-r) <= 1 && abs(w.spotX(o)-x) < 2*cellUnits {
				return true
			}
		}
		return false
	}
	var spots []spot
	for r := range w.Lanes {
		switch k := w.Lanes[r].Kind; {
		case k == config.Road && road:
			for c := range w.Cols {
				if x := c * cellUnits; w.reach[r][c] && !near(r, x) && !taken(r, x) {
					spots = append(spots, spot{r, -1, x})
				}
			}
		case k == config.River && !road:
			for h, e := range w.Lanes[r].Entities {
				for c := range e.Kind.Length / config.Cell {
					off := c * cellUnits
					if x := e.X() + off; w.perch(r, e, off, config.FireflyLandMargin) && !near(r, x) && !taken(r, x) {
						spots = append(spots, spot{r, h, off})
					}
				}
			}
		}
	}
	if len(spots) == 0 {
		return false
	}
	s := spots[w.rng.IntN(len(spots))]
	f := &w.Swarm[i]
	f.row, f.host, f.offset = s.row, s.host, s.offset
	return true
}

// swarmSpot sends firefly i, coming in, to a spot on the side drawn for
// it, and counts it there. It reports whether it found one; if not, it
// tries again later, drawing afresh.
func (w *World) swarmSpot(i int) bool {
	road := w.swarmSide()
	if !w.bonusSpot(i, road) {
		return false
	}
	if road {
		w.cameRoad++
	} else {
		w.cameRiver++
	}
	return true
}

// updateSwarm runs the bonus level's fireflies: each one due comes in
// from a side, flies to its spot and settles there; one on a log rides it,
// and lifts off for another log as it nears the edge, so it never goes out
// of reach.
func (w *World) updateSwarm() {
	for i := range w.Swarm {
		f := &w.Swarm[i]
		if f.due > 0 {
			if f.due--; f.due == 0 {
				w.swarmIn(i)
			}
			continue
		}
		if !f.Visiting {
			continue
		}
		if f.host >= 0 && !w.perch(f.row, w.Lanes[f.row].Entities[f.host], f.offset, 1) {
			w.reland(i)
			continue
		}
		if f.Landed {
			f.X = w.spotX(f)
			continue
		}
		if f.host >= 0 {
			f.X += w.Lanes[f.row].Step() // it flies with its log, so it always gets there
		}
		if f.flyTo(w.spotX(f), f.row*cellUnits) {
			f.Landed = true
		}
	}
}

// swarmIn brings firefly i in from either side of the field, flying to a
// spot; with nowhere to go just now, it tries again next tick.
func (w *World) swarmIn(i int) {
	f := &w.Swarm[i]
	if !w.swarmSpot(i) {
		f.due = 1
		return
	}
	f.X, f.Facing = -cellUnits, 1
	if w.rng.IntN(2) == 0 {
		f.X, f.Facing = fieldWidth, -1
	}
	f.Y = f.row * cellUnits
	f.Visiting, f.Landed = true, false

}

// reland lifts firefly i off its log as it nears the edge and sends it to
// another log: once come to a side, it stays there until caught (D61).
// With none free, it keeps riding as it is.
func (w *World) reland(i int) {
	if w.bonusSpot(i, false) {
		w.Swarm[i].Landed = false
	}
}

// catchSwarm catches a firefly landed on row r over left–right, in SubPx
// units, where the toad is down: its points, and the next one due. The
// last one ends the level (D61). Its box is its cell, trimmed like a
// heron's.
func (w *World) catchSwarm(r, left, right int) {
	b := w.Level.Bonus
	m := config.HitMargin * config.SubPx
	for i := range w.Swarm {
		f := &w.Swarm[i]
		if !f.Visiting || !f.Landed || f.row != r || f.X+m >= right || left >= f.X+cellUnits-m {
			continue
		}
		f.Visiting, f.Landed = false, false
		w.Stats.Fireflies++
		w.score(b.Firefly)
		w.Gained += b.Firefly
		w.emit(FireflyCaught)
		if w.sent < b.Fireflies { // counted as it is sent for, so a quick run of catches never sends too many
			f.due = between(w, config.BonusComeMinTicks, config.BonusComeMaxTicks)
			w.sent++
		}
		w.checkSet()
		return
	}
}

// collect collects whatever of the set is on row r over left–right, in
// SubPx units, where the toad is down: a firefly landed there, and fruit.
func (w *World) collect(r, left, right int) {
	w.catchSwarm(r, left, right)
	if !w.Cleared {
		w.pickFruit(r, left, right)
	}
}

// Collected is how much of the bonus level's set has been collected:
// fireflies caught and fruit picked.
func (w *World) Collected() int { return w.Stats.Fireflies + w.Stats.Picked }

// checkSet ends the bonus level early once the whole set is collected
// (D61).
func (w *World) checkSet() {
	if !w.Cleared && w.Collected() == w.Level.Bonus.Size() {
		w.endBonus(true)
	}
}

// updateBonusClock runs the level's clock down, through slips and nooks:
// the low time warning, and the end when it runs out.
func (w *World) updateBonusClock() {
	w.TicksLeft--
	switch w.TicksLeft {
	case config.LowTimeSecs * config.TPS:
		w.emit(LowTime)
	case 0:
		w.endBonus(false)
	}
}

// endBonus ends the bonus level: early, with every firefly caught, or as
// its time runs out. All caught pays its points, and each whole second
// left pays the time bonus (D61).
func (w *World) endBonus(all bool) {
	w.Cleared = true
	if !all {
		w.emit(BonusOver)
		return
	}
	b := w.Level.Bonus
	secs := w.TicksLeft / config.TPS
	w.score(b.All + secs*config.PointsBonusSec)
	w.Stats.All = b.All
	w.Stats.TimeBonus = secs * config.PointsBonusSec
	w.emit(AllCaught)
}

// bonusNook is the toad reaching nook n on a bonus level: its points, and
// back to the start row once it has shown there; the nook never fills.
func (w *World) bonusNook() {
	w.score(config.PointsBonusNook)
	w.Stats.Goals++
	w.Home = config.GoalTicks
	w.emit(Goal)
}

// HomeNook is the nook the toad is showing in, between reaching it and
// starting again, or -1.
func (w *World) HomeNook() int {
	if w.Home == 0 {
		return -1
	}
	return NookAt(w.Player.Col)
}
