package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// sirenWorld is mode's level n with every random event held quiet, run on
// for wait ticks past its calm start from seed, then the hero put at col,
// row and an emergency vehicle sent down road row lane at once, in the
// rain if wet (DESIGN §6.9).
func sirenWorld(mode config.Mode, n int, seed uint64, wait, lane, col, row int, wet bool) *World {
	w := quietWorld(mode, n, seed, wait, col, row, wet)
	w.Spell(config.Emergency).Phase = SpellWarning
	w.startSiren(lane, 0)
	return w
}

// quietWorld is sirenWorld with no vehicle sent.
func quietWorld(mode config.Mode, n int, seed uint64, wait, col, row int, wet bool) *World {
	w := New(session.New(mode, config.StartLives), mode.Levels.Level(n), rand.New(rand.NewPCG(seed, seed)))
	for i := range w.Spells {
		w.Spells[i].Phase, w.Spells[i].Left = SpellQuiet, 1<<30
	}
	for range config.LevelCalmTicks + wait {
		w.Update(None)
	}
	if wet {
		s := w.Spell(config.Rain)
		s.Phase, s.Left = SpellOn, 1<<30
	}
	w.Player = newPlayer(col, row)
	return w
}

// ceilingLevel is the first level of mode played at its speed ceiling.
func ceilingLevel(mode config.Mode) int {
	n := len(mode.Levels.Levels) + 1
	for ; ; n++ {
		if _, s := mode.Levels.Def(n); s >= mode.Levels.CeilingPct {
			return n
		}
	}
}

// sirenLevels are the levels the checks play: each from the first with
// emergency vehicles to the top level, and the first at the ceiling.
func sirenLevels(mode config.Mode) []int {
	var ns []int
	for n := config.EmergencyFrom; n <= config.EventTopLevel; n++ {
		ns = append(ns, n)
	}
	return append(ns, ceilingLevel(mode))
}

// gaps are road row r's gaps between its cars, in SubPx units, from each
// car's front to the next one's back round the loop, sorted.
func gaps(l *Lane) []int {
	var us []Entity
	for _, e := range l.Entities {
		if !e.Wave {
			us = append(us, e)
		}
	}
	slices.SortFunc(us, func(a, b Entity) int { return a.U - b.U })
	var out []int
	for i, e := range us {
		next := us[(i+1)%len(us)]
		out = append(out, l.wrap(next.U-e.U-e.Kind.Length*config.SubPx))
	}
	slices.Sort(out)
	return out
}

func TestNoEmergenciesBeforeTheirLevel(t *testing.T) {
	for _, m := range config.Modes {
		for n := 1; n <= 12; n++ {
			on := m.Levels.Level(n).Events[config.Emergency].On()
			want := n >= config.EmergencyFrom && m.ID != config.Stroll.ID && m.ID != config.LastTrain.ID
			if on != want {
				t.Errorf("%s level %d: emergency vehicles %v, want %v", m.Name, n, on, want)
			}
		}
	}
}

// TestSirenCycle: through an emergency vehicle's coming, from the hero on
// the start row, the emergency lane empties and stays empty until the
// vehicle has been, the lanes either side close no sooner than
// config.EscapeTicks after the warning, the vehicle comes through its
// lane once the trap is closed, and afterwards every lane gets its
// pattern back, with no extra cars left (D72).
func TestSirenCycle(t *testing.T) {
	for _, mode := range []config.Mode{config.RushHour, config.Storm, config.SchoolRun} {
		for _, n := range sirenLevels(mode) {
			for _, lane := range config.EmergencyRows {
				for seed := uint64(1); seed <= 6; seed++ {
					w := sirenWorld(mode, n, seed, int(seed)*97, lane, config.StartCol, config.StartRow, false)
					before := map[int][]int{}
					for _, r := range []int{lane - 1, lane, lane + 1} {
						before[r] = gaps(&w.Lanes[r])
					}
					start, shut, came := w.Tick, int64(0), int64(0)
					for w.Siren.Phase != SirenAway && w.Tick-start < 60*config.TPS {
						w.TicksLeft = 1 << 20
						for _, e := range w.Update(None) {
							if e == SirenComing {
								came = w.Tick
							}
						}
						if shut == 0 && (came == 0 || came == w.Tick) && w.shut(lane-1) && w.shut(lane+1) {
							shut = w.Tick
						}
						if w.Siren.Phase == SirenRun && !w.laneClear(lane) {
							t.Fatalf("%s level %d row %d: a car in the lane as the vehicle comes", mode.Name, n, lane)
						}
					}
					switch {
					case came == 0:
						t.Errorf("%s level %d row %d seed %d: no vehicle", mode.Name, n, lane, seed)
					case shut == 0:
						t.Errorf("%s level %d row %d seed %d: the trap never closed", mode.Name, n, lane, seed)
					case shut-start < config.EscapeTicks:
						t.Errorf("%s level %d row %d seed %d: the trap closed after %d ticks, want at least %d", mode.Name, n, lane, seed, shut-start, config.EscapeTicks)
					}
					for w.Tick-start < 120*config.TPS && (w.Lanes[lane-1].held || w.Lanes[lane].held || w.Lanes[lane+1].held) {
						w.TicksLeft = 1 << 20
						w.Update(None)
					}
					for r, g := range before {
						l := &w.Lanes[r]
						if l.held {
							t.Errorf("%s level %d row %d seed %d: row %d still finding its way back", mode.Name, n, lane, seed, r)
							continue
						}
						for _, e := range l.Entities {
							if e.Extra || e.Away {
								t.Errorf("%s level %d row %d seed %d: row %d has an extra or away car left", mode.Name, n, lane, seed, r)
							}
						}
						if got := gaps(l); !slices.Equal(got, g) {
							t.Errorf("%s level %d row %d seed %d: row %d's gaps %v, were %v", mode.Name, n, lane, seed, r, got, g)
						}
					}
				}
			}
		}
	}
}

// TestQueueGivesWay: a hero standing anywhere in a lane either side of the
// emergency lane is never hit while it bunches up and stands, and the
// queue leaves it a pocket; once the queues move off, their cars are
// ordinary traffic again (D72).
func TestQueueGivesWay(t *testing.T) {
	for _, n := range sirenLevels(config.RushHour) {
		for _, lane := range config.EmergencyRows {
			for _, row := range []int{lane - 1, lane + 1} {
				for col := range config.Cols {
					w := sirenWorld(config.RushHour, n, uint64(col), col*13, lane, col, row, false)
					if w.Lanes[row].hits(w.Player.X, w.Player.X+cellUnits, config.HitMargin) {
						continue // put in a car
					}
					for w.Siren.Phase < SirenMoveOff {
						w.Update(None)
						if w.Dying > 0 {
							t.Fatalf("level %d row %d: hero at %d, %d %v", n, lane, col, row, w.Player.Death)
						}
					}
				}
			}
		}
	}
}

// TestRunDown: a hero in the emergency lane when the vehicle comes is run
// down.
func TestRunDown(t *testing.T) {
	for _, lane := range config.EmergencyRows {
		w := sirenWorld(config.RushHour, 5, 3, 0, lane, config.StartCol, config.StartRow, false)
		for !w.laneClear(lane) {
			w.Update(None)
		}
		w.Player = newPlayer(config.StartCol, lane)
		for w.Dying == 0 && w.Siren.Phase != SirenAway {
			w.Update(None)
		}
		if w.Player.Death != RunDown {
			t.Errorf("row %d: the hero %v, want run down", lane, w.Player.Death)
		}
	}
}

// escapes reports whether the hero, from where it is in road row lane, has
// moves that take it out alive into another row by tick by, and before an
// emergency vehicle comes. It tries every move, a hop or a short wait, on
// copies of w.
func escapes(w *World, lane int, by int64) bool {
	type at struct {
		tick     int64
		col, row int
	}
	seen := map[at]bool{}
	var try func(w *World) bool
	try = func(w *World) bool {
		if w.Dying > 0 || w.Tick > by || w.Siren.Phase > SirenWarning {
			return false
		}
		p := &w.Player
		if !p.Hopping() && p.Row != lane {
			return true
		}
		k := at{w.Tick / config.HopTicks, columnOf(p.X), p.Row}
		if seen[k] {
			return false
		}
		seen[k] = true
		for _, d := range []Dir{Up, Down, Left, Right, None} {
			c := w.Clone(rand.New(rand.NewPCG(1, 1)))
			c.Update(d)
			for range config.HopTicks - 1 {
				c.Update(None)
			}
			if try(c) {
				return true
			}
		}
		return false
	}
	return try(w)
}

// TestEveryEmergencyCanBeEscaped is the crossing check's escape window
// (D21, D72): a hero in the emergency lane when the warning starts, who
// with the traffic as usual could get out into a lane either side within
// config.EscapeTicks, can always get out before the vehicle comes; on
// every level from the first with them to the top and at the ceiling, in
// the dry and in the rain.
func TestEveryEmergencyCanBeEscaped(t *testing.T) {
	tried := 0
	for _, n := range sirenLevels(config.RushHour) {
		for _, lane := range config.EmergencyRows {
			for _, wet := range []bool{false, true} {
				for col := range config.Cols {
					for seed := uint64(1); seed <= 6; seed++ {
						wait := col*31 + int(seed)*7
						if q := quietWorld(config.RushHour, n, seed, wait, col, lane, wet); !escapes(q, lane, q.Tick+config.EscapeTicks) {
							continue // no way out even as usual: a hero couldn't be standing there
						}
						tried++
						w := sirenWorld(config.RushHour, n, seed, wait, lane, col, lane, wet)
						if !escapes(w, lane, 1<<40) {
							t.Errorf("level %d row %d col %d seed %d wet %v: no way out", n, lane, col, seed, wet)
						}
					}
				}
			}
		}
	}
	if tried == 0 {
		t.Error("no hero could stand in an emergency lane")
	}
}

// TestEmergencyAlone: in long plays of Rush Hour and Storm, emergency
// vehicles come, and never while a rush hour wave or a storm is announced
// or on (D48, D72).
func TestEmergencyAlone(t *testing.T) {
	for _, mode := range []config.Mode{config.RushHour, config.Storm} {
		w := New(session.New(mode, config.StartLives), mode.Levels.Level(6), rand.New(rand.NewPCG(3, 3)))
		came := 0
		for range 15 * 60 * config.TPS {
			w.TicksLeft = 1 << 20
			for _, e := range w.Update(None) {
				if e == SirenComing {
					came++
				}
			}
			if w.Siren.Phase == SirenAway {
				continue
			}
			for _, s := range w.Spells {
				if s.Kind != config.Emergency && config.Events[s.Kind].Big && s.Phase != SpellQuiet {
					t.Fatalf("%s tick %d: an emergency vehicle with event %d", mode.Name, w.Tick, s.Kind)
				}
			}
		}
		if came < 10 {
			t.Errorf("%s: %d emergency vehicles in 15 minutes", mode.Name, came)
		}
	}
}
