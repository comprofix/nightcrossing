package config

// Random events (DESIGN §6.1, D28): each is announced, lasts a set time
// and comes from the seeded RNG, more often each level. Starting values,
// tuned in M11–M12.

// EventKind is a kind of random event.
type EventKind int

// The kinds. Fog and rain come in M12.
const (
	Wave EventKind = iota // a rush hour wave on one road lane (D43)
	NumEventKinds
)

// EventTiming is how one kind of event runs on a level: the quiet between
// them, a random length from QuietMin to QuietMax ticks, and how long each
// lasts. A zero QuietMax means the event is off.
type EventTiming struct {
	QuietMin, QuietMax int
	Last               int
}

// On reports whether the event runs at all.
func (t EventTiming) On() bool { return t.QuietMax > 0 }

// EventDef is a kind of event's rules: its warning, the message shown in
// the hint strip through the warning, and its timing on the level it
// arrives and from EventTopLevel on (D41: each ramps like the trains).
type EventDef struct {
	WarnTicks  int
	Message    string
	First, Top EventTiming
}

// Events are the event kinds' rules, by kind.
var Events = [NumEventKinds]EventDef{
	Wave: {
		WarnTicks: 2 * TPS,
		Message:   "TRAFFIC BUILDING: A RUSH IS COMING",
		First:     EventTiming{QuietMin: 16 * TPS, QuietMax: 22 * TPS, Last: 8 * TPS},
		Top:       EventTiming{QuietMin: 7 * TPS, QuietMax: 11 * TPS, Last: 12 * TPS},
	},
}

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
	}
}

// LevelCalmTicks is how long a level's start stays free of events: as long
// as its opening hint shows, since warnings use the same strip (DESIGN
// §6.1).
const LevelCalmTicks = 6 * TPS

// EventRetryTicks is how soon an event that could not start yet (the calm
// start, or another weather on) tries again.
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
