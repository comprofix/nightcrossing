package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Lane is one row of the playfield as it runs.
//
// Each lane is a loop: positions run from 0 to Loop and wrap. A position u
// is drawn at x = u - Buffer from the field's left edge. Buffer is at least
// the longest entity and Loop at least the field's width plus Buffer, so an
// entity wraps only while it is wholly off the field and never pops in
// (PLAN §3).
type Lane struct {
	Kind     config.LaneKind
	Dir      int // +1 right, -1 left
	Speed    int // SubPx units per tick, after the skill's speed
	Loop     int // SubPx units
	Entities []Entity
}

// Entity is one vehicle on a road lane or platform on a river lane.
type Entity struct {
	Kind  config.Entity
	U     int  // its left end's position along the loop, in SubPx units
	Dives bool // lily pads that dive (DESIGN §5.2, D33)
	Phase int  // ticks into its dive cycle, for one that dives
	Sinks bool // a ship that sinks for good (DESIGN §5.2, D33)
	Sink  int  // ticks into sinking, for one that sinks; 0 while it floats
	Wait  int  // ticks wholly on the field left before it starts to sink
}

// Stage is where a diving platform is in its cycle (DESIGN §5.2).
type Stage int

// The stages. Lily pads go Afloat, Warning, Sinking, Under, Rising and round
// again. A ship goes Afloat, Warning, Breaking, Sinking, Sunk, and its
// pieces go under one by one while it is Sinking (see PieceStage). A
// platform that neither dives nor sinks is always Afloat.
const (
	Afloat   Stage = iota
	Warning        // still afloat, flashing
	Sinking        // going down, still holds the hero
	Under          // under: no platform here
	Rising         // half under, still holds the hero
	Breaking       // a ship breaking up, still holds the hero
	Sunk           // a ship gone for good: no platform here
)

// Buffer is how far off the field's left edge a lane's loop begins, in
// SubPx units: the longest entity.
const Buffer = config.Longest * config.SubPx

// fieldWidth is the playfield's width in SubPx units.
const fieldWidth = config.Cols * config.Cell * config.SubPx

// newLane lays out a level's lane for a skill. The pattern repeats until
// it fills the loop, and the whole lane starts at a random point along it,
// so each game looks a little different but the pattern holds. Each diving
// platform starts at a random point in its cycle, so they don't all go
// under together.
func newLane(l config.Lane, skill config.Skill, rng *rand.Rand) Lane {
	lane := Lane{Kind: l.Kind, Dir: l.Dir}
	if len(l.Pattern) == 0 {
		return lane
	}
	lane.Speed = max(1, l.Speed*skill.SpeedPct/100)

	pattern := 0
	for _, s := range l.Pattern {
		pattern += (s.Entity.Length + s.Gap) * config.SubPx
	}
	repeats := (fieldWidth + Buffer + pattern - 1) / pattern
	lane.Loop = repeats * pattern

	u := rng.IntN(lane.Loop)
	for range repeats {
		for _, s := range l.Pattern {
			e := Entity{Kind: s.Entity, U: u % lane.Loop, Dives: s.Dives, Sinks: s.Sinks}
			if e.Dives {
				e.Phase = rng.IntN(config.DiveCycle)
			}
			if e.Sinks {
				e.Wait = sinkWait(rng)
			}
			lane.Entities = append(lane.Entities, e)
			u += (s.Entity.Length + s.Gap) * config.SubPx
		}
	}
	return lane
}

// update moves the lane's entities one tick, moves each diving platform on
// through its cycle, and each sinking ship on towards the bottom.
func (l *Lane) update(rng *rand.Rand) {
	for i := range l.Entities {
		e := &l.Entities[i]
		e.U = ((e.U+l.Dir*l.Speed)%l.Loop + l.Loop) % l.Loop
		if e.Dives {
			e.Phase = (e.Phase + 1) % config.DiveCycle
		}
		if e.Sinks {
			e.updateSink(rng)
		}
	}
}

// sinkWait is how long a ship floats once it is wholly on the field.
func sinkWait(rng *rand.Rand) int {
	return config.SinkAfterMinTicks + rng.IntN(config.SinkAfterMaxTicks-config.SinkAfterMinTicks+1)
}

// updateSink moves a ship that sinks one tick on. Its wait runs down only
// while it is wholly on the field, so it never sinks out of sight. Once it
// has started, it goes on to the bottom. When it is wholly off the field
// again, a sunk or sinking ship is replaced by a new one in its place
// (DESIGN §5.2, D33).
func (e *Entity) updateSink(rng *rand.Rand) {
	left, right := e.span(0)
	switch {
	case right <= 0 || left >= fieldWidth:
		if e.Sink > 0 {
			e.Sink, e.Wait = 0, sinkWait(rng)
		}
	case e.Sink > 0:
		e.Sink = min(e.Sink+1, sinkTicks(e.Kind))
	case left >= 0 && right <= fieldWidth:
		if e.Wait--; e.Wait <= 0 {
			e.Sink = 1
		}
	}
}

// pieces is how many one-cell pieces a ship has.
func pieces(ship config.Entity) int { return ship.Length / config.Cell }

// sinkTicks is how long a ship takes from its warning to the bottom.
func sinkTicks(ship config.Entity) int {
	return config.SinkWarnTicks + config.SinkBreakTicks + pieces(ship)*config.SinkPieceTicks
}

// Step is how far the lane moves things in one tick, in SubPx units, with
// its direction's sign.
func (l *Lane) Step() int { return l.Dir * l.Speed }

// X is where e's left end is, from the field's left edge, in SubPx units.
func (e Entity) X() int { return e.U - Buffer }

// Stage is where e is in its dive cycle, or in sinking.
func (e Entity) Stage() Stage {
	if e.Sinks {
		return e.sinkStage()
	}
	if !e.Dives {
		return Afloat
	}
	t := e.Phase
	for _, s := range []struct {
		stage Stage
		ticks int
	}{
		{Afloat, config.DiveAfloatTicks},
		{Warning, config.DiveWarnTicks},
		{Sinking, config.DiveSinkTicks},
		{Under, config.DiveUnderTicks},
	} {
		if t < s.ticks {
			return s.stage
		}
		t -= s.ticks
	}
	return Rising
}

// sinkStage is where a ship that sinks is in sinking.
func (e Entity) sinkStage() Stage {
	switch s := e.Sink; {
	case s == 0:
		return Afloat
	case s <= config.SinkWarnTicks:
		return Warning
	case s <= config.SinkWarnTicks+config.SinkBreakTicks:
		return Breaking
	case s < sinkTicks(e.Kind):
		return Sinking
	}
	return Sunk
}

// PieceStage is where piece i of a ship is, counting from its bow: while
// the ship is Sinking, each piece is Breaking until its turn, Sinking as
// it goes down, then Under. At every other stage, it is the ship's stage.
func (e Entity) PieceStage(i int) Stage {
	st := e.Stage()
	if st != Sinking {
		return st
	}
	t := e.Sink - config.SinkWarnTicks - config.SinkBreakTicks // ticks into losing pieces
	switch {
	case t >= (i+1)*config.SinkPieceTicks:
		return Under
	case t > i*config.SinkPieceTicks:
		return Sinking
	}
	return Breaking
}

// FlashPhase is the count the warning flash of e runs from.
func (e Entity) FlashPhase() int {
	if e.Sinks {
		return e.Sink
	}
	return e.Phase
}

// span is the part of the field e covers, left to right, in SubPx units,
// trimmed by margin pixels at each end. A negative margin widens it.
func (e Entity) span(margin int) (left, right int) {
	m := margin * config.SubPx
	x := e.X()
	return x + m, x + e.Kind.Length*config.SubPx - m
}

// hits reports whether any vehicle in the lane overlaps left–right,
// with each vehicle's ends trimmed by margin pixels.
func (l *Lane) hits(left, right, margin int) bool {
	for _, e := range l.Entities {
		el, er := e.span(margin)
		if el < right && left < er {
			return true
		}
	}
	return false
}

// holds reports whether a platform is under the point x, in SubPx units
// from the field's left edge. Each platform reaches config.StandGrace
// pixels past its ends. Lily pads that are under and ships that have sunk
// hold nothing, and nor does a piece of a sinking ship that has gone under
// (DESIGN §5.2).
func (l *Lane) holds(x int) bool {
	for _, e := range l.Entities {
		if st := e.Stage(); st == Under || st == Sunk {
			continue
		}
		el, er := e.span(-config.StandGrace)
		if x < el || x >= er {
			continue
		}
		if e.Sinks && e.PieceStage(l.pieceFromBow(e, x)) == Under {
			continue
		}
		return true
	}
	return false
}

// pieceFromBow is which piece of ship e is at x, counting from its bow,
// the end it is heading towards. Points in the grace past an end count as
// the end piece.
func (l *Lane) pieceFromBow(e Entity, x int) int {
	left, right := e.span(0)
	from := x - left // from the left end
	if l.Dir > 0 {
		from = right - 1 - x
	}
	return min(max(from/cellUnits, 0), pieces(e.Kind)-1)
}
