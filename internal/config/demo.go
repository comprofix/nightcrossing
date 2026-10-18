package config

// The title's demo (DESIGN §7): a game played by the autopilot behind the
// logo.
const (
	DemoSeed      = 1981 // the demo's own seed, so it is the same every time and leaves the game's RNG alone
	DemoLookahead = 3    // hops the autopilot looks ahead
	DemoWaitTicks = 6    // how long "wait" lasts as one of its moves
	DemoTailTicks = 30   // after its last move, how long the hero must then survive standing still
	DemoLevels    = 3    // the demo plays levels 1 to DemoLevels in turn
	DemoFirefly   = 3    // a firefly caught is worth this many rows of progress to the autopilot (D58)
)

// DemoSwarmFirefly is what a firefly caught on a bonus level is worth to
// the autopilot, in rows of progress: more than the furthest the next one
// can be, so it never shies from a catch (D61).
const DemoSwarmFirefly = 2 * Rows * Cols / 4
