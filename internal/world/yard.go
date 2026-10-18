package world

import "github.com/comprofix/nightcrossing/internal/config"

// A bonus level's goods yard (DESIGN §6.7): trains standing on the track
// rows. The toad can hop onto a carriage's roof from any side and ride
// it, but never onto an engine: a hop into one is blocked. Only a rake
// with an engine shunts. These are slips:
//   - on the rails, being run over by a moving rake's leading end, an
//     engine or a carriage pushed ahead of one;
//   - on a roof, being carried off the field.

// Rake is a train standing in the yard: carriages, with an engine or not.
// One that shunts runs slowly between two points, pausing at each.
type Rake struct {
	Row    int
	X      int  // its left end from the field's left edge, in SubPx units
	Cars   int  // its carriages
	Engine bool // an engine at the end it faces
	Facing int  // the way its engine faces: +1 right, -1 left
	Last   bool // the last train, which pulls out when the clock runs out
	Step   int  // how far it moved this tick, in SubPx units

	ends    [2]int // the two ends of its shunt, as X; the same for one that stands
	heading int    // which of ends it is running to
	wait    int    // ticks left of its pause at an end
}

// newRake is r as it starts: at its column, about to shunt.
func newRake(r config.Rake) Rake {
	x := r.Col * cellUnits
	k := Rake{Row: r.Row, X: x, Cars: r.Cars, Engine: r.Engine, Facing: 1, Last: r.Last,
		ends: [2]int{x, x + r.Shunt*cellUnits}, heading: 1}
	if r.Engine && r.Facing < 0 {
		k.Facing = -1
	}
	return k
}

// engineSpan is the part of the field k's engine covers, left to right,
// in SubPx units; empty for a rake without one.
func (k *Rake) engineSpan() (left, right int) {
	if !k.Engine {
		return 0, 0
	}
	e := config.EngineCells * cellUnits
	if k.Facing < 0 {
		return k.X, k.X + e
	}
	return k.X + k.Length() - e, k.X + k.Length()
}

// carriageAt is the rake on row r with a carriage under the point x, or
// -1: an engine's cells don't count.
func (w *World) carriageAt(r, x int) int {
	for i := range w.Yard {
		k := &w.Yard[i]
		if el, er := k.engineSpan(); k.Row == r && k.holds(x) && (x < el || x >= er) {
			return i
		}
	}
	return -1
}

// engineAt reports whether an engine is under the point x on row r.
func (w *World) engineAt(r, x int) bool {
	for i := range w.Yard {
		if k := &w.Yard[i]; k.Row == r {
			if el, er := k.engineSpan(); x >= el && x < er {
				return true
			}
		}
	}
	return false
}

// rakeHits reports whether a rake on row r overlaps left–right, in SubPx
// units, trimmed by config.HitMargin at each end like a vehicle.
func (w *World) rakeHits(r, left, right int) bool {
	m := config.HitMargin * config.SubPx
	for i := range w.Yard {
		if k := &w.Yard[i]; k.Row == r && k.X+m < right && left < k.X+k.Length()-m {
			return true
		}
	}
	return false
}

// inYard is the hero on a track row of a bonus level's yard. Landing, it
// is on the roof of the carriage under it, or else on the rails. On a
// roof it is safe until carried off the field; on the rails, a rake
// running into it runs it over. Either way it collects what is there.
func (w *World) inYard(left, right int) {
	p := &w.Player
	r := p.LaneRow()
	switch mid := p.Middle(); {
	case mid < 0 || mid >= fieldWidth:
		w.die(Swept) // carried off the field on a roof
	case p.Hopping():
	default:
		if p.stood == 0 { // just landed
			w.roof = w.carriageAt(r, mid)
		}
		if w.roof < 0 && w.rakeHits(r, left, right) {
			w.die(Struck) // run over by an engine, or a carriage pushed ahead of one
			return
		}
		w.collect(r, left, right)
	}
}

// Length is how long k is, in SubPx units.
func (k *Rake) Length() int {
	return config.Rake{Cars: k.Cars, Engine: k.Engine}.Cells() * cellUnits
}

// update runs k one tick: a standing rake stays put; a shunting one runs
// to the end it is heading for, then waits there and heads back.
func (k *Rake) update() {
	k.Step = 0
	goal := k.ends[k.heading]
	switch {
	case k.ends[0] == k.ends[1]:
	case k.wait > 0:
		k.wait--
	default:
		d := min(config.ShuntSpeed, abs(goal-k.X))
		if goal < k.X {
			d = -d
		}
		k.X += d
		k.Step = d
		if k.X == goal {
			k.heading ^= 1
			k.wait = config.ShuntPauseTicks
		}
	}
}

// holds reports whether k is under the point x, in SubPx units from the
// field's left edge, reaching config.StandGrace pixels past its ends as a
// platform does.
func (k *Rake) holds(x int) bool {
	g := config.StandGrace * config.SubPx
	return x >= k.X-g && x < k.X+k.Length()+g
}

// LastTrain is the yard's last train, or nil if the level has none.
func (w *World) LastTrain() *Rake {
	for i := range w.Yard {
		if w.Yard[i].Last {
			return &w.Yard[i]
		}
	}
	return nil
}

// Departing reports whether the last train is about to leave: the clock is
// low on a level with one. The crossing lights flash and the bell rings
// (DESIGN §6.7).
func (w *World) Departing() bool {
	return w.LastTrain() != nil && w.TicksLeft <= config.LowTimeSecs*config.TPS
}

// Pick is one fruit of a bonus level's set (DESIGN §6.7): which it is,
// and where it lies, riding a carriage or a log, or on the ground.
type Pick struct {
	Fruit int // which of config.Fruits it is
	Row   int
	Taken bool

	rake   int // the rake it rides, in Yard, or -1
	host   int // the log it rides, in its row's lane, or -1
	offset int // from what it rides's left end, or else the field's, in SubPx units
}

// layFruit lays the bonus level's fruit for this run, each in a random
// spot on its side: on the river, a cell of a log well in from the
// field's edges; in the yard, the rails or a carriage's roof, never an
// engine, nor rails a rake shunts over. None lies next to another, even
// corner to corner, nor near the toad's start (DESIGN §6.7).
func (w *World) layFruit() {
	yard, river := w.fruitSpots()
	for _, f := range w.Level.Bonus.Fruit {
		spots := yard
		if f.River {
			spots = river
		}
		var free []Pick
		for _, s := range spots {
			if w.apart(s) {
				free = append(free, s)
			}
		}
		if len(free) == 0 {
			free = spots // too crowded to keep them apart: anywhere on its side
		}
		p := free[w.rng.IntN(len(free))]
		p.Fruit = f.Fruit
		w.Picks = append(w.Picks, p)
	}
}

// fruitSpots are every spot fruit could lie at the start, in the yard and
// on the river, as Picks still to be given their fruit.
func (w *World) fruitSpots() (yard, river []Pick) {
	start := func(r, x int) bool {
		return abs(r-config.StartRow) <= config.BonusClearCells && abs(columnOf(x)-config.StartCol) <= config.BonusClearCells
	}
	for r := range w.Lanes {
		switch w.Lanes[r].Kind {
		case config.Rail:
			for c := range w.Cols {
				x := c * cellUnits
				mid := x + cellUnits/2
				switch k := w.carriageAt(r, mid); {
				case start(r, x), w.engineAt(r, mid):
				case k >= 0:
					yard = append(yard, Pick{Row: r, rake: k, host: -1, offset: x - w.Yard[k].X})
				case !w.shuntedOver(r, mid):
					yard = append(yard, Pick{Row: r, rake: -1, host: -1, offset: x})
				}
			}
		case config.River:
			for h, e := range w.Lanes[r].Entities {
				for c := range e.Kind.Length / config.Cell {
					off := c * cellUnits
					if w.perch(r, e, off, config.FireflyLandMargin) && !start(r, e.X()+off) {
						river = append(river, Pick{Row: r, rake: -1, host: h, offset: off})
					}
				}
			}
		}
	}
	return yard, river
}

// shuntedOver reports whether a shunting rake on row r runs over the
// point x: fruit there would look as if it rode the rake.
func (w *World) shuntedOver(r, x int) bool {
	for i := range w.Yard {
		k := &w.Yard[i]
		lo, hi := min(k.ends[0], k.ends[1]), max(k.ends[0], k.ends[1])+k.Length()
		if k.Row == r && k.ends[0] != k.ends[1] && x >= lo && x < hi {
			return true
		}
	}
	return false
}

// apart reports whether spot s is clear of every fruit laid so far, by
// config.BonusFruitApart cells across or up, and on a log or carriage
// with no fruit on it yet: one each (DESIGN §6.7).
func (w *World) apart(s Pick) bool {
	x := w.pickX(&s)
	for i := range w.Picks {
		o := &w.Picks[i]
		if abs(o.Row-s.Row) < config.BonusFruitApart && abs(w.pickX(o)-x) < config.BonusFruitApart*cellUnits {
			return false
		}
		if o.Row == s.Row && (s.host >= 0 && o.host == s.host || s.rake >= 0 && o.rake == s.rake && carriage(o) == carriage(&s)) {
			return false
		}
	}
	return true
}

// carriage is which carriage, or the engine, of its rake fruit p rides,
// counted from the rake's left end: an engine is as long as a carriage.
func carriage(p *Pick) int { return p.offset / (config.CarCells * cellUnits) }

// PickX is where fruit i is across the field, in SubPx units.
func (w *World) PickX(i int) int { return w.pickX(&w.Picks[i]) }

// pickX is where fruit p is across the field, in SubPx units.
func (w *World) pickX(p *Pick) int {
	switch {
	case p.rake >= 0:
		return w.Yard[p.rake].X + p.offset
	case p.host >= 0:
		return w.Lanes[p.Row].Entities[p.host].X() + p.offset
	}
	return p.offset
}

// Showing reports whether fruit i is still to be collected and on the
// field.
func (w *World) Showing(i int) bool {
	x := w.PickX(i)
	return !w.Picks[i].Taken && x+cellUnits > 0 && x < fieldWidth
}

// pickFruit collects any fruit on row r over left–right, in SubPx units,
// where the toad is down: its points, and the end of the level once the
// set is complete. Its box is its cell, trimmed like a firefly's.
func (w *World) pickFruit(r, left, right int) {
	m := config.HitMargin * config.SubPx
	for i := range w.Picks {
		p := &w.Picks[i]
		if p.Row != r || !w.Showing(i) {
			continue
		}
		if x := w.PickX(i); x+m >= right || left >= x+cellUnits-m {
			continue
		}
		p.Taken = true
		pts := config.BonusFruitPoints[p.Fruit]
		w.Stats.Picked++
		w.Stats.Fruit += pts
		w.score(pts)
		w.Gained += pts
		w.emit(FruitTaken)
		w.checkSet()
	}
}

// rideRake carries the hero with the carriage it stands on, on a track
// row of a bonus level, as a platform carries it on the river. A hop
// always lands on its column, so mid-hop it is carried by nothing.
func (w *World) rideRake() {
	p := &w.Player
	if w.roof < 0 || p.Hopping() || w.Yard[w.roof].Step == 0 {
		return
	}
	step := w.Yard[w.roof].Step
	p.fromX += step
	p.toX += step
	p.place()
}
