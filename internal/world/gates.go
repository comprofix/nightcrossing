package world

import "github.com/comprofix/nightcrossing/internal/config"

// Straggler is a toadlet waiting at a kerb on the school gates, School
// Run's bonus level (DESIGN §6.7): on the start row, the verge or the
// track, in a random place each run. Landing on it adds it to the toad's
// line, which follows as in School Run; a nook brings the whole line home.
// A slip sends the line back to where each was waiting.
type Straggler struct {
	Row, Col int
	InLine   bool // in the toad's line, on its way home
	Home     bool // brought home
}

// Waiting reports whether s is still at its kerb.
func (s *Straggler) Waiting() bool { return !s.InLine && !s.Home }

// kerb reports whether row r is a kerb where stragglers wait: the start
// row, the verge or the track.
func (w *World) kerb(r int) bool {
	return r == config.StartRow || r == config.VergeRow || w.Lanes[r].Kind == config.Rail
}

// layStragglers lays the level's stragglers at random along the kerbs,
// apart from each other and clear of the toad's start (DESIGN §6.7).
func (w *World) layStragglers() {
	n := w.Level.Bonus.Stragglers
	var spots []Straggler
	for r := range w.Lanes {
		if !w.kerb(r) {
			continue
		}
		for c := range w.Cols {
			if r == config.StartRow && abs(c-config.StartCol) <= config.BonusClearCells {
				continue
			}
			spots = append(spots, Straggler{Row: r, Col: c})
		}
	}
	for len(w.Stragglers) < n && len(spots) > 0 {
		i := w.rng.IntN(len(spots))
		s := spots[i]
		spots = append(spots[:i], spots[i+1:]...)
		apart := true
		for _, o := range w.Stragglers {
			if abs(o.Row-s.Row) < config.BonusFruitApart && abs(o.Col-s.Col) < config.BonusFruitApart {
				apart = false
				break
			}
		}
		if apart {
			w.Stragglers = append(w.Stragglers, s)
		}
	}
}

// gather adds the straggler the toad has just landed on at row r, column
// col, to the front of its line, in the toad's cell, so the rest of the
// line files on behind it. With the line full, it peeps and waits.
func (w *World) gather(r, col int) {
	b := w.Level.Bonus
	for i := range w.Stragglers {
		s := &w.Stragglers[i]
		if !s.Waiting() || s.Row != r || s.Col != col {
			continue
		}
		if len(w.Line) >= b.LineMax {
			w.emit(Refused)
			return
		}
		s.InLine = true
		t := Toadlet{Player: newPlayer(col, r)}
		t.Facing = w.Player.Facing
		w.Line = append([]Toadlet{t}, w.Line...)
		w.emit(Joined)
		return
	}
}

// homeLine brings the toad's line home through a nook: each straggler in
// it is home, for its points, doubled for a trip of config's TripMin or
// more (DESIGN §6.7).
func (w *World) homeLine() {
	b := w.Level.Bonus
	n := 0
	for i := range w.Stragglers {
		if s := &w.Stragglers[i]; s.InLine {
			s.InLine, s.Home = false, true
			n++
		}
	}
	if n == 0 {
		return
	}
	pts := n * b.Straggler
	if n >= b.TripMin {
		w.Stats.Trip += pts
		pts *= 2
	}
	w.Stats.Toadlets += n
	w.score(pts)
	w.Gained += pts
	w.checkSet()
}

// strayBack sends the stragglers in the toad's line back to where each
// was waiting, as a slip ends the attempt (DESIGN §6.7).
func (w *World) strayBack() {
	for i := range w.Stragglers {
		w.Stragglers[i].InLine = false
	}
}
