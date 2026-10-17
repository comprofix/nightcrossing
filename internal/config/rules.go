package config

// The base numbers, the same for every mode (DESIGN §5.6, D13). Starting
// values, tuned in M4.
const (
	// HitMargin is how many pixels are trimmed from each end of a hazard's
	// collision box, so a graze at the very end of its art is not a death
	// (DESIGN §5.1).
	HitMargin = 4

	// The timer for each attempt (DESIGN §5.4, D22): TimerStartSecs at
	// level 1, TimerCutSecs less each level after, never under
	// TimerFloorSecs.
	TimerStartSecs = 45
	TimerCutSecs   = 2
	TimerFloorSecs = 20

	// LowTimeSecs is when the low time warning sounds (DESIGN §5.4).
	LowTimeSecs = 10

	// ToadletSecs is the time added to the timer for each toadlet in
	// School Run's line, so it can file in (D53).
	ToadletSecs = 1
)

// TimerTicks is the time for each attempt on level n (1 for the first), in
// ticks.
func TimerTicks(n int) int {
	return max(TimerStartSecs-TimerCutSecs*(n-1), TimerFloorSecs) * TPS
}

// Scoring (DESIGN §5.5, first numbers, to be tuned).
const (
	PointsHop      = 10   // a hop into a row further than reached this attempt
	PointsGoal     = 100  // filling a nook
	PointsPerSec   = 10   // the time bonus, for each whole second left
	PointsAllGoals = 1000 // filling the fifth nook
	PointsToadlet  = 50   // each toadlet home with the hero, added to the nook's points (D53)
	ExtraLifeEvery = 20_000
)

// The goal nooks on the far bank (DESIGN §5.3): each is NookWidth cells
// wide, starting at these columns. They are spread evenly, with the same
// bank at each side. A hop that lands in any cell of an empty nook fills it.
var Nooks = [...]int{2, 7, 12, 17, 22}

// NookWidth is how many cells wide each nook is.
const NookWidth = 2

// GoalTicks is how long a toad that reached a nook shows there before the
// next attempt starts.
const GoalTicks = 30

// CrossOpenTicks is how long a gap in traffic must stay open over the
// hero's cell for it to hop in, wait and hop out: two hops (DESIGN §5.7,
// D21). Starting value, tuned in M4.
const CrossOpenTicks = 2 * HopTicks
