// Package world is Night Crossing's game logic: the playfield, its lanes and
// everything on them, the hero, goals and the timer. It runs headless and
// must not import graphics or audio packages; the play scene draws its state
// and reacts to the events it gives out.
//
// So far it holds the grid, the skill, the clock and the hero's hopping.
package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Event is something that happened during a tick, for the play scene to
// show or sound.
type Event int

// The events.
const (
	Hopped Event = iota + 1 // the hero left a cell
	Landed                  // the hero arrived in a cell
)

// World is one game's playfield.
type World struct {
	Cols, Rows int
	Skill      config.Skill
	Tick       int64 // ticks since the game started
	TicksLeft  int   // time left on this attempt (DESIGN §5.4)
	Player     Player

	rng    *rand.Rand
	events []Event
}

// New makes an empty playfield of cols×rows cells for a game at skill, with
// the hero on the start row. Every random choice comes from rng, so a seed
// gives the same game.
func New(cols, rows int, skill config.Skill, rng *rand.Rand) *World {
	w := &World{
		Cols:      cols,
		Rows:      rows,
		Skill:     skill,
		TicksLeft: AttemptTicks(skill),
		rng:       rng,
	}
	w.Player = newPlayer(config.StartCol, config.StartRow)
	return w
}

// AttemptTicks is the time limit for each attempt at skill, in ticks.
func AttemptTicks(skill config.Skill) int { return skill.TimerSecs * config.TPS }

// Update advances the world by one tick. press is the direction pressed
// this tick, or None. It returns what happened; the slice is reused by the
// next call.
func (w *World) Update(press Dir) []Event {
	w.Tick++
	w.events = w.events[:0]
	w.updatePlayer(press)
	return w.events
}

func (w *World) emit(e Event) { w.events = append(w.events, e) }

// inside reports whether a cell is on the playfield.
func (w *World) inside(col, row int) bool {
	return col >= 0 && col < w.Cols && row >= 0 && row < w.Rows
}
