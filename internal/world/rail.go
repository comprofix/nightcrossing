package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// TrainPhase is where the track is in its train cycle (DESIGN §6.2).
type TrainPhase int

// The phases, in order, round and round.
const (
	RailQuiet   TrainPhase = iota // no train: the track is safe
	RailWarning                   // the crossing lights flash and the bell rings
	RailPassing                   // a train is on the field
)

// Rail is the track and its trains, in a mode with trains (DESIGN §6.2,
// D17, D37). Each train comes after a quiet spell of seeded length and a
// warning, from the left or the right, and is longer than the field.
type Rail struct {
	On    bool // trains run; the track is only scenery when false
	Row   int  // the track's row
	Phase TrainPhase
	Left  int // ticks left of RailQuiet or RailWarning
	Dir   int // the coming or passing train's way: +1 right, -1 left
	X     int // the train's left end from the field's left edge, in SubPx units, while RailPassing

	Timing config.TrainTiming // the level's train speed and quiet spells (D17)
}

// trainUnits is the train's length in SubPx units.
var trainUnits = config.Train.Length * config.SubPx

// newRail is the track for level: with trains if the level has them,
// starting with a quiet spell.
func newRail(level config.Level, rng *rand.Rand) Rail {
	if !level.Trains {
		return Rail{}
	}
	for r, l := range level.Lanes {
		if l.Kind == config.Rail {
			rl := Rail{On: true, Row: r, Phase: RailQuiet, Timing: level.Train}
			rl.Left = rl.quietTicks(rng)
			return rl
		}
	}
	return Rail{}
}

// quietTicks is a seeded length for the quiet before the next train.
func (rl *Rail) quietTicks(rng *rand.Rand) int {
	return rl.Timing.QuietMin + rng.IntN(rl.Timing.QuietMax-rl.Timing.QuietMin+1)
}

// update runs the train cycle one tick and gives out its events.
func (rl *Rail) update(w *World) {
	if !rl.On {
		return
	}
	switch rl.Phase {
	case RailQuiet:
		if rl.Left--; rl.Left == 0 {
			rl.Phase, rl.Left = RailWarning, config.TrainWarnTicks
			rl.Dir = 1
			if w.rng.IntN(2) == 0 {
				rl.Dir = -1
			}
			w.emit(TrainWarning)
		}
	case RailWarning:
		if rl.Left--; rl.Left == 0 {
			rl.Phase = RailPassing
			rl.X = fieldWidth // coming in from the right
			if rl.Dir > 0 {
				rl.X = -trainUnits
			}
			w.emit(TrainComing)
		}
	case RailPassing:
		rl.X += rl.Dir * rl.Timing.Speed
		if rl.X >= fieldWidth || rl.X+trainUnits <= 0 {
			rl.Phase, rl.Left = RailQuiet, rl.quietTicks(w.rng)
			w.emit(TrainGone)
		}
	}
}

// Hits reports whether the train is on the field over left to right, in
// SubPx units from the field's left edge, trimmed at each end by
// config.HitMargin like a vehicle.
func (rl *Rail) Hits(left, right int) bool {
	if !rl.On || rl.Phase != RailPassing {
		return false
	}
	m := config.HitMargin * config.SubPx
	return rl.X+m < right && left < rl.X+trainUnits-m
}
