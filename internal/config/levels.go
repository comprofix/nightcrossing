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
	HeronPct int    // the chance, per log, that a heron stands on it (D34)
	Hint     string // shown when the level starts, for a new mechanic
}

// Level is a level as it is played: its lanes at their real speeds and
// gaps, its heron chance and its timer.
type Level struct {
	Number     int // 1 for the first
	Lanes      [Rows]Lane
	HeronPct   int
	TimerTicks int // the time for each attempt
	Hint       string
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
	d.Hint = "" // each hint shows once, on the level that brings it in
	return d, min(d.SpeedPct+(k/LoopLevels+1)*LoopSpeedStepPct, m.CeilingPct)
}

// ModeLevels is a mode's levels and speed ceiling.
type ModeLevels struct {
	Levels     []LevelDef
	CeilingPct int // the speed % the loops stop rising at (D20)
}

// Level is level n (1 for the first) as it is played.
func (m ModeLevels) Level(n int) Level {
	def, speed := m.Def(n)
	lv := Level{Number: n, HeronPct: def.HeronPct, TimerTicks: TimerTicks(n), Hint: def.Hint}
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
	return lv
}
