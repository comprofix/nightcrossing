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
	c.Swarm = slices.Clone(w.Swarm)
	c.Yard = slices.Clone(w.Yard)
	c.Picks = slices.Clone(w.Picks)
	c.Stragglers = slices.Clone(w.Stragglers)
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
		if v := value(c, d, config.DemoLookahead, w.Collected()); v > bestV {
			best, bestV = d, v
		}
	}
	if best != None && w.Bonus() && w.refused(best) {
		return None // wait on the verge for the line to gather, rather than peep at it (DESIGN §6.7)
	}
	return best
}

// refused reports whether a hop in d now would be refused: off the verge
// onto the river with the line not yet gathered (D53).
func (w *World) refused(d Dir) bool {
	p := &w.Player
	return d == Up && p.Row > 0 && w.Lanes[p.Row-1].Kind == config.River && w.Lanes[p.Row].Kind != config.River && !w.gathered()
}

// value plays move d on w, then the best of depth-1 more moves, and is how
// good the best line is. caught is how many fireflies, and on a bonus
// level fruit, were collected before the line began. w is changed.
func value(w *World, d Dir, depth, caught int) int {
	before := w.Collected()
	if t, ok := step(w, d); !ok {
		return dead + t
	}
	worth, soon := config.DemoFirefly, 0
	if w.Bonus() {
		// On a bonus level the fireflies are the point, and one caught
		// sooner is worth a little more, so it never puts a catch off
		// (D61).
		worth, soon = config.DemoSwarmFirefly, (w.Collected()-before)*depth
	}
	bonus := (w.Collected() - caught) * worth * config.Cols * 2
	if w.Cleared || w.Home > 0 && !w.Bonus() {
		return 1000*config.Rows + bonus + soon // a nook filled: as good as it gets
	}
	if depth == 1 {
		for t := range config.DemoTailTicks {
			if w.Update(None); w.Dying > 0 {
				return dead + config.DemoTailTicks*config.DemoLookahead + t
			}
		}
		return progress(w) + bonus + soon
	}
	best := dead - 1
	for _, d := range moves {
		best = max(best, value(w.Clone(w.rng), d, depth-1, caught))
	}
	return best + soon
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
			ev := w.Update(d)
			if d != None && w.Bonus() && w.playing() && !w.Player.Hopping() && !slices.Contains(ev, Refused) {
				return 0, false // a parked car in the way: no move at all (D61)
			}
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
	if w.Bonus() {
		// A straggler in the line is half way home: worth more than the
		// walk to the next, but less than bringing it home (DESIGN §6.7).
		return -hunt(w)*config.Cols*2 + len(w.Line)*config.DemoSwarmFirefly*config.Cols
	}
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

// hunt is how many hops the hero is from the nearest firefly or fruit on
// a bonus level, round the parked cars, or 0 with none about (D61). The
// river and the yard count as open, since logs and carriages come round.
func hunt(w *World) int {
	type cell struct{ col, row int }
	var dist [config.Rows][config.Cols]int
	for r := range dist {
		for c := range dist[r] {
			dist[r][c] = -1
		}
	}
	p := w.Player
	start := cell{columnOf(p.X), p.Row}
	dist[start.row][start.col] = 0
	queue := []cell{start}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range [...]cell{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			n := cell{c.col + d.col, c.row + d.row}
			if n.row < 0 || n.row >= w.Rows || n.col < 0 || n.col >= w.Cols || dist[n.row][n.col] >= 0 ||
				w.Lanes[n.row].Kind == config.Bank || w.Parked(n.col, n.row) {
				continue
			}
			dist[n.row][n.col] = dist[c.row][c.col] + 1
			queue = append(queue, n)
		}
	}
	best := -1
	for i := range w.Swarm {
		f := &w.Swarm[i]
		if !f.Visiting {
			continue
		}
		if d := dist[f.row][columnOf(w.spotX(f))]; d >= 0 && (best < 0 || d < best) {
			best = d
		}
	}
	for i := range w.Picks {
		if !w.Showing(i) {
			continue
		}
		col := min(max(columnOf(w.PickX(i)+cellUnits/2), 0), w.Cols-1)
		if d := dist[w.Picks[i].Row][col]; d >= 0 && (best < 0 || d < best) {
			best = d
		}
	}
	// On the school gates, a straggler waiting while the line has room;
	// with the line full, or none left waiting, a nook, one hop up from
	// the river's top row (DESIGN §6.7).
	waiting := 0
	for i := range w.Stragglers {
		if w.Stragglers[i].Waiting() {
			waiting++
		}
	}
	if b := w.Level.Bonus; b != nil && b.Stragglers > 0 {
		if waiting > 0 && len(w.Line) < b.LineMax {
			for _, s := range w.Stragglers {
				if d := dist[s.Row][s.Col]; s.Waiting() && d >= 0 && (best < 0 || d < best) {
					best = d
				}
			}
		} else if len(w.Line) > 0 {
			for _, c := range config.Nooks {
				for col := c; col < c+config.NookWidth; col++ {
					if d := dist[config.GoalRow+1][col]; d >= 0 && (best < 0 || d+1 < best) {
						best = d + 1
					}
				}
			}
		}
	}
	return max(best, 0)
}
