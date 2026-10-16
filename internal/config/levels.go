package config

// LaneKind is what a row of the playfield holds (DESIGN §5.2).
type LaneKind int

// The lane kinds.
const (
	Safe  LaneKind = iota // nothing can hurt the hero
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
	Speed   int     // SubPx units per tick at a skill's 100%
	Pattern []Spawn // repeated along the lane
}

// Level is the playfield's rows, top to bottom. Row GoalRow is the far
// bank and row StartRow the start; both are always safe.
type Level struct {
	Lanes [Rows]Lane
}

// Levels are the game's levels, in order. M3 has one level with every
// river platform on the rows of DESIGN §3.2 (D32); Stroll's real layouts
// and levels come in M4.
var Levels = []Level{
	{Lanes: [Rows]Lane{
		1: {Kind: River, Dir: 1, Speed: 10, Pattern: []Spawn{
			{Entity: LogLong, Gap: 96}, {Entity: LogShort, Gap: 120},
		}},
		2: {Kind: River, Dir: -1, Speed: 14, Pattern: []Spawn{
			{Entity: Pads3, Gap: 72}, {Entity: Pads3, Gap: 72, Dives: true}, {Entity: Pads2, Gap: 96},
		}},
		3: {Kind: River, Dir: 1, Speed: 8, Pattern: []Spawn{
			{Entity: Ferry, Gap: 120}, {Entity: Barge, Gap: 144, Sinks: true},
		}},
		4: {Kind: River, Dir: -1, Speed: 16, Pattern: []Spawn{
			{Entity: Pads2, Gap: 72}, {Entity: Pads3, Gap: 96, Dives: true}, {Entity: Pads3, Gap: 72},
		}},
		5: {Kind: River, Dir: 1, Speed: 12, Pattern: []Spawn{
			{Entity: LogShort, Gap: 96}, {Entity: LogShort, Gap: 72}, {Entity: LogLong, Gap: 120},
		}},
		7: {Kind: Rail},
		8: {Kind: Road, Dir: -1, Speed: 10, Pattern: []Spawn{
			{Entity: Truck, Gap: 150}, {Entity: Truck, Gap: 230},
		}},
		9: {Kind: Road, Dir: 1, Speed: 16, Pattern: []Spawn{
			{Entity: CarRed, Gap: 90}, {Entity: CarBlue, Gap: 90}, {Entity: CarYellow, Gap: 170},
		}},
		10: {Kind: Road, Dir: -1, Speed: 28, Pattern: []Spawn{
			{Entity: Bike, Gap: 260},
		}},
		11: {Kind: Road, Dir: 1, Speed: 12, Pattern: []Spawn{
			{Entity: Van, Gap: 110}, {Entity: Van, Gap: 190},
		}},
	}},
}
