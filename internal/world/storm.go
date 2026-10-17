package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// The storm front (DESIGN §6.4, D51). Storm's level has one storm spell:
// its quiet is the clear spell, and while it is on the storm holds. A wet
// storm rains all through, and from level 3 floods the verge a little
// after the rain sets in; a dry storm's wind pushes the hero one cell on a
// beat. Each storm's kind, and a dry storm's direction, are chosen as the
// clear spell before it begins, so the weather can be fitted round it.

// nextStorm chooses the kind of storm s brings next: every storm the
// testing kind if there is one; wet while the level has no wind; the
// level's first storm the kind it brings in; then a seeded coin flip,
// never more than config.StormRunMax of a kind in a row (D51).
func (w *World) nextStorm(s *Spell) {
	lv := &w.Level
	switch {
	case lv.Storms != config.AnyStorm:
		s.Wet = lv.Storms == config.WetStorm
	case s.Timing.Beat == 0:
		s.Wet = true
	case w.storms == 0 && lv.FirstStorm != config.AnyStorm:
		s.Wet = lv.FirstStorm == config.WetStorm
	case w.stormRun >= config.StormRunMax:
		s.Wet = !w.lastWet
	default:
		s.Wet = w.rng.IntN(2) == 0
	}
	if w.storms > 0 && s.Wet == w.lastWet {
		w.stormRun++
	} else {
		w.stormRun = 1
	}
	w.storms++
	w.lastWet = s.Wet
	s.Dir = 0
	if !s.Wet {
		s.Dir = 1
		if w.rng.IntN(2) == 0 {
			s.Dir = -1
		}
	}
}

// warnStorm announces storm s: its seeded length and, for a wet storm with
// floods, the verge cells it will take.
func (w *World) warnStorm(s *Spell) {
	s.Length = s.Timing.Last + w.rng.IntN(s.Timing.Spread()+1)
	s.Cells = [config.Cols]bool{}
	if s.Wet && s.Timing.Size > 0 && w.Lanes[config.VergeRow].Kind == config.Safe {
		s.Cells = floodCells(s.Timing.Size, w.rng)
	}
	if s.Wet {
		w.emit(WetStormWarning)
	} else {
		w.emit(DryStormWarning)
	}
}

// endStorm rolls storm s away: its flood drains, a clear stretch follows,
// and the next storm's kind is chosen.
func (w *World) endStorm(s *Spell) {
	if s.Floods() {
		w.emit(FloodEnd)
	}
	s.Cells = [config.Cols]bool{}
	w.emit(StormEnd)
	gap := config.StormClearMin + w.rng.IntN(config.StormClearMax-config.StormClearMin+1)
	w.clearUntil = w.Tick + int64(gap)
	w.nextStorm(s)
}

// stormAllows reports whether weather s may be announced now, as the
// storm front allows (D51):
//   - in the clear stretch after a storm, no weather starts;
//   - in a clear spell, weather may run on into the next storm only if
//     that storm allows it, rain into a wet one and fog into a dry one;
//     otherwise it must be over before the storm's warning;
//   - in a storm, only fog, in a dry one, once the storm is on, so its
//     warning never covers the storm's.
func (w *World) stormAllows(s *Spell) bool {
	st := w.Spell(config.StormFront)
	if st == nil {
		return true
	}
	if w.Tick < w.clearUntil {
		return false
	}
	if st.Phase != SpellQuiet {
		return s.Kind == config.Fog && w.inStorm()
	}
	need := config.Events[s.Kind].WarnTicks
	if st.Wet != (s.Kind == config.Rain) {
		need += s.Timing.Last
	}
	return need < st.Left
}

// inStorm reports whether a dry storm is on.
func (w *World) inStorm() bool {
	s := w.Spell(config.StormFront)
	return s != nil && s.Phase == SpellOn && !s.Wet
}

// beat is how often dry storm s pushes: a storm made dry for testing on a
// level with no wind yet blows at the first level's beat.
func (s *Spell) beat() int {
	if s.Timing.Beat > 0 {
		return s.Timing.Beat
	}
	return config.WindBeatFirst
}

// NextPush is how many ticks until a dry storm's next push, or -1 if it
// pushes no more: it pushes every beat while it is on (D47).
func (s *Spell) NextPush() int {
	if s.Kind != config.StormFront || s.Wet {
		return -1
	}
	beat := s.beat()
	switch s.Phase {
	case SpellWarning:
		if beat < s.Length {
			return s.Left + beat
		}
	case SpellOn:
		if n := beat - s.since()%beat; n < s.Left {
			return n
		}
	}
	return -1 // quiet, or the storm rolls away first
}

// pushesNow reports whether a dry storm pushes this tick.
func (s *Spell) pushesNow() bool {
	return s.Phase == SpellOn && s.Kind == config.StormFront && !s.Wet && s.since()%s.beat() == 0
}

// blow pushes the hero one cell dir, now if it is still or when it next
// lands (D47). A hero dying or home in a nook is not pushed. The wind
// doesn't blow on the toadlets: they take the hero's push as they follow
// (D53).
func (w *World) blow(dir int) {
	if w.out(Hero) {
		w.Player.push = dir
	}
}

// Floods reports whether storm s brings a flood.
func (s *Spell) Floods() bool {
	if s.Kind != config.StormFront || !s.Wet {
		return false
	}
	for _, c := range s.Cells {
		if c {
			return true
		}
	}
	return false
}

// floodsNow reports whether storm s's flood comes in this tick, a little
// after its rain (D48).
func (s *Spell) floodsNow() bool {
	return s.Phase == SpellOn && s.since() == config.StormFloodDelay && s.Floods()
}

// flood is the storm's flood, if one is in, and how long it has been in.
func (w *World) flood() (s *Spell, t int, ok bool) {
	s = w.Spell(config.StormFront)
	if s == nil || s.Phase != SpellOn || !s.Floods() || s.since() < config.StormFloodDelay {
		return nil, 0, false
	}
	return s, s.since() - config.StormFloodDelay, true
}

// floodCells picks the verge cells a flood of about n cells takes, in
// stretches of config.FloodStretchMin to FloodStretchMax with at least
// config.FloodDryMin dry cells between them and at the field's edges, and
// never more than config.FloodMostCells (D48). It tries seeded layouts
// until one fits, and floods fewer cells if none does.
func floodCells(n int, rng *rand.Rand) [config.Cols]bool {
	const lo, hi, dry = config.FloodStretchMin, config.FloodStretchMax, config.FloodDryMin
	n = min(n, config.FloodMostCells)
	for ; n >= lo; n-- {
		for range 50 {
			var cells [config.Cols]bool
			if placeStretches(&cells, n, lo, hi, dry, rng) {
				return cells
			}
		}
	}
	return [config.Cols]bool{}
}

// placeStretches floods n cells of cells in stretches of lo to hi, each
// with dry cells either side of it, and reports whether they all fit.
func placeStretches(cells *[config.Cols]bool, n, lo, hi, dry int, rng *rand.Rand) bool {
	for n > 0 {
		l := n
		if n > hi {
			l = lo + rng.IntN(hi-lo+1)
			l = min(l, n-lo) // never leave a stretch too short to make
		}
		placed := false
		for range 20 {
			start := rng.IntN(config.Cols - l + 1)
			if stretchFits(cells, start, l, dry) {
				for c := start; c < start+l; c++ {
					cells[c] = true
				}
				placed = true
				break
			}
		}
		if !placed {
			return false
		}
		n -= l
	}
	return true
}

// stretchFits reports whether l cells from start can flood, keeping dry
// cells either side of them, out to the field's edges.
func stretchFits(cells *[config.Cols]bool, start, l, dry int) bool {
	end := start + l
	if start > 0 && start < dry || end < config.Cols && end > config.Cols-dry {
		return false // too narrow a strip left dry at the edge
	}
	for c := max(start-dry, 0); c < min(end+dry, config.Cols); c++ {
		if cells[c] {
			return false
		}
	}
	return true
}

// Flooded reports whether verge column col is under water: a flood is in,
// past its first puddles and not yet draining (D48).
func (w *World) Flooded(col int) bool {
	s, t, ok := w.flood()
	return ok && s.Cells[col] && t >= config.FloodPuddleTicks && s.Left > config.FloodPuddleTicks
}

// Puddled reports whether verge column col shows puddles: a flood is in
// and it is about to go under, or draining (D48). It is still dry ground.
func (w *World) Puddled(col int) bool {
	s, _, ok := w.flood()
	return ok && s.Cells[col] && !w.Flooded(col)
}

// surge is how much faster a flood makes the river run now, in percent:
// easing in as it comes and out as it drains (D48).
func (w *World) surge() int {
	s, t, ok := w.flood()
	if !ok {
		return 0
	}
	t = min(t, s.Left, config.FloodSurgeTicks)
	return config.FloodSurgePct * t / config.FloodSurgeTicks
}
