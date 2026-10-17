package world

import "github.com/comprofix/nightcrossing/internal/config"

// Toadlet is one of the line that escorts the hero in School Run (DESIGN
// §6.5, D53). On land it follows the hero's trail one move behind: when
// the one ahead of it moves, it moves into the cell that one has just
// left. It makes no moves of its own, and the wind and rain don't act on
// it directly, but every hazard kills it. On the river it rides on the
// hero's back.
type Toadlet struct {
	Player
	Riding bool // on the hero's back, crossing the river

	trail []spot // the cells the one ahead has left, for it to move into in turn
}

// moveKind is a kind of move, which the line copies as it is.
type moveKind int

const (
	moveHop moveKind = iota
	moveSlide
	movePush
)

// spot is the cell a move began from: its row and x, in SubPx units.
type spot struct {
	row, x int
	kind   moveKind // the move that left it
}

// newLine is the level's line of toadlets, all in the hero's start cell.
func newLine(n int) []Toadlet {
	if n == 0 {
		return nil
	}
	line := make([]Toadlet, n)
	for i := range line {
		line[i].Player = newPlayer(config.StartCol, config.StartRow)
	}
	return line
}

// passOn tells the toadlet behind body i that i has begun a move of kind
// from where it stood, landing at row, x. The toadlet will move into the
// cell i left. One sharing i's cell is pushed with it instead, at the same
// moment, so a bunched line stays bunched in the wind. Riders go with the
// hero, not along a trail (D53).
func (w *World) passOn(i int, from spot, row, x int) {
	j := i + 1
	if j >= len(w.Line) || w.Line[j].Riding {
		return
	}
	t := &w.Line[j]
	if from.kind == movePush && len(t.trail) == 0 && !t.Hopping() && t.Row == from.row && t.X == from.x {
		t.begin(row, x)
		t.Pushed = true
		w.passOn(j, from, row, x)
		return
	}
	t.trail = append(t.trail, from)
}

// follow starts toadlet i's next move along its trail, if it has one.
func (w *World) follow(i int) {
	t := &w.Line[i]
	for len(t.trail) > 0 {
		s := t.trail[0]
		t.trail = t.trail[1:]
		if w.moveTo(i, s) {
			return
		}
	}
}

// moveTo begins toadlet i's move into the cell at spot s, and reports
// whether it began: a toadlet already there stays put. A hop turns it to
// face its way; a slide or a push keeps its facing.
func (w *World) moveTo(i int, s spot) bool {
	p := &w.Line[i].Player
	if s.row == p.Row && s.x == p.X {
		return false
	}
	d := Right
	switch {
	case s.row < p.Row:
		d = Up
	case s.row > p.Row:
		d = Down
	case s.x < p.X:
		d = Left
	}
	p.begin(s.row, s.x)
	p.Sliding, p.Pushed = s.kind == moveSlide, s.kind == movePush
	if s.kind == moveHop {
		p.Facing = d
	}
	w.passOn(i, spot{row: p.FromRow, x: p.fromX, kind: s.kind}, p.Row, p.toX)
	return true
}

// gathered reports whether the whole line is in the hero's cell, landed:
// the hero may take it onto the river (D53).
func (w *World) gathered() bool {
	h := &w.Player
	for _, t := range w.Line {
		if t.Hopping() || t.Row != h.Row || t.X != h.X {
			return false
		}
	}
	return true
}

// mount puts the whole line on the hero's back as it hops onto the river
// (D53).
func (w *World) mount() {
	for i := range w.Line {
		w.Line[i].Riding = true
		w.Line[i].trail = nil
	}
}

// carry keeps the riders on the hero's back, and lets them off, bunched in
// its cell, once it has landed back on land (D53).
func (w *World) carry() {
	h := &w.Player
	off := !h.Hopping() && w.Lanes[h.Row].Kind != config.River
	for i := range w.Line {
		t := &w.Line[i]
		if !t.Riding {
			continue
		}
		t.Player = *h
		t.push, t.queued = 0, None
		t.Riding = !off
	}
}

// bunch gathers the line into the hero's cell once the hero has stood
// still on a safe row for config.BunchTicks: each toadlet joins it at once
// and safely (D53).
func (w *World) bunch() {
	p := &w.Player
	if p.Hopping() || p.stood < config.BunchTicks || w.Lanes[p.Row].Kind != config.Safe {
		return
	}
	for i := range w.Line {
		t := &w.Line[i]
		t.Player = Player{Facing: t.Facing, Still: config.LandTicks, stood: p.stood}
		t.Row, t.FromRow = p.Row, p.Row
		t.fromX, t.toX = p.X, p.X
		t.place()
		t.trail = nil
	}
}

// Fallen is the one whose death is under way, the hero or a toadlet, or
// nil when none is dying.
func (w *World) Fallen() *Player {
	if w.Dying == 0 {
		return nil
	}
	return w.body(w.fallen)
}

// cloneLine is a copy of line that shares nothing with it.
func cloneLine(line []Toadlet) []Toadlet {
	if line == nil {
		return nil
	}
	c := make([]Toadlet, len(line))
	for i, t := range line {
		c[i] = t
		c[i].trail = append([]spot(nil), t.trail...)
	}
	return c
}
