package config

// Random events (DESIGN §6.1, D28): each is announced, lasts a set time
// and comes from the seeded RNG, more often each level. Starting values,
// tuned in M11–M12.

// EventKind is a kind of random event.
type EventKind int

// The kinds.
const (
	Wave       EventKind = iota // a rush hour wave on one road lane (D43)
	Fog                         // fog over the road (D29, D44)
	Rain                        // rain: the road is slippery (D31, D44)
	StormFront                  // a storm front rolling in and away: wet (rain, a flood) or dry (wind) (D51)
	NumEventKinds
)

// EventTiming is how one kind of event runs on a level: the quiet between
// them, a random length from QuietMin to QuietMax ticks, and how long each
// lasts, Last ticks, or a random length from Last to LastMax if LastMax is
// more. A storm also has the ticks between a dry storm's pushes, 0 while
// every storm is wet (D47), and how many verge cells a wet storm floods, 0
// for no flood (D48). A zero QuietMax means the event is off.
type EventTiming struct {
	QuietMin, QuietMax int
	Last, LastMax      int
	Beat               int
	Size               int
}

// On reports whether the event runs at all.
func (t EventTiming) On() bool { return t.QuietMax > 0 }

// EventDef is a kind of event's rules: its warning, the message shown in
// the hint strip through the warning, and its timing on the level it
// arrives and from EventTopLevel on (D41: each ramps like the trains).
type EventDef struct {
	WarnTicks  int
	Message    string
	Weather    bool // only one weather at a time (D42)
	Big        bool // only one big event at a time: a wave or a storm (D48, D51)
	First, Top EventTiming
}

// Events are the event kinds' rules, by kind.
var Events = [NumEventKinds]EventDef{
	Wave: {
		WarnTicks: 2 * TPS,
		Message:   "TRAFFIC BUILDING: A RUSH IS COMING",
		Big:       true,
		First:     EventTiming{QuietMin: 16 * TPS, QuietMax: 22 * TPS, Last: 8 * TPS},
		Top:       EventTiming{QuietMin: 7 * TPS, QuietMax: 11 * TPS, Last: 12 * TPS},
	},
	Fog: {
		WarnTicks: 2 * TPS,
		Message:   "FOG ROLLING IN: WATCH FOR HEADLIGHTS",
		Weather:   true,
		First:     EventTiming{QuietMin: 24 * TPS, QuietMax: 32 * TPS, Last: 10 * TPS},
		Top:       EventTiming{QuietMin: 14 * TPS, QuietMax: 20 * TPS, Last: 14 * TPS},
	},
	Rain: {
		WarnTicks: 2 * TPS,
		Message:   "RAIN: ROADS SLIPPERY",
		Weather:   true,
		First:     EventTiming{QuietMin: 24 * TPS, QuietMax: 32 * TPS, Last: 10 * TPS},
		Top:       EventTiming{QuietMin: 14 * TPS, QuietMax: 20 * TPS, Last: 14 * TPS},
	},
	// The quiet is the clear spell between storms, and Last to LastMax the
	// storm holding. The wind's beat and the flood's size come from the
	// level (WindBeat, FloodSize).
	StormFront: {
		WarnTicks: 2 * TPS,
		Message:   StormMessages[0], // each storm names its own kind: see StormMessage
		Big:       true,
		First:     EventTiming{QuietMin: 45 * TPS, QuietMax: 60 * TPS, Last: 15 * TPS, LastMax: 18 * TPS},
		Top:       EventTiming{QuietMin: 20 * TPS, QuietMax: 30 * TPS, Last: 22 * TPS, LastMax: 25 * TPS},
	},
}

// StormKind is a kind of storm (D51).
type StormKind int

// The kinds. AnyStorm leaves it to the level's seeded mix.
const (
	AnyStorm StormKind = iota
	WetStorm           // rain for the whole storm, and from level 3 a flood
	DryStorm           // the wind for the whole storm, pushing the hero one cell on a beat
)

// StormRunMax is the most storms of one kind in a row; the next is the
// other kind (D51).
const StormRunMax = 2

// StormMessages are a storm's warnings: wet, and dry with the wind from
// the east (pushing the hero west, left) or from the west (D51).
var StormMessages = [3]string{
	"STORM ROLLING IN: HEAVY RAIN",
	"STORM ROLLING IN: HIGH WINDS FROM THE EAST",
	"STORM ROLLING IN: HIGH WINDS FROM THE WEST",
}

// StormMessage is the warning for a storm: wet, or dry with its wind
// pushing the hero dir, -1 left or +1 right.
func StormMessage(wet bool, dir int) string {
	switch {
	case wet:
		return StormMessages[0]
	case dir < 0:
		return StormMessages[1]
	}
	return StormMessages[2]
}

// StormClearMin to StormClearMax is how long after each storm no rain or
// fog starts, so a properly clear stretch follows it (D51).
const (
	StormClearMin = 8 * TPS
	StormClearMax = 10 * TPS
)

// StormFogPct is the chance fog that comes due in a dry storm starts,
// rather than waiting a whole quiet again: half as often as in a clear
// spell. StormFogDensityPct is how thick it is then, so it is thinner
// (D51).
const (
	StormFogPct        = 50
	StormFogDensityPct = 60
)

// A dry storm's wind pushes every WindBeatFirst ticks on the level dry
// storms arrive, WindBeatNext on the level after, and on a quicker beat
// each level to WindBeatTop at EventTopLevel (D47).
const (
	WindBeatFirst = 4 * TPS
	WindBeatNext  = 3 * TPS
	WindBeatTop   = 2 * TPS
)

// WindBeat is the wind's beat on level n of a mode with dry storms from
// level from, or 0 before that, when every storm is wet.
func WindBeat(from, n int) int {
	switch {
	case from <= 0 || n < from:
		return 0
	case n == from:
		return WindBeatFirst
	}
	span := max(EventTopLevel-from-1, 1)
	step := min(n-from-1, span)
	return WindBeatNext + (WindBeatTop-WindBeatNext)*step/span
}

// A wet storm's flood takes about FloodSizeFirst verge cells on the level
// floods arrive, rising to FloodSizeTop at EventTopLevel (D48).
const (
	FloodSizeFirst = 4
	FloodSizeTop   = 12
)

// FloodSize is how many verge cells a wet storm floods on level n of a
// mode with floods from level from, or 0 before that.
func FloodSize(from, n int) int {
	if from <= 0 || n < from {
		return 0
	}
	span := max(EventTopLevel-from, 1)
	step := min(n-from, span)
	return FloodSizeFirst + (FloodSizeTop-FloodSizeFirst)*step/span
}

// A flood's verge and river (D48). It comes StormFloodDelay ticks after a
// wet storm's rain sets in and lasts until the storm rolls away: its
// cells show puddles for FloodPuddleTicks, are water, then show puddles
// again as they drain at the storm's end. The river rows and their currents
// run FloodSurgePct faster, easing in and out over FloodSurgeTicks.
// Flooded cells come in stretches of FloodStretchMin to FloodStretchMax,
// with at least FloodDryMin dry cells between them and at the field's
// edges, and never more than half the verge goes under.
const (
	StormFloodDelay  = 3 * TPS
	FloodPuddleTicks = TPS
	FloodSurgePct    = 20
	FloodSurgeTicks  = TPS
	FloodStretchMin  = 2
	FloodStretchMax  = 4
	FloodDryMin      = 3
	FloodMostCells   = Cols / 2
)

// WindTellTicks is how long the wind streaks build before each push, so a
// push is never a surprise (D47).
const WindTellTicks = TPS / 2

// EventTopLevel is the level events reach their most often and longest on,
// as the trains do (D17, D41).
const EventTopLevel = TrainTopLevel

// EventsOn is how event kind k runs on level n of a mode that brings it in
// on level from; off before that, or when from is 0.
func EventsOn(k EventKind, from, n int) EventTiming {
	if from <= 0 || n < from {
		return EventTiming{}
	}
	d := Events[k]
	span := max(EventTopLevel-from, 1)
	step := min(n-from, span)
	at := func(first, top int) int { return first + (top-first)*step/span }
	return EventTiming{
		QuietMin: at(d.First.QuietMin, d.Top.QuietMin),
		QuietMax: at(d.First.QuietMax, d.Top.QuietMax),
		Last:     at(d.First.Last, d.Top.Last),
		LastMax:  at(d.First.LastMax, d.Top.LastMax),
	}
}

// Spread is how many ticks longer than Last an event of timing t can be.
func (t EventTiming) Spread() int { return max(t.LastMax-t.Last, 0) }

// LevelCalmTicks is how long a level's start stays free of events: as long
// as its opening hint shows, since warnings use the same strip (DESIGN
// §6.1).
const LevelCalmTicks = 6 * TPS

// EventRetryTicks is how soon an event that could not start yet (the calm
// start, another weather on, or another big event on) tries again.
const EventRetryTicks = TPS

// WaveRushPct shortens the quiet between waves at dawn and dusk, the
// morning and evening rush (D43).
const WaveRushPct = 60

// GapOpenTicks is how long a gap of gap pixels between vehicles in a road
// lane moving at speed (SubPx units per tick) stays open over the hero's
// cell, with each vehicle trimmed by HitMargin and the hero by HeroMargin
// (DESIGN §5.7, D21). It must be at least CrossOpenTicks.
func GapOpenTicks(gap, speed int) int {
	free := (gap + 2*HitMargin) - (Cell - 2*HeroMargin)
	return free * SubPx / speed
}

// WaveFits reports whether a wave can put a vehicle n pixels long in the
// middle of a gap of gap pixels in a lane moving at speed, leaving both
// halves crossable (D42, D43: a wave always leaves a way across).
func WaveFits(gap, n, speed int) bool {
	return GapOpenTicks((gap-n)/2, speed) >= CrossOpenTicks
}

// OvercastTicks is how long fog or rain takes to darken the light, and to
// lift it again (D44): as long as its warning, so it is overcast by the
// time it starts.
const OvercastTicks = 2 * TPS

// StormSkyTicks is how long a storm's sky takes to roll in once the storm
// begins, and to roll away after it ends (D49, D51).
const StormSkyTicks = 3 * TPS
