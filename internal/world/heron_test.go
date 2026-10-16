package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// heronLane is a fast lane of short logs with herons at pct.
func heronLane(pct int, seed uint64) (*Lane, *rand.Rand) {
	rng := rand.New(rand.NewPCG(seed, seed))
	l := config.Lane{Kind: config.River, Dir: 1, Speed: 40, Pattern: []config.Spawn{
		{Entity: config.LogShort, Gap: 48}, {Entity: config.LogLong, Gap: 60},
	}}
	lane := newLane(l, pct, rng)
	return &lane, rng
}

// TestHeronsComeOnWithTheirLog runs a lane for a long while. A heron is
// only ever placed or taken away while its log is wholly off the field, and
// the log after a heron's log, coming on, never has one.
func TestHeronsComeOnWithTheirLog(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3} {
		lane, rng := heronLane(100, seed) // as many as the rules allow
		seen, last := 0, false
		for range 20_000 {
			before := slices.Clone(lane.Entities)
			lane.update(rng)
			for i, e := range lane.Entities {
				b := before[i]
				if e.Heron != b.Heron && !(b.off() || e.off()) {
					t.Fatalf("seed %d: log %d's heron changed while on the field", seed, i)
				}
				if e.Heron && (e.HeronCell < 0 || e.HeronCell >= e.Kind.Length/config.Cell) {
					t.Fatalf("seed %d: heron on cell %d of a %dpx log", seed, e.HeronCell, e.Kind.Length)
				}
				if b.off() && !e.off() { // coming on
					if e.Heron && last {
						t.Fatalf("seed %d: two heron logs in a row", seed)
					}
					last = e.Heron
					if e.Heron {
						seen++
					}
				}
			}
		}
		if seen == 0 {
			t.Errorf("seed %d: no herons at 100%%", seed)
		}
	}
}

func TestNoHeronsAtZero(t *testing.T) {
	lane, rng := heronLane(0, 1)
	for range 5_000 {
		lane.update(rng)
		for _, e := range lane.Entities {
			if e.Heron {
				t.Fatal("a heron at 0%")
			}
		}
	}
}

// TestLandingOnAHeron: the heron's cell is deadly, the rest of its log
// safe (DESIGN §5.2, D34).
func TestLandingOnAHeron(t *testing.T) {
	const x0 = 8 * cellUnits
	for _, c := range []struct {
		col    int
		caught bool
	}{{8, false}, {9, true}, {10, false}} {
		w := riverWorld(config.LogShort, false, x0, 1, 0)
		e := &w.Lanes[riverRow].Entities[0]
		e.Heron, e.HeronCell = true, 1
		w.Player = newPlayer(c.col, riverRow+1)
		ev := run(w, Up, config.HopTicks)
		if got := slices.Contains(ev, Caught); got != c.caught {
			t.Errorf("landing in column %d (heron in 9): caught %v, events %v", c.col, got, ev)
		}
	}
}

func TestHeronsOnlyOnLogs(t *testing.T) {
	w := New(session.New(config.Modes[0], 3), rich, rand.New(rand.NewPCG(4, 4)))
	for range 6000 {
		w.Update(None)
		for r, l := range w.Lanes {
			for _, e := range l.Entities {
				if e.Heron && !slices.Contains(config.Logs(), e.Kind) {
					t.Fatalf("row %d: a heron on %s", r, e.Kind.ID)
				}
			}
		}
		w.Player.Death, w.Dying, w.TicksLeft = 0, 0, 1000 // the hero is not the point here
	}
}
