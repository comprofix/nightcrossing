package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// gatesWorld is School Run's bonus level, the school gates, from a seed.
func gatesWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	lv, ok := config.BonusLevel(config.SchoolRun)
	if !ok {
		t.Fatal("School Run has no bonus level")
	}
	return New(session.New(config.SchoolRun, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
}

// TestStragglersWait: the stragglers wait at random along the kerbs, the
// start row, the verge and the track, apart and clear of the start, a
// different place each run; the toad starts with no line (DESIGN §6.7).
func TestStragglersWait(t *testing.T) {
	places := map[[2]int]bool{}
	for seed := range uint64(50) {
		w := gatesWorld(t, seed)
		b := w.Level.Bonus
		if len(w.Stragglers) != b.Stragglers || len(w.Line) != 0 {
			t.Fatalf("seed %d: %d stragglers, a line of %d", seed, len(w.Stragglers), len(w.Line))
		}
		for i, s := range w.Stragglers {
			if !w.kerb(s.Row) || !s.Waiting() {
				t.Errorf("seed %d: straggler %d at row %d, waiting %v", seed, i, s.Row, s.Waiting())
			}
			if s.Row == config.StartRow && abs(s.Col-config.StartCol) <= config.BonusClearCells {
				t.Errorf("seed %d: straggler %d by the start", seed, i)
			}
			for _, o := range w.Stragglers[:i] {
				if abs(o.Row-s.Row) < config.BonusFruitApart && abs(o.Col-s.Col) < config.BonusFruitApart {
					t.Errorf("seed %d: stragglers next to each other at %v and %v", seed, o, s)
				}
			}
		}
		places[[2]int{w.Stragglers[0].Row, w.Stragglers[0].Col}] = true
	}
	if len(places) < 10 {
		t.Errorf("the first straggler waited in %d places in 50 runs", len(places))
	}
}

// besideStraggler puts the toad one cell to the left of straggler i, on
// the grid and landed, with the rest of the field clear of traffic.
func besideStraggler(w *World, i int) {
	for r := range w.Lanes {
		w.Lanes[r].Entities = nil
	}
	s := &w.Stragglers[i]
	if s.Col == 0 {
		s.Col = 1
	}
	w.Player = newPlayer(s.Col-1, s.Row)
}

// TestGathering: landing on a straggler adds it to the front of the line,
// in the toad's cell; the line holds config's LineMax at most, and with it
// full a straggler peeps and waits (DESIGN §6.7).
func TestGathering(t *testing.T) {
	w := gatesWorld(t, 3)
	b := w.Level.Bonus
	for i := range b.LineMax + 1 {
		besideStraggler(w, i)
		w.Line = cloneLine(w.Line) // as if it had filed in behind
		for j := range w.Line {
			w.Line[j].Player = newPlayer(w.Player.Col, w.Player.Row)
		}
		ev := run(w, Right, 1)
		ev = append(ev, run(w, None, config.HopTicks)...)
		s := w.Stragglers[i]
		switch {
		case i < b.LineMax:
			if !s.InLine || len(w.Line) != i+1 || !slices.Contains(ev, Joined) ||
				w.Line[0].Row != w.Player.Row || w.Line[0].X != w.Player.X {
				t.Fatalf("straggler %d: in line %v, line of %d, events %v", i, s.InLine, len(w.Line), ev)
			}
		default:
			if s.InLine || len(w.Line) != b.LineMax || !slices.Contains(ev, Refused) {
				t.Errorf("with the line full: in line %v, line of %d, events %v", s.InLine, len(w.Line), ev)
			}
		}
	}
}

// TestTheLineFollows: a straggler gathered follows the toad's hops one
// behind, as in School Run (DESIGN §6.5, §6.7).
func TestTheLineFollows(t *testing.T) {
	w := gatesWorld(t, 3)
	besideStraggler(w, 0)
	run(w, Right, 1)
	run(w, None, config.HopTicks)
	at := w.Player.Col
	run(w, Right, 1)
	run(w, None, 2*config.HopTicks)
	if len(w.Line) != 1 || w.Line[0].Col != at || w.Player.Col != at+1 {
		t.Errorf("toad at %d, line %d at %d: want the straggler one behind at %d", w.Player.Col, len(w.Line), w.Line[0].Col, at)
	}
}

// TestHomeTogether: a nook brings the line home, each straggler for its
// points, doubled for a trip of config's TripMin or more (DESIGN §6.7).
func TestHomeTogether(t *testing.T) {
	b := config.SchoolGates
	for _, n := range []int{1, b.TripMin, b.LineMax} {
		w := gatesWorld(t, 3)
		for i := range n {
			w.Stragglers[i].InLine = true
		}
		w.Line = newLine(n)
		before := w.Session.Score
		w.bonusNook()
		want := n * b.Straggler
		if n >= b.TripMin {
			want *= 2
		}
		if got := w.Session.Score - before - config.PointsBonusNook; got != want || w.Stats.Toadlets != n || w.Collected() != n {
			t.Errorf("a trip of %d: paid %d, %d home; want %d", n, got, w.Stats.Toadlets, want)
		}
		for i := range n {
			if s := w.Stragglers[i]; !s.Home || s.InLine {
				t.Errorf("a trip of %d: straggler %d %+v, want home", n, i, s)
			}
		}
	}
}

// TestASlipSendsThemBack: a slip sends the stragglers in the line back to
// where each was waiting (DESIGN §6.7).
func TestASlipSendsThemBack(t *testing.T) {
	w := gatesWorld(t, 3)
	w.Stragglers[0].InLine = true
	w.Line = newLine(1)
	w.die(Splashed)
	run(w, None, config.SlipTicks+1)
	if s := w.Stragglers[0]; !s.Waiting() || len(w.Line) != 0 || w.Session.Lives != config.StartLives {
		t.Errorf("after a slip: straggler %+v, line of %d, %d lives", s, len(w.Line), w.Session.Lives)
	}
}

// TestAllHome: bringing the last straggler home ends the level with the
// all-home bonus (DESIGN §6.7).
func TestAllHome(t *testing.T) {
	w := gatesWorld(t, 3)
	for i := range w.Stragglers {
		w.Stragglers[i].Home = true
	}
	w.Stats.Toadlets = len(w.Stragglers) - 1
	w.Stragglers[0].Home, w.Stragglers[0].InLine = false, true
	w.Line = newLine(1)
	w.bonusNook()
	if !w.Cleared || w.Stats.All != w.Level.Bonus.All {
		t.Errorf("cleared %v, all %d", w.Cleared, w.Stats.All)
	}
}

// TestAutopilotAtTheSchoolGates: the autopilot gathers stragglers and
// walks them home, never losing a life (DESIGN §6.7).
func TestAutopilotAtTheSchoolGates(t *testing.T) {
	for seed := range uint64(4) {
		w := gatesWorld(t, seed)
		slips, trips := 0, 0
		for !w.Cleared {
			for _, e := range w.Update(Autopilot{}.Next(w)) {
				switch e {
				case Slipped:
					slips++
				case Goal:
					trips++
				}
			}
		}
		t.Logf("seed %d: %d of %d home in %d trips, trip bonus %d, %d slips, %d s left, score %d", seed, w.Stats.Toadlets,
			len(w.Stragglers), trips, w.Stats.Trip, slips, w.TicksLeft/config.TPS, w.Session.Score)
		if w.Stats.Toadlets < len(w.Stragglers)/2 || w.Session.Lives != config.StartLives {
			t.Errorf("seed %d: %d home, %d lives", seed, w.Stats.Toadlets, w.Session.Lives)
		}
	}
}
