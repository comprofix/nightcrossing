package config

// The hero's hop (DESIGN §5.1, D16). Starting values, tuned in M1.
const (
	// HopTicks is how long one hop from cell to cell takes.
	HopTicks = 8
	// HopQueueTicks is how close to landing a press must be to be kept
	// and carried out on landing. Earlier presses during a hop are
	// ignored.
	HopQueueTicks = 3
	// LandTicks is how long the landing squash shows after a hop.
	LandTicks = 4
	// HopLiftPx is how high the hero rises at the top of a hop, drawn as
	// a lift towards the top of the screen.
	HopLiftPx = 3
)

// StartCol is the column each attempt starts in.
const StartCol = Cols / 2

// SubPx is the number of position units in a pixel. Positions are whole
// numbers of these, so the game stays deterministic (PLAN §3).
const SubPx = 16

// HeroMargin is how many pixels are trimmed from each side of the hero's
// collision box, so a graze at the very edge of the art is not a death
// (DESIGN §1).
const HeroMargin = 3

// DeathTicks is how long the hero's death shows before the next attempt
// or game over (DESIGN §5.4).
const DeathTicks = 60
