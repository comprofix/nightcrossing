package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// rushWorld is a Rush Hour game on its level n.
func rushWorld(n int, seed uint64) *World {
	return New(session.New(config.RushHour, config.StartLives), config.RushHour.Levels.Level(n),
		rand.New(rand.NewPCG(seed, seed)))
}

// onRoad counts the lane's vehicles on the lane and at least partly on the
// field.
func onRoad(l *Lane) int {
	n := 0
	for _, e := range l.Entities {
		if !e.Out && !e.off() {
			n++
		}
	}
	return n
}

func TestNoWavesOutsideRushHour(t *testing.T) {
	for _, m := range []config.Mode{config.Stroll, config.LastTrain} {
		w := New(session.New(m, config.StartLives), m.Levels.Level(8), rand.New(rand.NewPCG(1, 1)))
		if len(w.Spells) != 0 {
			t.Errorf("%s has events %v", m.Name, w.Spells)
		}
		for r := range w.Lanes {
			if w.Lanes[r].canWave() {
				t.Errorf("%s row %d has wave vehicles", m.Name, r)
			}
		}
	}
}

// TestWaveSlotsLeaveAWayAcross: every wave vehicle leaves the gaps either
// side of it crossable at the lane's speed, on every level and loop up to
// the ceiling, and each of levels 1–8 has a lane that can wave (D42, D43).
func TestWaveSlotsLeaveAWayAcross(t *testing.T) {
	for n := 1; n <= 40; n++ {
		w := rushWorld(n, 1)
		any := false
		for r := range w.Lanes {
			l := &w.Lanes[r]
			if l.canWave() {
				any = true
			}
			if l.Kind != config.Road {
				continue
			}
			es := slices.Clone(l.Entities)
			slices.SortFunc(es, func(a, b Entity) int { return a.U - b.U })
			for i, e := range es {
				next := es[(i+1)%len(es)]
				gap := (next.U - e.U - e.Kind.Length*config.SubPx + l.Loop) % l.Loop / config.SubPx
				if open := config.GapOpenTicks(gap, l.Speed); open < config.CrossOpenTicks {
					t.Errorf("level %d row %d: a %dpx gap in a wave, open %d ticks", n, r, gap, open)
				}
			}
		}
		if !any && n <= config.EventTopLevel {
			t.Errorf("level %d: no lane can take a wave", n)
		}
	}
}

// TestWaveCycle: a calm start, then a warning of WarnTicks on one lane, the
// wave for its length, and quiet again; the wave's vehicles come on only
// while the wave is on, and only from off the field.
func TestWaveCycle(t *testing.T) {
	w := rushWorld(1, 7)
	w.Player = newPlayer(config.StartCol, config.StartRow) // safe on the start row
	until(t, w, WaveWarning, 60*config.TPS)
	if int(w.Tick) < config.LevelCalmTicks {
		t.Errorf("a wave warned at tick %d, in the calm start", w.Tick)
	}
	s := w.Spell(config.Wave)
	lane := &w.Lanes[s.Lane]
	if lane.Kind != config.Road || !lane.canWave() || s.Phase != SpellWarning {
		t.Fatalf("spell %+v on a %v lane", s, lane.Kind)
	}
	before := onRoad(lane)
	if got := until(t, w, WaveStart, config.Events[config.Wave].WarnTicks); got != config.Events[config.Wave].WarnTicks {
		t.Errorf("warning of %d ticks, want %d", got, config.Events[config.Wave].WarnTicks)
	}
	most := 0
	for range s.Timing.Last - 1 {
		prev := slices.Clone(lane.Entities)
		w.Update(None)
		for i, e := range lane.Entities {
			if prev[i].Out && !e.Out && !e.off() {
				t.Fatalf("a wave vehicle popped in on the field at %d", e.X())
			}
		}
		most = max(most, onRoad(lane))
	}
	if most <= before {
		t.Errorf("the wave brought no more traffic: %d on the field before, at most %d during", before, most)
	}
	until(t, w, WaveEnd, 2)
	if lane.Waving {
		t.Error("the lane still waving after the wave")
	}
	if w.Dying != 0 {
		t.Errorf("the hero on the start row died: %v", w.Player.Death)
	}
}

func TestWavesSameSeedSameTimes(t *testing.T) {
	times := func() []int64 {
		w := rushWorld(8, 11)
		w.Player = newPlayer(config.StartCol, config.StartRow)
		var out []int64
		for range 120 * config.TPS {
			for _, e := range w.Update(None) {
				if e == WaveWarning || e == WaveStart || e == WaveEnd {
					out = append(out, w.Tick, int64(w.Spell(config.Wave).Lane))
				}
			}
			w.TicksLeft = w.Level.TimerTicks // keep the clock from running out
		}
		return out
	}
	a, b := times(), times()
	if !slices.Equal(a, b) || len(a) < 9 {
		t.Errorf("two runs: %v and %v", a, b)
	}
}

// TestWavesComeSoonerAtDawnAndDusk (D43).
func TestWavesComeSoonerAtDawnAndDusk(t *testing.T) {
	w := rushWorld(1, 1)
	if !w.rushHour() {
		t.Error("dawn, with no nooks filled, is not a rush")
	}
	w.Filled[0] = true
	if w.rushHour() {
		t.Error("day is a rush")
	}
	w.Filled[1], w.Filled[2] = true, true
	if !w.rushHour() {
		t.Error("dusk is not a rush")
	}
}

func TestWavesRampUp(t *testing.T) {
	first, top := config.RushHour.Levels.Level(1).Events[config.Wave], config.RushHour.Levels.Level(config.EventTopLevel).Events[config.Wave]
	if first != config.Events[config.Wave].First || top != config.Events[config.Wave].Top {
		t.Errorf("level 1 %+v, top %+v", first, top)
	}
	if top.QuietMax >= first.QuietMax || top.Last <= first.Last {
		t.Errorf("waves don't get more frequent: %+v to %+v", first, top)
	}
}
