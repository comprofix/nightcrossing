package world

import (
	"math/rand/v2"
	"slices"

	"github.com/comprofix/nightcrossing/internal/config"
)

// Clone is a copy of w that runs on its own: its lanes, session and events
// are its own, and its random choices come from rng.
func (w *World) Clone(rng *rand.Rand) *World {
	c := *w
	s := *w.Session
	c.Session = &s
	for r := range c.Lanes {
		c.Lanes[r].Entities = slices.Clone(w.Lanes[r].Entities)
	}
	c.Spells = slices.Clone(w.Spells)
	c.Line = cloneLine(w.Line)
	c.rng = rng
	c.events = nil
	return &c
}

// Autopilot plays the hero for the title's demo (DESIGN §7). Before each
// hop it tries every move config.DemoLookahead moves ahead on copies of the
// world, and takes the first move of the line that gets furthest toward an
// empty nook without dying. A toadlet's death counts as the hero's, so it
// escorts a line of them (D53, D55). A firefly caught on the way counts
// for a few rows, so it takes one its route can reach (D58).
type Autopilot struct{}

// moves are the autopilot's moves, best first: on a tie the earlier wins.
var moves = [...]Dir{Up, Left, Right, None, Down}

// dead is the value of a line in which the hero dies; a later death is
// worth a little more than an earlier one.
const dead = -1_000_000

// Next is the direction to press this tick, or None.
func (Autopilot) Next(w *World) Dir {
	if !w.playing() || w.Player.Hopping() {
		return None
	}
	best, bestV := None, dead-1
	for _, d := range moves {
		c := w.Clone(rand.New(rand.NewPCG(uint64(w.Tick), 0)))
		if v := value(c, d, config.DemoLookahead, w.Stats.Fireflies); v > bestV {
			best, bestV = d, v
		}
	}
	return best
}

// value plays move d on w, then the best of depth-1 more moves, and is how
// good the best line is. caught is how many fireflies were caught before
// the line began. w is changed.
func value(w *World, d Dir, depth, caught int) int {
	if t, ok := step(w, d); !ok {
		return dead + t
	}
	bonus := (w.Stats.Fireflies - caught) * config.DemoFirefly * config.Cols * 2
	if w.Home > 0 || w.Cleared {
		return 1000*config.Rows + bonus // a nook filled: as good as it gets
	}
	if depth == 1 {
		for t := range config.DemoTailTicks {
			if w.Update(None); w.Dying > 0 {
				return dead + config.DemoTailTicks*config.DemoLookahead + t
			}
		}
		return progress(w) + bonus
	}
	best := dead - 1
	for _, d := range moves {
		best = max(best, value(w.Clone(w.rng), d, depth-1, caught))
	}
	return best
}

// step plays one move: a hop until it lands, or a short wait. It reports
// how many ticks the hero lived and whether it is still alive.
func step(w *World, d Dir) (int, bool) {
	n := config.DemoWaitTicks
	if d != None {
		n = config.HopTicks
	}
	for t := range n {
		if t == 0 {
			w.Update(d)
		} else {
			w.Update(None)
		}
		if w.Dying > 0 || w.Over {
			return t, false
		}
		if w.Home > 0 || w.Cleared {
			break
		}
	}
	return n, true
}

// progress is how far the hero has got: rows up the field, less how far its
// column is from an empty nook, which counts for more the nearer the bank
// it is, so it does not ride the top row away from the last nook.
func progress(w *World) int {
	p := w.Player
	far := config.Cols
	for i, c := range config.Nooks {
		if w.Filled[i] {
			continue
		}
		for col := c; col < c+config.NookWidth; col++ {
			far = min(far, abs(p.Col-col))
		}
	}
	up := config.StartRow - p.Row
	return up*config.Cols*2 - far*up
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
