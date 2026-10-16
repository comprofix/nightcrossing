// Package world is Night Crossing's game logic: the playfield, its lanes and
// everything on them, the hero, goals and the timer. It runs headless and
// must not import graphics or audio packages; the play scene draws its state
// and reacts to the events it gives out.
package world

import (
	"math/rand/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// Event is something that happened during a tick, for the play scene to
// show or sound.
type Event int

// The events.
const (
	Hopped    Event = iota + 1 // the hero left a cell
	Landed                     // the hero arrived in a cell
	Squashed                   // a vehicle hit the hero
	Splashed                   // the hero landed in the water, or its platform went under
	Swept                      // a platform carried the hero off the field
	Respawned                  // a new attempt began on the start row
	GameOver                   // the last life is gone
)

// World is one game's playfield.
type World struct {
	Cols, Rows int
	Session    *session.Session
	Skill      config.Skill
	Tick       int64 // ticks since the game started
	TicksLeft  int   // time left on this attempt (DESIGN §5.4)
	Player     Player
	Lanes      [config.Rows]Lane
	Dying      int  // ticks left of the hero's death; 0 when alive
	Over       bool // the game has ended

	rng    *rand.Rand
	events []Event
}

// New starts a game for s on level. Every random choice comes from rng,
// so a seed gives the same game.
func New(s *session.Session, level config.Level, rng *rand.Rand) *World {
	w := &World{
		Cols:    config.Cols,
		Rows:    config.Rows,
		Session: s,
		Skill:   s.Skill,
		rng:     rng,
	}
	for r, l := range level.Lanes {
		w.Lanes[r] = newLane(l, s.Skill, rng)
	}
	w.newAttempt()
	return w
}

// AttemptTicks is the time limit for each attempt at skill, in ticks.
func AttemptTicks(skill config.Skill) int { return skill.TimerSecs * config.TPS }

// newAttempt puts the hero on the start row with a full clock.
func (w *World) newAttempt() {
	w.Player = newPlayer(config.StartCol, config.StartRow)
	w.TicksLeft = AttemptTicks(w.Skill)
}

// Update advances the world by one tick. press is the direction pressed
// this tick, or None. It returns what happened; the slice is reused by the
// next call.
func (w *World) Update(press Dir) []Event {
	w.events = w.events[:0]
	if w.Over {
		return w.events
	}
	w.Tick++
	for r := range w.Lanes {
		if len(w.Lanes[r].Entities) > 0 {
			w.Lanes[r].update(w.rng)
		}
	}
	if w.Dying > 0 {
		w.updateDeath()
		return w.events
	}
	w.updatePlayer(press)
	w.ride()
	w.checkHazards()
	return w.events
}

func (w *World) emit(e Event) { w.events = append(w.events, e) }

// checkHazards kills the hero if its row is a danger to it. Only the lane
// the hero is over is checked (PLAN §3).
//   - Road: a vehicle overlaps it. The hero's box is a cell trimmed by
//     config.HeroMargin; vehicles are trimmed by the skill's HitMargin
//     (DESIGN §5.1, §5.6).
//   - River: a platform carried its middle off the field, or it has landed
//     with no platform under its middle (DESIGN §5.2). Water only counts
//     once the hero is down, so a hop over a gap onto a platform is safe.
func (w *World) checkHazards() {
	p := &w.Player
	lane := &w.Lanes[p.LaneRow()]
	switch lane.Kind {
	case config.Road:
		m := config.HeroMargin * config.SubPx
		left, right := p.X+m, p.X+cellUnits-m
		if lane.hits(left, right, w.Skill.HitMargin) {
			w.die(Squashed)
		}
	case config.River:
		switch mid := p.Middle(); {
		case mid < 0 || mid >= fieldWidth:
			w.die(Swept)
		case !p.Hopping() && !lane.holds(mid):
			w.die(Splashed)
		}
	}
}

// die starts the hero's death. The life goes at once, so the status bar
// shows it during the death.
func (w *World) die(how Event) {
	w.Player.Death = how
	w.Player.queued = None
	w.Dying = config.DeathTicks
	w.Session.LoseLife()
	w.emit(how)
}

// updateDeath runs out the death, then starts a new attempt or ends the
// game.
func (w *World) updateDeath() {
	w.Dying--
	if w.Dying > 0 {
		return
	}
	if w.Session.Lives == 0 {
		w.Over = true
		w.emit(GameOver)
		return
	}
	w.newAttempt()
	w.emit(Respawned)
}
