package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// bankWorld is a game on an all-safe level with the far bank on row 0, the
// hero standing on row 1 in col.
func bankWorld(col int) *World {
	lv := testLevel()
	lv.Lanes[config.GoalRow] = config.Lane{Kind: config.Bank}
	w := New(session.New(config.Modes[0], config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
	w.Player = newPlayer(col, 1)
	w.furthest = 1
	return w
}

func TestNookAt(t *testing.T) {
	for i, c := range config.Nooks {
		for dc := range config.NookWidth {
			if got := NookAt(c + dc); got != i {
				t.Errorf("column %d: nook %d, want %d", c+dc, got, i)
			}
		}
		if NookAt(c-1) == i || NookAt(c+config.NookWidth) == i {
			t.Errorf("nook %d reaches past its columns", i)
		}
	}
	// The nooks are spread evenly, with the same bank at each side.
	first, last := config.Nooks[0], config.Nooks[len(config.Nooks)-1]
	if first != config.Cols-(last+config.NookWidth) {
		t.Errorf("bank of %d cells on the left, %d on the right", first, config.Cols-(last+config.NookWidth))
	}
}

func TestFillANook(t *testing.T) {
	w := bankWorld(config.Nooks[2] + 1)
	w.TicksLeft = 30*config.TPS + 20 // 30 whole seconds left when it lands
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, Goal) || !w.Filled[2] {
		t.Fatalf("events %v, filled %v: want nook 2 filled", ev, w.Filled)
	}
	want := config.PointsHop + config.PointsGoal + 30*config.PointsPerSec
	if w.Session.Score != want {
		t.Errorf("score %d, want %d (hop, goal and time bonus)", w.Session.Score, want)
	}
	if w.Home != config.GoalTicks {
		t.Errorf("home %d, want %d", w.Home, config.GoalTicks)
	}
	// The toad shows in its nook, then a new attempt starts with a full clock.
	if ev := run(w, None, config.GoalTicks); !slices.Equal(ev, []Event{Respawned}) {
		t.Fatalf("events %v after the goal, want a respawn", ev)
	}
	at(t, w, config.StartCol, config.StartRow)
	if w.TicksLeft != w.Level.TimerTicks || !w.Filled[2] {
		t.Errorf("clock %d, filled %v after the respawn", w.TicksLeft, w.Filled)
	}
}

func TestBankAndFilledNooksBump(t *testing.T) {
	for name, setup := range map[string]func() *World{
		"the bank": func() *World { return bankWorld(config.Nooks[1] - 1) },
		"a filled nook": func() *World {
			w := bankWorld(config.Nooks[1])
			w.Filled[1] = true
			return w
		},
	} {
		w := setup()
		ev := run(w, Up, config.HopTicks)
		if !slices.Contains(ev, Bumped) || w.Player.Death != Bumped {
			t.Errorf("%s: events %v, want a bump", name, ev)
		}
	}
}

func TestFifthNookClearsTheLevel(t *testing.T) {
	w := bankWorld(config.Nooks[4])
	for i := range 4 {
		w.Filled[i] = true
	}
	before := w.Session.Score
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, LevelClear) || !w.Cleared {
		t.Fatalf("events %v: want the level cleared", ev)
	}
	if got := w.Session.Score - before; got < config.PointsGoal+config.PointsAllGoals {
		t.Errorf("scored %d for the last nook, want at least %d", got, config.PointsGoal+config.PointsAllGoals)
	}
	if w.Stats.Clear != config.PointsAllGoals || w.Stats.Goals != 1 {
		t.Errorf("stats %+v", w.Stats)
	}
	tick := w.Tick
	if ev := run(w, Down, 30); len(ev) != 0 || w.Tick != tick {
		t.Errorf("the world ran on after the level was cleared: %v", ev)
	}
}

// TestNookStaysFilledAfterADeath: filled nooks stay filled (DESIGN §5.4).
func TestNookStaysFilledAfterADeath(t *testing.T) {
	w := bankWorld(config.Nooks[0])
	w.Filled[3] = true
	w.die(Squashed)
	run(w, None, config.DeathTicks)
	if !w.Filled[3] {
		t.Error("a death emptied a nook")
	}
}

func TestTimeRunsOut(t *testing.T) {
	w := newWorld(1)
	low := w.Level.TimerTicks - config.LowTimeSecs*config.TPS
	ev := run(w, None, low)
	if !slices.Equal(ev, []Event{LowTime}) {
		t.Fatalf("events %v by %d ticks, want the low time warning", ev, low)
	}
	ev = run(w, None, config.LowTimeSecs*config.TPS)
	if !slices.Equal(ev, []Event{TimeUp}) || w.Player.Death != TimeUp {
		t.Fatalf("events %v when the clock ran out, want time up", ev)
	}
	if w.Session.Lives != config.StartLives-1 {
		t.Errorf("%d lives, want %d", w.Session.Lives, config.StartLives-1)
	}
}

func TestClockStopsWhileDyingOrHome(t *testing.T) {
	w := newWorld(1)
	w.die(Squashed)
	left := w.TicksLeft
	run(w, None, config.DeathTicks-1)
	if w.TicksLeft != left {
		t.Errorf("the clock ran during a death: %d → %d", left, w.TicksLeft)
	}
}

func TestHopPointsOnlyForNewRows(t *testing.T) {
	w := newWorld(1)
	run(w, Up, config.HopTicks)
	run(w, Up, config.HopTicks)
	run(w, Down, config.HopTicks)
	run(w, Up, config.HopTicks)
	run(w, Left, config.HopTicks)
	if want := 2 * config.PointsHop; w.Session.Score != want {
		t.Errorf("score %d after two new rows, want %d", w.Session.Score, want)
	}
	// A new attempt starts counting rows again.
	w.die(Squashed)
	run(w, None, config.DeathTicks)
	run(w, Up, config.HopTicks)
	if want := 3 * config.PointsHop; w.Session.Score != want {
		t.Errorf("score %d after a row on the next attempt, want %d", w.Session.Score, want)
	}
}

func TestExtraLife(t *testing.T) {
	w := newWorld(1)
	w.Session.Score = config.ExtraLifeEvery - config.PointsHop
	if ev := run(w, Up, config.HopTicks); !slices.Contains(ev, ExtraLife) {
		t.Fatalf("events %v crossing %d points, want an extra life", ev, config.ExtraLifeEvery)
	}
	if w.Session.Lives != config.StartLives+1 {
		t.Errorf("%d lives, want %d", w.Session.Lives, config.StartLives+1)
	}
	if ev := run(w, Up, config.HopTicks); slices.Contains(ev, ExtraLife) {
		t.Error("another life without passing the next threshold")
	}
}

// TestQueuedHopDoesNotSkipASplash: a press queued for the landing must not
// carry the hero on before the water counts.
func TestQueuedHopDoesNotSkipASplash(t *testing.T) {
	w := riverWorld(config.LogShort, false, 0, 1, 0) // the only log is far off
	w.Player = newPlayer(config.Cols-2, riverRow+1)
	w.Update(Up)
	run(w, None, config.HopTicks-2)
	ev := w.Update(Up) // queued, and the hop lands this tick
	if !slices.Contains(ev, Splashed) || slices.Contains(ev, Hopped) {
		t.Errorf("events %v: want a splash and no queued hop", ev)
	}
}
