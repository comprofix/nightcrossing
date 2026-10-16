package config

// The playfield layout, in screen pixels (DESIGN §3.2): a 26×13 grid of
// 24px cells, with a status bar above, a hint strip below and a margin
// either side. Together they fill the Modern preset's 640×360 screen.
const (
	Cell        = 24
	Cols        = 26
	Rows        = 13
	FieldMargin = 8  // left and right of the grid
	HUDHeight   = 32 // the status bar along the top
	HintHeight  = 16 // the hint and message strip along the bottom
)

// FieldX and FieldY are the screen position of the grid's top-left cell.
const (
	FieldX = FieldMargin
	FieldY = HUDHeight
)

// Rows with a fixed job on every level (DESIGN §3.2).
const (
	GoalRow  = 0        // the far bank with the goal nooks
	StartRow = Rows - 1 // where each attempt starts
)

// StartLives is the number of lives a game starts with (DESIGN §5.4).
const StartLives = 3
