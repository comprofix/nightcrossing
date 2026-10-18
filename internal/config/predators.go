package config

// Predators (DESIGN §6.8, D66–D68): from level 3 of every mode but Stroll,
// the heron stalks its log, and from level 6 it strikes; a fox works the
// nooks at dusk and night. Each gives a warning and is slow enough to
// read. Starting values, tuned by play.

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
