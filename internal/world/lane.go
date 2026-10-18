package world

import (
	"math/rand/v2"
	"slices"

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
	Speed    int // SubPx units per tick
	Loop     int // SubPx units
	Entities []Entity
	HeronPct int  // the chance a log coming on has a heron (DESIGN §5.2, D34)
	Stalk    bool // herons stalk along their logs (D67)
	Strike   bool // and strike the cell they face (D67)
	Waving   bool // a rush hour wave is on: its vehicles come on as they come round (D43)
	Current  int  // the current's slide along a platform, SubPx units per tick: + with the flow, - against (D46)
	Surge    int  // how much faster a flood makes the river run, in percent (D48)
	Role     Role // the lane's part in an emergency vehicle's coming (D72)
	Brakes   bool // a queue's cars show their brake lights: they are standing for the vehicle

	lastHeron bool // the last log to come on had a heron
	held      bool // its cars move the emergency's way, or find their way back to their places, not with the lane
	pace      int  // how fast its cars move while held, in SubPx units per tick
	extras    int  // queue cars brought on so far, so they take the lane's kinds in turn
}

// Role is a road lane's part in an emergency vehicle's coming (D72).
type Role int

// The roles.
const (
	Unheld Role = iota // as usual, or its cars finding their way back to their places
	Clear              // the emergency lane: its cars drive off and none come on
	Queue              // a lane either side: its cars bunch up and stand, giving way to the hero
)

// Entity is one vehicle on a road lane or platform on a river lane.
type Entity struct {
	Kind  config.Entity
	U     int  // its left end's position along the loop, in SubPx units
	Dives bool // lily pads that dive (DESIGN §5.2, D33)
	Phase int  // ticks into its dive cycle, for one that dives
	Sinks bool // a ship that sinks for good (DESIGN §5.2, D33)
	Sink  int  // ticks into sinking, for one that sinks; 0 while it floats
	Wait  int  // ticks wholly on the field left before it starts to sink

	Heron     bool // a heron stands on this log (D34)
	HeronCell int  // which cell of the log it stands on, from the left end
	HeronDir  int  // the way it faces: +1 right, -1 left; a stalking heron walks this way (D67)
	Cock      int  // ticks left of a stalking heron cocking its head, the warning before it strikes; 0 when not (D67)
	Jab       int  // ticks left of its jab into the cell it faces; 0 when not
	Stride    int  // ticks left of a stalking heron's walk into the next cell; 0 while it stands
	heronStep int  // ticks until a stalking heron's next step
	strikeIn  int  // ticks until it next strikes

	Wave bool // an extra vehicle that only runs in a rush hour wave (D43)
	Out  bool // not on the lane: a wave vehicle between waves

	Home   int  // where the lane's pattern has it, while its lane is held for an emergency vehicle (D72)
	Away   bool // off the lane until its place comes round off the field: an emergency lane's car, or one back from a queue
	Extra  bool // a car come on to close a queue, gone once off the field after it
	Halted bool // a queue's car that stood this tick

	log bool // a log, which herons can stand on
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

// newLane lays out a level's lane, with herons on its logs at heronPct. If
// waves is true, each road gap that can take one more of the vehicle before
// it and stay crossable gets a wave vehicle in its middle, out until a wave
// (D43). The pattern repeats until it fills the loop, and the whole lane starts at a random point along it,
// so each game looks a little different but the pattern holds. Each diving
// platform starts at a random point in its cycle, so they don't all go
// under together.
func newLane(l config.Lane, heronPct int, waves bool, rng *rand.Rand) Lane {
	lane := Lane{Kind: l.Kind, Dir: l.Dir, Speed: l.Speed, HeronPct: heronPct, Current: l.Current}
	if len(l.Pattern) == 0 {
		return lane
	}

	pattern := 0
	for _, s := range l.Pattern {
		pattern += (s.Entity.Length + s.Gap) * config.SubPx
	}
	repeats := (fieldWidth + Buffer + pattern - 1) / pattern
	lane.Loop = repeats * pattern

	u := rng.IntN(lane.Loop)
	for range repeats {
		for _, s := range l.Pattern {
			e := Entity{Kind: s.Entity, U: u % lane.Loop, Dives: s.Dives, Sinks: s.Sinks,
				log: slices.Contains(config.Logs(), s.Entity)}
			if e.Dives {
				e.Phase = rng.IntN(config.DiveCycle)
			}
			if e.Sinks {
				e.Wait = sinkWait(rng)
			}
			lane.Entities = append(lane.Entities, e)
			if waves && l.Kind == config.Road && config.WaveFits(s.Gap, s.Entity.Length, l.Speed) {
				w := (u + (s.Entity.Length+(s.Gap-s.Entity.Length)/2)*config.SubPx) % lane.Loop
				lane.Entities = append(lane.Entities, Entity{Kind: s.Entity, U: w, Wave: true, Out: true})
			}
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
		wasOff := e.off()
		if l.held {
			e.Home = l.wrap(e.Home + l.Step()) // its place moves on; the emergency moves the car (D72)
		} else {
			e.U = l.wrap(e.U + l.Step())
		}
		if e.Wave && e.off() {
			e.Out = !l.Waving // only while wholly off the field, so none pops in or out
		}
		if e.log {
			switch off := e.off(); {
			case wasOff && !off:
				l.placeHeron(e, rng)
			case !wasOff && off:
				e.Heron = false
			}
			if e.Heron && l.Stalk {
				l.stalk(e, rng)
			}
		}
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
		e.Sink = min(e.Sink+1, config.SinkTicks(e.Kind))
	case left >= 0 && right <= fieldWidth:
		if e.Wait--; e.Wait <= 0 {
			e.Sink = 1
		}
	}
}

// wrap is u brought back onto the lane's loop.
func (l *Lane) wrap(u int) int { return (u%l.Loop + l.Loop) % l.Loop }

// Hidden reports whether e is off the lane: a wave vehicle between waves,
// or a car away for an emergency vehicle (D43, D72).
func (e Entity) Hidden() bool { return e.Out || e.Away }

// Step is how far the lane moves things in one tick, in SubPx units, with
// its direction's sign: faster in a flood (D48).
func (l *Lane) Step() int { return l.Dir * (l.Speed + l.Speed*l.Surge/100) }

// Slide is how far the lane's current moves a hero on one of its
// platforms in one tick, on top of the platform's own Step, in SubPx units
// with its direction's sign (D46): faster in a flood, like the river (D48).
func (l *Lane) Slide() int { return l.Dir * (l.Current + l.Current*l.Surge/100) }

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
	case s < config.SinkTicks(e.Kind):
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
		if e.Hidden() {
			continue
		}
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
	return min(max(from/cellUnits, 0), config.Pieces(e.Kind)-1)
}

// off reports whether e is wholly off the field.
func (e Entity) off() bool {
	left, right := e.span(0)
	return right <= 0 || left >= fieldWidth
}

// placeHeron decides, as log e comes on, whether a heron stands on it and
// where. It is decided while the log is still wholly off the field, so a
// heron never appears under the hero or pops into view: it rides in with
// its log. The log after a heron's log never has one, so a log without a
// heron is never more than one log away (DESIGN §5.2, D34). A standing
// heron faces the way its log moves; a stalking one faces either way and
// sets off along the log (D67).
func (l *Lane) placeHeron(e *Entity, rng *rand.Rand) {
	e.Heron = false
	if l.HeronPct <= 0 {
		return
	}
	if !l.lastHeron && rng.IntN(100) < l.HeronPct {
		e.Heron, e.HeronCell, e.HeronDir = true, rng.IntN(e.cells()), l.Dir
		e.Cock, e.Jab, e.Stride = 0, 0, 0
		if l.Stalk {
			if rng.IntN(2) == 0 {
				e.HeronDir = -e.HeronDir
			}
			e.heronStep, e.strikeIn = config.HeronStepTicks, strikeWait(rng)
			e.turn()
		}
	}
	l.lastHeron = e.Heron
}

// HeronX is how far e's heron is from the log's left end, in SubPx units:
// its cell, and part of the way to the next while it walks (D67).
func (e Entity) HeronX() int {
	x := e.HeronCell * cellUnits
	if e.Stride > 0 {
		x += e.HeronDir * cellUnits * (config.HeronStrideTicks - e.Stride) / config.HeronStrideTicks
	}
	return x
}

// cells is how many cells long e is.
func (e Entity) cells() int { return e.Kind.Length / config.Cell }

// turn turns e's heron round if the cell it faces is off the end of its
// log, so it always faces a cell of the log.
func (e *Entity) turn() {
	if c := e.HeronCell + e.HeronDir; c < 0 || c >= e.cells() {
		e.HeronDir = -e.HeronDir
	}
}

// strikeWait is how long a stalking heron walks before it next strikes.
func strikeWait(rng *rand.Rand) int {
	return config.HeronStrikeMinTicks + rng.IntN(config.HeronStrikeMaxTicks-config.HeronStrikeMinTicks+1)
}

// stalk moves e's heron one tick on (D67): a step along its log every
// config.HeronStepTicks, standing and then walking smoothly into the next
// cell, turning at the ends; and on a lane where herons strike, now and
// then, standing, it cocks its head, the warning, then jabs the cell it
// faces, and walks on.
func (l *Lane) stalk(e *Entity, rng *rand.Rand) {
	switch {
	case e.Jab > 0:
		if e.Jab--; e.Jab == 0 {
			e.strikeIn = strikeWait(rng)
		}
		return
	case e.Cock > 0:
		if e.Cock--; e.Cock == 0 {
			e.Jab = config.HeronJabTicks
		}
		return
	case e.Stride > 0:
		if e.Stride--; e.Stride == 0 {
			e.HeronCell += e.HeronDir
			e.turn()
			e.heronStep = config.HeronStepTicks - config.HeronStrideTicks
		}
		return
	}
	if l.Strike {
		if e.strikeIn--; e.strikeIn <= 0 {
			e.Cock = config.HeronCockTicks
			return
		}
	}
	if e.heronStep--; e.heronStep <= 0 {
		e.Stride = config.HeronStrideTicks
	}
}

// heronIn reports whether a heron overlaps left–right, in SubPx units:
// where it is, mid-walk too, and while it jabs, the cell it faces (D67).
// Each box is a cell trimmed by config.HitMargin at each side.
func (l *Lane) heronIn(left, right int) bool {
	m := config.HitMargin * config.SubPx
	in := func(e Entity, at int) bool {
		hl := e.X() + at + m
		return hl < right && left < hl+cellUnits-2*m
	}
	for _, e := range l.Entities {
		if !e.Heron {
			continue
		}
		if in(e, e.HeronX()) || e.Jab > 0 && in(e, (e.HeronCell+e.HeronDir)*cellUnits) {
			return true
		}
	}
	return false
}
