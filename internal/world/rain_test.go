package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// rain makes it rain on w from now on.
func rain(w *World) {
	w.Spells = []Spell{{Kind: config.Rain, Timing: config.Events[config.Rain].Top, Phase: SpellOn, Left: 1 << 30}}
}

// wetWorld is a game on an all-safe level whose rows 8–11 are empty road,
// in the rain.
func wetWorld() *World {
	lv := testLevel()
	for r := 8; r <= 11; r++ {
		lv.Lanes[r] = config.Lane{Kind: config.Road}
	}
	w := New(session.New(config.RushHour, config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
	rain(w)
	return w
}

// TestRainSlidesOneMoreCell: a hop that lands on the road slides one more
// cell the same way, and the slide doesn't slide again; the row it slides
// into scores like a row hopped into (D31).
func TestRainSlidesOneMoreCell(t *testing.T) {
	w := wetWorld()
	ev := hops(w, Up, 1)
	ev = append(ev, run(w, None, config.HopTicks+config.LandTicks)...)
	if count(ev, Slid) != 1 || count(ev, Hopped) != 1 {
		t.Fatalf("events %v: want one hop and one slide", ev)
	}
	at(t, w, config.StartCol, config.StartRow-2)
	if w.Player.Facing != Up || w.Player.Sliding {
		t.Errorf("facing %v, sliding %v after the slide", w.Player.Facing, w.Player.Sliding)
	}
	if want := 2 * config.PointsHop; w.Session.Score != want {
		t.Errorf("score %d, want %d for two new rows", w.Session.Score, want)
	}
}

func TestNoSlideWithoutRain(t *testing.T) {
	w := wetWorld()
	w.Spells = nil
	if ev := hops(w, Up, 1); slices.Contains(ev, Slid) {
		t.Errorf("slid without rain: %v", ev)
	}
	at(t, w, config.StartCol, config.StartRow-1)
}

// TestNoSlideOffTheRoad: landing on the verge, the track or the start row
// doesn't slide.
func TestNoSlideOffTheRoad(t *testing.T) {
	w := wetWorld()
	w.Player = newPlayer(config.StartCol, 8)
	w.furthest = 8
	if ev := hops(w, Up, 1); slices.Contains(ev, Slid) {
		t.Errorf("slid onto row 7: %v", ev)
	}
	at(t, w, config.StartCol, 7)
	w.Player = newPlayer(config.StartCol, 11)
	if ev := hops(w, Down, 1); slices.Contains(ev, Slid) {
		t.Errorf("slid onto the start row: %v", ev)
	}
}

// TestSlideStopsAtTheEdge: a slide that would leave the field doesn't
// happen.
func TestSlideStopsAtTheEdge(t *testing.T) {
	w := wetWorld()
	w.Player = newPlayer(1, 10)
	if ev := hops(w, Left, 1); slices.Contains(ev, Slid) {
		t.Errorf("slid off the field: %v", ev)
	}
	at(t, w, 0, 10)
	w.Player = newPlayer(3, 10)
	hops(w, Right, 1)
	run(w, None, config.HopTicks)
	at(t, w, 5, 10)
}

// TestSlideIntoAVehicle: a vehicle in the slide cell is a death (D31).
func TestSlideIntoAVehicle(t *testing.T) {
	w := roadWorld(config.Truck, Buffer+(config.StartCol-1)*cellUnits, 1, 0)
	w.Lanes[10] = Lane{Kind: config.Road}
	rain(w)
	w.Player = newPlayer(config.StartCol, 9)
	w.Lanes[9] = Lane{Kind: config.Road}
	ev := hops(w, Down, 1)
	ev = append(ev, run(w, None, config.HopTicks)...)
	if !slices.Contains(ev, Slid) || !slices.Contains(ev, Squashed) {
		t.Errorf("events %v: want a slide into the truck on row 11", ev)
	}
}

// TestSlideIsNotAHopForTheCombo: a slide neither adds to the combo nor
// breaks it (D31).
func TestSlideIsNotAHopForTheCombo(t *testing.T) {
	w := wetWorld()
	w.Level.Combo = true
	hops(w, Up, 1)
	run(w, None, config.HopTicks)
	if w.Combo.Hops != 1 {
		t.Errorf("combo %+v after a hop and a slide, want 1", w.Combo)
	}
}

// TestOneWeatherAtATime: fog and rain both come on Rush Hour's top level,
// never at once, and never in the calm start (D42).
func TestOneWeatherAtATime(t *testing.T) {
	w := rushWorld(config.EventTopLevel, 3)
	w.Player = newPlayer(config.StartCol, config.StartRow)
	seen := map[config.EventKind]bool{}
	for range 300 * config.TPS {
		ev := w.Update(None)
		w.TicksLeft = w.Level.TimerTicks
		if (slices.Contains(ev, FogWarning) || slices.Contains(ev, RainWarning)) && int(w.Tick) < config.LevelCalmTicks {
			t.Fatalf("weather warned at tick %d", w.Tick)
		}
		on := 0
		for _, s := range w.Spells {
			if config.Events[s.Kind].Weather && s.Phase != SpellQuiet {
				on++
				seen[s.Kind] = true
			}
		}
		if on > 1 {
			t.Fatalf("tick %d: two weathers at once: %+v", w.Tick, w.Spells)
		}
	}
	if !seen[config.Fog] || !seen[config.Rain] {
		t.Errorf("weathers seen: %v", seen)
	}
}

// TestRushHourBringsItsEventsIn: waves from level 1, fog from 2, rain
// from 3 (D41).
func TestRushHourBringsItsEventsIn(t *testing.T) {
	for n, want := range map[int][3]bool{1: {true, false, false}, 2: {true, true, false}, 3: {true, true, true}, 9: {true, true, true}} {
		lv := config.RushHour.Levels.Level(n)
		got := [3]bool{lv.Events[config.Wave].On(), lv.Events[config.Fog].On(), lv.Events[config.Rain].On()}
		if got != want {
			t.Errorf("level %d: wave, fog, rain %v, want %v", n, got, want)
		}
	}
}
