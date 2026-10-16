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
	Hopped       Event = iota + 1 // the hero left a cell
	Landed                        // the hero arrived in a cell
	Squashed                      // a vehicle hit the hero
	Splashed                      // the hero landed in the water, or its platform went under
	Swept                         // a platform carried the hero off the field
	Caught                        // the hero landed on a heron
	Bumped                        // the hero landed on the bank or a filled nook
	TimeUp                        // the attempt's time ran out
	LowTime                       // the attempt's time is running low
	Goal                          // the hero filled a nook
	LevelClear                    // the last nook is filled
	ExtraLife                     // the score earned a life
	Respawned                     // a new attempt began on the start row
	GameOver                      // the last life is gone
	Struck                        // a train hit the hero (D37)
	TrainWarning                  // the crossing lights start to flash: a train is coming
	TrainComing                   // a train comes onto the field
	TrainGone                     // the train has left the field
	ComboUp                       // the hop combo's multiplier rose (D38)
	WaveWarning                   // a rush hour wave is announced on a lane (D43)
	WaveStart                     // the wave begins: its traffic comes on
	WaveEnd                       // the wave ends: its extra traffic goes as it leaves
	FogWarning                    // fog is announced (D29)
	FogStart                      // the fog is in
	FogEnd                        // the fog clears
	RainWarning                   // rain is announced (D31)
	RainStart                     // it rains: the road is slippery
	RainEnd                       // the rain stops
	Slid                          // the hero began a rain slide (D31)
)

// Stats are the points earned on this level, for the level-clear tally.
type Stats struct {
	Hops      int // points for hops into new rows
	Goals     int // nooks filled
	TimeBonus int // points for time left when filling them
	Clear     int // the bonus for filling the last one
	Combo     int // what the hop combo's multiplier added (D38)
}

// World is one level of a game.
type World struct {
	Cols, Rows int
	Session    *session.Session
	Level      config.Level
	Tick       int64 // ticks since the level started
	TicksLeft  int   // time left on this attempt (DESIGN §5.4)
	Player     Player
	Lanes      [config.Rows]Lane
	Rail       Rail                    // the track's trains, if the level has them (DESIGN §6.2)
	Spells     []Spell                 // the random events, if the level has them (DESIGN §6.1)
	Filled     [len(config.Nooks)]bool // the nooks filled so far; they stay filled (DESIGN §5.4)
	Home       int                     // ticks left of a toad showing in its nook; 0 otherwise
	Dying      int                     // ticks left of the hero's death; 0 when alive
	Cleared    bool                    // every nook is filled
	Over       bool                    // the game has ended
	Stats      Stats
	Combo      Combo // the hop combo, on levels with it (D38)
	Gained     int   // points multiplied by the combo this tick, for the pop-ups

	furthest int // the furthest row reached this attempt
	rng      *rand.Rand
	events   []Event
}

// New starts level for the game in s. Every random choice comes from rng,
// so a seed gives the same game.
func New(s *session.Session, level config.Level, rng *rand.Rand) *World {
	w := &World{
		Cols:    config.Cols,
		Rows:    config.Rows,
		Session: s,
		Level:   level,
		rng:     rng,
	}
	for r, l := range level.Lanes {
		w.Lanes[r] = newLane(l, level.HeronPct, level.Events[config.Wave].On(), rng)
	}
	w.Rail = newRail(level, rng)
	w.Spells = newSpells(level, rng)
	w.newAttempt()
	return w
}

// newAttempt puts the hero on the start row with a full clock.
func (w *World) newAttempt() {
	w.Player = newPlayer(config.StartCol, config.StartRow)
	w.TicksLeft = w.Level.TimerTicks
	w.furthest = config.StartRow
	w.Combo = Combo{}
}

// Update advances the world by one tick. press is the direction pressed
// this tick, or None. It returns what happened; the slice is reused by the
// next call. Once the level is cleared or the game is over, it does
// nothing.
func (w *World) Update(press Dir) []Event {
	w.events = w.events[:0]
	w.Gained = 0
	if w.Over || w.Cleared {
		return w.events
	}
	w.Tick++
	for r := range w.Lanes {
		if len(w.Lanes[r].Entities) > 0 {
			w.Lanes[r].update(w.rng)
		}
	}
	w.Rail.update(w)
	w.updateSpells()
	switch {
	case w.Dying > 0:
		w.updateDeath()
	case w.Home > 0:
		if w.Home--; w.Home == 0 {
			w.newAttempt()
			w.emit(Respawned)
		}
	default:
		w.updatePlayer(press)
		w.updateCombo()
		if w.playing() {
			w.ride()
			w.checkHazards()
		}
		if w.playing() {
			w.updateTimer()
		}
	}
	return w.events
}

// playing reports whether the hero is out on an attempt: not dying, not
// home in a nook, and the level not yet cleared.
func (w *World) playing() bool { return w.Dying == 0 && w.Home == 0 && !w.Cleared }

func (w *World) emit(e Event) { w.events = append(w.events, e) }

// score adds points, and a life each time the score passes a multiple of
// config.ExtraLifeEvery (DESIGN §5.4).
func (w *World) score(points int) {
	s := w.Session
	before := s.Score
	s.AddScore(points)
	if lives := s.Score/config.ExtraLifeEvery - before/config.ExtraLifeEvery; lives > 0 {
		s.Lives += lives
		w.emit(ExtraLife)
	}
}

// landed scores a hop into a row further than the hero has reached on
// this attempt (DESIGN §5.5). updatePlayer calls it on each landing.
func (w *World) landed() {
	if r := w.Player.Row; r < w.furthest {
		w.furthest = r
		if !w.Player.Sliding {
			w.countHop() // a slide is not a hop (D31)
		}
		w.score(w.comboPoints(config.PointsHop))
		w.Stats.Hops += config.PointsHop
	}
}

// updateTimer runs the attempt's clock down: a warning when time is low,
// and a death when it runs out (DESIGN §5.4).
func (w *World) updateTimer() {
	w.TicksLeft--
	switch w.TicksLeft {
	case config.LowTimeSecs * config.TPS:
		w.emit(LowTime)
	case 0:
		w.die(TimeUp)
	}
}

// checkHazards kills the hero if its row is a danger to it. Only the lane
// the hero is over is checked (PLAN §3).
//   - Road: a vehicle overlaps it. The hero's box is a cell trimmed by
//     config.HeroMargin; vehicles are trimmed by config.HitMargin (DESIGN
//     §5.1, D13).
//   - River: a platform carried its middle off the field, or it has landed
//     with no platform under its middle, or on a heron (DESIGN §5.2). Water
//     and herons only count once the hero is down, so a hop over a gap onto
//     a platform is safe.
//   - Rail: a passing train overlaps it, trimmed like a vehicle (DESIGN
//     §6.2).
//   - Bank: on landing, the hero fills an empty nook or bumps into the bank
//     (DESIGN §5.3).
func (w *World) checkHazards() {
	p := &w.Player
	lane := &w.Lanes[p.LaneRow()]
	m := config.HeroMargin * config.SubPx
	left, right := p.X+m, p.X+cellUnits-m
	switch lane.Kind {
	case config.Road:
		if lane.hits(left, right, config.HitMargin) {
			w.die(Squashed)
		}
	case config.River:
		switch mid := p.Middle(); {
		case mid < 0 || mid >= fieldWidth:
			w.die(Swept)
		case p.Hopping():
		case !lane.holds(mid):
			w.die(Splashed)
		case lane.heronIn(left, right):
			w.die(Caught)
		}
	case config.Rail:
		if w.Rail.Hits(left, right) {
			w.die(Struck)
		}
	case config.Bank:
		if !p.Hopping() {
			w.reachBank()
		}
	}
}

// NookAt is the nook column col is in, or -1 for the bank between them.
func NookAt(col int) int {
	for i, c := range config.Nooks {
		if col >= c && col < c+config.NookWidth {
			return i
		}
	}
	return -1
}

// reachBank fills the nook the hero landed in, ending the attempt with a
// success, or bumps the hero on the bank or a filled nook (DESIGN §5.3).
// The fifth nook clears the level.
func (w *World) reachBank() {
	n := NookAt(w.Player.Col)
	if n < 0 || w.Filled[n] {
		w.die(Bumped)
		return
	}
	w.Filled[n] = true
	secs := w.TicksLeft / config.TPS
	w.score(w.comboPoints(config.PointsGoal) + secs*config.PointsPerSec)
	w.Combo = Combo{} // the attempt is over; the combo was cashed in (D38)
	w.Stats.Goals++
	w.Stats.TimeBonus += secs * config.PointsPerSec
	w.emit(Goal)
	for _, f := range w.Filled {
		if !f {
			w.Home = config.GoalTicks
			return
		}
	}
	w.score(config.PointsAllGoals)
	w.Stats.Clear += config.PointsAllGoals
	w.Cleared = true
	w.emit(LevelClear)
}

// die starts the hero's death. The life goes at once, so the status bar
// shows it during the death.
func (w *World) die(how Event) {
	w.Player.Death = how
	w.Player.queued = None
	w.Combo = Combo{}
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
