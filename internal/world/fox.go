package world

import "github.com/comprofix/nightcrossing/internal/config"

// Fox is the fox that works the nooks at dusk and night (D68).
type Fox struct {
	Nook  int      // the nook it is in, or -1
	Phase FoxPhase // where it is in its visit
	ticks int      // ticks left of this phase
}

// FoxPhase is where the fox is in a visit to a nook.
type FoxPhase int

// The phases. Away covers the time it is gone and the quiet after.
const (
	FoxAway  FoxPhase = iota
	FoxEyes           // its eyes glint in the nook: the warning, and the nook still safe
	FoxThere          // it is in the nook: hopping in is a death
)

// newFox starts the level's fox, if it has one, away.
func (w *World) newFox() {
	w.Fox = Fox{Nook: -1}
	if w.Level.Fox {
		w.foxAway()
	}
}

// foxAway sends the fox away: gone, then a quiet before it comes again.
func (w *World) foxAway() {
	w.Fox = Fox{Nook: -1, ticks: config.FoxGoneTicks + between(w, config.FoxQuietMinTicks, config.FoxQuietMaxTicks)}
}

// foxHour reports whether it is the fox's time: dusk and night on a level
// with the day and night cycle, and always on one without, which is night
// (D39, D68).
func (w *World) foxHour() bool {
	if !w.Level.DayCycle {
		return true
	}
	t := config.DayCycle[w.filled()]
	return t == config.Dusk || t == config.Night
}

// updateFox moves the fox one tick on (D68). Once it is away long enough,
// in its hours, it picks an empty nook without fruit: its eyes glint, then
// it is there, then it goes. Its clock runs on through deaths and nooks
// filled.
func (w *World) updateFox() {
	f := &w.Fox
	if !w.Level.Fox {
		return
	}
	if f.ticks--; f.ticks > 0 {
		return
	}
	switch f.Phase {
	case FoxEyes:
		f.Phase, f.ticks = FoxThere, config.FoxOutTicks
		w.emit(FoxIn)
		return
	case FoxThere:
		w.foxAway()
		return
	}
	var empty []int
	for n, filled := range w.Filled {
		if !filled && n != w.FruitNook {
			empty = append(empty, n)
		}
	}
	if !w.foxHour() || len(empty) == 0 {
		f.ticks = config.TPS // look again in a while
		return
	}
	f.Nook, f.Phase, f.ticks = empty[w.rng.IntN(len(empty))], FoxEyes, config.FoxEyesTicks
	w.emit(EyesGlint)
}

// foxIn reports whether the fox is in nook n, and not just its eyes.
func (w *World) foxIn(n int) bool { return w.Fox.Phase == FoxThere && w.Fox.Nook == n }

// foxShowing reports whether the fox, or its eyes, is in nook n.
func (w *World) foxShowing(n int) bool { return w.Fox.Phase != FoxAway && w.Fox.Nook == n }
