package config

// StandGrace is how many pixels past each end of a platform the middle of
// the hero can be and still stand on it, so landing on the very end is not
// a splash (DESIGN §1, §5.2).
const StandGrace = 4

// A diving platform's cycle, in ticks (DESIGN §5.2, §3.3). It floats, flashes
// as a warning, sinks, stays under, then rises again. Only while it is under
// is standing on it a death; sinking and rising, it is half under but still
// holds the hero. Starting values, tuned in M4.
const (
	DiveAfloatTicks = 240
	DiveWarnTicks   = 60
	DiveSinkTicks   = 20
	DiveUnderTicks  = 90
	DiveRiseTicks   = 20
	DiveCycle       = DiveAfloatTicks + DiveWarnTicks + DiveSinkTicks + DiveUnderTicks + DiveRiseTicks

	// DiveFlashTicks is how long each on and off of the warning flash lasts.
	DiveFlashTicks = 8
)

// A sinking ship's timing, in ticks (DESIGN §5.2, D33). Once it is wholly
// on the screen it floats for a random time between SinkAfterMinTicks and
// SinkAfterMaxTicks, then flashes as a warning, breaks up, and loses one
// piece every SinkPieceTicks, bow first. Starting values, tuned in M4.
const (
	SinkAfterMinTicks = 2 * TPS
	SinkAfterMaxTicks = 8 * TPS
	SinkWarnTicks     = 120
	SinkBreakTicks    = 30
	SinkPieceTicks    = 40
)

// Pieces is how many one-cell pieces a ship has.
func Pieces(ship Entity) int { return ship.Length / Cell }

// SinkTicks is how long a ship takes from the start of its warning until
// its stern has gone under.
func SinkTicks(ship Entity) int {
	return SinkWarnTicks + SinkBreakTicks + Pieces(ship)*SinkPieceTicks
}
