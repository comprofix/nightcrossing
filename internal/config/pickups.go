package config

// MaxLives is the most lives a game can hold, what the status bar shows.
// At the cap no life comes from points, and a firefly gives points instead
// (DESIGN §6.6, D57).
const MaxLives = 8

// Host is a kind of platform a firefly can ride (D58, D60).
type Host int

// The hosts.
const (
	HostLogs  Host = iota // logs
	HostPads              // lily pads
	HostShips             // the barge and the ferry
	NumHosts
)

// HostOf is the kind of host platform e is.
func HostOf(e Entity) Host {
	switch e {
	case Pads2, Pads3:
		return HostPads
	case Barge, Ferry:
		return HostShips
	}
	return HostLogs
}

// The firefly (D58): one roll a crossing, its chance rising from its first
// level to FireflyTopLevel, then staying at the top. Starting values.
const (
	FireflyPctFirst    = 30
	FireflyPctTop      = 50
	FireflyTopLevel    = 8   // the level the chance reaches the top, in every mode
	PointsFireflyAtCap = 500 // a firefly caught with lives at the cap
)

// The firefly's flight (D58): when it comes in, how long it sits on each
// platform and stays on the field, how fast it flies, and how often it
// flits along its row rather than up or down. Starting values, fine-tuned
// in M20.
const (
	FireflyComeMinTicks  = 1 * TPS
	FireflyComeMaxTicks  = 3 * TPS
	FireflySitMinTicks   = 2 * TPS
	FireflySitMaxTicks   = 4 * TPS
	FireflyVisitMinTicks = 16 * TPS
	FireflyVisitMaxTicks = 20 * TPS
	FireflySpeed         = 3 * Cell * SubPx / TPS // SubPx units a tick: 3 cells a second
	FireflySameRowPct    = 25
	FireflyLandMargin    = 2 // it lands at least this many cells in from either edge of the field
	FireflyReach         = 5 // it flits to a spot this many cells across at most, if there is one
)

// Fruit in the nooks (D59): it shows in one empty nook at a time, then
// goes, and comes back after a seeded wait. Starting values.
const (
	FruitShowTicks    = 9 * TPS
	FruitWaitMinTicks = 8 * TPS
	FruitWaitMaxTicks = 12 * TPS
	PointsFruitFirst  = 200 // on the level fruit first comes in, in the mode
	PointsFruitStep   = 100 // more for each level after
)

// Fruits are the fruit of each level, from level 1; the loops keep the
// last (D59).
var Fruits = [...]string{"strawberry", "apple", "pear", "banana", "grapes", "orange", "pineapple", "watermelon"}

// FireflyPct is the chance, in percent, of a firefly in a crossing on
// level n of a mode where fireflies first come in on level from, rising
// to the top by FireflyTopLevel; 0 before it or with none (D58).
func FireflyPct(from, n int) int {
	if from == 0 || n < from {
		return 0
	}
	if FireflyTopLevel <= from {
		return FireflyPctTop
	}
	n = min(n, FireflyTopLevel)
	return FireflyPctFirst + (FireflyPctTop-FireflyPctFirst)*(n-from)/(FireflyTopLevel-from)
}

// FruitPoints is what fruit is worth on level n of a mode with last
// levels, where fruit first comes in on level from; 0 before it or with
// none (D59).
func FruitPoints(from, last, n int) int {
	if from == 0 || n < from {
		return 0
	}
	return PointsFruitFirst + PointsFruitStep*(min(n, last)-from)
}

// FruitOf is the index in Fruits of level n's fruit.
func FruitOf(n int) int { return min(n, len(Fruits)) - 1 }

// PickupsAlso is the line on each later mode's intro, under ALSO, for the
// pickups it has from level 1 (D60).
const PickupsAlso = "Fruit and fireflies, and dragonflies by day"
