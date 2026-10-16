package config

// Last Train's own mechanics beyond the trains (DESIGN §6.2): the hop combo
// (D38) and the day and night cycle (D39). Starting values, tuned in M10.

// The hop combo (D38). Each forward hop into a row not yet reached this
// attempt adds 1 to the combo; every ComboStep of them raise the
// multiplier by 1, up to ComboMaxMult. The combo ends after ComboTicks
// without such a hop (riding a platform pauses the count), on a hop
// backwards, or on a death.
const (
	ComboStep    = 3
	ComboMaxMult = 4
	ComboTicks   = 3 * TPS / 2
)

// ComboMult is the score multiplier for a combo of n hops: ×1 below
// ComboStep, then ×2, ×3 and at most ComboMaxMult.
func ComboMult(n int) int { return min(1+n/ComboStep, ComboMaxMult) }

// TimeOfDay is a stage of the day and night cycle (D39), in the order the
// clock turns through them.
type TimeOfDay int

// The stages.
const (
	Dawn TimeOfDay = iota
	Day
	Dusk
	Night
)

// DayCycle is the time of day for each number of nooks filled, from none
// to all five (D39): each level is a full turn of the clock, starting at
// dawn, and the level clear is at night.
var DayCycle = [len(Nooks) + 1]TimeOfDay{Dawn, Day, Day, Dusk, Night, Night}

// DayEaseTicks is how long the light takes to ease from one stage to the
// next.
const DayEaseTicks = 2 * TPS
