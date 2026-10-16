package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

func TestLaneLayout(t *testing.T) {
	for r, l := range rich.Lanes {
		if len(l.Pattern) == 0 {
			continue
		}
		lane := newLane(l, 0, rand.New(rand.NewPCG(3, 3)))
		pattern := 0
		for _, s := range l.Pattern {
			pattern += (s.Entity.Length + s.Gap) * config.SubPx
		}
		if lane.Loop < fieldWidth+Buffer || lane.Loop%pattern != 0 {
			t.Errorf("row %d: loop %d is not whole patterns of %d covering the field and buffer", r, lane.Loop, pattern)
		}
		if want := lane.Loop / pattern * len(l.Pattern); len(lane.Entities) != want {
			t.Errorf("row %d: %d vehicles, want %d", r, len(lane.Entities), want)
		}
		// Each vehicle sits its pattern's length and gap behind the one
		// before, all the way round the loop.
		for i, v := range lane.Entities {
			next := lane.Entities[(i+1)%len(lane.Entities)]
			s := l.Pattern[i%len(l.Pattern)]
			if got := ((next.U-v.U)%lane.Loop + lane.Loop) % lane.Loop; got != (s.Entity.Length+s.Gap)*config.SubPx {
				t.Errorf("row %d vehicle %d: next one %d units on, want %d", r, i, got, (s.Entity.Length+s.Gap)*config.SubPx)
			}
		}
	}
}

// TestLanesWrapWithoutPopping runs every lane of level 1 a full loop. Each
// entity comes back to where it started, and an entity only wraps when at
// most one tick's movement of it is on the field, on either side.
func TestLanesWrapWithoutPopping(t *testing.T) {
	for r, l := range rich.Lanes {
		if len(l.Pattern) == 0 {
			continue
		}
		rng := rand.New(rand.NewPCG(9, 9))
		lane := newLane(l, 0, rng)
		start := slices.Clone(lane.Entities)
		for range lane.Loop {
			before := slices.Clone(lane.Entities)
			lane.update(rng)
			for i, v := range lane.Entities {
				if abs(v.U-before[i].U) <= lane.Speed {
					continue
				}
				for _, at := range []Entity{before[i], v} {
					if on := onField(at); on > lane.Speed {
						t.Fatalf("row %d: entity %d wrapped with %d units on the field", r, i, on)
					}
				}
			}
		}
		for i := range start {
			e := lane.Entities[i] // only where they are counts here
			start[i].Phase, start[i].Sink, start[i].Wait = e.Phase, e.Sink, e.Wait
		}
		if !slices.Equal(lane.Entities, start) {
			t.Errorf("row %d: entities not back where they started after a full loop", r)
		}
	}
}

// onField is how much of v is on the field, in SubPx units.
func onField(v Entity) int {
	left, right := v.span(0)
	return max(0, min(right, fieldWidth)-max(left, 0))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// roadWorld is a game with one road lane on row 11, just above the start,
// holding a single vehicle at u, moving dir at speed units a tick.
func roadWorld(v config.Entity, u, dir, speed int) *World {
	lv := testLevel()
	lv.Lanes[11] = config.Lane{Kind: config.Road, Dir: dir, Speed: speed, Pattern: []config.Spawn{{Entity: v, Gap: 600}}}
	w := New(session.New(config.Skills[0], config.StartLives), lv, rand.New(rand.NewPCG(1, 1)))
	lane := &w.Lanes[11]
	lane.Speed = speed
	lane.Entities = []Entity{{Kind: v, U: u}}
	return w
}

func TestAVehicleSquashesTheHero(t *testing.T) {
	const speed = config.SubPx // a pixel a tick
	w := roadWorld(config.CarRed, 0, 1, speed)
	w.Player = newPlayer(config.StartCol, 11)

	// Start the car's trimmed nose 10px short of the hero's trimmed side.
	heroLeft := w.Player.X + config.HeroMargin*config.SubPx
	nose := (config.CarRed.Length - config.HitMargin) * config.SubPx
	w.Lanes[11].Entities[0].U = heroLeft - nose - 10*config.SubPx + Buffer

	for tick := 1; ; tick++ {
		ev := w.Update(None)
		if slices.Contains(ev, Squashed) {
			if tick != 11 {
				t.Errorf("squashed on tick %d, want 11 (the nose meets the hero on 10)", tick)
			}
			break
		}
		if tick > 20 {
			t.Fatal("never squashed")
		}
	}
	if w.Player.Death != Squashed || w.Dying != config.DeathTicks {
		t.Errorf("death %v, dying %d", w.Player.Death, w.Dying)
	}
	if w.Session.Lives != config.StartLives-1 {
		t.Errorf("%d lives after a death, want %d", w.Session.Lives, config.StartLives-1)
	}
}

func TestAVehicleInAnotherLaneIsHarmless(t *testing.T) {
	w := roadWorld(config.Truck, 0, 1, 0)
	w.Player = newPlayer(config.StartCol, config.StartRow)
	w.Lanes[11].Entities[0].U = w.Player.X + Buffer // right above the hero
	if ev := run(w, None, 30); len(ev) != 0 {
		t.Errorf("events %v standing below a vehicle", ev)
	}
}

// TestHopCountsForTheNewRowFromHalfway parks a truck over the hero's next
// cell and hops into it: the first half of the hop is still over the safe
// row.
func TestHopCountsForTheNewRowFromHalfway(t *testing.T) {
	w := roadWorld(config.Truck, 0, 1, 0)
	w.Lanes[11].Entities[0].U = w.Player.X - config.Cell*config.SubPx + Buffer
	for tick := 1; tick <= config.HopTicks; tick++ {
		d := None
		if tick == 1 {
			d = Up
		}
		ev := w.Update(d)
		squashed := slices.Contains(ev, Squashed)
		if want := tick == config.HopTicks/2; squashed != want {
			t.Fatalf("tick %d of the hop: squashed %v, want %v", tick, squashed, want)
		}
		if squashed {
			return
		}
	}
}

func TestDeathThenRespawn(t *testing.T) {
	w := roadWorld(config.Truck, 0, 1, 0)
	w.Player = newPlayer(config.StartCol, 11)
	w.Lanes[11].Entities[0].U = w.Player.X + Buffer
	w.TicksLeft = 100
	if ev := w.Update(None); !slices.Equal(ev, []Event{Squashed}) {
		t.Fatalf("events %v, want a squash", ev)
	}
	w.Lanes[11].Entities[0].U = 0 // move it off, so the respawn is safe

	// Presses do nothing while dying.
	if ev := run(w, Down, config.DeathTicks-1); len(ev) != 0 {
		t.Fatalf("events %v while dying", ev)
	}
	if w.Player.Row != 11 {
		t.Fatalf("the hero moved while dying, to row %d", w.Player.Row)
	}
	if ev := w.Update(None); !slices.Equal(ev, []Event{Respawned}) {
		t.Fatalf("events %v at the end of the death, want a respawn", ev)
	}
	at(t, w, config.StartCol, config.StartRow)
	if w.Player.Death != 0 || w.Player.Facing != Up || w.Dying != 0 {
		t.Errorf("after respawn: %+v, dying %d", w.Player, w.Dying)
	}
	if w.TicksLeft != w.Level.TimerTicks {
		t.Errorf("clock %d after respawn, want it full", w.TicksLeft)
	}
	if w.Session.Lives != config.StartLives-1 {
		t.Errorf("%d lives, want %d", w.Session.Lives, config.StartLives-1)
	}
}

func TestLastLifeEndsTheGame(t *testing.T) {
	w := roadWorld(config.Truck, 0, 1, 0)
	w.Session.Lives = 1
	w.Player = newPlayer(config.StartCol, 11)
	w.Lanes[11].Entities[0].U = w.Player.X + Buffer
	w.Update(None)
	ev := run(w, None, config.DeathTicks)
	if !slices.Equal(ev, []Event{GameOver}) || !w.Over {
		t.Fatalf("events %v, over %v: want game over", ev, w.Over)
	}
	tick := w.Tick
	if ev := run(w, Up, 30); len(ev) != 0 || w.Tick != tick {
		t.Errorf("the world ran on after game over: events %v, tick %d → %d", ev, tick, w.Tick)
	}
}

func TestLevelOneIsRandomPerSeedButRepeatable(t *testing.T) {
	start := func(seed uint64) []Entity {
		w := New(session.New(config.Skills[0], 3), rich, rand.New(rand.NewPCG(seed, seed)))
		return w.Lanes[9].Entities
	}
	if !slices.Equal(start(5), start(5)) {
		t.Error("one seed gave two layouts")
	}
	if slices.Equal(start(5), start(6)) {
		t.Error("two seeds gave the same layout")
	}
}
