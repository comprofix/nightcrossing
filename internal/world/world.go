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
	Hopped          Event = iota + 1 // the hero left a cell
	Landed                           // the hero arrived in a cell
	Squashed                         // a vehicle hit the hero
	Splashed                         // the hero landed in the water, or its platform went under
	Swept                            // a platform carried the hero off the field
	Caught                           // the hero landed on a heron
	Bumped                           // the hero landed on the bank or a filled nook
	TimeUp                           // the attempt's time ran out
	LowTime                          // the attempt's time is running low
	Goal                             // the hero filled a nook
	LevelClear                       // the last nook is filled
	ExtraLife                        // the score earned a life
	Respawned                        // a new attempt began on the start row
	GameOver                         // the last life is gone
	Struck                           // a train hit the hero (D37)
	TrainWarning                     // the crossing lights start to flash: a train is coming
	TrainComing                      // a train comes onto the field
	TrainGone                        // the train has left the field
	ComboUp                          // the hop combo's multiplier rose (D38)
	WaveWarning                      // a rush hour wave is announced on a lane (D43)
	WaveStart                        // the wave begins: its traffic comes on
	WaveEnd                          // the wave ends: its extra traffic goes as it leaves
	FogWarning                       // fog is announced (D29)
	FogStart                         // the fog is in
	FogEnd                           // the fog clears
	RainWarning                      // rain is announced (D31)
	RainStart                        // it rains: the road is slippery
	RainEnd                          // the rain stops
	Slid                             // the hero began a rain slide (D31)
	WetStormWarning                  // a wet storm is announced: heavy rain (D51)
	DryStormWarning                  // a dry storm is announced: high winds, with their direction (D51)
	StormStart                       // the storm is in: its rain or its wind begins
	StormEnd                         // the storm rolls away
	Pushed                           // the wind pushed the hero one cell (D47)
	FloodStart                       // a wet storm's flood begins: puddles on the verge, the river speeding up (D48)
	FloodEnd                         // the flood has drained
	Refused                          // a hop off the verge onto the river, refused: the line isn't gathered (D53)
	FireflyCaught                    // the hero caught a firefly: a life, with ExtraLife, or points at the cap (D58)
	FireflyGone                      // the firefly flew away, its visit over (D58)
	FruitTaken                       // the nook filled had fruit in it (D59)
)

// Stats are the points earned on this level, for the level-clear tally.
type Stats struct {
	Hops      int // points for hops into new rows
	Goals     int // nooks filled
	TimeBonus int // points for time left when filling them
	Clear     int // the bonus for filling the last one
	Combo     int // what the hop combo's multiplier added (D38)
	Toadlets  int // toadlets brought home (D53)
	Fruit     int // points for fruit taken, before the combo (D59)
	Fireflies int // fireflies caught (D58)
	AtCap     int // points for fireflies caught with lives at the cap (D57)
}

// World is one level of a game.
type World struct {
	Cols, Rows int
	Session    *session.Session
	Level      config.Level
	Tick       int64 // ticks since the level started
	TicksLeft  int   // time left on this attempt (DESIGN §5.4)
	Player     Player
	Line       []Toadlet // the toadlets following the hero, first to last, in School Run (D53)
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
	Gained     int   // points to pop up this tick: what the combo multiplied, and a firefly's at the cap
	FruitNook  int   // the nook fruit shows in, or -1 (D59)
	Flyer      Flyer // the firefly, or by day the dragonfly (D58)

	fruitTicks int   // ticks until the fruit goes, or until it next shows
	furthest   int   // the furthest row reached this attempt
	fallen     int   // whose death is under way: Hero, or a toadlet (D53)
	clearUntil int64 // no weather starts before this tick: the clear stretch after a storm (D51)
	storms     int   // storms chosen this level, counting the next
	stormRun   int   // how many of them in a row are the latest's kind
	lastWet    bool  // the latest storm chosen is wet
	rng        *rand.Rand
	events     []Event
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
	if s := w.Spell(config.StormFront); s != nil {
		w.nextStorm(s)
	}
	w.newFruit()
	w.newAttempt()
	w.rollFirefly()
	return w
}

// newAttempt puts the hero, and its line if it has one, on the start row
// with a full clock.
func (w *World) newAttempt() {
	w.Player = newPlayer(config.StartCol, config.StartRow)
	w.Line = newLine(w.Level.Toadlets)
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
	surge := w.surge()
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.River {
			w.Lanes[r].Surge = surge
		}
		if len(w.Lanes[r].Entities) > 0 {
			w.Lanes[r].update(w.rng)
		}
	}
	w.updateFirefly()
	w.updateFruit()
	w.Rail.update(w)
	w.updateSpells()
	switch {
	case w.Dying > 0:
		w.updateDeath()
	case w.Home > 0:
		if w.Home--; w.Home == 0 {
			w.newAttempt()
			w.rollFirefly() // a new crossing, after a nook filled (D58)
			w.emit(Respawned)
		}
	default:
		// The hero goes first, so its line moves on the same beat (D53).
		w.updatePlayer(Hero, press)
		w.updateCombo()
		for i := range w.Line {
			if w.out(i) {
				w.updatePlayer(i, None)
			}
		}
		w.bunch()
		if w.out(Hero) {
			w.ride()
		}
		for i := Hero; i < len(w.Line); i++ {
			if w.out(i) {
				w.checkHazards(i)
			}
		}
		if w.playing() {
			w.carry()
			w.updateTimer()
		}
	}
	return w.events
}

// playing reports whether the attempt is on: no one dying, no toad home in
// a nook, and the level not yet cleared.
func (w *World) playing() bool { return w.Dying == 0 && w.Home == 0 && !w.Cleared }

func (w *World) emit(e Event) { w.events = append(w.events, e) }

// score adds points, and a life each time the score passes a multiple of
// config.ExtraLifeEvery, up to the cap (DESIGN §5.4, D57).
func (w *World) score(points int) {
	s := w.Session
	before := s.Score
	s.AddScore(points)
	if lives := s.Score/config.ExtraLifeEvery - before/config.ExtraLifeEvery; lives > 0 {
		w.gainLives(lives)
	}
}

// gainLives adds n lives, up to config.MaxLives, with ExtraLife if any
// came (D57).
func (w *World) gainLives(n int) {
	s := w.Session
	if s.Lives >= config.MaxLives {
		return
	}
	s.Lives = min(s.Lives+n, config.MaxLives)
	w.emit(ExtraLife)
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

// checkHazards kills body i, the hero or a toadlet, if its row is a danger
// to it; the world acts on toadlets as on the hero (D53). Only the lane it
// is over is checked (PLAN §3).
//   - Road: a vehicle overlaps it. Its box is a cell trimmed by
//     config.HeroMargin; vehicles are trimmed by config.HitMargin (DESIGN
//     §5.1, D13).
//   - River: a platform carried its middle off the field, or it has landed
//     with no platform under its middle, or on a heron (DESIGN §5.2). Water
//     and herons only count once it is down, so a hop over a gap onto a
//     platform is safe.
//   - Rail: a passing train overlaps it, trimmed like a vehicle (DESIGN
//     §6.2).
//   - Bank: on landing, the hero fills an empty nook or bumps into the bank
//     (DESIGN §5.3); a toadlet goes into the hero's nook.
//   - Verge: in a flood, a cell under water is a splash once it is down,
//     whether it landed there or the water came to it (D48).
func (w *World) checkHazards(i int) {
	p := w.body(i)
	lane := &w.Lanes[p.LaneRow()]
	m := config.HeroMargin * config.SubPx
	left, right := p.X+m, p.X+cellUnits-m
	switch lane.Kind {
	case config.Road:
		if lane.hits(left, right, config.HitMargin) {
			w.dieAs(i, Squashed)
		}
	case config.River:
		switch mid := p.Middle(); {
		case mid < 0 || mid >= fieldWidth:
			w.dieAs(i, Swept)
		case p.Hopping():
		case !lane.holds(mid):
			w.dieAs(i, Splashed)
		case lane.heronIn(left, right):
			w.dieAs(i, Caught)
		case i == Hero:
			w.catchFirefly(p.LaneRow(), left, right)
		}
	case config.Rail:
		if w.Rail.Hits(left, right) {
			w.dieAs(i, Struck)
		}
	case config.Bank:
		if !p.Hopping() && i == Hero { // toadlets reach it on the hero's back (D53)
			w.reachBank()
		}
	case config.Safe:
		if !p.Hopping() && p.Row == config.VergeRow && w.Flooded(columnOf(p.X)) {
			w.dieAs(i, Splashed)
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

// reachBank fills the nook the hero landed in, with any toadlets on its
// back, ending the attempt with a success, or bumps the hero on the bank or
// a filled nook (DESIGN §5.3). The fifth nook clears the level. Each
// toadlet home, and fruit in the nook, add to the nook's points,
// multiplied by the combo with them (D53, D59).
func (w *World) reachBank() {
	n := NookAt(w.Player.Col)
	if n < 0 || w.Filled[n] {
		w.die(Bumped)
		return
	}
	w.Filled[n] = true
	secs := w.TicksLeft / config.TPS
	toadlets := len(w.Line) * config.PointsToadlet
	fruit := w.takeFruit(n)
	w.score(w.comboPoints(config.PointsGoal+toadlets+fruit) + secs*config.PointsPerSec)
	w.Combo = Combo{} // the attempt is over; the combo was cashed in (D38)
	w.Stats.Goals++
	w.Stats.Toadlets += len(w.Line)
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

// die starts the hero's death.
func (w *World) die(how Event) { w.dieAs(Hero, how) }

// dieAs starts the death of body i, the hero or a toadlet. Either ends the
// attempt (D53). The life goes at once, so the status bar shows it during
// the death.
func (w *World) dieAs(i int, how Event) {
	w.body(i).Death = how
	w.fallen = i
	w.Player.queued = None
	w.Player.push = 0
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
