package config

// Predators (DESIGN §6.8, D66–D70): from level 3 of every mode but Stroll,
// the heron stalks its log, and from level 6 it strikes; a fox works the
// nooks at dusk and night; a water rat swims for the ends of platforms from
// level 4, and a snake patrols the verge from level 5. Each gives a warning
// and is slow enough to read. Starting values, tuned by play.

// PredatorsFrom is the first level with the stalking heron and the fox, and
// StrikeFrom the first with the heron's strike (D66).
const (
	PredatorsFrom = 3
	StrikeFrom    = 6
)

// The heron's stalk and strike (D67). It steps a cell along its log every
// HeronStepTicks, turning at the ends: it stands, then walks smoothly into
// the next cell over HeronStrideTicks. From StrikeFrom it strikes every
// HeronStrikeMinTicks to HeronStrikeMaxTicks: it stops and cocks its head
// for HeronCockTicks, the warning, then jabs the cell it faces for
// HeronJabTicks.
const (
	HeronStepTicks      = 2 * TPS
	HeronStrideTicks    = TPS / 2
	HeronStrikeMinTicks = 4 * TPS
	HeronStrikeMaxTicks = 6 * TPS
	HeronCockTicks      = TPS / 2
	HeronJabTicks       = TPS / 4
)

// The chance, per log, of a stalking heron: HeronPctFirst on PredatorsFrom,
// rising evenly to HeronPctTop on HeronPctTopLevel and staying there (D67).
const (
	HeronPctFirst    = 15
	HeronPctTop      = 35
	HeronPctTopLevel = 8
)

// StalkingHeronPct is the chance of a stalking heron on level n.
func StalkingHeronPct(n int) int {
	step := min(n, HeronPctTopLevel) - PredatorsFrom
	return HeronPctFirst + (HeronPctTop-HeronPctFirst)*step/(HeronPctTopLevel-PredatorsFrom)
}

// The fox in a nook (D68): its eyes glint for FoxEyesTicks, safe; then it
// is there for FoxOutTicks, deadly; then it is gone for FoxGoneTicks and a
// quiet of FoxQuietMinTicks to FoxQuietMaxTicks before it comes again.
const (
	FoxEyesTicks     = 3 * TPS / 2
	FoxOutTicks      = 3 * TPS / 2
	FoxGoneTicks     = 3 * TPS / 2
	FoxQuietMinTicks = 3 * TPS
	FoxQuietMaxTicks = 6 * TPS
)

// The hints for the levels predators come in on (D66). Each shows in turn
// with the level's own hint.
const (
	StalkHint  = "A HERON STALKS ITS LOG, A FOX WATCHES THE NOOKS"
	StrikeHint = "A HERON COCKS ITS HEAD BEFORE IT STRIKES"
)

// The water rat and the snake (D66, D69, D70): the rat from RatFrom, the
// snake from SnakeFrom, and from PairFrom two of them out at once.
const (
	RatFrom   = 4
	SnakeFrom = 5
	PairFrom  = 6
)

// The water rat (D69). It comes up with a ripple for RippleTicks, the
// warning, RatStartCells cells off the front end of a platform at least
// RatTargetCells long, and swims for it against the flow, a cell every
// RatCellTicks relative to the water, until its nose touches the end.
// Then it grabs the end for RatGrabTicks, its head and paws RatOverPx onto
// the platform, the only part of it that is deadly, and dives.
const (
	RippleTicks    = TPS
	RatStartCells  = 2
	RatCellTicks   = TPS
	RatGrabTicks   = TPS / 2
	RatOverPx      = Cell / 2
	RatTargetCells = 3
)

// RatSwimTicks is how long the water rat swims, from its ripple to its
// grab.
const RatSwimTicks = RatStartCells * RatCellTicks

// The snake on the verge (D70). The grass rustles for RustleTicks, the
// warning; then a snake SnakeCells long slides back and forth along a
// stretch of SnakeStretch cells, a cell every SnakeCellTicks, for
// SnakeOutMinTicks to SnakeOutMaxTicks, and slips away. The verge keeps
// at least VergeFreeMin cells free of its stretch.
const (
	RustleTicks      = TPS
	SnakeCells       = 2
	SnakeStretch     = 6
	SnakeCellTicks   = 2 * TPS
	SnakeOutMinTicks = 12 * TPS
	SnakeOutMaxTicks = 16 * TPS
	VergeFreeMin     = 18
)

// SnakePct is the chance that the creature coming next is the snake, when
// it can come; otherwise it is a water rat.
const SnakePct = 40

// The quiet between one creature coming and the next: ProwlFirstMin to
// ProwlFirstMax ticks on RatFrom, shortening evenly to ProwlTopMin to
// ProwlTopMax on HeronPctTopLevel and staying there (D66).
const (
	ProwlFirstMinTicks = 10 * TPS
	ProwlFirstMaxTicks = 14 * TPS
	ProwlTopMinTicks   = 5 * TPS
	ProwlTopMaxTicks   = 8 * TPS
)

// ProwlQuiet is the quiet between creatures on level n, shortest and
// longest.
func ProwlQuiet(n int) (lo, hi int) {
	step, steps := min(n, HeronPctTopLevel)-RatFrom, HeronPctTopLevel-RatFrom
	return ProwlFirstMinTicks - (ProwlFirstMinTicks-ProwlTopMinTicks)*step/steps,
		ProwlFirstMaxTicks - (ProwlFirstMaxTicks-ProwlTopMaxTicks)*step/steps
}

// The hints for the water rat and the snake (D66).
const (
	RatHint   = "A WATER RAT GOES FOR THE END OF A LOG"
	SnakeHint = "A SNAKE ON THE VERGE: DON'T LAND ON IT"
)
