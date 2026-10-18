package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// nightWorld is Stroll's bonus level, firefly night, from a seed.
func nightWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	lv, ok := config.BonusLevel(config.Stroll)
	if !ok {
		t.Fatal("Stroll has no bonus level")
	}
	return New(session.New(config.Stroll, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
}

// visiting is how many of w's fireflies are on the field.
func visiting(w *World) int {
	n := 0
	for _, f := range w.Swarm {
		if f.Visiting {
			n++
		}
	}
	return n
}

// TestFireflyNightStarts: five fireflies are landed on the field at the
// start, each on a road gap the toad can reach or on a log, none near the
// toad and no two together (D61).
func TestFireflyNightStarts(t *testing.T) {
	for seed := range uint64(20) {
		w := nightWorld(t, seed)
		b := w.Level.Bonus
		if len(w.Swarm) != b.AtOnce || visiting(w) != b.AtOnce || w.sent != b.AtOnce {
			t.Fatalf("seed %d: %d fireflies, %d visiting, %d sent for; want %d", seed, len(w.Swarm), visiting(w), w.sent, b.AtOnce)
		}
		if w.TicksLeft != b.Secs*config.TPS {
			t.Errorf("seed %d: clock %d, want %d s", seed, w.TicksLeft, b.Secs)
		}
		seen := map[[2]int]bool{}
		for _, f := range w.Swarm {
			col := columnOf(f.X)
			if !f.Landed {
				t.Errorf("seed %d: a firefly is not landed at the start", seed)
			}
			switch w.Lanes[f.row].Kind {
			case config.Road:
				if w.Parked(col, f.row) || !w.Reachable(col, f.row) {
					t.Errorf("seed %d: a firefly in a parked car or a gap out of reach, %d,%d", seed, col, f.row)
				}
			case config.River:
				if e := w.Lanes[f.row].Entities[f.host]; !slices.Contains(config.Logs(), e.Kind) {
					t.Errorf("seed %d: a firefly on a %s, want logs only", seed, e.Kind.ID)
				}
			default:
				t.Errorf("seed %d: a firefly on row %d, neither road nor river", seed, f.row)
			}
			if abs(col-config.StartCol) <= config.BonusClearCells && abs(f.row-config.StartRow) <= config.BonusClearCells {
				t.Errorf("seed %d: a firefly at %d,%d, within %d cells of the toad", seed, col, f.row, config.BonusClearCells)
			}
			if seen[[2]int{col, f.row}] {
				t.Errorf("seed %d: two fireflies at %d,%d", seed, col, f.row)
			}
			seen[[2]int{col, f.row}] = true
		}
	}
}

// TestFireflyNightReach: the toad can get from the start to the verge past
// the parked cars, and to most of the road's gaps.
func TestFireflyNightReach(t *testing.T) {
	w := nightWorld(t, 1)
	if !w.Reachable(config.StartCol, config.VergeRow) {
		t.Fatal("the verge is out of reach of the start")
	}
	gaps, out := 0, 0
	for r := range w.Lanes {
		if w.Lanes[r].Kind != config.Road {
			continue
		}
		for c := range w.Cols {
			if !w.Parked(c, r) {
				gaps++
				if !w.Reachable(c, r) {
					out++
				}
			}
		}
	}
	if out*4 > gaps {
		t.Errorf("%d of %d road gaps out of reach", out, gaps)
	}
}

// TestParkedCarsBlock: a hop into a parked car is refused; the toad stays
// put, losing nothing (D61).
func TestParkedCarsBlock(t *testing.T) {
	w := nightWorld(t, 1)
	b := w.Level.Bonus
	col := -1
	for c := range w.Cols {
		if b.Parked[11][c] != '.' && b.Parked[11][c] != 0 {
			col = c
			break
		}
	}
	w.Player = newPlayer(col, config.StartRow)
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, Blocked) || slices.Contains(ev, Hopped) {
		t.Fatalf("events %v: want the hop blocked", ev)
	}
	at(t, w, col, config.StartRow)
	if w.Dying != 0 || w.Session.Lives != config.StartLives {
		t.Errorf("a blocked hop cost something: dying %d, %d lives", w.Dying, w.Session.Lives)
	}
}

// TestASlip: a hop into the water is a slip: no life lost, the toad back
// at the start row a second later, the clock running all through, and the
// fireflies caught still caught (D61).
func TestASlip(t *testing.T) {
	w := nightWorld(t, 1)
	w.Stats.Fireflies = 2
	river := config.VergeRow - 1
	w.Lanes[river].Speed = 0
	for c := range w.Cols {
		if !w.Lanes[river].holds(c*cellUnits + cellUnits/2) {
			w.Player = newPlayer(c, config.VergeRow)
			break
		}
	}
	before := w.TicksLeft
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, Slipped) || slices.Contains(ev, Splashed) {
		t.Fatalf("events %v: want a slip, and no death", ev)
	}
	if w.Session.Lives != config.StartLives {
		t.Errorf("%d lives after a slip, want %d", w.Session.Lives, config.StartLives)
	}
	ev = run(w, None, config.SlipTicks)
	if !slices.Contains(ev, Respawned) || w.Player.Row != config.StartRow {
		t.Errorf("events %v, row %d: want the toad back on the start row", ev, w.Player.Row)
	}
	if want := before - config.HopTicks - config.SlipTicks; w.TicksLeft != want {
		t.Errorf("clock %d after the slip, want %d: it never stops", w.TicksLeft, want)
	}
	if w.Stats.Fireflies != 2 {
		t.Errorf("%d fireflies after the slip, want the 2 caught", w.Stats.Fireflies)
	}
}

// TestABonusNook: a nook pays its points and sends the toad back to the
// start row, and never fills (D61).
func TestABonusNook(t *testing.T) {
	w := nightWorld(t, 1)
	w.Player = newPlayer(config.Nooks[0], 1)
	w.Lanes[1], w.Swarm = Lane{Kind: config.Safe}, nil // no river under it, for the test
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, Goal) || w.Session.Score != config.PointsBonusNook {
		t.Fatalf("events %v, score %d: want the nook's %d", ev, w.Session.Score, config.PointsBonusNook)
	}
	if w.Filled[0] || w.Cleared {
		t.Error("the nook filled, or the level ended")
	}
	if w.HomeNook() != 0 {
		t.Errorf("toad showing in nook %d, want 0", w.HomeNook())
	}
	run(w, None, config.GoalTicks)
	if w.Player.Row != config.StartRow {
		t.Errorf("row %d after the nook, want the start row", w.Player.Row)
	}
}

// catchAt has the toad catch firefly i where it is, settling it first if
// it is still flying in, and is the events.
func catchAt(w *World, i int) []Event {
	f := &w.Swarm[i]
	f.X, f.Y, f.Landed = w.spotX(f), f.row*cellUnits, true
	w.events = w.events[:0]
	w.catchSwarm(f.row, f.X, f.X+cellUnits)
	return slices.Clone(w.events)
}

// TestCatchInARoadGap: a hop that lands on a firefly in a road gap catches
// it (D61).
func TestCatchInARoadGap(t *testing.T) {
	w := nightWorld(t, 1)
	const col, row = 2, 9 // a gap with a gap below it
	if w.Parked(col, row) || w.Parked(col, row+1) {
		t.Fatalf("cells %d,%d and below are not both gaps", col, row)
	}
	w.Swarm[0] = Flyer{Visiting: true, Landed: true, X: col * cellUnits, Y: row * cellUnits, row: row, host: -1, offset: col * cellUnits}
	w.Player = newPlayer(col, row+1)
	if ev := run(w, Up, config.HopTicks); !slices.Contains(ev, FireflyCaught) {
		t.Fatalf("events %v: want the firefly caught", ev)
	}
	if w.Stats.Fireflies != 1 || w.Session.Score != w.Level.Bonus.Firefly {
		t.Errorf("caught %d, score %d", w.Stats.Fireflies, w.Session.Score)
	}
}

// TestCatchingTheSwarm: each firefly caught pays its points and no life,
// and the next one comes in 2–3 s later, until all ten have come; the
// last ends the level early with the all-caught points and the time
// bonus (D61).
func TestCatchingTheSwarm(t *testing.T) {
	w := nightWorld(t, 3)
	b := w.Level.Bonus
	w.Session.Score = config.ExtraLifeEvery - 10 // a catch would pass a life's points
	score := w.Session.Score
	for n := 1; n <= b.Fireflies; n++ {
		i := slices.IndexFunc(w.Swarm, func(f Flyer) bool { return f.Visiting })
		for i < 0 {
			run(w, None, 1)
			i = slices.IndexFunc(w.Swarm, func(f Flyer) bool { return f.Visiting })
		}
		sent := w.sent
		ev := catchAt(w, i)
		if !slices.Contains(ev, FireflyCaught) || w.Stats.Fireflies != n {
			t.Fatalf("catch %d: events %v, caught %d", n, ev, w.Stats.Fireflies)
		}
		if slices.Contains(ev, ExtraLife) || w.Session.Lives != config.StartLives {
			t.Fatalf("catch %d gave a life", n)
		}
		score += b.Firefly
		if n == b.Fireflies {
			break
		}
		if w.Session.Score != score {
			t.Fatalf("catch %d: score %d, want %d", n, w.Session.Score, score)
		}
		due := w.Swarm[i].due
		if sent < b.Fireflies && (due < config.BonusComeMinTicks || due > config.BonusComeMaxTicks || w.sent != sent+1) {
			t.Fatalf("catch %d: next due in %d ticks, want 2–3 s", n, due)
		}
		if sent >= b.Fireflies && (due != 0 || w.sent != sent) {
			t.Fatalf("catch %d: another due after all %d have come", n, b.Fireflies)
		}
	}
	if !w.Cleared || w.Stats.All != b.All {
		t.Fatalf("all caught: cleared %v, all-caught points %d", w.Cleared, w.Stats.All)
	}
	secs := w.TicksLeft / config.TPS
	if w.Stats.TimeBonus != secs*config.PointsBonusSec || w.Session.Score != score+b.All+secs*config.PointsBonusSec {
		t.Errorf("time bonus %d, score %d; want %d s at %d", w.Stats.TimeBonus, w.Session.Score, secs, config.PointsBonusSec)
	}
	if w.sent != b.Fireflies {
		t.Errorf("%d fireflies sent for, want %d", w.sent, b.Fireflies)
	}
}

// TestBonusTimeUp: the clock runs out, ending the level with no all-caught
// points or time bonus, and no life lost (D61).
func TestBonusTimeUp(t *testing.T) {
	w := nightWorld(t, 1)
	ev := run(w, None, w.TicksLeft)
	if !slices.Contains(ev, LowTime) || !slices.Contains(ev, BonusOver) || !w.Cleared {
		t.Fatalf("want the low time warning and the end; cleared %v", w.Cleared)
	}
	if w.Stats.All != 0 || w.Stats.TimeBonus != 0 || w.Session.Lives != config.StartLives || w.Over {
		t.Errorf("stats %+v, %d lives, over %v", w.Stats, w.Session.Lives, w.Over)
	}
	if ev := run(w, Up, 10); len(ev) != 0 {
		t.Errorf("events %v after the end", ev)
	}
}

// TestFireflyNightLogsCarry: a firefly on a log rides it, and lifts off
// for a new spot as it nears the edge, so none is carried off (D61).
func TestFireflyNightLogsCarry(t *testing.T) {
	w := nightWorld(t, 4)
	for range 60 * config.TPS {
		w.Update(None)
		for _, f := range w.Swarm {
			if f.Visiting && f.Landed && (f.X < 0 || f.X+cellUnits > fieldWidth) {
				t.Fatalf("tick %d: a landed firefly at %d, off the field", w.Tick, f.X)
			}
		}
		if w.Cleared {
			break
		}
	}
}

// TestAutopilotOnFireflyNight: the autopilot plays the bonus level
// through, catching fireflies and never losing a life (D61).
func TestAutopilotOnFireflyNight(t *testing.T) {
	for seed := range uint64(3) {
		w := nightWorld(t, seed)
		slips := 0
		for !w.Cleared {
			for _, e := range w.Update(Autopilot{}.Next(w)) {
				if e == Slipped {
					slips++
				}
			}
		}
		t.Logf("seed %d: %d of %d caught, %d slips, %d s left, score %d", seed, w.Stats.Fireflies,
			w.Level.Bonus.Fireflies, slips, w.TicksLeft/config.TPS, w.Session.Score)
		if w.Stats.Fireflies < w.Level.Bonus.Fireflies/2 || w.Session.Lives != config.StartLives {
			t.Errorf("seed %d: caught %d, %d lives", seed, w.Stats.Fireflies, w.Session.Lives)
		}
	}
}

// TestNeverMoreThanTen: catching fireflies as fast as they land, never
// more than the set's ten come, however quickly the catches run (D61).
func TestNeverMoreThanTen(t *testing.T) {
	for seed := range uint64(10) {
		w := nightWorld(t, seed)
		b := w.Level.Bonus
		came := visiting(w)
		was := make([]bool, len(w.Swarm))
		for i, f := range w.Swarm {
			was[i] = f.Visiting
		}
		for !w.Cleared && w.TicksLeft > 1 {
			for i := range w.Swarm {
				if w.Swarm[i].Visiting {
					catchAt(w, i)
					was[i] = false
				}
			}
			w.Update(None)
			for i, f := range w.Swarm {
				if f.Visiting && !was[i] {
					came++
				}
				was[i] = f.Visiting
			}
		}
		if came != b.Fireflies || w.Stats.Fireflies != b.Fireflies {
			t.Errorf("seed %d: %d came, %d caught; want %d", seed, came, w.Stats.Fireflies, b.Fireflies)
		}
	}
}

// TestFirefliesShareAndStay: of the set, the road's share come to the road
// and the rest to the river, mixed as they come; each stays on the side it came to
// until caught, and none lands next to another, even corner to corner
// (D61).
func TestFirefliesShareAndStay(t *testing.T) {
	onRoad := func(w *World, f Flyer) bool { return w.Lanes[f.row].Kind == config.Road }
	for seed := range uint64(10) {
		w := nightWorld(t, seed)
		b := w.Level.Bonus
		visit := make([]bool, len(w.Swarm))
		landed := make([]bool, len(w.Swarm))
		road := make([]bool, len(w.Swarm))
		came := map[bool]int{}
		for i, f := range w.Swarm {
			if f.Visiting {
				visit[i], landed[i], road[i] = true, f.Landed, onRoad(w, f)
				came[road[i]]++
			}
		}
		for tick := 0; !w.Cleared && w.TicksLeft > 1; tick++ {
			if tick%config.TPS == 0 { // a catch now and then, so more come
				if i := slices.IndexFunc(w.Swarm, func(f Flyer) bool { return f.Visiting && f.Landed }); i >= 0 {
					catchAt(w, i)
					visit[i], landed[i] = false, false
				}
			}
			w.Update(None)
			for i, f := range w.Swarm {
				switch {
				case !f.Visiting:
				case !visit[i]: // it has just come in
					road[i] = onRoad(w, f)
					came[road[i]]++
				case onRoad(w, f) != road[i]:
					t.Fatalf("seed %d, tick %d: a firefly left the road %v for the other side", seed, tick, road[i])
				}
				if f.Visiting && f.Landed && !landed[i] { // it has just landed
					for j, o := range w.Swarm {
						if j != i && o.Visiting && o.Landed && abs(o.row-f.row) <= 1 && abs(o.X-f.X) < 2*cellUnits {
							t.Fatalf("seed %d, tick %d: fireflies landed next to each other at %d,%d and %d,%d",
								seed, tick, columnOf(f.X), f.row, columnOf(o.X), o.row)
						}
					}
				}
				visit[i], landed[i] = f.Visiting, f.Visiting && f.Landed
			}
		}
		if came[true] != b.OnRoad || came[false] != b.Fireflies-b.OnRoad {
			t.Errorf("seed %d: %d came to the road and %d to the river, want %d and %d",
				seed, came[true], came[false], b.OnRoad, b.Fireflies-b.OnRoad)
		}
	}
}
