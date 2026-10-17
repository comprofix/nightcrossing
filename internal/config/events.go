package config

// Random events (DESIGN §6.1, D28): each is announced, lasts a set time
// and comes from the seeded RNG, more often each level. Starting values,
// tuned in M11–M12.

// EventKind is a kind of random event.
type EventKind int

// The kinds.
const (
	Wave  EventKind = iota // a rush hour wave on one road lane (D43)
	Fog                    // fog over the road (D29, D44)
	Rain                   // rain: the road is slippery (D31, D44)
	Gust                   // a gust spell: the wind pushes the hero one cell at a time (D47)
	Flood                  // a flood: verge cells go under and the river runs faster (D48)
	NumEventKinds
)

// EventTiming is how one kind of event runs on a level: the quiet between
// them, a random length from QuietMin to QuietMax ticks, how long each
// lasts, for a gust spell the ticks between its pushes (D47), and for a
// flood how many verge cells go under (D48). A zero QuietMax means the
// event is off.
type EventTiming struct {
	QuietMin, QuietMax int
	Last               int
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
	Big        bool // only one big event at a time: a wave, a gust spell or a flood (D48)
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
	Gust: {
		WarnTicks: 2 * TPS,
		Message:   GustMessages[1], // each spell shows its own direction: see GustMessage
		Big:       true,
		First:     EventTiming{QuietMin: 20 * TPS, QuietMax: 28 * TPS, Last: 8 * TPS, Beat: 3 * TPS},
		Top:       EventTiming{QuietMin: 12 * TPS, QuietMax: 18 * TPS, Last: 12 * TPS, Beat: 2 * TPS},
	},
	Flood: {
		WarnTicks: 2 * TPS,
		Message:   "FLOOD: THE RIVER IS RISING",
		Big:       true,
		First:     EventTiming{QuietMin: 24 * TPS, QuietMax: 32 * TPS, Last: 10 * TPS, Size: 4},
		Top:       EventTiming{QuietMin: 14 * TPS, QuietMax: 20 * TPS, Last: 14 * TPS, Size: 12},
	},
}

// A flood's verge and river (D48). Its Last runs from the first puddles
// to the last: the cells show puddles for FloodPuddleTicks, are water,
// then show puddles again as they drain. The river rows and their currents
// run FloodSurgePct faster, easing in and out over FloodSurgeTicks.
// Flooded cells come in stretches of FloodStretchMin to FloodStretchMax,
// with at least FloodDryMin dry cells between them and at the field's
// edges, and never more than half the verge goes under.
const (
	FloodPuddleTicks = TPS
	FloodSurgePct    = 20
	FloodSurgeTicks  = TPS
	FloodStretchMin  = 2
	FloodStretchMax  = 4
	FloodDryMin      = 3
	FloodMostCells   = Cols / 2
)

// GustMessages are a gust spell's warnings, by the way it blows: a wind
// from the east pushes the hero west (left), one from the west east (D47).
var GustMessages = [2]string{"GUSTS FROM THE EAST", "GUSTS FROM THE WEST"}

// GustMessage is the warning for a gust spell that pushes the hero dir:
// -1 left, +1 right.
func GustMessage(dir int) string {
	if dir < 0 {
		return GustMessages[0]
	}
	return GustMessages[1]
}

// GustTellTicks is how long the wind streaks build before each push, so a
// push is never a surprise (D47).
const GustTellTicks = TPS / 2

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
		Beat:     at(d.First.Beat, d.Top.Beat),
		Size:     at(d.First.Size, d.Top.Size),
	}
}

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
