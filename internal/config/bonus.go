package config

// Bonus is a mode's bonus level, played once a run after level 8's tally
// (DESIGN §6.7, D61). There are no deaths: a slip costs time, and the
// clock is the same for every run. Its pickups give points, never lives.
type Bonus struct {
	Name   string  // its title card, e.g. "FIREFLY NIGHT"
	Hint   string  // shown in the hint strip as it starts
	Layout *Layout // its lanes; road rows run no traffic
	// Parked is the parked cars on each road row, one character a cell:
	// '.' for a gap, or a letter of ParkedCars repeated for each cell the
	// car takes. A hop into a parked car is blocked.
	Parked    [Rows]string
	Secs      int // the clock
	Fireflies int // fireflies in all
	OnRoad    int // of them, how many come to the road; the rest come to the river
	AtOnce    int // on the field at the start
	Firefly   int // points for each
	All       int // points for collecting the whole set: every firefly and fruit
	// Dawn lights the level as Last Train's dawn, rather than the night,
	// and its fireflies come as dragonflies; Wet leaves the road shining
	// from the night's rain, only as a look, and Slippery makes it slide
	// as in the rain (D31).
	Dawn, Wet, Slippery bool
	// Rainbow hangs a rainbow over the river, and mist on the water: only
	// the look (DESIGN §6.7).
	Rainbow bool
	// RunMax is the most a run of catches multiplies a firefly's points
	// by, each caught without setting foot on land worth one more times
	// than the last; 0 for no runs (DESIGN §6.7).
	RunMax int
	// Stragglers are toadlets waiting at the kerbs (the start row, the
	// verge and the track) to be walked home: landing on one adds it to
	// the toad's line, which holds LineMax at most, and a nook brings the
	// line home for Straggler points each, doubled for a trip of TripMin
	// or more (DESIGN §6.7).
	Stragglers, LineMax, Straggler, TripMin int
	// Dusk lights the level in the late afternoon's golden light, and
	// Gates puts a school sign by each nook: only the look.
	Dusk, Gates bool
	// Ending follows the bonus tally with the game's ending and credits
	// (D62).
	Ending bool
	// Sweeper sends two street sweepers along the road lanes, one down
	// from the top and one up from the bottom, sweeping up fruit in their
	// path (DESIGN §6.7).
	Sweeper bool
	// Yard is the trains standing on the track rows, a goods yard, and
	// Fruit the fruit laid out at the start, in random places each run
	// (DESIGN §6.7).
	Yard  []Rake
	Fruit []BonusFruit
}

// Rake is a train standing in a bonus level's goods yard: carriages, with
// an engine or not, which the toad can stand on and ride (DESIGN §6.7).
type Rake struct {
	Row, Col int  // its track row, and the column its left end starts at
	Cars     int  // its carriages
	Engine   bool // an engine, at the end it faces; only a rake with one can shunt
	Facing   int  // the way its engine faces, and so the end it is at: +1 right, -1 left
	Shunt    int  // how far it shunts, in cells, and the way: + right, - left; 0 for one that stands
	Last     bool // the last train, which pulls out the way it faces when the clock runs out
}

// Cells is how many cells r takes.
func (r Rake) Cells() int {
	n := r.Cars * CarCells
	if r.Engine {
		n += EngineCells
	}
	return n
}

// A train's carriage and engine, in cells (DESIGN §6.2, D37).
const (
	CarCells    = 3
	EngineCells = 3
)

// BonusFruit is one fruit of a bonus level's set: which of Fruits it is,
// and which side of the field it lies on. Where exactly is chosen at
// random each run, so no run's route is the last's (DESIGN §6.7).
type BonusFruit struct {
	Fruit int
	Side  FruitSide
}

// FruitSide is where on the field a bonus level's fruit can lie.
type FruitSide int

// The sides.
const (
	FruitYard  FruitSide = iota // a goods yard: the rails, or a carriage's roof
	FruitRiver                  // riding a log
	FruitRoad                   // the road lanes and the verge
)

// BonusFruitApart is how many cells apart a bonus level's fruit are at
// the least, across and up, so none lies next to another, even corner to
// corner; none lies within BonusClearCells of the toad's start either.
const BonusFruitApart = 2

// The goods yard (DESIGN §6.7): how fast a rake shunts, in SubPx units a
// tick, and how long it waits at each end. Starting values.
const (
	ShuntSpeed      = 8
	ShuntPauseTicks = TPS
)

// BonusFruitPoints are what each of Fruits is worth on a bonus level, more
// for the more exotic (DESIGN §6.7).
var BonusFruitPoints = [len(Fruits)]int{100, 150, 200, 250, 300, 350, 400, 500}

// Size is how many pickups b's set has: its fireflies and its fruit.
func (b *Bonus) Size() int { return b.Fireflies + len(b.Fruit) + b.Stragglers }

// ParkedCars are the letters of Bonus.Parked and the vehicles they are.
var ParkedCars = map[byte]Entity{
	'b': Bike, 'c': CarRed, 'u': CarBlue, 'y': CarYellow, 'v': Van, 't': Truck,
}

// ParkedCells is how many cells a parked e takes.
func ParkedCells(e Entity) int { return (e.Length + Cell - 1) / Cell }

// ParkedCar is one parked car: its vehicle and the first column it takes.
type ParkedCar struct {
	Entity Entity
	Col    int
}

// Cars are the parked cars on row r, left to right. A letter not in
// ParkedCars is a gap; the bonus tests check there are none.
func (b *Bonus) Cars(r int) []ParkedCar {
	var out []ParkedCar
	s := b.Parked[r]
	for c := 0; c < len(s); {
		e, ok := ParkedCars[s[c]]
		if !ok {
			c++
			continue
		}
		out = append(out, ParkedCar{Entity: e, Col: c})
		c += ParkedCells(e)
	}
	return out
}

// The bonus level's rules, the same in every mode (DESIGN §6.7, D61).
// Starting values, tuned by play.
const (
	PointsBonusSec    = 50      // each whole second left when the set is cleared early
	PointsBonusNook   = 100     // a nook, which sends the toad back to the start row
	SlipTicks         = TPS     // a slip's splash or bump, before the toad starts again
	BonusComeMinTicks = 2 * TPS // a firefly comes in this long after one is caught
	BonusComeMaxTicks = 3 * TPS
	BonusClearCells   = 3 // a firefly never lands within this many cells of the toad
)

// The river on firefly night: slow, with long logs (DESIGN §6.7).
var (
	nightLogsRight = Lane{Kind: River, Dir: 1, Speed: 6, Pattern: []Spawn{
		{Entity: LogLong, Gap: 72}, {Entity: LogLong, Gap: 96},
	}}
	nightLogsLeft = Lane{Kind: River, Dir: -1, Speed: 7, Pattern: []Spawn{
		{Entity: LogLong, Gap: 96}, {Entity: LogShort, Gap: 72},
	}}
)

// fireflyNightLayout is the far bank, the slow river, the verge, the track
// as scenery and the road, its traffic gone: the cars stand parked.
var fireflyNightLayout = func() *Layout {
	l := strollLayout("Firefly night", [5]Lane{nightLogsRight, nightLogsLeft, nightLogsRight, nightLogsLeft, nightLogsRight})
	for r, lane := range l.Lanes {
		if lane.Kind == Road {
			l.Lanes[r] = Lane{Kind: Road, Dir: lane.Dir}
		}
	}
	return l
}()

// FireflyNight is Stroll's bonus level (DESIGN §6.7): the road a maze of
// parked cars, the river slow, and ten fireflies to catch, five at a time.
var FireflyNight = &Bonus{
	Name:   "FIREFLY NIGHT",
	Hint:   "CATCH THE FIREFLIES BEFORE THE CLOCK RUNS OUT",
	Layout: fireflyNightLayout,
	Parked: [Rows]string{
		8:  "....cc....uu...ttt..b.vv..",
		9:  "b...yy....b....uu.......vv",
		10: "cc.ttt......uu....ttt.b.yy",
		11: "...vv...ttt......b...ttt..",
	},
	Secs:      40,
	Fireflies: 10,
	OnRoad:    5,
	AtOnce:    5,
	Firefly:   200,
	All:       1000,
}

// lastTrainHomeLayout is the far bank, firefly night's slow river, the
// verge, and the main line with the road rows turned to sidings: a goods
// yard.
var lastTrainHomeLayout = func() *Layout {
	l := strollLayout("The last train home", [5]Lane{nightLogsRight, nightLogsLeft, nightLogsRight, nightLogsLeft, nightLogsRight})
	for r, lane := range l.Lanes {
		if lane.Kind == Road {
			l.Lanes[r] = Lane{Kind: Rail}
		}
	}
	return l
}()

// LastTrainHome is Last Train's bonus level (DESIGN §6.7): dawn over a
// goods yard of carriages to ride, some standing, some shunting, and a set
// of fruit to collect before the last train leaves.
var LastTrainHome = &Bonus{
	Name:   "THE LAST TRAIN HOME",
	Hint:   "COLLECT THE FRUIT BEFORE THE LAST TRAIN LEAVES",
	Layout: lastTrainHomeLayout,
	Secs:   40,
	All:    1000,
	Dawn:   true,
	Yard: []Rake{
		{Row: 7, Col: 5, Cars: 4, Engine: true, Facing: 1, Last: true},
		{Row: 8, Col: 0, Cars: 2, Engine: true, Facing: 1, Shunt: 8},
		{Row: 8, Col: 18, Cars: 2},
		{Row: 9, Col: 3, Cars: 3},
		{Row: 10, Col: 14, Cars: 3, Engine: true, Facing: -1, Shunt: 6}, // it pushes its carriages off the field
		{Row: 11, Col: 4, Cars: 1},
		{Row: 11, Col: 20, Cars: 2},
	},
	// Half in the yard, half on the river, the bigger fruit on the river,
	// which is harder to reach (DESIGN §6.7).
	Fruit: []BonusFruit{
		{Fruit: 0}, {Fruit: 0}, {Fruit: 1}, {Fruit: 1}, {Fruit: 2}, {Fruit: 4}, // strawberries, apples, a pear and grapes in the yard
		{Fruit: 2, Side: FruitRiver}, {Fruit: 3, Side: FruitRiver}, {Fruit: 3, Side: FruitRiver}, // a pear, bananas,
		{Fruit: 5, Side: FruitRiver}, {Fruit: 6, Side: FruitRiver}, {Fruit: 7, Side: FruitRiver}, // an orange, a pineapple and a watermelon on the river
	},
}

// The street sweepers (DESIGN §6.7): how long they are, how fast they go,
// in SubPx units a tick (about 6 s a lane), and how long they wait off
// the field before they set off. No fruit is laid
// where a sweeper would get to it before a toad hopping straight there
// from the start, given SweeperFairTicks to set off and SweeperFairHopTicks
// a hop, so every fruit can be had. Starting values.
const (
	SweeperCells        = 2
	SweeperSpeed        = 28
	SweeperHoldTicks    = 3 * TPS
	SweeperFairTicks    = TPS
	SweeperFairHopTicks = 2 * HopTicks
)

// emptyRoadsLayout is the far bank, firefly night's slow river, the
// verge, the track as scenery and the road, its traffic gone.
var emptyRoadsLayout = func() *Layout {
	l := strollLayout("Empty roads at dawn", [5]Lane{nightLogsRight, nightLogsLeft, nightLogsRight, nightLogsLeft, nightLogsRight})
	for r, lane := range l.Lanes {
		if lane.Kind == Road {
			l.Lanes[r] = Lane{Kind: Road, Dir: lane.Dir}
		}
	}
	return l
}()

// EmptyRoads is Rush Hour's bonus level (DESIGN §6.7): dawn, the road
// empty but for two street sweepers, and a set of fruit to collect before
// they sweep it up.
var EmptyRoads = &Bonus{
	Name:    "EMPTY ROADS AT DAWN",
	Hint:    "BEAT THE SWEEPERS TO THE FRUIT",
	Layout:  emptyRoadsLayout,
	Secs:    40,
	All:     1000,
	Dawn:    true,
	Wet:     true,
	Sweeper: true,
	Fruit: []BonusFruit{
		{Fruit: 0, Side: FruitRoad}, {Fruit: 0, Side: FruitRoad}, {Fruit: 1, Side: FruitRoad}, {Fruit: 1, Side: FruitRoad},
		{Fruit: 2, Side: FruitRoad}, {Fruit: 2, Side: FruitRoad}, {Fruit: 3, Side: FruitRoad}, {Fruit: 3, Side: FruitRoad},
		{Fruit: 4, Side: FruitRoad}, {Fruit: 5, Side: FruitRoad}, {Fruit: 6, Side: FruitRoad}, {Fruit: 7, Side: FruitRoad},
	},
}

// afterTheStormLayout is the far bank, firefly night's slow river with two
// of Storm's currents at its level 1 strength, one with the flow and one
// against it, both on rows of long logs, the verge, the track as scenery
// and the road, its traffic gone.
var afterTheStormLayout = func() *Layout {
	with, against := nightLogsRight, nightLogsRight
	with.Current, against.Current = 3, -3
	l := strollLayout("After the storm", [5]Lane{with, nightLogsLeft, against, nightLogsLeft, nightLogsRight})
	for r, lane := range l.Lanes {
		if lane.Kind == Road {
			l.Lanes[r] = Lane{Kind: Road, Dir: lane.Dir}
		}
	}
	return l
}()

// AfterTheStorm is Storm's bonus level (DESIGN §6.7): dawn as the storm
// clears, a rainbow over the river, the road still slippery, and twelve
// dragonflies on the logs, each caught in a run without setting foot on
// land worth more than the last.
var AfterTheStorm = &Bonus{
	Name:      "AFTER THE STORM",
	Hint:      "STAY ON THE LOGS: A RUN OF CATCHES SCORES MORE",
	Layout:    afterTheStormLayout,
	Secs:      40,
	Fireflies: 12,
	AtOnce:    5,
	Firefly:   200,
	All:       1000,
	Dawn:      true,
	Wet:       true,
	Slippery:  true,
	Rainbow:   true,
	RunMax:    5,
}

// schoolZone is the school gates' road: a quiet school-zone road, its
// traffic slow and far apart, so the line behind the toad is what has to
// be kept safe (DESIGN §6.7). Starting values, tuned by play.
var schoolZone = [Rows]Lane{
	8: {Kind: Road, Dir: -1, Speed: 8, Pattern: []Spawn{
		{Entity: CarRed, Gap: 200}, {Entity: CarBlue, Gap: 260},
	}},
	9: {Kind: Road, Dir: 1, Speed: 9, Pattern: []Spawn{
		{Entity: Van, Gap: 230}, {Entity: CarYellow, Gap: 280},
	}},
	10: {Kind: Road, Dir: -1, Speed: 8, Pattern: []Spawn{
		{Entity: CarBlue, Gap: 250}, {Entity: Bike, Gap: 210},
	}},
	11: {Kind: Road, Dir: 1, Speed: 7, Pattern: []Spawn{
		{Entity: CarYellow, Gap: 210}, {Entity: CarRed, Gap: 270},
	}},
}

// schoolGatesLayout is the far bank, firefly night's slow river, the
// verge, the track as scenery and the school-zone road.
var schoolGatesLayout = func() *Layout {
	l := strollLayout("The school gates", [5]Lane{nightLogsRight, nightLogsLeft, nightLogsRight, nightLogsLeft, nightLogsRight})
	for r, lane := range schoolZone {
		if lane.Kind == Road {
			l.Lanes[r] = lane
		}
	}
	return l
}()

// SchoolGates is School Run's bonus level (DESIGN §6.7): the late
// afternoon, toadlets waiting at the kerbs, and the toad walking them home
// a line at a time across a quiet road, then the game's ending.
var SchoolGates = &Bonus{
	Name:       "THE SCHOOL GATES",
	Hint:       "WALK THE TOADLETS HOME: HOP ONTO ONE AND IT FOLLOWS",
	Layout:     schoolGatesLayout,
	Secs:       40,
	All:        1000,
	Stragglers: 10,
	LineMax:    4,
	Straggler:  200,
	TripMin:    3,
	Dusk:       true,
	Gates:      true,
	Ending:     true,
}

// BonusLevel is m's bonus level as it is played, numbered as the level it
// follows; ok is false if m has none yet, so clearing its level 8 goes
// straight to the MASTERED screen (D61).
func BonusLevel(m Mode) (lv Level, ok bool) {
	b := m.Bonus
	if b == nil {
		return Level{}, false
	}
	lv = Level{Number: len(m.Levels.Levels), Lanes: b.Layout.Lanes, Bonus: b, Hint: b.Hint,
		TimerTicks: b.Secs * TPS, DayCycle: b.Dawn || b.Dusk} // its nooks never fill, so it stays dawn, or dusk
	lv.FireflyOn[HostLogs] = true // never on lily pads, which dive
	return lv, true
}

// PointsFinishLife is the bonus for each life left when a run finishes
// at the MASTERED screen, rather than carrying on (D62).
const PointsFinishLife = 1000
