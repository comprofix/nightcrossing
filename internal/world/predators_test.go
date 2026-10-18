package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// stalkWorld is riverWorld with a still long log at x0 and a stalking
// heron on cell cell of it, facing dir, due to step next tick and not to
// strike for a long while.
func stalkWorld(cell, dir int, strike bool) *World {
	const x0 = 8 * cellUnits
	w := riverWorld(config.LogLong, false, x0, 1, 0)
	lane := &w.Lanes[riverRow]
	lane.Stalk, lane.Strike = true, strike
	e := &lane.Entities[0]
	e.log, e.Heron, e.HeronCell, e.HeronDir = true, true, cell, dir
	e.heronStep, e.strikeIn = 1, 1_000_000
	return w
}

// TestHeronStalks: a stalking heron steps a cell along its log every
// config.HeronStepTicks, standing and then walking smoothly into the next
// cell over config.HeronStrideTicks, and turns round at the ends (D67).
func TestHeronStalks(t *testing.T) {
	w := stalkWorld(0, 1, false)
	lane := &w.Lanes[riverRow]
	e := &lane.Entities[0]
	want := []int{1, 2, 3, 4, 3, 2, 1, 0, 1, 2}
	stride := (cellUnits + config.HeronStrideTicks - 1) / config.HeronStrideTicks
	lane.update(w.rng)
	for i, c := range want {
		x := e.HeronX()
		for range config.HeronStrideTicks {
			lane.update(w.rng)
			if d := e.HeronX() - x; d == 0 || d > stride || d < -stride {
				t.Fatalf("step %d: heron moved %d units in a tick, want 1–%d", i+1, d, stride)
			}
			x = e.HeronX()
		}
		if e.HeronCell != c || e.HeronX() != c*cellUnits {
			t.Fatalf("step %d: heron on cell %d at %d, want cell %d", i+1, e.HeronCell, e.HeronX(), c)
		}
		if f := e.HeronCell + e.HeronDir; f < 0 || f >= e.cells() {
			t.Fatalf("step %d: heron on cell %d faces off the log", i+1, e.HeronCell)
		}
		for range config.HeronStepTicks - config.HeronStrideTicks {
			lane.update(w.rng)
			if e.HeronX() != x {
				t.Fatalf("step %d: moved while standing", i+1)
			}
		}
	}
}

// TestHeronStrikes: where herons strike, it stops, cocks its head for
// config.HeronCockTicks, then jabs the cell it faces for
// config.HeronJabTicks; only then is that cell deadly. Then it waits to
// strike again (D67).
func TestHeronStrikes(t *testing.T) {
	w := stalkWorld(1, 1, true)
	lane := &w.Lanes[riverRow]
	e := &lane.Entities[0]
	e.heronStep, e.strikeIn = config.HeronStepTicks, 1
	jabCell := func() bool {
		l := e.X() + 2*cellUnits
		return lane.heronIn(l+config.HeroMargin*config.SubPx, l+cellUnits-config.HeroMargin*config.SubPx)
	}
	lane.update(w.rng)
	if e.Cock != config.HeronCockTicks || jabCell() {
		t.Fatalf("cock %d, jab cell deadly %v: want the cock begun, the cell safe", e.Cock, jabCell())
	}
	for range config.HeronCockTicks {
		lane.update(w.rng)
	}
	if e.Jab != config.HeronJabTicks || !jabCell() {
		t.Fatalf("jab %d, jab cell deadly %v: want the jab", e.Jab, jabCell())
	}
	for range config.HeronJabTicks {
		lane.update(w.rng)
	}
	if e.Jab != 0 || jabCell() || e.HeronCell != 1 {
		t.Fatalf("after the jab: jab %d, deadly %v, cell %d", e.Jab, jabCell(), e.HeronCell)
	}
	if e.strikeIn < config.HeronStrikeMinTicks || e.strikeIn > config.HeronStrikeMaxTicks {
		t.Errorf("next strike in %d ticks, want %d–%d", e.strikeIn, config.HeronStrikeMinTicks, config.HeronStrikeMaxTicks)
	}
}

// TestAStalkingHeronCatches: a heron walking into the hero's cell catches
// it.
func TestAStalkingHeronCatches(t *testing.T) {
	w := stalkWorld(1, 1, false)
	standOn(w, w.Lanes[riverRow].Entities[0].X()+2*cellUnits)
	if ev := run(w, None, 1+config.HeronStrideTicks); !slices.Contains(ev, Caught) {
		t.Errorf("events %v: want the hero caught", ev)
	}
}

// TestStrollHeronsStand: Stroll's herons never move, and face the way
// their logs go (DESIGN §5.2, D34).
func TestStrollHeronsStand(t *testing.T) {
	w := New(session.New(config.Stroll, 3), config.Stroll.Levels.Level(8), rand.New(rand.NewPCG(4, 4)))
	seen := 0
	for range 6000 {
		before := map[int]int{}
		for r, l := range w.Lanes {
			for i, e := range l.Entities {
				if e.Heron {
					before[r*100+i] = e.HeronCell
				}
			}
		}
		w.Update(None)
		for r, l := range w.Lanes {
			for i, e := range l.Entities {
				if !e.Heron {
					continue
				}
				seen++
				if c, ok := before[r*100+i]; ok && c != e.HeronCell {
					t.Fatalf("row %d: a Stroll heron moved from cell %d to %d", r, c, e.HeronCell)
				}
				if e.HeronDir != l.Dir || e.Cock > 0 || e.Jab > 0 {
					t.Fatalf("row %d: a Stroll heron facing %d, cock %d, jab %d", r, e.HeronDir, e.Cock, e.Jab)
				}
			}
		}
		w.Player.Death, w.Dying, w.TicksLeft = 0, 0, 1000
	}
	if seen == 0 {
		t.Error("no herons on Stroll's level 8")
	}
}

// foxWorld is bankWorld with the fox on.
func foxWorld(col int) *World {
	w := bankWorld(col)
	w.Level.Fox = true
	w.newFox()
	return w
}

// TestFoxVisits: the fox's eyes glint in an empty nook, then the fox is
// there, then it goes for a while (D68).
func TestFoxVisits(t *testing.T) {
	w := foxWorld(0)
	w.Fox.ticks = 1
	if ev := w.Update(None); !slices.Contains(ev, EyesGlint) || w.Fox.Phase != FoxEyes || w.Fox.Nook < 0 {
		t.Fatalf("events %v, fox %+v: want its eyes in a nook", ev, w.Fox)
	}
	n := w.Fox.Nook
	if ev := run(w, None, config.FoxEyesTicks); !slices.Contains(ev, FoxIn) || !w.foxIn(n) {
		t.Fatalf("events %v, fox %+v: want the fox in nook %d", ev, w.Fox, n)
	}
	run(w, None, config.FoxOutTicks)
	if w.Fox.Phase != FoxAway || w.Fox.Nook != -1 {
		t.Fatalf("fox %+v: want it gone", w.Fox)
	}
	lo, hi := config.FoxGoneTicks+config.FoxQuietMinTicks, config.FoxGoneTicks+config.FoxQuietMaxTicks
	if w.Fox.ticks < lo-1 || w.Fox.ticks > hi {
		t.Errorf("away for %d ticks, want %d–%d", w.Fox.ticks, lo, hi)
	}
}

// TestFoxKeepsToEmptyNooks: the fox and fruit never share a nook, and the
// fox never comes to a filled one (D68).
func TestFoxKeepsToEmptyNooks(t *testing.T) {
	w := foxWorld(0)
	w.Level.FruitPts = 200
	w.newFruit()
	w.Filled[1], w.Filled[3] = true, true
	visits := 0
	for range 20_000 {
		w.Update(None)
		if w.Fox.Phase == FoxEyes && w.Fox.ticks == config.FoxEyesTicks-1 {
			visits++
		}
		if n := w.Fox.Nook; n >= 0 && (w.Filled[n] || n == w.FruitNook) {
			t.Fatalf("fox in nook %d: filled %v, fruit in %d", n, w.Filled, w.FruitNook)
		}
		w.TicksLeft = 1000
	}
	if visits == 0 {
		t.Error("the fox never came")
	}
}

// TestFoxHours: with the day and night cycle the fox comes only at dusk
// and night (D39, D68).
func TestFoxHours(t *testing.T) {
	for _, c := range []struct {
		filled int
		comes  bool
	}{{0, false}, {2, false}, {3, true}, {4, true}} {
		w := foxWorld(0)
		w.Level.DayCycle = true
		for i := range c.filled {
			w.Filled[i] = true
		}
		came := false
		for range 3000 {
			w.Update(None)
			came = came || w.Fox.Phase != FoxAway
			w.TicksLeft = 1000
		}
		if came != c.comes {
			t.Errorf("%d nooks filled: fox came %v, want %v", c.filled, came, c.comes)
		}
	}
}

// TestHoppingIntoTheFoxsNook: with only its eyes showing, the nook fills
// and the fox slinks off; with the fox there, the hero is snatched and
// the nook stays empty (D68).
func TestHoppingIntoTheFoxsNook(t *testing.T) {
	for _, phase := range []FoxPhase{FoxEyes, FoxThere} {
		w := foxWorld(config.Nooks[2])
		w.Fox = Fox{Nook: 2, Phase: phase, ticks: 1000}
		lives := w.Session.Lives
		ev := run(w, Up, config.HopTicks)
		switch phase {
		case FoxEyes:
			if !slices.Contains(ev, Goal) || !w.Filled[2] || w.Fox.Phase != FoxAway {
				t.Errorf("eyes: events %v, filled %v, fox %+v: want the nook filled and the fox gone", ev, w.Filled, w.Fox)
			}
		case FoxThere:
			if !slices.Contains(ev, Snatched) || w.Filled[2] || w.Session.Lives != lives-1 {
				t.Errorf("fox: events %v, filled %v, lives %d: want the hero snatched", ev, w.Filled, w.Session.Lives)
			}
		}
	}
}

// TestNoFoxWithoutTheLevel: levels without the fox never show it.
func TestNoFoxWithoutTheLevel(t *testing.T) {
	w := bankWorld(0)
	for range 5000 {
		w.Update(None)
		if w.Fox.Phase != FoxAway || w.Fox.Nook != -1 {
			t.Fatalf("fox %+v on a level without it", w.Fox)
		}
		w.TicksLeft = 1000
	}
}
