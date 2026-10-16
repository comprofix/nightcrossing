// Package world is Night Crossing's game logic: the playfield, its lanes and
// everything on them, the hero, goals and the timer. It runs headless and
// must not import graphics or audio packages; the play scene draws its state
// and reacts to the events it gives out.
//
// In M0 it holds only the grid, the skill and the clock.
package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
)

// World is one game's playfield.
type World struct {
	Cols, Rows int
	Skill      config.Skill
	Tick       int64 // ticks since the game started
	TicksLeft  int   // time left on this attempt (DESIGN §5.4)

	rng *rand.Rand
}

// New makes an empty playfield of cols×rows cells for a game at skill.
// Every random choice comes from rng, so a seed gives the same game.
func New(cols, rows int, skill config.Skill, rng *rand.Rand) *World {
	return &World{
		Cols:      cols,
		Rows:      rows,
		Skill:     skill,
		TicksLeft: AttemptTicks(skill),
		rng:       rng,
	}
}

// AttemptTicks is the time limit for each attempt at skill, in ticks.
func AttemptTicks(skill config.Skill) int { return skill.TimerSecs * config.TPS }

// Update advances the world by one tick.
func (w *World) Update() {
	w.Tick++
}
