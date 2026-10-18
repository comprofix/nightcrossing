package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// afterWorld is Storm's bonus level, after the storm, from a seed.
func afterWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	lv, ok := config.BonusLevel(config.Storm)
	if !ok {
		t.Fatal("Storm has no bonus level")
	}
	return New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
}

// TestDragonfliesOnTheLogs: after the storm's dragonflies come only to the
// logs, and stay there, as dragonflies (DESIGN §6.7).
func TestDragonfliesOnTheLogs(t *testing.T) {
	for seed := range uint64(5) {
		w := afterWorld(t, seed)
		for tick := 0; tick < 20*config.TPS; tick++ {
			for i, f := range w.Swarm {
				if f.Visiting && (w.Lanes[f.row].Kind != config.River || !f.Day) {
					t.Fatalf("seed %d tick %d: flyer %d on row %d, day %v; want a dragonfly on the river", seed, tick, i, f.row, f.Day)
				}
			}
			w.Update(None)
		}
	}
}

// catchOnLog lands flyer i of w where the toad is, on river row r, and
// catches it, returning the points it paid.
func catchOnLog(w *World, i, r int) int {
	f := &w.Swarm[i]
	f.Visiting, f.Landed, f.row, f.X = true, true, r, w.Player.X
	before := w.Session.Score
	m := config.HeroMargin * config.SubPx
	w.catchSwarm(r, w.Player.X+m, w.Player.X+cellUnits-m)
	return w.Session.Score - before
}

// TestRuns: each dragonfly caught without setting foot on land is worth
// one more times than the last, up to the most; standing on land ends the
// run (DESIGN §6.7).
func TestRuns(t *testing.T) {
	w := afterWorld(t, 1)
	b := w.Level.Bonus
	w.Player = newPlayer(config.StartCol, 3)
	var got []int
	for range b.RunMax + 1 {
		if m := w.RunMult(); m != min(w.Run+1, b.RunMax) {
			t.Errorf("run %d: next worth x%d", w.Run, m)
		}
		got = append(got, catchOnLog(w, 0, 3))
	}
	want := []int{200, 400, 600, 800, 1000, 1000}
	if !slices.Equal(got, want) || w.Stats.BestRun != b.RunMax+1 || w.Stats.Run != 200+400+600+800+800 {
		t.Errorf("paid %v, best run %d, runs added %d; want %v", got, w.Stats.BestRun, w.Stats.Run, want)
	}
	w.Player = newPlayer(config.StartCol, config.VergeRow)
	w.endRun()
	w.Player = newPlayer(config.StartCol, 3)
	if pts := catchOnLog(w, 0, 3); pts != b.Firefly || w.Run != 1 {
		t.Errorf("after the verge: paid %d, run %d; want a fresh run", pts, w.Run)
	}
}

// TestHoppingOnTheRiverKeepsTheRun: hopping from log to log, the run goes
// on; only landing on land ends it, not passing over it mid-hop (DESIGN
// §6.7).
func TestHoppingOnTheRiverKeepsTheRun(t *testing.T) {
	w := afterWorld(t, 1)
	w.Player = newPlayer(config.StartCol, 3)
	w.Run = 2
	w.Player.hopLeft = 1
	w.Player.Row = config.VergeRow
	w.endRun()
	if w.Run != 2 {
		t.Errorf("run %d mid-hop onto the verge, want it kept until landing", w.Run)
	}
	w.Player.hopLeft = 0
	w.endRun()
	if w.Run != 0 {
		t.Errorf("run %d landed on the verge, want it ended", w.Run)
	}
}

// TestSlipperyRoad: after the storm the road is slippery, as in the rain:
// a hop onto it slides one more cell; empty roads at dawn only looks wet
// (D31, DESIGN §6.7).
func TestSlipperyRoad(t *testing.T) {
	w := afterWorld(t, 1)
	ev := run(w, Up, 1)
	ev = append(ev, run(w, None, 3*config.HopTicks)...)
	if !slices.Contains(ev, Slid) || w.Player.Row != config.StartRow-2 {
		t.Errorf("events %v, on row %d: want a slide to row %d", ev, w.Player.Row, config.StartRow-2)
	}
	w = roadsWorld(t, 1)
	w.Sweepers = [2]Sweeper{}
	ev = run(w, Up, 1)
	ev = append(ev, run(w, None, 3*config.HopTicks)...)
	if slices.Contains(ev, Slid) || w.Player.Row != config.StartRow-1 {
		t.Errorf("empty roads: events %v, on row %d: want no slide", ev, w.Player.Row)
	}
}

// TestAfterTheStormCurrents: two of the river's rows have a current, one
// with the flow and one against it (DESIGN §6.7).
func TestAfterTheStormCurrents(t *testing.T) {
	w := afterWorld(t, 1)
	var with, against int
	for r := range w.Lanes {
		switch l := &w.Lanes[r]; {
		case l.Current > 0:
			with++
		case l.Current < 0:
			against++
		}
	}
	if with != 1 || against != 1 {
		t.Errorf("%d rows with the flow and %d against, want one of each", with, against)
	}
}

// TestAutopilotOnAfterTheStorm: the autopilot plays the level through,
// catching dragonflies on the logs and never losing a life (DESIGN §6.7).
func TestAutopilotOnAfterTheStorm(t *testing.T) {
	for seed := range uint64(4) {
		w := afterWorld(t, seed)
		slips := 0
		for !w.Cleared {
			for _, e := range w.Update(Autopilot{}.Next(w)) {
				if e == Slipped {
					slips++
				}
			}
		}
		t.Logf("seed %d: %d of %d, best run %d, runs added %d, %d slips, %d s left, score %d", seed, w.Stats.Fireflies,
			w.Level.Bonus.Fireflies, w.Stats.BestRun, w.Stats.Run, slips, w.TicksLeft/config.TPS, w.Session.Score)
		if w.Stats.Fireflies < w.Level.Bonus.Fireflies/2 || w.Session.Lives != config.StartLives {
			t.Errorf("seed %d: caught %d, %d lives", seed, w.Stats.Fireflies, w.Session.Lives)
		}
	}
}
