package config

// LaneKind is what a row of the playfield holds (DESIGN §5.2).
type LaneKind int

// The lane kinds. River and rail arrive in M3 and M4.
const (
	Safe LaneKind = iota // nothing can hurt the hero
	Road                 // vehicles; touching one is a death
)

// Vehicle is a kind of road vehicle. Its ID names its sprite.
type Vehicle struct {
	ID     string
	Length int // pixels, nose to tail
}

// The vehicles.
var (
	Bike      = Vehicle{ID: "bike", Length: 24}
	CarRed    = Vehicle{ID: "car_red", Length: 32}
	CarBlue   = Vehicle{ID: "car_blue", Length: 32}
	CarYellow = Vehicle{ID: "car_yellow", Length: 32}
	Van       = Vehicle{ID: "van", Length: 40}
	Truck     = Vehicle{ID: "truck", Length: 72}
)

// Vehicles lists every vehicle, for checks and tests.
func Vehicles() []Vehicle { return []Vehicle{Bike, CarRed, CarBlue, CarYellow, Van, Truck} }

// LongestVehicle is the length of the longest vehicle, in pixels. Lanes
// hide vehicles this far off the field before they come on.
const LongestVehicle = 72

// Spawn is one vehicle in a lane's pattern and the gap behind it.
type Spawn struct {
	Vehicle Vehicle
	Gap     int // pixels of clear road before the next vehicle
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

// Levels are the game's levels, in order. M2 has one road level; river
// rows, the rail and the rest of the levels come in M3–M5.
var Levels = []Level{
	{Lanes: [Rows]Lane{
		7: {Kind: Road, Dir: -1, Speed: 22, Pattern: []Spawn{
			{CarYellow, 120}, {CarRed, 200},
		}},
		8: {Kind: Road, Dir: -1, Speed: 10, Pattern: []Spawn{
			{Truck, 150}, {Truck, 230},
		}},
		9: {Kind: Road, Dir: 1, Speed: 16, Pattern: []Spawn{
			{CarRed, 90}, {CarBlue, 90}, {CarYellow, 170},
		}},
		10: {Kind: Road, Dir: -1, Speed: 28, Pattern: []Spawn{
			{Bike, 260},
		}},
		11: {Kind: Road, Dir: 1, Speed: 12, Pattern: []Spawn{
			{Van, 110}, {Van, 190},
		}},
	}},
}
