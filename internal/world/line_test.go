package world

import (
	"math/rand/v2"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// lineWorld is a School Run game on an all-safe level with the far bank on
// row 0 and a line of n toadlets.
func lineWorld(n int) *World {
	lv := testLevel()
	lv.Lanes[config.GoalRow] = config.Lane{Kind: config.Bank}
	lv.Toadlets = n
	return New(session.New(config.SchoolRun, config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
}

// placeLine puts the hero and its whole line, landed, in one cell, as at
// the start of an attempt.
func placeLine(w *World, col, row int) {
	w.Player = newPlayer(col, row)
	w.Line = newLine(len(w.Line))
	for i := range w.Line {
		w.Line[i].Player = newPlayer(col, row)
	}
	w.furthest = row
}

// toadletAt fails unless toadlet i is landed in cell col, row.
func toadletAt(t *testing.T, w *World, i, col, row int) {
	t.Helper()
	p := w.Line[i].Player
	if p.Col != col || p.Row != row || p.Hopping() {
		t.Fatalf("toadlet %d in cell %d,%d (moving %v), want %d,%d", i, p.Col, p.Row, p.Hopping(), col, row)
	}
	if p.X != col*cellUnits || p.Y != row*cellUnits {
		t.Fatalf("toadlet %d at %d,%d units, want the corner of cell %d,%d", i, p.X, p.Y, col, row)
	}
}

// TestLineStartsInTheHerosCell: each attempt starts the line in the
// hero's start cell, and the timer has a second more for each toadlet
// (D53).
func TestLineStartsInTheHerosCell(t *testing.T) {
	lv := config.SchoolRun.Levels.Level(5)
	w := New(session.New(config.SchoolRun, config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
	if len(w.Line) != 2 {
		t.Fatalf("%d toadlets on level 5, want 2", len(w.Line))
	}
	for i := range w.Line {
		toadletAt(t, w, i, config.StartCol, config.StartRow)
	}
	if want := config.TimerTicks(5) + 2*config.TPS; w.TicksLeft != want {
		t.Errorf("%d ticks on the clock, want %d", w.TicksLeft, want)
	}
	if w := newWorld(1); w.Line != nil {
		t.Errorf("a Stroll game has a line of %d", len(w.Line))
	}
}

// TestToadletsFollowTheTrail: when the one ahead moves, each toadlet moves
// into the cell it has just left, on the same beat (D53).
func TestToadletsFollowTheTrail(t *testing.T) {
	w := lineWorld(2)
	c, r := config.StartCol, config.StartRow
	ev := hops(w, Up, 1)
	at(t, w, c, r-1)
	toadletAt(t, w, 0, c, r)
	toadletAt(t, w, 1, c, r)
	ev = append(ev, hops(w, Left, 1)...)
	at(t, w, c-1, r-1)
	toadletAt(t, w, 0, c, r-1)
	toadletAt(t, w, 1, c, r)
	ev = append(ev, hops(w, Up, 1)...)
	at(t, w, c-1, r-2)
	toadletAt(t, w, 0, c-1, r-1)
	toadletAt(t, w, 1, c, r-1)
	if w.Line[0].Facing != Left || w.Line[1].Facing != Up {
		t.Errorf("toadlets facing %v and %v, want left and up", w.Line[0].Facing, w.Line[1].Facing)
	}
	if n := count(ev, Hopped); n != 3 {
		t.Errorf("%d Hopped events, want the hero's 3: the toadlets' moves are quiet", n)
	}
}

// TestTheLineMovesOnTheSameBeat: a toadlet's move starts on the tick the
// hero's does, and lands with it.
func TestTheLineMovesOnTheSameBeat(t *testing.T) {
	w := lineWorld(1)
	hops(w, Up, 1)
	w.Update(Up)
	if !w.Player.Hopping() || !w.Line[0].Hopping() || w.Line[0].HopProgress() != w.Player.HopProgress() {
		t.Fatalf("hero %d ticks into its hop, toadlet %d", w.Player.HopProgress(), w.Line[0].HopProgress())
	}
}

// TestABlockedHopIsNotPassedOn: a hop the field's edge stops isn't a move,
// so the line doesn't move either (D53).
func TestABlockedHopIsNotPassedOn(t *testing.T) {
	w := lineWorld(1)
	placeLine(w, 0, config.StartRow)
	hops(w, Up, 1)
	if ev := hops(w, Left, 1); count(ev, Hopped) != 0 {
		t.Fatalf("hopped off the field's left edge")
	}
	toadletAt(t, w, 0, 0, config.StartRow)
	hops(w, Up, 1)
	toadletAt(t, w, 0, 0, config.StartRow-1)
}

// TestHoppingBackPastAToadlet: the hero and the toadlets never collide,
// so a hop back into a toadlet's cell is just a hop, and the toadlet moves
// into the cell the hero left.
func TestHoppingBackPastAToadlet(t *testing.T) {
	w := lineWorld(1)
	c, r := config.StartCol, config.StartRow
	hops(w, Up, 2)
	hops(w, Down, 1)
	at(t, w, c, r-1)
	toadletAt(t, w, 0, c, r-2)
	if w.Dying > 0 {
		t.Fatalf("a death sharing a cell")
	}
}

// checkpointWorld is a School Run game with a long log on the river row
// just above the verge, drifting under the start column, and the hero on the verge
// with a line of n toadlets behind it, one a row, not yet gathered.
func checkpointWorld(n int) *World {
	w := riverWorld(config.LogLong, false, (config.StartCol-3)*cellUnits, 1, config.SubPx)
	w.Level.Toadlets = n
	w.Player = newPlayer(config.StartCol, config.VergeRow)
	w.Line = newLine(n)
	for i := range w.Line {
		w.Line[i].Player = newPlayer(config.StartCol, config.VergeRow+1+i)
	}
	return w
}

// TestTheVergeIsACheckpoint: a hop from the verge onto the river is refused
// until the whole line is in the hero's cell, with Refused for the peep;
// then the toadlets ride on its back (D53, D56).
func TestTheVergeIsACheckpoint(t *testing.T) {
	w := checkpointWorld(2)
	if ev := hops(w, Up, 1); count(ev, Hopped) != 0 || count(ev, Refused) != 1 {
		t.Fatalf("events %v: want the hop onto the river refused with the line behind", ev)
	}
	at(t, w, config.StartCol, config.VergeRow)
	run(w, None, config.BunchTicks) // the line bunches up
	if ev := hops(w, Up, 1); count(ev, Hopped) != 1 || count(ev, Splashed) != 0 || count(ev, Refused) != 0 {
		t.Fatalf("events %v: want a hop onto the log with the line gathered", ev)
	}
	for i, tl := range w.Line {
		if !tl.Riding || tl.Player.X != w.Player.X || tl.Row != config.VergeRow-1 {
			t.Fatalf("toadlet %d riding %v at %d in row %d, want on the hero's back at %d", i, tl.Riding, tl.Player.X, tl.Row, w.Player.X)
		}
	}
}

// TestRidersGoWithTheHero: riders are carried with the hero, and get off,
// bunched in its cell, when it hops back down onto the verge (D53).
func TestRidersGoWithTheHero(t *testing.T) {
	w := checkpointWorld(1)
	run(w, None, config.BunchTicks)
	hops(w, Up, 1)
	run(w, None, 20)
	if tl := w.Line[0]; !tl.Riding || tl.Player.X != w.Player.X || tl.Hopping() {
		t.Fatalf("toadlet at %d, want riding at the hero's %d", tl.Player.X, w.Player.X)
	}
	hops(w, Down, 1)
	if tl := w.Line[0]; tl.Riding || tl.Row != config.VergeRow || tl.Player.X != w.Player.X {
		t.Fatalf("toadlet riding %v in row %d, want off on the verge in the hero's cell", tl.Riding, tl.Row)
	}
	hops(w, Down, 1) // the line draws out again
	if tl := w.Line[0]; tl.Row != config.VergeRow {
		t.Errorf("toadlet in row %d, want left on the verge", tl.Row)
	}
}

// TestRidersComeHomeWithTheHero: landing in a nook brings the riders home
// at once, with +50 each (D53).
func TestRidersComeHomeWithTheHero(t *testing.T) {
	w := lineWorld(2)
	placeLine(w, config.Nooks[1], 1)
	w.mount()
	score := w.Session.Score
	ev := hops(w, Up, 1)
	if count(ev, Goal) != 1 || !w.Filled[1] {
		t.Fatalf("events %v: want the nook filled", ev)
	}
	secs := w.TicksLeft / config.TPS
	if want := score + config.PointsHop + config.PointsGoal + 2*config.PointsToadlet + secs*config.PointsPerSec; w.Session.Score != want {
		t.Errorf("score %d, want %d", w.Session.Score, want)
	}
	if w.Stats.Toadlets != 2 {
		t.Errorf("%d toadlets home in the stats, want 2", w.Stats.Toadlets)
	}
}

// TestAToadletsDeathEndsTheAttempt: every hazard kills a toadlet as it
// kills the hero; its death loses a life, and the hero and line start
// again on the start row (D53).
func TestAToadletsDeathEndsTheAttempt(t *testing.T) {
	w := roadWorld(config.CarRed, config.StartCol*cellUnits+Buffer, 1, config.SubPx)
	w.Level.Toadlets = 1
	w.Player = newPlayer(config.StartCol, 10)
	w.Line = newLine(1)
	w.Line[0].Player = newPlayer(config.StartCol, 11) // under the car
	lives := w.Session.Lives
	ev := run(w, None, 1)
	if count(ev, Squashed) != 1 || w.Dying == 0 {
		t.Fatalf("events %v: want the toadlet squashed", ev)
	}
	if w.Fallen() != &w.Line[0].Player || w.Line[0].Death != Squashed || w.Player.Death != 0 {
		t.Fatalf("the death is the hero's (%v), want the toadlet's", w.Player.Death)
	}
	if w.Session.Lives != lives-1 {
		t.Errorf("%d lives, want %d", w.Session.Lives, lives-1)
	}
	ev = run(w, None, config.DeathTicks)
	if count(ev, Respawned) != 1 {
		t.Fatalf("events %v: want a new attempt", ev)
	}
	at(t, w, config.StartCol, config.StartRow)
	toadletAt(t, w, 0, config.StartCol, config.StartRow)
	if w.Line[0].Death != 0 || w.Fallen() != nil {
		t.Errorf("the toadlet is still dead")
	}
}

// TestTheLineTakesTheHerosPush: the wind pushes only the hero; a toadlet
// behind takes the push as it reaches the spot, so it goes where the hero
// was (D47, D53).
func TestTheLineTakesTheHerosPush(t *testing.T) {
	w := lineWorld(1)
	c, r := config.StartCol, config.StartRow
	hops(w, Up, 2)
	windy(w, 1, 90)
	if ev := run(w, None, config.HopTicks+1); count(ev, Pushed) != 1 {
		t.Fatalf("events %v: want one push", ev)
	}
	at(t, w, c+1, r-2)
	toadletAt(t, w, 0, c, r-2)
	if w.Line[0].Facing != Up {
		t.Errorf("toadlet facing %v: a push keeps the facing", w.Line[0].Facing)
	}
}

// TestABunchedLineIsPushedWithTheHero: toadlets in the hero's cell are
// pushed with it, at the same moment, so the bunch stays together (D53).
func TestABunchedLineIsPushedWithTheHero(t *testing.T) {
	w := lineWorld(2)
	windy(w, 1, 90)
	run(w, None, 2)
	if !w.Line[0].Hopping() || !w.Line[0].Pushed || !w.Line[1].Pushed {
		t.Fatalf("the bunched line isn't pushed with the hero")
	}
	run(w, None, config.HopTicks)
	at(t, w, config.StartCol+1, config.StartRow)
	toadletAt(t, w, 0, config.StartCol+1, config.StartRow)
	toadletAt(t, w, 1, config.StartCol+1, config.StartRow)
}

// TestRainSlidesTheLine: the rain doesn't slide a toadlet itself; it takes
// the hero's slides as it follows (D31, D53).
func TestRainSlidesTheLine(t *testing.T) {
	w := wetWorld()
	w.Level.Toadlets = 1
	w.Line = newLine(1)
	ev := hops(w, Up, 1)
	ev = append(ev, run(w, None, config.HopTicks)...)
	at(t, w, config.StartCol, config.StartRow-2)
	toadletAt(t, w, 0, config.StartCol, config.StartRow-1) // where the hero's slide began: no slide of its own
	ev = append(ev, hops(w, Up, 1)...)
	ev = append(ev, run(w, None, config.HopTicks)...)
	at(t, w, config.StartCol, config.StartRow-4)
	toadletAt(t, w, 0, config.StartCol, config.StartRow-3)
	if n := count(ev, Slid); n != 2 {
		t.Errorf("%d Slid events, want the hero's 2", n)
	}
}

// TestTheLineBunchesUpOnASafeRow: standing still on a safe row, the hero
// gathers the toadlets behind it into its cell (D53).
func TestTheLineBunchesUpOnASafeRow(t *testing.T) {
	w := lineWorld(2)
	c, r := config.StartCol, config.StartRow
	hops(w, Up, 2)
	run(w, None, config.BunchTicks-1)
	toadletAt(t, w, 0, c, r-1)
	toadletAt(t, w, 1, c, r)
	run(w, None, 1)
	toadletAt(t, w, 0, c, r-2)
	toadletAt(t, w, 1, c, r-2)
	hops(w, Up, 1) // the next move draws the line out again
	toadletAt(t, w, 0, c, r-2)
	hops(w, Up, 1)
	toadletAt(t, w, 0, c, r-3)
	toadletAt(t, w, 1, c, r-2)
}

// TestNoBunchingOffASafeRow: the hero standing on the road gathers no one.
func TestNoBunchingOffASafeRow(t *testing.T) {
	w := wetWorld()
	w.Spells = nil // no rain: plain empty road
	w.Level.Toadlets = 1
	w.Line = newLine(1)
	hops(w, Up, 2)
	run(w, None, 2*config.BunchTicks)
	toadletAt(t, w, 0, config.StartCol, config.StartRow-1)
}

// TestCloneCopiesTheLine: running a clone leaves the world's line alone.
func TestCloneCopiesTheLine(t *testing.T) {
	w := lineWorld(2)
	hops(w, Up, 2)
	before := w.Line[1]
	c := w.Clone(rand.New(rand.NewPCG(2, 2)))
	hops(c, Left, 3)
	if w.Line[1].Player != before.Player || len(w.Line[1].trail) != len(before.trail) {
		t.Errorf("running the clone changed the world's line")
	}
}
