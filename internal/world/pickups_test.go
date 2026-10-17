package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// landedOn is riverWorld with platform p at x, moving right at speed, and
// a firefly landed on it offset from its left end, staying a good while.
// Fireflies may land on every kind of platform.
func landedOn(p config.Entity, dives bool, x, speed, offset int) *World {
	w := riverWorld(p, dives, x, 1, speed)
	w.Level.FireflyOn = [config.NumHosts]bool{true, true, true}
	w.Flyer = Flyer{Visiting: true, Landed: true, Facing: 1, X: x + offset, Y: riverRow * cellUnits,
		row: riverRow, offset: offset, visit: 20 * config.TPS, sit: 10 * config.TPS}
	return w
}

// fireflyWorld is a still log in columns 8–10 with a firefly landed in
// column 9, and the hero below it in column col.
func fireflyWorld(col, lives int) *World {
	w := landedOn(config.LogShort, false, 8*cellUnits, 0, cellUnits)
	w.Player = newPlayer(col, riverRow+1)
	w.Session.Lives = lives
	return w
}

// TestCatchingAFirefly: landing on the firefly's cell catches it, a life at
// once; the rest of its log is just a log (D58).
func TestCatchingAFirefly(t *testing.T) {
	for _, c := range []struct {
		col    int
		caught bool
	}{{8, false}, {9, true}, {10, false}} {
		w := fireflyWorld(c.col, config.StartLives)
		ev := run(w, Up, config.HopTicks)
		if got := slices.Contains(ev, FireflyCaught); got != c.caught {
			t.Fatalf("landing in column %d (firefly in 9): caught %v, events %v", c.col, got, ev)
		}
		if !c.caught {
			continue
		}
		if !slices.Contains(ev, ExtraLife) || w.Session.Lives != config.StartLives+1 {
			t.Errorf("events %v, %d lives: want a life at once", ev, w.Session.Lives)
		}
		if w.Flyer.Visiting || w.Stats.Fireflies != 1 {
			t.Errorf("the firefly is still there, or not counted (%d)", w.Stats.Fireflies)
		}
		if ev := run(w, None, config.TPS); slices.Contains(ev, FireflyCaught) {
			t.Error("caught twice")
		}
	}
}

// TestNoCatchInFlight: a firefly in the air over the hero's cell can't be
// caught, only one that has landed (D58).
func TestNoCatchInFlight(t *testing.T) {
	w := fireflyWorld(9, config.StartLives)
	w.flyAway()
	if ev := run(w, Up, config.HopTicks); slices.Contains(ev, FireflyCaught) {
		t.Errorf("events %v: caught in flight", ev)
	}
}

// TestAFireflyAtTheCap: with lives at the cap, a firefly gives points
// instead (D57, D58).
func TestAFireflyAtTheCap(t *testing.T) {
	w := fireflyWorld(9, config.MaxLives)
	ev := run(w, Up, config.HopTicks)
	if !slices.Contains(ev, FireflyCaught) || slices.Contains(ev, ExtraLife) {
		t.Fatalf("events %v: want the firefly caught and no life", ev)
	}
	if w.Session.Lives != config.MaxLives {
		t.Errorf("%d lives, want the cap, %d", w.Session.Lives, config.MaxLives)
	}
	if want := config.PointsHop + config.PointsFireflyAtCap; w.Session.Score != want || w.Stats.AtCap != config.PointsFireflyAtCap {
		t.Errorf("score %d (at the cap %d), want %d", w.Session.Score, w.Stats.AtCap, want)
	}
}

// TestNoLifeFromPointsAtTheCap: points still pass the threshold, but the
// lives stop at the cap (D57).
func TestNoLifeFromPointsAtTheCap(t *testing.T) {
	for _, lives := range []int{config.MaxLives - 1, config.MaxLives} {
		w := newWorld(1)
		w.Session.Lives = lives
		w.Session.Score = config.ExtraLifeEvery - config.PointsHop
		ev := run(w, Up, config.HopTicks)
		if got, want := slices.Contains(ev, ExtraLife), lives < config.MaxLives; got != want {
			t.Errorf("%d lives: extra life %v, want %v", lives, got, want)
		}
		if w.Session.Lives != config.MaxLives {
			t.Errorf("from %d lives: %d, want the cap", lives, w.Session.Lives)
		}
	}
}

// TestAFireflyLiftsOff: it lifts off at once when its lily pad starts to
// flash, its ship starts to break up, or its platform nears the field's
// edge; but sits on through a ship's flashing (D58).
func TestAFireflyLiftsOff(t *testing.T) {
	cases := []struct {
		name  string
		w     func() *World
		ticks int
		lifts bool
	}{
		{"a pad flashing", func() *World {
			w := landedOn(config.Pads3, true, 8*cellUnits, 0, cellUnits)
			w.Lanes[riverRow].Entities[0].Phase = config.DiveAfloatTicks - 1
			return w
		}, 2, true},
		{"a ship flashing", func() *World {
			w := landedOn(config.Barge, false, 8*cellUnits, 0, cellUnits)
			e := &w.Lanes[riverRow].Entities[0]
			e.Sinks, e.Sink = true, 1
			return w
		}, config.SinkWarnTicks - 2, false},
		{"a ship breaking up", func() *World {
			w := landedOn(config.Barge, false, 8*cellUnits, 0, cellUnits)
			e := &w.Lanes[riverRow].Entities[0]
			e.Sinks, e.Sink = true, config.SinkWarnTicks
			return w
		}, 2, true},
		{"nearing the edge", func() *World {
			return landedOn(config.LogShort, false, fieldWidth-3*cellUnits, config.SubPx, cellUnits)
		}, 2, true},
	}
	for _, c := range cases {
		w := c.w()
		run(w, None, c.ticks)
		if lifted := !w.Flyer.Landed; lifted != c.lifts {
			t.Errorf("%s: lifted off %v, want %v", c.name, lifted, c.lifts)
		}
	}
}

// visitLevel is Stroll's level 8, with every kind of platform, where a
// firefly comes in every crossing.
func visitLevel() config.Level {
	lv := config.StrollLevels.Level(8)
	lv.FireflyPct = 100
	return lv
}

// TestAFireflyVisits plays a firefly's visit with the hero waiting at the
// start: it comes in from the side after 1–3 s, only ever lands on a
// platform that may hold it, sits there and flits about, and flies away
// when its visit is up (D58).
func TestAFireflyVisits(t *testing.T) {
	for seed := range uint64(8) {
		w := New(session.New(config.Stroll, 3), visitLevel(), rand.New(rand.NewPCG(seed, seed)))
		came, gone, landings := 0, 0, 0
		landed := false
		for tick := 1; tick <= 40*config.TPS && gone == 0; tick++ {
			ev := w.Update(None)
			w.TicksLeft = w.Level.TimerTicks
			f := &w.Flyer
			if f.Visiting && came == 0 {
				came = tick
				if tick < config.FireflyComeMinTicks || tick > config.FireflyComeMaxTicks {
					t.Errorf("seed %d: came in at tick %d, want %d–%d", seed, tick, config.FireflyComeMinTicks, config.FireflyComeMaxTicks)
				}
				if f.X > 0 && f.X < fieldWidth-cellUnits {
					t.Errorf("seed %d: came in at %d, not from a side", seed, f.X)
				}
				if f.Day {
					t.Errorf("seed %d: a dragonfly at night", seed)
				}
			}
			if slices.Contains(ev, FireflyGone) {
				gone = tick
			}
			if f.Landed {
				e := w.Lanes[f.row].Entities[f.host]
				if !w.perch(f.row, e, f.offset, 1) || f.X != e.X()+f.offset || f.Y != f.row*cellUnits {
					t.Fatalf("seed %d tick %d: landed on %s at stage %d, at %d,%d", seed, tick, e.Kind.ID, e.Stage(), f.X, f.Y)
				}
				if !landed {
					landings++
				}
			}
			landed = f.Landed
		}
		switch {
		case came == 0:
			t.Errorf("seed %d: no firefly came", seed)
		case gone == 0:
			t.Errorf("seed %d: the firefly never left", seed)
		case landings == 0:
			t.Errorf("seed %d: the firefly never landed", seed)
		case gone-came < config.FireflyVisitMinTicks:
			t.Errorf("seed %d: a visit of %d ticks, want at least %d", seed, gone-came, config.FireflyVisitMinTicks)
		}
		if w.Flyer.Visiting {
			t.Errorf("seed %d: still visiting once gone", seed)
		}
	}
}

// TestFireflyRolls: the chance of a firefly is rolled at the level's
// start, at about the level's chance (D58, D60).
func TestFireflyRolls(t *testing.T) {
	for _, n := range []int{4, 8} {
		lv := config.StrollLevels.Level(n)
		const games = 400
		rolled := 0
		for seed := range uint64(games) {
			w := New(session.New(config.Stroll, 3), lv, rand.New(rand.NewPCG(seed, seed)))
			if w.Flyer.due > 0 {
				rolled++
			}
		}
		pct := 100 * rolled / games
		if want := lv.FireflyPct; pct < want*3/4 || pct > want*5/4 {
			t.Errorf("level %d: %d%% of games rolled a firefly, want about %d%%", n, pct, want)
		}
	}
}

// TestNoRollAfterADeath: dying doesn't roll again, so it can't fish for a
// firefly; but one already visiting carries on (D58).
func TestNoRollAfterADeath(t *testing.T) {
	w := New(session.New(config.Stroll, 3), visitLevel(), rand.New(rand.NewPCG(1, 1)))
	w.Flyer = Flyer{}
	w.die(TimeUp)
	for range config.DeathTicks + 5*config.TPS {
		w.Update(None)
		w.TicksLeft = w.Level.TimerTicks
		if w.Flyer.due > 0 || w.Flyer.Visiting {
			t.Fatal("a firefly rolled for after a death")
		}
	}

	w = fireflyWorld(config.StartCol, config.StartLives)
	w.die(TimeUp)
	run(w, None, config.DeathTicks+config.TPS)
	if !w.Flyer.Visiting {
		t.Error("the firefly left when the hero died")
	}
}

// TestRollAfterANook: filling a nook rolls for a firefly for the next
// crossing (D58).
func TestRollAfterANook(t *testing.T) {
	lv := testLevel()
	lv.Lanes[config.GoalRow] = config.Lane{Kind: config.Bank}
	lv.FireflyPct = 100
	w := New(session.New(config.Stroll, config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
	w.Flyer = Flyer{}
	w.Player = newPlayer(config.StartCol, config.GoalRow+1)
	for range config.HopTicks + 10*config.TPS {
		ev := w.Update(None)
		if slices.Contains(ev, Respawned) {
			if w.Flyer.due == 0 {
				t.Error("no firefly rolled for after a nook")
			}
			return
		}
		if w.Player.Row == config.GoalRow+1 && !w.Player.Hopping() && w.Home == 0 {
			w.Update(Up)
		}
	}
	t.Error("the nook was never filled")
}

// TestADragonflyByDay: by day it is a dragonfly; at dawn, dusk and night,
// a firefly (D58).
func TestADragonflyByDay(t *testing.T) {
	for filled, tod := range config.DayCycle {
		w := New(session.New(config.Stroll, 3), visitLevel(), rand.New(rand.NewPCG(2, 2)))
		w.Level.DayCycle = true
		for n := range filled {
			w.Filled[n] = true
		}
		w.Flyer = Flyer{due: 1}
		w.Update(None)
		if !w.Flyer.Visiting {
			t.Fatalf("%d filled: none came", filled)
		}
		if want := tod == config.Day; w.Flyer.Day != want {
			t.Errorf("%d filled (time %d): a dragonfly %v, want %v", filled, tod, w.Flyer.Day, want)
		}
	}
}

// TestAutopilotTakesAFirefly: a landed firefly a cell off its way up is
// worth the detour to the autopilot (D58).
func TestAutopilotTakesAFirefly(t *testing.T) {
	w := fireflyWorld(8, config.StartLives) // the firefly in column 9
	var a Autopilot
	for range 3 * config.TPS {
		w.Update(a.Next(w))
		w.TicksLeft = w.Level.TimerTicks
	}
	if w.Stats.Fireflies != 1 {
		t.Errorf("the autopilot caught %d fireflies, want the one beside its way", w.Stats.Fireflies)
	}
}

// fruitWorld is an all-safe level with the far bank on row 0 and fruit.
func fruitWorld() *World {
	lv := testLevel()
	lv.Lanes[config.GoalRow] = config.Lane{Kind: config.Bank}
	lv.FruitPts, lv.Fruit = 300, 2
	return New(session.New(config.Stroll, config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
}

// TestFruitComesAndGoes: fruit shows after its wait, in an empty nook, for
// its time, then goes and comes back (D59).
func TestFruitComesAndGoes(t *testing.T) {
	w := fruitWorld()
	only := 3
	for n := range w.Filled {
		w.Filled[n] = n != only
	}
	shown, gone := 0, 0
	for tick := 1; tick <= 4*config.FruitWaitMaxTicks; tick++ {
		before := w.FruitNook
		w.Update(None)
		w.TicksLeft = config.TimerTicks(1)
		switch {
		case before < 0 && w.FruitNook >= 0:
			if w.FruitNook != only {
				t.Fatalf("fruit in nook %d, a filled one", w.FruitNook)
			}
			if shown == 0 && (tick < config.FruitWaitMinTicks || tick > config.FruitWaitMaxTicks+1) {
				t.Errorf("first fruit at tick %d, want after %d–%d", tick, config.FruitWaitMinTicks, config.FruitWaitMaxTicks)
			}
			shown = tick
		case before >= 0 && w.FruitNook < 0:
			if tick-shown != config.FruitShowTicks {
				t.Errorf("fruit showed %d ticks, want %d", tick-shown, config.FruitShowTicks)
			}
			gone++
		}
	}
	if gone < 2 {
		t.Errorf("fruit came and went %d times, want it back again", gone)
	}
}

// TestNoFruitWithoutPoints: a level without fruit never shows any.
func TestNoFruitWithoutPoints(t *testing.T) {
	w := newWorld(1)
	for range 4 * config.FruitWaitMaxTicks {
		if w.Update(None); w.FruitNook >= 0 {
			t.Fatal("fruit on a level without it")
		}
		w.TicksLeft = config.TimerTicks(1)
	}
}

// TestFruitInTheNook: filling the nook with fruit in it scores the fruit
// with the nook, multiplied by the combo (D59).
func TestFruitInTheNook(t *testing.T) {
	for _, combo := range []bool{false, true} {
		w := fruitWorld()
		w.Level.Combo = combo
		n := NookAt(config.StartCol)
		w.FruitNook, w.fruitTicks = n, config.FruitShowTicks
		w.Player = newPlayer(config.StartCol, config.GoalRow+1)
		w.furthest = config.GoalRow + 1
		if combo {
			w.Combo = Combo{Hops: 2*config.ComboStep - 1, Left: config.ComboTicks}
		}
		ev := run(w, Up, config.HopTicks)
		if !slices.Contains(ev, FruitTaken) || w.FruitNook >= 0 || w.Stats.Fruit != w.Level.FruitPts {
			t.Fatalf("combo %v: events %v, fruit %d, stats %d: want the fruit taken", combo, ev, w.FruitNook, w.Stats.Fruit)
		}
		mult := 1
		if combo {
			mult = 3
		}
		secs := w.TicksLeft / config.TPS
		want := (config.PointsHop+config.PointsGoal+w.Level.FruitPts)*mult + secs*config.PointsPerSec
		if w.Session.Score != want {
			t.Errorf("combo %v: score %d, want %d", combo, w.Session.Score, want)
		}
	}
}

// TestFruitElsewhere: filling a nook without the fruit leaves it showing.
func TestFruitElsewhere(t *testing.T) {
	w := fruitWorld()
	n := NookAt(config.StartCol)
	w.FruitNook, w.fruitTicks = (n+1)%len(config.Nooks), config.FruitShowTicks
	w.Player = newPlayer(config.StartCol, config.GoalRow+1)
	if ev := run(w, Up, config.HopTicks); slices.Contains(ev, FruitTaken) || w.FruitNook < 0 {
		t.Errorf("events %v: the fruit in another nook was taken", ev)
	}
}
