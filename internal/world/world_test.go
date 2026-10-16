package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

const cellUnits = config.Cell * config.SubPx

// newWorld is a game at the second skill on an all-safe level, so only
// the hero moves.
func newWorld(seed uint64) *World {
	return New(session.New(config.Skills[1], config.StartLives), config.Level{}, rand.New(rand.NewPCG(seed, seed)))
}

// run updates w once with press, then n-1 more times with nothing pressed,
// and returns every event in order.
func run(w *World, press Dir, n int) []Event {
	var all []Event
	for i := range n {
		d := None
		if i == 0 {
			d = press
		}
		all = append(all, w.Update(d)...)
	}
	return all
}

func at(t *testing.T, w *World, col, row int) {
	t.Helper()
	p := w.Player
	if p.Col != col || p.Row != row {
		t.Fatalf("hero in cell %d,%d, want %d,%d", p.Col, p.Row, col, row)
	}
	if p.Hopping() {
		t.Fatalf("hero is still hopping")
	}
	if p.X != col*cellUnits || p.Y != row*cellUnits {
		t.Fatalf("hero at %d,%d units, want the corner of cell %d,%d", p.X, p.Y, col, row)
	}
}

func TestNew(t *testing.T) {
	sk := config.Skills[1]
	w := newWorld(1)
	if w.Cols != 26 || w.Rows != 13 {
		t.Errorf("grid %d×%d, want 26×13", w.Cols, w.Rows)
	}
	if w.Skill.ID != sk.ID {
		t.Errorf("skill %q, want %q", w.Skill.ID, sk.ID)
	}
	if want := sk.TimerSecs * config.TPS; w.TicksLeft != want {
		t.Errorf("%d ticks on the clock, want %d", w.TicksLeft, want)
	}
	at(t, w, config.StartCol, config.StartRow)
	if w.Player.Facing != Up {
		t.Errorf("hero faces %v, want up", w.Player.Facing)
	}
}

func TestHopTakesHopTicksAndMovesSmoothly(t *testing.T) {
	w := newWorld(1)
	startY := w.Player.Y
	if ev := w.Update(Up); !slices.Equal(ev, []Event{Hopped}) {
		t.Fatalf("pressing up gave %v, want a hop", ev)
	}
	if w.Player.Row != config.StartRow-1 || w.Player.Facing != Up {
		t.Fatalf("hopping to row %d facing %v", w.Player.Row, w.Player.Facing)
	}
	prev := startY
	for i := 2; i <= config.HopTicks; i++ {
		if w.Player.Y >= prev {
			t.Fatalf("tick %d: Y %d did not move up from %d", i-1, w.Player.Y, prev)
		}
		prev = w.Player.Y
		ev := w.Update(None)
		if i < config.HopTicks && len(ev) != 0 {
			t.Fatalf("tick %d: events %v mid-hop", i, ev)
		}
		if i == config.HopTicks && !slices.Equal(ev, []Event{Landed}) {
			t.Fatalf("tick %d: events %v, want a landing", i, ev)
		}
	}
	at(t, w, config.StartCol, config.StartRow-1)
	if w.Player.Still != 0 {
		t.Errorf("Still %d just after landing", w.Player.Still)
	}
	run(w, None, config.LandTicks+5)
	if w.Player.Still != config.LandTicks {
		t.Errorf("Still %d after resting, want it capped at %d", w.Player.Still, config.LandTicks)
	}
}

func TestEveryDirection(t *testing.T) {
	w := newWorld(1)
	c, r := config.StartCol, config.StartRow
	for _, step := range []struct {
		d      Dir
		dc, dr int
	}{{Up, 0, -1}, {Up, 0, -1}, {Right, 1, 0}, {Down, 0, 1}, {Left, -1, 0}, {Left, -1, 0}} {
		run(w, step.d, config.HopTicks)
		c, r = c+step.dc, r+step.dr
		at(t, w, c, r)
		if w.Player.Facing != step.d {
			t.Errorf("facing %v after hopping %v", w.Player.Facing, step.d)
		}
	}
}

func TestEdgesBlockHops(t *testing.T) {
	cases := []struct {
		name     string
		col, row int
		d        Dir
	}{
		{"below the start row", config.StartCol, config.StartRow, Down},
		{"left edge", 0, 5, Left},
		{"right edge", config.Cols - 1, 5, Right},
		{"top row", 3, 0, Up},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(1)
			w.Player = newPlayer(c.col, c.row)
			w.Player.Facing = Left + Right - c.d // anything but c.d
			if ev := run(w, c.d, config.HopTicks); len(ev) != 0 {
				t.Errorf("events %v from a blocked hop", ev)
			}
			at(t, w, c.col, c.row)
			if w.Player.Facing == c.d {
				t.Error("a blocked hop turned the hero")
			}
		})
	}
}

func TestEarlyPressDuringAHopIsIgnored(t *testing.T) {
	w := newWorld(1)
	w.Update(Up)
	w.Update(Right) // far from landing
	run(w, None, 2*config.HopTicks)
	at(t, w, config.StartCol, config.StartRow-1)
}

func TestLatePressIsQueuedForTheLanding(t *testing.T) {
	w := newWorld(1)
	w.Update(Up)
	run(w, None, config.HopTicks-config.HopQueueTicks-1)
	// The hop has HopQueueTicks ticks left: presses now are kept.
	w.Update(Left)
	var ev []Event
	for i := 1; i < config.HopQueueTicks; i++ {
		ev = w.Update(Right) // the latest press wins
	}
	if !slices.Equal(ev, []Event{Landed, Hopped}) {
		t.Fatalf("landing tick gave %v, want a landing and the queued hop", ev)
	}
	if w.Player.Facing != Right || w.Player.Col != config.StartCol+1 {
		t.Fatalf("queued hop went %v to column %d", w.Player.Facing, w.Player.Col)
	}
	run(w, None, config.HopTicks)
	at(t, w, config.StartCol+1, config.StartRow-1)
}

func TestHeldRhythm(t *testing.T) {
	// Pressing again on every landing tick gives one hop every HopTicks.
	w := newWorld(1)
	for i := range 4 {
		run(w, Up, config.HopTicks)
		at(t, w, config.StartCol, config.StartRow-1-i)
	}
	if w.Tick != 4*config.HopTicks {
		t.Errorf("4 hops took %d ticks, want %d", w.Tick, 4*config.HopTicks)
	}
}

func TestUpdateIsDeterministic(t *testing.T) {
	path := []Dir{Up, Up, Left, Up, Right, Right, Down, Up}
	play := func() (int64, Player) {
		w := newWorld(42)
		for i := range 600 {
			d := None
			if i%5 == 0 {
				d = path[(i/5)%len(path)]
			}
			w.Update(d)
		}
		return w.Tick, w.Player
	}
	t1, p1 := play()
	t2, p2 := play()
	if t1 != t2 || p1 != p2 {
		t.Errorf("two runs from one seed differ:\n%d %+v\n%d %+v", t1, p1, t2, p2)
	}
}
