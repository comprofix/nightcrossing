package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// SpellPhase is where a random event is in its run (DESIGN §6.1).
type SpellPhase int

// The phases, in order, round and round.
const (
	SpellQuiet   SpellPhase = iota // waiting for the next one
	SpellWarning                   // announced: the hint strip and cues say it's coming
	SpellOn                        // happening
)

// Spell is one kind of random event as it runs on a level: a quiet of
// seeded length, a warning, then the event for a set time, and round
// again (DESIGN §6.1, D28).
type Spell struct {
	Kind   config.EventKind
	Timing config.EventTiming
	Phase  SpellPhase
	Left   int // ticks left of this phase
	Lane   int // the wave's road row, from its warning to its end
}

// newSpells are the level's random events, each starting with a quiet.
func newSpells(level config.Level, rng *rand.Rand) []Spell {
	var out []Spell
	for k, t := range level.Events {
		if t.On() {
			s := Spell{Kind: config.EventKind(k), Timing: t}
			s.Left = s.quiet(rng, false)
			out = append(out, s)
		}
	}
	return out
}

// quiet is a seeded length for the quiet before the next one. Waves come
// sooner at dawn and dusk if rush is true (D43).
func (s *Spell) quiet(rng *rand.Rand, rush bool) int {
	n := s.Timing.QuietMin + rng.IntN(s.Timing.QuietMax-s.Timing.QuietMin+1)
	if rush && s.Kind == config.Wave {
		n = n * config.WaveRushPct / 100
	}
	return n
}

// Spell is the level's event of kind k, or nil if it has none.
func (w *World) Spell(k config.EventKind) *Spell {
	for i := range w.Spells {
		if w.Spells[i].Kind == k {
			return &w.Spells[i]
		}
	}
	return nil
}

// updateSpells runs each random event one tick: its quiet, its warning,
// then the event itself, giving out the events for each change.
func (w *World) updateSpells() {
	for i := range w.Spells {
		s := &w.Spells[i]
		if s.Left--; s.Left > 0 {
			continue
		}
		switch s.Phase {
		case SpellQuiet:
			w.warn(s)
		case SpellWarning:
			s.Phase, s.Left = SpellOn, s.Timing.Last
			w.start(s)
		case SpellOn:
			s.Phase, s.Left = SpellQuiet, s.quiet(w.rng, w.rushHour())
			w.end(s)
		}
	}
}

// warn announces s, unless it can't start yet: not during a level's calm
// start, while its opening hint shows (DESIGN §6.1), and a wave only if a
// road lane can take one. Then it tries again shortly.
func (w *World) warn(s *Spell) {
	if w.Tick < config.LevelCalmTicks {
		s.Left = config.EventRetryTicks
		return
	}
	if config.Events[s.Kind].Weather && w.weatherBut(s.Kind) {
		s.Left = config.EventRetryTicks // one weather at a time (D42)
		return
	}
	switch s.Kind {
	case config.Wave:
		var lanes []int
		for r := range w.Lanes {
			if w.Lanes[r].canWave() {
				lanes = append(lanes, r)
			}
		}
		if len(lanes) == 0 {
			s.Left = config.EventRetryTicks
			return
		}
		s.Lane = lanes[w.rng.IntN(len(lanes))]
		w.emit(WaveWarning)
	case config.Fog:
		w.emit(FogWarning)
	case config.Rain:
		w.emit(RainWarning)
	}
	s.Phase, s.Left = SpellWarning, config.Events[s.Kind].WarnTicks
}

// start begins s.
func (w *World) start(s *Spell) {
	switch s.Kind {
	case config.Wave:
		w.Lanes[s.Lane].Waving = true
		w.emit(WaveStart)
	case config.Fog:
		w.emit(FogStart)
	case config.Rain:
		w.emit(RainStart)
	}
}

// end ends s.
func (w *World) end(s *Spell) {
	switch s.Kind {
	case config.Wave:
		w.Lanes[s.Lane].Waving = false
		w.emit(WaveEnd)
	case config.Fog:
		w.emit(FogEnd)
	case config.Rain:
		w.emit(RainEnd)
	}
}

// rushHour reports whether it is dawn or dusk on a level with the day and
// night cycle: the morning and evening rush, when waves come sooner (D43).
func (w *World) rushHour() bool {
	if !w.Level.DayCycle {
		return false
	}
	filled := 0
	for _, f := range w.Filled {
		if f {
			filled++
		}
	}
	t := config.DayCycle[filled]
	return t == config.Dawn || t == config.Dusk
}

// canWave reports whether the lane has wave vehicles to bring on.
func (l *Lane) canWave() bool {
	for _, e := range l.Entities {
		if e.Wave {
			return true
		}
	}
	return false
}

// weatherBut reports whether a weather other than k is announced or on.
func (w *World) weatherBut(k config.EventKind) bool {
	for _, s := range w.Spells {
		if s.Kind != k && config.Events[s.Kind].Weather && s.Phase != SpellQuiet {
			return true
		}
	}
	return false
}

// Raining reports whether it is raining: the road is slippery (D31).
func (w *World) Raining() bool {
	s := w.Spell(config.Rain)
	return s != nil && s.Phase == SpellOn
}
