package config

// LaneKind is what a row of the playfield holds (DESIGN §5.2).
type LaneKind int

// The lane kinds.
const (
	Safe  LaneKind = iota // nothing can hurt the hero
	Bank                  // the far bank: the goal nooks, with bank between them (DESIGN §5.3)
	Road                  // vehicles; touching one is a death
	River                 // platforms to ride; the water is a death
	Rail                  // the track: safe scenery in Stroll, trains from Last Train (D15)
)

// Entity is a kind of thing that moves along a lane: a vehicle on the road
// or a platform on the river. Its ID names its sprite, or the pieces it is
// built from.
type Entity struct {
	ID     string
	Length int // pixels, end to end
}

// The road vehicles.
var (
	Bike      = Entity{ID: "bike", Length: 24}
	CarRed    = Entity{ID: "car_red", Length: 32}
	CarBlue   = Entity{ID: "car_blue", Length: 32}
	CarYellow = Entity{ID: "car_yellow", Length: 32}
	Van       = Entity{ID: "van", Length: 40}
	Truck     = Entity{ID: "truck", Length: 72}
)

// The river platforms (DESIGN §2, §5.2): logs, a barge, the ferry, and
// clumps of lily pads, which can dive.
var (
	LogShort = Entity{ID: "log_short", Length: 72}
	LogLong  = Entity{ID: "log_long", Length: 120}
	Barge    = Entity{ID: "barge", Length: 96}
	Ferry    = Entity{ID: "ferry", Length: 144}
	Pads2    = Entity{ID: "pads2", Length: 48}
	Pads3    = Entity{ID: "pads3", Length: 72}
)

// Logs lists the platforms herons can stand on (DESIGN §5.2, D34).
func Logs() []Entity { return []Entity{LogShort, LogLong} }

// Vehicles lists every road vehicle, for checks and tests.
func Vehicles() []Entity { return []Entity{Bike, CarRed, CarBlue, CarYellow, Van, Truck} }

// Ships lists the platforms that can sink. Each is built from pieces one
// cell long, which go under one by one (DESIGN §5.2).
func Ships() []Entity { return []Entity{Barge, Ferry} }

// Pads lists the platforms that can dive.
func Pads() []Entity { return []Entity{Pads2, Pads3} }

// Platforms lists every river platform, for checks and tests.
func Platforms() []Entity { return []Entity{LogShort, LogLong, Barge, Ferry, Pads2, Pads3} }

// Entities lists everything that moves along a lane.
func Entities() []Entity { return append(Vehicles(), Platforms()...) }

// Longest is the length of the longest entity, in pixels. Lanes hide
// entities this far off the field before they come on.
const Longest = 144

// Spawn is one entity in a lane's pattern and the gap behind it.
type Spawn struct {
	Entity Entity
	Gap    int  // pixels of clear road or open water before the next one
	Dives  bool // lily pads that dive and come back up (DESIGN §5.2, D33)
	Sinks  bool // a ship that sinks for good (DESIGN §5.2, D33)
}

// Lane is one row of a level. A lane's pattern repeats along the row, so
// it can be learned (DESIGN §1, §5.2).
type Lane struct {
	Kind    LaneKind
	Dir     int     // +1 moves right, -1 moves left
	Speed   int     // SubPx units per tick: at 100% in a layout, as it runs in a Level
	Pattern []Spawn // repeated along the lane
	Current int     // how fast a river row's current slides the hero along a platform, in SubPx units per tick: + with the row's flow, - against it; 0 for none (D46)
}

// Current is a river row's current on a level (D46, D50): Push is its
// strength in SubPx units per tick at 100% speed, + with the row's flow
// and - against it.
type Current struct {
	Row  int
	Push int
}

// Layout is a hand-written set of lanes, top to bottom, at 100% speed
// (DESIGN §5.7, D18). Row GoalRow is the far bank and row StartRow the
// start.
type Layout struct {
	Name  string
	Lanes [Rows]Lane
}

// LevelDef is one level of a mode: a layout made harder (D18). SpeedPct
// scales every lane's speed. GapPct scales the gaps between vehicles on
// the road, so the traffic gets busier; river gaps are left alone, since
// more platforms would make the river easier.
type LevelDef struct {
	Layout   *Layout
	SpeedPct int
	GapPct   int
	HeronPct int       // the chance, per log, that a heron stands on it (D34)
	Hint     string    // shown when the level starts, for a new mechanic
	Currents []Current // the river rows with a current (D46, D50)
	Toadlets int       // the toadlets in the hero's line (D53, D54)
}

// Level is a level as it is played: its lanes at their real speeds and
// gaps, its heron chance and its timer.
type Level struct {
	Number     int // 1 for the first
	Lanes      [Rows]Lane
	HeronPct   int
	Trains     bool                       // trains run on the track (DESIGN §6.2)
	Train      TrainTiming                // how the trains run, when they do (D17)
	Combo      bool                       // the hop combo is on (D38)
	DayCycle   bool                       // the day and night cycle runs (D39)
	Events     [NumEventKinds]EventTiming // the random events and how they run (DESIGN §6.1)
	FirstStorm StormKind                  // the kind of the level's first storm, to teach it; AnyStorm for the mix (D51)
	Storms     StormKind                  // every storm this kind, for testing (-storm, D52); AnyStorm for the mix
	Toadlets   int                        // the toadlets that follow the hero (D53)
	FireflyPct int                        // the chance of a firefly in a crossing; 0 for none (D58)
	FireflyOn  [NumHosts]bool             // the platforms a firefly can ride (D60)
	FruitPts   int                        // what fruit in a nook is worth; 0 for no fruit (D59)
	Fruit      int                        // which of Fruits it is
	TimerTicks int                        // the time for each attempt; on a bonus level, for the whole level
	Hint       string
	Bonus      *Bonus // the bonus level this is, or nil (D61)
}

// After the last level, the last LoopLevels levels loop. Each time round,
// only the speed rises, by LoopSpeedStepPct, and the road gaps stay as on
// the last level (DESIGN §5.7, D19). The speed stops rising at the
// ceiling (D20).
const (
	LoopLevels       = 3
	LoopSpeedStepPct = 10
)

// Def is the definition level n is played from, and its speed after the
// loops' rises and the ceiling.
func (m ModeLevels) Def(n int) (def LevelDef, speedPct int) {
	last := len(m.Levels)
	if n <= last {
		d := m.Levels[n-1]
		return d, d.SpeedPct
	}
	k := n - last - 1 // levels played since the loops began
	d := m.Levels[last-LoopLevels+k%LoopLevels]
	d.GapPct = m.Levels[last-1].GapPct
	d.Currents = m.Levels[last-1].Currents // the loops keep the last level's currents (D50)
	d.Toadlets = m.Levels[last-1].Toadlets // and its line of toadlets (D54)
	d.Hint = ""                            // each hint shows once, on the level that brings it in
	return d, min(d.SpeedPct+(k/LoopLevels+1)*LoopSpeedStepPct, m.CeilingPct)
}

// ModeLevels is a mode's levels and speed ceiling, and the levels its own
// mechanics come in on (D27).
type ModeLevels struct {
	Levels     []LevelDef
	CeilingPct int                // the speed % the loops stop rising at (D20)
	Trains     bool               // trains from level 1 (DESIGN §6.2)
	ComboFrom  int                // the first level with the hop combo; 0 for none (D38)
	CycleFrom  int                // the first level with the day and night cycle; 0 for none (D39)
	EventsFrom [NumEventKinds]int // the first level with each random event; 0 for none (D41)
	QuietPct   [NumEventKinds]int // each event's quiet at this % of its usual; 0 for 100 (D45)
	WindFrom   int                // the first level with dry storms; 0 for none (D47, D51)
	FloodFrom  int                // the first level with floods in wet storms; 0 for none (D48)

	FruitFrom   int           // the first level with fruit in the nooks; 0 for none (D59, D60)
	FireflyFrom [NumHosts]int // the first level with a firefly on each kind of platform; 0 for none (D58, D60)
}

// Level is level n (1 for the first) as it is played.
func (m ModeLevels) Level(n int) Level {
	def, speed := m.Def(n)
	lv := Level{Number: n, HeronPct: def.HeronPct, Trains: m.Trains, Toadlets: def.Toadlets, Hint: def.Hint,
		TimerTicks: TimerTicks(n) + def.Toadlets*ToadletSecs*TPS} // a second a toadlet, to file in (D53)
	if m.Trains {
		lv.Train = TrainsOn(n)
	}
	last := len(m.Levels)
	if lv.FruitPts = FruitPoints(m.FruitFrom, last, n); lv.FruitPts > 0 {
		lv.Fruit = FruitOf(n)
	}
	first := 0 // the first level with a firefly on any platform
	for h, from := range m.FireflyFrom {
		if lv.FireflyOn[h] = from > 0 && n >= from; from > 0 && (first == 0 || from < first) {
			first = from
		}
	}
	lv.FireflyPct = FireflyPct(first, n)
	lv.Combo = m.ComboFrom > 0 && n >= m.ComboFrom
	lv.DayCycle = m.CycleFrom > 0 && n >= m.CycleFrom
	for k := range lv.Events {
		t := EventsOn(EventKind(k), m.EventsFrom[k], n)
		if pct := m.QuietPct[k]; pct > 0 {
			t.QuietMin, t.QuietMax = t.QuietMin*pct/100, t.QuietMax*pct/100
		}
		lv.Events[k] = t
	}
	if s := &lv.Events[StormFront]; s.On() {
		s.Beat, s.Size = WindBeat(m.WindFrom, n), FloodSize(m.FloodFrom, n)
		switch { // each level that brings a kind of storm in starts with it (D51)
		case m.WindFrom == m.FloodFrom:
			// Both kinds come in at once, carried over from Storm: the mix (D54).
		case n == m.WindFrom:
			lv.FirstStorm = DryStorm
		case n == m.FloodFrom:
			lv.FirstStorm = WetStorm
		}
	}
	for r, l := range def.Layout.Lanes {
		if len(l.Pattern) > 0 {
			l.Speed = max(1, l.Speed*speed/100)
			pattern := make([]Spawn, len(l.Pattern))
			for i, s := range l.Pattern {
				if l.Kind == Road {
					s.Gap = max(Cell, s.Gap*def.GapPct/100)
				}
				pattern[i] = s
			}
			l.Pattern = pattern
		}
		lv.Lanes[r] = l
	}
	for _, c := range def.Currents {
		lv.Lanes[c.Row].Current = scaleSigned(c.Push, speed)
	}
	return lv
}

// scaleSigned is v at pct%, keeping its sign and never rounding to 0.
func scaleSigned(v, pct int) int {
	switch {
	case v > 0:
		return max(1, v*pct/100)
	case v < 0:
		return -max(1, -v*pct/100)
	}
	return 0
}
