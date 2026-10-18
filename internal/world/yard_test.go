package world

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// homeWorld is Last Train's bonus level, the last train home, from a
// seed.
func homeWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	lv, ok := config.BonusLevel(config.LastTrain)
	if !ok {
		t.Fatal("Last Train has no bonus level")
	}
	return New(session.New(config.LastTrain, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
}

// rakeOn is the index in w.Yard of the rake on row r that starts at col.
func rakeOn(t *testing.T, w *World, r, col int) int {
	t.Helper()
	for i, k := range w.Yard {
		if k.Row == r && k.X == col*cellUnits {
			return i
		}
	}
	t.Fatalf("no rake on row %d at column %d", r, col)
	return -1
}

// TestTheLastTrainHomeStarts: dawn, the rakes standing where they start,
// the last train on the main line, and all twelve fruit showing, laid at
// random each run: each on its side, on a log on the river, or on the
// rails or a carriage in the yard, never an engine nor rails a rake
// shunts over; none next to another, nor near the toad, and one at most
// on each log or carriage (DESIGN §6.7).
func TestTheLastTrainHomeStarts(t *testing.T) {
	layouts := map[string]bool{}
	for seed := range uint64(100) {
		w := homeWorld(t, seed)
		b := w.Level.Bonus
		if !w.Level.DayCycle || len(w.Swarm) != 0 || len(w.Yard) != len(b.Yard) || len(w.Picks) != len(b.Fruit) {
			t.Fatalf("dawn %v, %d fireflies, %d rakes, %d fruit", w.Level.DayCycle, len(w.Swarm), len(w.Yard), len(w.Picks))
		}
		if k := w.LastTrain(); k == nil || k.Row != 7 || k.Length() != config.Train.Length*config.SubPx {
			t.Fatalf("the last train %+v", k)
		}
		var layout strings.Builder
		for i, f := range b.Fruit {
			p := w.Picks[i]
			x, mid := w.PickX(i), w.PickX(i)+cellUnits/2
			fmt.Fprintf(&layout, "%d,%d ", columnOf(mid), p.Row)
			if !w.Showing(i) || p.Fruit != f.Fruit || (w.Lanes[p.Row].Kind == config.River) != f.River {
				t.Fatalf("seed %d: fruit %d is %d on row %d, showing %v", seed, i, p.Fruit, p.Row, w.Showing(i))
			}
			switch {
			case f.River:
				e := w.Lanes[p.Row].Entities[p.host]
				if p.host < 0 || x < e.X() || x+cellUnits > e.X()+e.Kind.Length*config.SubPx {
					t.Errorf("seed %d: fruit %d isn't on a log", seed, i)
				}
			case p.rake >= 0:
				if w.carriageAt(p.Row, mid) != p.rake || w.engineAt(p.Row, mid) {
					t.Errorf("seed %d: fruit %d isn't on a carriage", seed, i)
				}
			case w.carriageAt(p.Row, mid) >= 0 || w.engineAt(p.Row, mid) || w.shuntedOver(p.Row, mid):
				t.Errorf("seed %d: fruit %d on the rails is under a rake, or in one's way", seed, i)
			}
			if abs(p.Row-config.StartRow) <= config.BonusClearCells && abs(columnOf(x)-config.StartCol) <= config.BonusClearCells {
				t.Errorf("seed %d: fruit %d is by the toad's start", seed, i)
			}
			for j, o := range w.Picks[:i] {
				if abs(o.Row-p.Row) < config.BonusFruitApart && abs(w.PickX(j)-x) < config.BonusFruitApart*cellUnits {
					t.Errorf("seed %d: fruit %d and %d lie next to each other", seed, j, i)
				}
				if o.Row == p.Row && (p.host >= 0 && o.host == p.host || p.rake >= 0 && o.rake == p.rake && carriage(&o) == carriage(&p)) {
					t.Errorf("seed %d: fruit %d and %d share a log or carriage", seed, j, i)
				}
			}
		}
		layouts[layout.String()] = true
	}
	if len(layouts) < 100 {
		t.Errorf("%d different layouts in 100 runs: want a new one each run", len(layouts))
	}
}

// TestShunting: a shunting rake runs to the far end of its shunt, waits
// there, and comes back; a standing one never moves (DESIGN §6.7).
func TestShunting(t *testing.T) {
	w := homeWorld(t, 1)
	i, still := rakeOn(t, w, 8, 0), rakeOn(t, w, 8, 18)
	far := 8 * cellUnits
	run(w, None, far/config.ShuntSpeed)
	if k := w.Yard[i]; k.X != far || k.wait != config.ShuntPauseTicks {
		t.Fatalf("rake at %d waiting %d, want at %d and waiting %d", k.X, k.wait, far, config.ShuntPauseTicks)
	}
	run(w, None, config.ShuntPauseTicks)
	if w.Yard[i].X != far {
		t.Fatalf("rake moved during its pause, to %d", w.Yard[i].X)
	}
	run(w, None, far/config.ShuntSpeed)
	if k := w.Yard[i]; k.X != 0 {
		t.Errorf("rake back at %d, want 0", k.X)
	}
	if k := w.Yard[still]; k.X != 18*cellUnits {
		t.Errorf("a standing rake moved, to %d", k.X)
	}
}

// onRoof puts the hero on the roof of the carriage at col on row r, as if
// it had hopped there.
func onRoof(t *testing.T, w *World, col, r int) {
	t.Helper()
	w.Player = newPlayer(col, r)
	if w.roof = w.carriageAt(r, col*cellUnits+cellUnits/2); w.roof < 0 {
		t.Fatalf("no carriage at %d,%d", col, r)
	}
}

// TestRidingARake: the toad on a shunting carriage's roof rides it; on
// the rails clear of it, it stays put (DESIGN §6.7).
func TestRidingARake(t *testing.T) {
	w := homeWorld(t, 1)
	onRoof(t, w, 1, 8)
	n := 2 * config.TPS
	ev := run(w, None, n)
	if slices.Contains(ev, Slipped) || w.Player.X != cellUnits+n*config.ShuntSpeed {
		t.Fatalf("events %v, toad at %d: want it carried to %d", ev, w.Player.X, cellUnits+n*config.ShuntSpeed)
	}
	w = homeWorld(t, 1)
	w.Player = newPlayer(17, 8) // on the rails, past where it shunts to
	ev = run(w, None, 20*config.TPS)
	if slices.Contains(ev, Slipped) {
		t.Fatalf("events %v: run over clear of the rake", ev)
	}
	at(t, w, 17, 8)
}

// TestHoppingOntoCarriages: a hop onto a carriage lands on its roof,
// from below or from the side, with no slip and nothing in the way
// (DESIGN §6.7).
func TestHoppingOntoCarriages(t *testing.T) {
	w := homeWorld(t, 1)
	w.Player = newPlayer(5, config.StartRow)
	if ev := run(w, Up, config.HopTicks); slices.Contains(ev, Slipped) || slices.Contains(ev, Blocked) {
		t.Fatalf("events %v: want a plain hop onto the carriage", ev)
	}
	at(t, w, 5, 11)
	if w.roof < 0 {
		t.Error("landed on a carriage, but not on its roof")
	}
	w.Player = newPlayer(3, 11)
	run(w, Right, config.HopTicks)
	at(t, w, 4, 11)
}

// TestEnginesBlock: a hop into an engine is blocked, as into a parked car:
// the toad stays put and nothing is lost (DESIGN §6.7).
func TestEnginesBlock(t *testing.T) {
	w := homeWorld(t, 1)
	w.Player = newPlayer(20, 7) // beside the last train's engine
	before := w.TicksLeft
	ev := run(w, Left, config.HopTicks)
	if !slices.Contains(ev, Blocked) || slices.Contains(ev, Hopped) || slices.Contains(ev, Slipped) {
		t.Fatalf("events %v: want the hop blocked", ev)
	}
	at(t, w, 20, 7)
	if w.TicksLeft != before-config.HopTicks {
		t.Errorf("clock %d, want %d", w.TicksLeft, before-config.HopTicks)
	}
}

// TestRunOver: on the rails in a shunting rake's way, the toad is run
// over, a slip back to the start: by its engine leading, and by its
// carriages as the engine pushes them back (DESIGN §6.7).
func TestRunOver(t *testing.T) {
	w := homeWorld(t, 1)
	w.Player = newPlayer(12, 8) // ahead of the engine
	if ev := run(w, None, 4*config.TPS); !slices.Contains(ev, Slipped) || w.Session.Lives != config.StartLives {
		t.Fatalf("events %v: want run over by the engine, a slip", ev)
	}
	w = homeWorld(t, 1)
	run(w, None, 8*cellUnits/config.ShuntSpeed) // at the far end of its shunt
	w.Player = newPlayer(3, 8)                  // behind it, as it comes back carriages first
	if ev := run(w, None, 8*config.TPS); !slices.Contains(ev, Slipped) {
		t.Fatalf("events %v: want run over by the carriages, a slip", ev)
	}
}

// TestCarriedOffTheYard: on the roof of the end carriage of a rake that
// pushes it off the field, it is a slip once the toad's middle goes over
// the edge (DESIGN §6.7).
func TestCarriedOffTheYard(t *testing.T) {
	w := homeWorld(t, 1)
	onRoof(t, w, config.Cols-1, 10)
	ev := run(w, None, cellUnits/2/config.ShuntSpeed+1)
	if !slices.Contains(ev, Slipped) || w.Session.Lives != config.StartLives {
		t.Fatalf("events %v, %d lives: want a slip", ev, w.Session.Lives)
	}
}

// TestPickingFruit: the toad landing on fruit collects it, for its
// fruit's points; the whole set ends the level early with the points for
// all and the time bonus (DESIGN §6.7).
func TestPickingFruit(t *testing.T) {
	w := homeWorld(t, 1)
	for i := range w.Picks {
		w.Picks[i].Taken = true
	}
	ground := func(col, r int) Pick { return Pick{Fruit: 0, Row: r, rake: -1, host: -1, offset: col * cellUnits} }
	w.Picks[0], w.Picks[1] = ground(9, 11), ground(18, 9) // strawberries on the rails
	w.Player = newPlayer(9, config.StartRow)
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, FruitTaken) || w.Stats.Picked != 1 || w.Session.Score != config.BonusFruitPoints[0] {
		t.Fatalf("events %v, %d picked, score %d", ev, w.Stats.Picked, w.Session.Score)
	}
	if w.Showing(0) {
		t.Error("the strawberries still show")
	}
	w.Stats.Picked = len(w.Picks) - 1 // all but the strawberries at 18,9
	onRoof(t, w, 18, 10)
	secs := (w.TicksLeft - config.HopTicks) / config.TPS
	ev = run(w, Up, config.HopTicks)
	if !slices.Contains(ev, AllCaught) || !w.Cleared || w.Stats.All != w.Level.Bonus.All ||
		w.Stats.TimeBonus != secs*config.PointsBonusSec {
		t.Errorf("events %v, cleared %v, all %d, time bonus %d", ev, w.Cleared, w.Stats.All, w.Stats.TimeBonus)
	}
}

// TestTheLastTrainDeparts: as the clock runs low the last train is about
// to leave (DESIGN §6.7).
func TestTheLastTrainDeparts(t *testing.T) {
	w := homeWorld(t, 1)
	if w.Departing() {
		t.Fatal("departing at the start")
	}
	w.TicksLeft = config.LowTimeSecs*config.TPS + 1
	run(w, None, 1)
	if !w.Departing() {
		t.Error("not departing with the clock low")
	}
	if nightWorld(t, 1).Departing() {
		t.Error("firefly night has a last train")
	}
}

// TestAutopilotOnTheLastTrainHome: the autopilot plays the level through,
// collecting fruit and never losing a life (DESIGN §6.7).
func TestAutopilotOnTheLastTrainHome(t *testing.T) {
	for seed := range uint64(3) {
		w := homeWorld(t, seed)
		slips := 0
		for !w.Cleared {
			for _, e := range w.Update(Autopilot{}.Next(w)) {
				if e == Slipped {
					slips++
				}
			}
		}
		t.Logf("seed %d: %d of %d fruit, %d slips, %d s left, score %d", seed, w.Stats.Picked,
			len(w.Picks), slips, w.TicksLeft/config.TPS, w.Session.Score)
		if w.Stats.Picked < len(w.Picks)/2 || w.Session.Lives != config.StartLives {
			t.Errorf("seed %d: picked %d, %d lives", seed, w.Stats.Picked, w.Session.Lives)
		}
	}
}
