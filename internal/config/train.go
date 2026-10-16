package config

// Trains on the track, from Last Train on (DESIGN §6.2, D17, D37).
// Starting values, tuned in M9.
const (
	// TrainWarnTicks is how long the crossing lights flash and the bell
	// rings before a train comes. It is the same on every level, so a
	// train never comes sooner than the lights promise.
	TrainWarnTicks = 2 * TPS

	// TrainTopLevel is the level the trains reach their fastest and most
	// often on, and stay at after (D17).
	TrainTopLevel = 8
)

// TrainTiming is how the trains run on one level: their speed, in SubPx
// units per tick, and the quiet between them, from one leaving the field
// to the next warning, a random length from QuietMin to QuietMax ticks.
// The speed is the train's own, not scaled by the level's speed (D37).
type TrainTiming struct {
	Speed              int
	QuietMin, QuietMax int
}

// The trains on level 1 and from TrainTopLevel on; the levels between
// step evenly from one to the other (D17).
var (
	TrainFirst = TrainTiming{Speed: 4 * SubPx, QuietMin: 10 * TPS, QuietMax: 14 * TPS}
	TrainTop   = TrainTiming{Speed: 8 * SubPx, QuietMin: 4 * TPS, QuietMax: 7 * TPS}
)

// TrainsOn is the train timing on level n (1 for the first).
func TrainsOn(n int) TrainTiming {
	step, span := min(max(n, 1), TrainTopLevel)-1, TrainTopLevel-1
	at := func(first, top int) int { return first + (top-first)*step/span }
	return TrainTiming{
		Speed:    at(TrainFirst.Speed, TrainTop.Speed),
		QuietMin: at(TrainFirst.QuietMin, TrainTop.QuietMin),
		QuietMax: at(TrainFirst.QuietMax, TrainTop.QuietMax),
	}
}

// Train is the train: an engine and four carriages, each as long as a
// truck, about half the field.
var Train = Entity{ID: "train", Length: 15 * Cell}
