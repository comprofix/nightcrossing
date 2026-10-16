package world

import "github.com/comprofix/nightcrossing/internal/config"

// Dir is a direction to hop in.
type Dir int

// The directions. None is no press.
const (
	None Dir = iota
	Up
	Down
	Left
	Right
)

// step is the change in column and row for one hop in d.
func (d Dir) step() (dc, dr int) {
	switch d {
	case Up:
		return 0, -1
	case Down:
		return 0, 1
	case Left:
		return -1, 0
	case Right:
		return 1, 0
	}
	return 0, 0
}

func (d Dir) String() string {
	return [...]string{"none", "up", "down", "left", "right"}[d]
}

// Player is the hero. It moves one cell per hop (DESIGN §5.1). While
// hopping, Col and Row are the cell it is hopping to, and X and Y move
// from the old cell to the new one.
type Player struct {
	Col, Row         int // the cell the hero is in, or landing in
	FromCol, FromRow int // the cell the current hop started from
	X, Y             int // top-left corner from the field's, in config.SubPx units
	Facing           Dir
	Still            int   // ticks since the last landing, up to config.LandTicks
	Death            Event // how the hero died, or 0 while alive

	hopLeft int // ticks left in the current hop; 0 when not hopping
	queued  Dir // a press kept for the landing
}

func newPlayer(col, row int) Player {
	p := Player{Col: col, Row: row, FromCol: col, FromRow: row, Facing: Up, Still: config.LandTicks}
	p.place()
	return p
}

// Hopping reports whether the hero is between cells.
func (p *Player) Hopping() bool { return p.hopLeft > 0 }

// HopProgress is how many ticks into the current hop the hero is, from 1
// to config.HopTicks, or 0 when not hopping.
func (p *Player) HopProgress() int {
	if !p.Hopping() {
		return 0
	}
	return config.HopTicks - p.hopLeft
}

// LaneRow is the row whose hazards can hit the hero: the row it is
// leaving for the first half of a hop, and the row it is landing in from
// halfway on.
func (p *Player) LaneRow() int {
	if p.Hopping() && 2*p.HopProgress() < config.HopTicks {
		return p.FromRow
	}
	return p.Row
}

// place sets X and Y from the hop's progress.
func (p *Player) place() {
	const cell = config.Cell * config.SubPx
	done := config.HopTicks - p.hopLeft
	p.X = p.FromCol*cell + (p.Col-p.FromCol)*cell*done/config.HopTicks
	p.Y = p.FromRow*cell + (p.Row-p.FromRow)*cell*done/config.HopTicks
}

// updatePlayer starts, moves and lands hops. A press starts a hop at once
// when the hero is still. During a hop, a press in the last
// config.HopQueueTicks ticks is kept and carried out on landing; the
// latest such press wins. Hops that would leave the playfield are ignored.
func (w *World) updatePlayer(press Dir) {
	p := &w.Player
	if press != None {
		switch {
		case !p.Hopping():
			w.startHop(press)
		case p.hopLeft <= config.HopQueueTicks:
			p.queued = press
		}
	}
	if !p.Hopping() {
		p.Still = min(p.Still+1, config.LandTicks)
		return
	}
	p.hopLeft--
	p.place()
	if p.Hopping() {
		return
	}
	p.FromCol, p.FromRow = p.Col, p.Row
	p.Still = 0
	w.emit(Landed)
	if d := p.queued; d != None {
		p.queued = None
		w.startHop(d)
	}
}

// startHop begins a hop in d, unless it would leave the playfield.
func (w *World) startHop(d Dir) {
	p := &w.Player
	dc, dr := d.step()
	col, row := p.Col+dc, p.Row+dr
	if !w.inside(col, row) {
		return
	}
	p.FromCol, p.FromRow = p.Col, p.Row
	p.Col, p.Row = col, row
	p.Facing = d
	p.hopLeft = config.HopTicks
	w.emit(Hopped)
}
