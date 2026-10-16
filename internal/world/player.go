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

// cellUnits is one cell in SubPx units.
const cellUnits = config.Cell * config.SubPx

// Player is the hero. It moves one cell per hop (DESIGN §5.1). While
// hopping, Row is the row it is hopping to, and X and Y move from where
// the hop began to where it lands. On the road and verge the hero keeps to
// the grid; a platform carries it off the grid, and it comes back to the
// nearest column when it hops off the river.
type Player struct {
	Col, Row int // the cell the hero is in, or landing in; Col is the column under its middle
	FromRow  int // the row the current hop started from
	X, Y     int // top-left corner from the field's, in config.SubPx units
	Facing   Dir
	Still    int   // ticks since the last landing, up to config.LandTicks
	Death    Event // how the hero died, or 0 while alive

	fromX, toX int // where the current hop began and lands, in SubPx units
	hopLeft    int // ticks left in the current hop; 0 when not hopping
	queued     Dir // a press kept for the landing
}

func newPlayer(col, row int) Player {
	p := Player{Col: col, Row: row, FromRow: row, Facing: Up, Still: config.LandTicks}
	p.fromX, p.toX = col*cellUnits, col*cellUnits
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

// Middle is the hero's middle across the field, in SubPx units.
func (p *Player) Middle() int { return p.X + cellUnits/2 }

// place sets X, Y and Col from the hop's progress.
func (p *Player) place() {
	done := config.HopTicks - p.hopLeft
	p.X = p.fromX + (p.toX-p.fromX)*done/config.HopTicks
	p.Y = p.FromRow*cellUnits + (p.Row-p.FromRow)*cellUnits*done/config.HopTicks
	p.Col = columnOf(p.toX)
}

// columnOf is the column under the middle of a hero at x, kept on the
// field.
func columnOf(x int) int {
	return min(max(floorDiv(x+cellUnits/2, cellUnits), 0), config.Cols-1)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
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
	p.FromRow = p.Row
	p.fromX = p.toX
	p.Still = 0
	w.emit(Landed)
	w.landed()
	// Where it landed counts before a queued hop can carry it on, so a
	// quick press never skips a splash or the bank.
	w.checkHazards()
	if d := p.queued; d != None && w.playing() {
		p.queued = None
		w.startHop(d)
	}
}

// startHop begins a hop in d, unless it would leave the playfield: below
// the start row, past the far bank, or with the hero's middle off either
// side. A hop onto any row but the river lands on the nearest column.
func (w *World) startHop(d Dir) {
	p := &w.Player
	dc, dr := d.step()
	row := p.Row + dr
	if row < 0 || row >= w.Rows {
		return
	}
	x := p.X
	if w.Lanes[row].Kind != config.River {
		x = columnOf(x) * cellUnits
	}
	x += dc * cellUnits
	if m := x + cellUnits/2; m < 0 || m >= fieldWidth {
		return
	}
	p.FromRow, p.Row = p.Row, row
	p.fromX, p.toX = p.X, x
	p.Facing = d
	p.hopLeft = config.HopTicks
	p.place()
	w.emit(Hopped)
}

// ride carries the hero with the river row it counts as being in
// (DESIGN §5.2). A hop that lands on the river is carried along with it;
// one that lands off the river still lands on its column.
func (w *World) ride() {
	p := &w.Player
	lane := &w.Lanes[p.LaneRow()]
	if lane.Kind != config.River {
		return
	}
	step := lane.Step()
	p.fromX += step
	if !p.Hopping() || w.Lanes[p.Row].Kind == config.River {
		p.toX += step
	}
	p.place()
}
