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
	All       int // points for catching every one
}

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

// BonusLevel is m's bonus level as it is played, numbered as the level it
// follows; ok is false if m has none yet, so clearing its level 8 goes
// straight to the MASTERED screen (D61).
func BonusLevel(m Mode) (lv Level, ok bool) {
	b := m.Bonus
	if b == nil {
		return Level{}, false
	}
	lv = Level{Number: len(m.Levels.Levels), Lanes: b.Layout.Lanes, Bonus: b, Hint: b.Hint,
		TimerTicks: b.Secs * TPS}
	lv.FireflyOn[HostLogs] = true // never on lily pads, which dive
	return lv, true
}

// PointsFinishLife is the bonus for each life left when a run finishes
// at the MASTERED screen, rather than carrying on (D62).
const PointsFinishLife = 1000
