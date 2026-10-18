package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// roadsWorld is Rush Hour's bonus level, empty roads at dawn, from a seed.
func roadsWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	lv, ok := config.BonusLevel(config.RushHour)
	if !ok {
		t.Fatal("Rush Hour has no bonus level")
	}
	return New(session.New(config.RushHour, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
}

// clearFruit takes every fruit off the field, so the sweepers sweep none.
func clearFruit(w *World) {
	for i := range w.Picks {
		w.Picks[i].Taken = true
	}
}

// sweeping reports whether sweeper s is on the field.
func sweeping(s *Sweeper) bool { return s.On && s.X+sweeperUnits > 0 && s.X < fieldWidth }

// TestSweeperPaths: after their wait, one sweeper comes onto the top road
// lane from a side and the other onto the bottom lane; each sweeps
// every lane in turn, the other way each time, never coming back to one,
// and is gone after its last; in a lane they share they go the same way
// and never meet (DESIGN §6.7).
func TestSweeperPaths(t *testing.T) {
	for seed := range uint64(4) {
		w := roadsWorld(t, seed)
		clearFruit(w)
		rows := w.roadRows()
		down := slices.Clone(rows)
		slices.Reverse(down)
		want := [2][]int{down, rows}
		var seen [2][]int
		dirs := [2]map[int]int{{}, {}}
		for tick := 0; (w.Sweepers[0].On || w.Sweepers[1].On) && tick < 60*config.TPS; tick++ {
			for i := range w.Sweepers {
				s := &w.Sweepers[i]
				if sweeping(s) && (len(seen[i]) == 0 || seen[i][len(seen[i])-1] != s.Row) {
					seen[i] = append(seen[i], s.Row)
					dirs[i][s.Row] = s.Dir
				}
			}
			if tick < config.SweeperHoldTicks && (sweeping(&w.Sweepers[0]) || sweeping(&w.Sweepers[1])) {
				t.Fatalf("seed %d: a sweeper set off at tick %d", seed, tick)
			}
			a, b := &w.Sweepers[0], &w.Sweepers[1]
			if a.On && b.On && a.Row == b.Row {
				if a.Dir != b.Dir && sweeping(a) && sweeping(b) {
					t.Fatalf("seed %d: the sweepers went opposite ways on row %d at tick %d", seed, a.Row, tick)
				}
				if a.X < b.X+sweeperUnits && b.X < a.X+sweeperUnits {
					t.Fatalf("seed %d: the sweepers met on row %d at tick %d", seed, a.Row, tick)
				}
			}
			w.Update(None)
		}
		for i := range w.Sweepers {
			if w.Sweepers[i].On || !slices.Equal(seen[i], want[i]) {
				t.Fatalf("seed %d: sweeper %d swept rows %v, still on %v; want %v", seed, i, seen[i], w.Sweepers[i].On, want[i])
			}
			for k := 1; k < len(want[i]); k++ {
				if dirs[i][want[i][k]] != -dirs[i][want[i][k-1]] {
					t.Errorf("seed %d: sweeper %d swept rows %d and %d the same way", seed, i, want[i][k-1], want[i][k])
				}
			}
		}
	}
}

// TestSweptAt: the tick worked out for the first sweeper to reach each
// road cell is when one first does (DESIGN §6.7).
func TestSweptAt(t *testing.T) {
	for seed := range uint64(4) {
		w := roadsWorld(t, seed)
		clearFruit(w)
		want := map[[2]int]int{}
		for _, r := range w.roadRows() {
			for c := range w.Cols {
				want[[2]int{r, c}] = w.sweptAt(r, c*cellUnits)
			}
		}
		got := map[[2]int]int{}
		for tick := 1; tick < 60*config.TPS; tick++ {
			w.Update(None)
			for _, s := range w.Sweepers {
				for c := range w.Cols {
					x := c * cellUnits
					if _, ok := got[[2]int{s.Row, c}]; !ok && s.On && x < s.X+sweeperUnits && s.X < x+cellUnits {
						got[[2]int{s.Row, c}] = tick
					}
				}
			}
		}
		for k, t0 := range want {
			if g, ok := got[k]; !ok || abs(g-t0) > 2 {
				t.Errorf("seed %d: row %d col %d swept at tick %d, worked out as %d", seed, k[0], k[1], g, t0)
			}
		}
	}
}

// TestSweepingFruit: fruit in a sweeper's path is swept up as it passes:
// gone for good, and not collected (DESIGN §6.7).
func TestSweepingFruit(t *testing.T) {
	w := roadsWorld(t, 1)
	clearFruit(w)
	s := &w.Sweepers[0]
	w.Picks[0] = Pick{Fruit: 7, Row: s.Row, rake: -1, host: -1, offset: 2 * cellUnits}
	w.Picks[1] = Pick{Fruit: 7, Row: s.Row, rake: -1, host: -1, offset: (config.Cols - 3) * cellUnits}
	ev := run(w, None, config.SweeperHoldTicks+sweeperLaneTicks)
	if !w.Picks[0].Gone || !w.Picks[1].Gone || w.Showing(0) || w.Collected() != 0 || !slices.Contains(ev, FruitSwept) {
		t.Errorf("events %v, fruit %+v %+v, %d collected: want both swept", ev, w.Picks[0], w.Picks[1], w.Collected())
	}
}

// TestTheSweeperIsASlip: on the road in a sweeper's way, the toad slips
// back to the start, losing no life (DESIGN §6.7).
func TestTheSweeperIsASlip(t *testing.T) {
	w := roadsWorld(t, 1)
	w.Player = newPlayer(config.StartCol, w.Sweepers[0].Row)
	ev := run(w, None, config.SweeperHoldTicks+sweeperLaneTicks)
	if !slices.Contains(ev, Slipped) || w.Session.Lives != config.StartLives {
		t.Errorf("events %v, %d lives: want a slip", ev, w.Session.Lives)
	}
}

// TestAllGone: once every fruit is collected or swept up, the level ends
// early, with no all-collected bonus and no time bonus (DESIGN §6.7).
func TestAllGone(t *testing.T) {
	w := roadsWorld(t, 1)
	clearFruit(w)
	w.Stats.Picked = len(w.Picks) - 1
	w.Picks[0] = Pick{Fruit: 0, Row: w.Sweepers[0].Row, rake: -1, host: -1, offset: (config.Cols / 2) * cellUnits}
	ev := run(w, None, config.SweeperHoldTicks+sweeperLaneTicks)
	if !slices.Contains(ev, BonusOver) || !w.Cleared || w.TicksLeft <= 0 || w.Stats.All != 0 || w.Stats.TimeBonus != 0 {
		t.Errorf("events %v, cleared %v, %d ticks left, all %d, time bonus %d", ev, w.Cleared, w.TicksLeft,
			w.Stats.All, w.Stats.TimeBonus)
	}
}

// TestEmptyRoadsFruit: the fruit lies at random each run on the road and
// the verge, apart, clear of the toad's start, and never where a sweeper
// would get to it before the toad could (DESIGN §6.7).
func TestEmptyRoadsFruit(t *testing.T) {
	layouts := map[int]bool{}
	for seed := range uint64(50) {
		w := roadsWorld(t, seed)
		if len(w.Picks) != w.Level.Bonus.Size() {
			t.Fatalf("seed %d: %d fruit", seed, len(w.Picks))
		}
		for i, p := range w.Picks {
			x := w.PickX(i)
			if w.Lanes[p.Row].Kind != config.Road && p.Row != config.VergeRow || p.rake >= 0 || p.host >= 0 {
				t.Errorf("seed %d: fruit %d on row %d isn't on the road or verge", seed, i, p.Row)
			}
			if abs(p.Row-config.StartRow) <= config.BonusClearCells && abs(columnOf(x)-config.StartCol) <= config.BonusClearCells {
				t.Errorf("seed %d: fruit %d is by the toad's start", seed, i)
			}
			if w.beaten(p.Row, x) {
				t.Errorf("seed %d: fruit %d on row %d col %d would be swept before the toad got there", seed, i, p.Row, columnOf(x))
			}
			for j, o := range w.Picks[:i] {
				if abs(o.Row-p.Row) < config.BonusFruitApart && abs(w.PickX(j)-x) < config.BonusFruitApart*cellUnits {
					t.Errorf("seed %d: fruit %d and %d lie next to each other", seed, j, i)
				}
			}
		}
		layouts[w.Picks[0].Row*config.Cols+columnOf(w.PickX(0))] = true
	}
	if len(layouts) < 10 {
		t.Errorf("the first fruit lay in %d places in 50 runs: want it to move about", len(layouts))
	}
}

// TestAutopilotOnEmptyRoads: the autopilot plays the level through,
// collecting fruit ahead of the sweepers and never losing a life (DESIGN
// §6.7).
func TestAutopilotOnEmptyRoads(t *testing.T) {
	for seed := range uint64(6) {
		w := roadsWorld(t, seed)
		slips, swept := 0, 0
		for !w.Cleared {
			for _, e := range w.Update(Autopilot{}.Next(w)) {
				switch e {
				case Slipped:
					slips++
				case FruitSwept:
					swept++
				}
			}
		}
		t.Logf("seed %d: %d of %d fruit, %d swept, %d slips, %d s left, score %d", seed, w.Stats.Picked,
			len(w.Picks), swept, slips, w.TicksLeft/config.TPS, w.Session.Score)
		if w.Stats.Picked < len(w.Picks)/2 || w.Session.Lives != config.StartLives {
			t.Errorf("seed %d: picked %d, %d lives", seed, w.Stats.Picked, w.Session.Lives)
		}
	}
}
