package world

import (
	"slices"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Rat is a water rat in a river row (D69): a ripple where it comes up,
// then it swims against the flow for the front end of a platform, grabs
// the end, and dives. Only its head and paws on the end are deadly.
type Rat struct {
	Phase  RatPhase
	Row    int // its river row
	Target int // the platform it swims for, in its lane's Entities
	ticks  int // ticks left of its phase
}

// RatPhase is where a water rat is in its visit.
type RatPhase int

// The phases.
const (
	RatAway   RatPhase = iota
	RatRipple          // a ripple where it comes up: the warning, harmless
	RatSwim            // swimming for its platform, harmless in the water
	RatGrab            // grabbing its platform's end: its head and paws there are deadly
)

// Snake is the snake on the verge (D70): the grass rustles where it will
// come out, then it slides back and forth along its stretch, all of it
// deadly, and slips away.
type Snake struct {
	Phase SnakePhase
	Col   int // the stretch's left column
	Dir   int // the way it sets off: +1 from the stretch's left end, -1 from its right
	T     int // ticks it has been out
	ticks int // ticks left of its phase
}

// SnakePhase is where the snake is in its visit.
type SnakePhase int

// The phases.
const (
	SnakeAway   SnakePhase = iota
	SnakeRustle            // the grass rustles where it will come out: the warning, harmless
	SnakeOut               // out on the verge: all of it deadly
)

// newProwl starts the level's water rats and snake, if it has them, away.
func (w *World) newProwl() {
	w.Rats, w.Snake = [len(w.Rats)]Rat{}, Snake{}
	if w.Level.Rats {
		w.prowlIn = between(w, w.Level.ProwlMin, w.Level.ProwlMax)
	}
}

// updateProwl moves the water rats and the snake one tick on, and after
// each quiet sends the next: the snake, now and then where it can come,
// or a water rat. One is out at a time, or two on levels with pairs,
// never two snakes (D66). Their clocks run on through deaths and nooks
// filled.
func (w *World) updateProwl() {
	if !w.Level.Rats {
		return
	}
	for i := range w.Rats {
		w.updateRat(&w.Rats[i])
	}
	w.updateSnake()
	if w.prowlIn--; w.prowlIn > 0 {
		return
	}
	w.prowlIn = config.TPS // look again in a while, unless one comes
	most := 1
	if w.Level.Pair {
		most = 2
	}
	if w.Tick < config.LevelCalmTicks || w.prowling() >= most {
		return
	}
	snake := w.Level.Snake && w.Snake.Phase == SnakeAway && !w.floody() && w.Lanes[config.VergeRow].Kind == config.Safe
	switch {
	case snake && w.rng.IntN(100) < config.SnakePct:
		w.sendSnake()
	case w.sendRat():
	case snake:
		w.sendSnake()
	default:
		return
	}
	w.prowlIn = between(w, w.Level.ProwlMin, w.Level.ProwlMax)
}

// prowling is how many water rats and snakes are out, or coming.
func (w *World) prowling() int {
	n := 0
	for _, r := range w.Rats {
		if r.Phase != RatAway {
			n++
		}
	}
	if w.Snake.Phase != SnakeAway {
		n++
	}
	return n
}

// sendRat sends a water rat for a platform it can swim for, in a river row
// with no rat in it, if there is one, and reports whether it did.
func (w *World) sendRat() bool {
	slot := -1
	for i, r := range w.Rats {
		if r.Phase == RatAway {
			slot = i
			break
		}
	}
	if slot < 0 {
		return false
	}
	type aim struct{ row, target int }
	var aims []aim
	for row := range w.Lanes {
		if w.Lanes[row].Kind != config.River || w.ratInRow(row) {
			continue
		}
		for i := range w.Lanes[row].Entities {
			if w.ratCanAim(row, i) {
				aims = append(aims, aim{row, i})
			}
		}
	}
	if len(aims) == 0 {
		return false
	}
	a := aims[w.rng.IntN(len(aims))]
	w.Rats[slot] = Rat{Phase: RatRipple, Row: a.row, Target: a.target, ticks: config.RippleTicks}
	w.emit(Ripple)
	return true
}

// ratInRow reports whether a water rat is out in row.
func (w *World) ratInRow(row int) bool {
	for _, r := range w.Rats {
		if r.Phase != RatAway && r.Row == row {
			return true
		}
	}
	return false
}

// ratCanAim reports whether a water rat can swim for platform i of river
// row (D69): a log with no heron on it or a ship that never sinks, at
// least config.RatTargetCells long; open water off its front end for the
// rat to come up in; the rat on the field from its ripple to its dive;
// and another platform on the field in the row, so the target is never
// the only one within reach.
func (w *World) ratCanAim(row, i int) bool {
	l := &w.Lanes[row]
	e := l.Entities[i]
	ship := slices.Contains(config.Ships(), e.Kind)
	if !e.log && !ship || e.Dives || e.Sinks || e.Heron || e.Kind.Length < config.RatTargetCells*config.Cell {
		return false
	}
	front := e.U // its end the flow carries it towards, along the loop
	if l.Dir > 0 {
		front += e.Kind.Length * config.SubPx
	}
	other := false
	for j, o := range l.Entities {
		if j == i {
			continue
		}
		gap := front - o.U - o.Kind.Length*config.SubPx // to o's back end
		if l.Dir > 0 {
			gap = o.U - front
		}
		if ((gap%l.Loop)+l.Loop)%l.Loop < (config.RatStartCells+1)*cellUnits {
			return false
		}
		if st := o.Stage(); !o.off() && st != Under && st != Sunk {
			other = true
		}
	}
	x := front - Buffer // on the field
	far := x + l.Dir*(config.RatStartCells+1)*cellUnits
	moved := x + l.Step()*(config.RatSwimTicks+config.RatGrabTicks)
	return other && inField(x) && inField(far) && inField(moved+l.Dir*cellUnits)
}

// inField reports whether x, in SubPx units, is on the field.
func inField(x int) bool { return x >= 0 && x <= fieldWidth }

// RatNose is where water rat r's nose is, in SubPx units from the field's
// left edge, and the way it faces: against its row's flow, towards its
// platform's front end, which it closes on smoothly as it swims, and
// touches while it grabs the end (D69).
func (w *World) RatNose(r Rat) (x, facing int) {
	l := &w.Lanes[r.Row]
	e := l.Entities[r.Target]
	front := e.X()
	if l.Dir > 0 {
		front += e.Kind.Length * config.SubPx
	}
	dist := config.RatStartCells * cellUnits
	switch r.Phase {
	case RatSwim:
		dist = dist * r.ticks / config.RatSwimTicks
	case RatGrab:
		dist = 0
	}
	return front + l.Dir*dist, -l.Dir
}

// updateRat moves water rat r one tick on: its ripple, then its swim, its
// grab of its platform's end, and its dive.
func (w *World) updateRat(r *Rat) {
	if r.Phase == RatAway {
		return
	}
	if r.ticks--; r.ticks > 0 {
		return
	}
	switch r.Phase {
	case RatRipple:
		r.Phase, r.ticks = RatSwim, config.RatSwimTicks
	case RatSwim:
		r.Phase, r.ticks = RatGrab, config.RatGrabTicks
		w.emit(RatGrabs)
	case RatGrab:
		*r = Rat{}
		w.emit(RatDives)
	}
}

// RatHead is the deadly part of water rat r as it grabs its platform's
// end, left to right in SubPx units from the field's left edge: its head
// and paws, config.RatOverPx onto the platform (D69).
func (w *World) RatHead(r Rat) (left, right int) {
	nose, facing := w.RatNose(r)
	over := config.RatOverPx * config.SubPx
	if facing < 0 {
		return nose - over, nose
	}
	return nose, nose + over
}

// ratBites reports whether a water rat grabbing its platform's end in row
// overlaps left–right, in SubPx units, with its head and paws (D69).
func (w *World) ratBites(row, left, right int) bool {
	for _, r := range w.Rats {
		if r.Phase != RatGrab || r.Row != row {
			continue
		}
		if hl, hr := w.RatHead(r); hl < right && left < hr {
			return true
		}
	}
	return false
}

// floody reports whether a storm with a flood is warned or on, when no
// snake may come out (D70).
func (w *World) floody() bool {
	s := w.Spell(config.StormFront)
	return s != nil && s.Phase != SpellQuiet && s.Floods()
}

// sendSnake sends the snake to a stretch of the verge: the grass rustles
// where it will come out, at one end of it.
func (w *World) sendSnake() {
	dir := 1
	if w.rng.IntN(2) == 0 {
		dir = -1
	}
	col := w.rng.IntN(config.Cols - config.SnakeStretch + 1)
	w.Snake = Snake{Phase: SnakeRustle, Col: col, Dir: dir, ticks: config.RustleTicks}
	w.emit(GrassRustles)
}

// updateSnake moves the snake one tick on: the rustle, then out for its
// time, turning at the ends of its stretch, then away.
func (w *World) updateSnake() {
	s := &w.Snake
	if s.Phase == SnakeAway {
		return
	}
	if s.Phase == SnakeOut {
		h := s.Heading()
		if s.T++; s.Heading() != h {
			w.emit(SnakeTurns)
		}
	}
	if s.ticks--; s.ticks > 0 {
		return
	}
	switch s.Phase {
	case SnakeRustle:
		s.Phase, s.ticks = SnakeOut, between(w, config.SnakeOutMinTicks, config.SnakeOutMaxTicks)
		w.emit(SnakeSlides)
	case SnakeOut:
		*s = Snake{}
		w.emit(SnakeGone)
	}
}

// X is where the snake's left end is, in SubPx units from the field's
// left edge, and Heading the way it slides: smoothly back and forth along
// its stretch, a cell every config.SnakeCellTicks (D70).
func (s Snake) X() int {
	off, _ := s.slide()
	return s.Col*cellUnits + off
}

// Heading is the way the snake is sliding: +1 right, -1 left.
func (s Snake) Heading() int {
	_, h := s.slide()
	return h
}

// slide is how far the snake is along its stretch from the left end, in
// SubPx units, and the way it is going.
func (s Snake) slide() (off, heading int) {
	run := (config.SnakeStretch - config.SnakeCells) * cellUnits
	d := s.T * cellUnits / config.SnakeCellTicks % (2 * run)
	off, heading = d, 1
	if d >= run {
		off, heading = 2*run-d, -1
	}
	if s.Dir < 0 {
		off, heading = run-off, -heading
	}
	return off, heading
}

// snakeBites reports whether the snake, out on the verge, overlaps
// left–right, in SubPx units, trimmed by config.HitMargin at each end
// (D70).
func (w *World) snakeBites(left, right int) bool {
	if w.Snake.Phase != SnakeOut {
		return false
	}
	m := config.HitMargin * config.SubPx
	x := w.Snake.X()
	return x+m < right && left < x+config.SnakeCells*cellUnits-m
}
