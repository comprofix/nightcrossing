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

// Player is the hero, or a toadlet in its line (D53). It moves one cell
// per hop (DESIGN §5.1). While
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
	Sliding  bool  // the move under way is a rain slide, not a hop (D31)
	Pushed   bool  // the move under way is the wind's push, not a hop (D47)

	fromX, toX int // where the current hop began and lands, in SubPx units
	hopLeft    int // ticks left in the current hop; 0 when not hopping
	queued     Dir // a press kept for the landing
	push       int // the wind's push waiting for the hero to land: -1 left, +1 right; 0 for none (D47)
	stood      int // ticks it has stood still since it last landed
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

// Hero is the body index of the hero; the toadlets in its line are 0 and
// up (D53).
const Hero = -1

// body is the hero, for Hero, or toadlet i.
func (w *World) body(i int) *Player {
	if i == Hero {
		return &w.Player
	}
	return &w.Line[i].Player
}

// out reports whether body i is out on the field by itself: the attempt
// is on, and it is the hero or a toadlet not riding on the hero's back.
func (w *World) out(i int) bool {
	return w.playing() && (i == Hero || !w.Line[i].Riding)
}

// updatePlayer starts, moves and lands body i's moves. For the hero, a
// press starts a hop at once when it is still; during a hop, a press in the
// last config.HopQueueTicks ticks is kept and carried out on landing, and
// the latest such press wins. Hops that would leave the playfield are
// ignored. A toadlet presses nothing: it follows the trail, each move as
// soon as it is still (D53).
func (w *World) updatePlayer(i int, press Dir) {
	p := w.body(i)
	switch {
	case i != Hero:
		if !p.Hopping() {
			w.follow(i)
		}
	case p.push != 0 && !p.Hopping():
		w.pushHero() // a press this tick comes mid-push, as mid-hop (D47)
	}
	if i == Hero && press != None {
		switch {
		case !p.Hopping():
			w.startHop(press)
		case p.hopLeft <= config.HopQueueTicks:
			p.queued = press
		}
	}
	if !p.Hopping() {
		p.Still = min(p.Still+1, config.LandTicks)
		p.stood++
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
	slid, pushed := p.Sliding, p.Pushed
	if i == Hero {
		w.emit(Landed)
		w.landed()
	}
	p.Sliding, p.Pushed = false, false
	// Where it landed counts before a queued move can carry it on, so a
	// quick press never skips a splash or the bank.
	w.checkHazards(i)
	if i != Hero {
		if w.out(i) {
			w.follow(i)
		}
		return
	}
	// In the rain a hop that lands on the road slides one more cell the
	// same way; a slide never slides again, and the wind's push never slides
	// (D31, D47).
	if !slid && !pushed && w.out(i) && w.Raining() && w.Lanes[p.Row].Kind == config.Road {
		if w.move(p.Facing, true) {
			return
		}
	}
	// A push from the wind that came mid-hop waits for the landing, and goes
	// before a queued press (D47).
	if p.push != 0 && w.out(i) && w.pushHero() {
		return
	}
	if d := p.queued; d != None && w.out(i) {
		p.queued = None
		w.startHop(d)
	}
}

// startHop begins a hop in d, unless it would leave the playfield: below
// the start row, past the far bank, or with the hero's middle off either
// side. A hop onto any row but the river lands on the nearest column.
func (w *World) startHop(d Dir) { w.move(d, false) }

// move begins a hop in d, or a rain slide if slide is true, unless it
// would leave the playfield, and reports whether it began. A slide takes
// a hop's time but is not a hop: it keeps the facing and the combo, and
// gives out Slid rather than Hopped (D31). Either is passed down the line
// (D53).
func (w *World) move(d Dir, slide bool) bool {
	p := &w.Player
	dc, dr := d.step()
	row := p.Row + dr
	if row < 0 || row >= w.Rows {
		return false
	}
	x := p.X
	if w.Lanes[row].Kind != config.River {
		x = columnOf(x) * cellUnits
	}
	x += dc * cellUnits
	if m := x + cellUnits/2; m < 0 || m >= fieldWidth {
		return false
	}
	// The verge is a checkpoint: the hero takes its line onto the river on
	// its back, once it is all in its cell (D53).
	onto := w.Lanes[row].Kind == config.River && w.Lanes[p.Row].Kind != config.River
	if onto && !w.gathered() {
		w.emit(Refused)
		return false
	}
	p.begin(row, x)
	if onto {
		w.mount()
	}
	p.Sliding = slide
	if slide {
		w.passOn(Hero, spot{row: p.FromRow, x: p.fromX, kind: moveSlide}, p.Row, p.toX)
		w.emit(Slid)
		return true
	}
	p.Facing = d
	w.passOn(Hero, spot{row: p.FromRow, x: p.fromX, kind: moveHop}, p.Row, p.toX)
	w.emit(Hopped)
	if d == Down {
		w.Combo = Combo{} // a hop back ends the combo (D38)
	}
	return true
}

// begin starts a move to row at x, taking a hop's time.
func (p *Player) begin(row, x int) {
	p.FromRow, p.Row = p.Row, row
	p.fromX, p.toX = p.X, x
	p.hopLeft = config.HopTicks
	p.stood = 0
	p.place()
}

// pushHero carries out the wind's waiting push, and reports whether it
// began (D47). It moves the hero exactly one cell sideways in a hop's
// time, but is not a hop: it keeps the facing and the combo. On land it
// keeps to the grid and doesn't happen at the field's edge; on a platform
// it drifts along it and stops at the edge. The line takes it as it
// reaches the spot, or at once if it shares the hero's cell (D53).
func (w *World) pushHero() bool {
	p := &w.Player
	d := p.push
	p.push = 0
	var x int
	if w.Lanes[p.Row].Kind == config.River {
		x = min(max(p.X+d*cellUnits, -cellUnits/2), fieldWidth-1-cellUnits/2)
		if x == p.X {
			return false
		}
	} else {
		x = (columnOf(p.X) + d) * cellUnits
		if m := x + cellUnits/2; m < 0 || m >= fieldWidth {
			return false
		}
	}
	p.begin(p.Row, x)
	p.Pushed = true
	w.passOn(Hero, spot{row: p.FromRow, x: p.fromX, kind: movePush}, p.Row, p.toX)
	w.emit(Pushed)
	return true
}

// ride carries the hero with the river row it counts as being in
// (DESIGN §5.2). A hop that lands on the river is carried along with it;
// one that lands off the river still lands on its column. Landed on a row
// with a current, or being pushed along it by the wind, the hero also
// slides along its platform; mid-hop the current does nothing (D46, D47).
// Toadlets are never on the river by themselves: they ride the hero
// (D53).
func (w *World) ride() {
	p := &w.Player
	lane := &w.Lanes[p.LaneRow()]
	if lane.Kind != config.River {
		return
	}
	step := lane.Step()
	if !p.Hopping() || p.Pushed {
		step += lane.Slide()
	}
	p.fromX += step
	if !p.Hopping() || w.Lanes[p.Row].Kind == config.River {
		p.toX += step
	}
	p.place()
}
