package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// currentWorld is a long log lying still on riverRow, with a current that
// slides the hero right at c SubPx units a tick, and the hero landed on it
// at column col.
func currentWorld(c, col int) *World {
	w := riverWorld(config.LogLong, false, 6*cellUnits, 1, 0)
	w.Lanes[riverRow].Current = c
	standOn(w, col*cellUnits)
	return w
}

// TestCurrentSlidesTheHeroAlongItsPlatform: landed on a platform in a
// current row, the hero slides along it at the current's speed, which way
// it pushes (D46).
func TestCurrentSlidesTheHeroAlongItsPlatform(t *testing.T) {
	for _, c := range []int{3, -3} {
		w := currentWorld(c, 8)
		x0 := w.Player.X
		run(w, None, 20)
		if got, want := w.Player.X-x0, 20*c; got != want {
			t.Errorf("current %d: slid %d units in 20 ticks, want %d", c, got, want)
		}
		if w.Dying > 0 {
			t.Errorf("current %d: died on the log", c)
		}
	}
}

// TestCurrentAddsToThePlatform: the slide comes on top of the platform's
// own ride.
func TestCurrentAddsToThePlatform(t *testing.T) {
	w := riverWorld(config.LogLong, false, 6*cellUnits, -1, 10)
	w.Lanes[riverRow].Current = 4 // with the flow: left, faster
	standOn(w, 8*cellUnits)
	x0 := w.Player.X
	run(w, None, 10)
	if got, want := w.Player.X-x0, -10*(10+4); got != want {
		t.Errorf("moved %d units in 10 ticks, want %d", got, want)
	}
}

// TestNoCurrentMidHop: a hop along the platform escapes the current for the
// hop: it lands exactly a cell on, and slides again from the landing tick.
func TestNoCurrentMidHop(t *testing.T) {
	w := currentWorld(5, 8)
	x0 := w.Player.X
	run(w, Left, config.HopTicks)
	if got, want := w.Player.X, x0-cellUnits+5; got != want {
		t.Errorf("landed at %d, want %d: a cell left, with no slide but the landing tick's", got, want)
	}
}

// TestCurrentSlidesTheHeroOffTheEnd: slid past the platform's end, beyond
// its grace, the hero splashes.
func TestCurrentSlidesTheHeroOffTheEnd(t *testing.T) {
	w := currentWorld(6, 10) // the log covers columns 6–10
	var ev []Event
	for range 3 * config.TPS {
		if ev = append(ev, w.Update(None)...); w.Dying > 0 {
			break
		}
	}
	if !slices.Contains(ev, Splashed) {
		t.Fatalf("events %v: want a splash off the log's end", ev)
	}
	end := (6*config.Cell + config.LogLong.Length + config.StandGrace) * config.SubPx
	if mid := w.Player.Middle(); mid < end {
		t.Errorf("splashed with its middle at %d, inside the log's end and grace at %d", mid, end)
	}
}

// TestCurrentSlidesTheHeroOntoAHeron: slid onto a heron's cell is a catch.
func TestCurrentSlidesTheHeroOntoAHeron(t *testing.T) {
	w := currentWorld(6, 7)
	e := &w.Lanes[riverRow].Entities[0]
	e.Heron, e.HeronCell = true, 3 // column 9
	ev := run(w, None, 2*config.TPS)
	if !slices.Contains(ev, Caught) {
		t.Errorf("events %v: want the heron to catch the hero", ev)
	}
}

// gusty starts a gust spell on w pushing dir, its next push in one tick,
// then one every beat ticks.
func gusty(w *World, dir, beat int) {
	w.Spells = []Spell{{Kind: config.Gust, Timing: config.EventTiming{QuietMin: 1 << 20, QuietMax: 1 << 20, Last: 1 << 20, Beat: beat},
		Phase: SpellOn, Left: 1<<20 - beat + 1, Dir: dir}}
}

// TestGustPushesOneCellOnLand: on land the push is a forced sideways hop
// of one cell on the grid, keeping the facing (D47).
func TestGustPushesOneCellOnLand(t *testing.T) {
	w := newWorld(1)
	gusty(w, 1, 90)
	ev := run(w, None, config.HopTicks+1)
	if count(ev, Pushed) != 1 || count(ev, Hopped) != 0 {
		t.Fatalf("events %v: want one push, no hop", ev)
	}
	at(t, w, config.StartCol+1, config.StartRow)
	if w.Player.Facing != Up {
		t.Errorf("facing %v after a push, want up still", w.Player.Facing)
	}
	ev = run(w, None, 90)
	if count(ev, Pushed) != 1 {
		t.Errorf("events %v over a beat: want the next push", ev)
	}
	at(t, w, config.StartCol+2, config.StartRow)
}

// TestNoGustPushOffTheField: at the field's edge, on land, the push doesn't
// happen.
func TestNoGustPushOffTheField(t *testing.T) {
	w := newWorld(1)
	w.Player = newPlayer(config.Cols-1, config.StartRow)
	gusty(w, 1, 90)
	if ev := run(w, None, config.HopTicks+1); slices.Contains(ev, Pushed) {
		t.Errorf("pushed off the field: %v", ev)
	}
	at(t, w, config.Cols-1, config.StartRow)
}

// TestGustWaitsForTheLanding: a push that comes mid-hop waits for the
// landing, then goes before a queued press; it is delayed, never dodged.
func TestGustWaitsForTheLanding(t *testing.T) {
	w := newWorld(1)
	run(w, Up, 3)
	gusty(w, -1, 1000)
	ev := run(w, None, config.HopTicks-4) // up to the tick before it lands
	if slices.Contains(ev, Pushed) {
		t.Fatalf("pushed mid-hop: %v", ev)
	}
	ev = run(w, None, config.HopTicks+1)
	if count(ev, Pushed) != 1 {
		t.Fatalf("events %v: want the push after landing", ev)
	}
	at(t, w, config.StartCol-1, config.StartRow-1)
}

// TestGustOnAPlatformDrifts: on a platform the push drifts the hero a cell
// along it; past its end is a splash.
func TestGustOnAPlatformDrifts(t *testing.T) {
	w := currentWorld(0, 8)
	gusty(w, 1, 1000)
	x0 := w.Player.X
	ev := run(w, None, config.HopTicks+1)
	if count(ev, Pushed) != 1 || w.Player.X != x0+cellUnits || w.Dying > 0 {
		t.Fatalf("events %v, at %d: want a drift of a cell from %d", ev, w.Player.X, x0)
	}

	w = currentWorld(0, 10) // the log's last cell
	gusty(w, 1, 1000)
	if ev := run(w, None, config.HopTicks+1); !slices.Contains(ev, Splashed) {
		t.Errorf("events %v: want a splash off the log's end", ev)
	}
}

// TestGustAddsToTheCurrent: while a push drifts the hero, the current
// still slides it.
func TestGustAddsToTheCurrent(t *testing.T) {
	w := currentWorld(4, 7)
	gusty(w, 1, 1000)
	x0 := w.Player.X
	run(w, None, 1) // the push begins: the hero is still before it
	run(w, None, config.HopTicks)
	if got, want := w.Player.X-x0, cellUnits+(config.HopTicks+1)*4; got != want {
		t.Errorf("moved %d units, want a cell and the slide: %d", got, want)
	}
}

// TestGustNeverSlides: in the rain a push onto the road doesn't slide, and
// the push is neither a hop for the combo nor breaks it.
func TestGustNeverSlides(t *testing.T) {
	w := wetWorld()
	w.Level.Combo = true
	w.Player = newPlayer(config.StartCol, 10)
	w.furthest = 10
	w.Combo = Combo{Hops: 2, Left: config.ComboTicks}
	w.Spells = append(w.Spells, Spell{Kind: config.Gust, Timing: config.EventTiming{QuietMin: 1, QuietMax: 1, Last: 1 << 20, Beat: 1000},
		Phase: SpellOn, Left: 1<<20 - 1000 + 1, Dir: 1})
	ev := run(w, None, config.HopTicks+2)
	if count(ev, Pushed) != 1 || slices.Contains(ev, Slid) {
		t.Errorf("events %v: want a push and no slide", ev)
	}
	if w.Combo.Hops != 2 {
		t.Errorf("combo %d hops after a push, want 2", w.Combo.Hops)
	}
}

// TestGustSpellRuns: a gust spell is announced with a direction, pushes on
// its beat while on, and ends; the same seed blows the same way.
func TestGustSpellRuns(t *testing.T) {
	lv := testLevel()
	lv.Events[config.Gust] = config.EventsOn(config.Gust, 1, 1)
	g := lv.Events[config.Gust]
	play := func(seed uint64) (*World, []Event) {
		w := New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
		var ev []Event
		for range g.QuietMax + 3*config.TPS + g.Last {
			ev = append(ev, w.Update(None)...)
		}
		return w, ev
	}
	dirs := map[int]bool{}
	for seed := uint64(1); seed <= 6; seed++ {
		w, ev := play(seed)
		if count(ev, GustWarning) != 1 || count(ev, GustStart) != 1 || count(ev, GustEnd) != 1 {
			t.Fatalf("seed %d: events %v, want one spell", seed, ev)
		}
		if got, want := count(ev, Pushed), (g.Last-1)/g.Beat; got != want {
			t.Errorf("seed %d: %d pushes, want %d", seed, got, want)
		}
		dirs[w.Spell(config.Gust).Dir] = true
		if again, ev2 := play(seed); !slices.Equal(ev, ev2) || again.Player.X != w.Player.X {
			t.Errorf("seed %d: the same seed played differently", seed)
		}
	}
	if len(dirs) != 2 {
		t.Errorf("over 6 seeds gusts only blew %v", dirs)
	}
}

// TestNextPush: the tell knows when the next push comes.
func TestNextPush(t *testing.T) {
	s := Spell{Kind: config.Gust, Timing: config.EventTiming{Last: 600, Beat: 180}, Phase: SpellWarning, Left: 120}
	if got := s.NextPush(); got != 300 {
		t.Errorf("in the warning: next push in %d, want 300", got)
	}
	s.Phase, s.Left = SpellOn, 600-170
	if got := s.NextPush(); got != 10 {
		t.Errorf("170 ticks in: next push in %d, want 10", got)
	}
	s.Left = 600 - 550 // the push at 540 has gone; 720 is past the end
	if got := s.NextPush(); got != -1 {
		t.Errorf("near the end: next push in %d, want none", got)
	}
}

// TestOneBigEventAtATime: a wave, a gust spell and a flood never overlap,
// and each still comes (D48).
func TestOneBigEventAtATime(t *testing.T) {
	lv := config.Storm.Levels.Level(config.EventTopLevel)
	big := []config.EventKind{config.Wave, config.Gust, config.Flood}
	for seed := uint64(1); seed <= 4; seed++ {
		w := New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
		came := map[Event]int{}
		for range 4 * 60 * config.TPS {
			for _, e := range w.Update(None) {
				came[e]++
			}
			if w.Dying > 0 {
				w.Session.Lives = config.StartLives
			}
			on := 0
			for _, k := range big {
				if w.Spell(k).Phase != SpellQuiet {
					on++
				}
			}
			if on > 1 {
				t.Fatalf("seed %d tick %d: %d big events at once", seed, w.Tick, on)
			}
		}
		for _, e := range []Event{WaveStart, GustStart, FloodStart} {
			if came[e] == 0 {
				t.Errorf("seed %d: no event %d in four minutes", seed, e)
			}
		}
	}
}

// TestFloodCells: a flood takes stretches of 2–4 verge cells with at least
// 3 dry cells between them and at the edges, never more than half the
// verge, and the same seed floods the same cells (D48).
func TestFloodCells(t *testing.T) {
	for n := 4; n <= 14; n++ {
		for seed := uint64(1); seed <= 40; seed++ {
			cells := floodCells(n, rand.New(rand.NewPCG(seed, seed)))
			if again := floodCells(n, rand.New(rand.NewPCG(seed, seed))); again != cells {
				t.Fatalf("n %d seed %d: the same seed flooded different cells", n, seed)
			}
			flooded, run := 0, 0
			var runs []int // alternating lengths, starting with a dry run
			wet := false
			for c, f := range cells {
				if f {
					flooded++
				}
				if c > 0 && f != wet {
					runs = append(runs, run)
					run = 0
				}
				wet = f
				run++
			}
			runs = append(runs, run)
			if cells[0] {
				runs = append([]int{0}, runs...) // no dry run at the left edge
			}
			if flooded == 0 || flooded > min(n, config.FloodMostCells) {
				t.Fatalf("n %d seed %d: %d cells flooded", n, seed, flooded)
			}
			for i, r := range runs {
				edge := i == 0 || i == len(runs)-1
				switch {
				case i%2 == 1 && (r < config.FloodStretchMin || r > config.FloodStretchMax):
					t.Fatalf("n %d seed %d: a stretch of %d flooded cells: %v", n, seed, r, cells)
				case i%2 == 0 && r < config.FloodDryMin && !(edge && r == 0):
					t.Fatalf("n %d seed %d: only %d dry cells between floods: %v", n, seed, r, cells)
				}
			}
		}
	}
}

// floodWorld is an all-safe level with a flood forced on over verge
// columns 10–12, just started, with the hero on the verge at col.
func floodWorld(col int) *World {
	w := newWorld(1)
	s := Spell{Kind: config.Flood, Timing: config.EventsOn(config.Flood, 1, 1), Phase: SpellOn}
	s.Left = s.Timing.Last
	for c := 10; c <= 12; c++ {
		s.Cells[c] = true
	}
	w.Spells = []Spell{s}
	w.Player = newPlayer(col, config.VergeRow)
	w.furthest = config.VergeRow
	return w
}

// TestFloodPuddlesThenWater: flooded cells show puddles and are still dry
// ground for a second, then are water: a hero standing on one splashes,
// one on a dry cell beside it doesn't. At the end they drain the same way.
func TestFloodPuddlesThenWater(t *testing.T) {
	w := floodWorld(11)
	run(w, None, config.FloodPuddleTicks-1)
	if !w.Puddled(11) || w.Flooded(11) || w.Dying > 0 {
		t.Fatalf("puddled %v flooded %v dying %d: want puddles on dry ground first", w.Puddled(11), w.Flooded(11), w.Dying)
	}
	if ev := run(w, None, 2); !slices.Contains(ev, Splashed) {
		t.Errorf("events %v: want a splash as the water comes", ev)
	}

	w = floodWorld(9)
	ev := run(w, None, w.Spells[0].Timing.Last+1)
	if slices.Contains(ev, Splashed) || w.Puddled(9) || w.Flooded(9) {
		t.Errorf("a hero beside the flood: events %v", ev)
	}
	if count(ev, FloodEnd) != 1 {
		t.Errorf("events %v: want the flood to end", ev)
	}

	w = floodWorld(9)
	s := &w.Spells[0]
	s.Left = config.FloodPuddleTicks + 1
	run(w, None, 1)
	if w.Flooded(10) || !w.Puddled(10) {
		t.Errorf("draining: flooded %v puddled %v, want puddles again", w.Flooded(10), w.Puddled(10))
	}
}

// TestHoppingIntoTheFlood: a hop that lands on flooded verge is a splash;
// one over it isn't, until it lands.
func TestHoppingIntoTheFlood(t *testing.T) {
	w := floodWorld(9)
	run(w, None, config.FloodPuddleTicks+1)
	ev := run(w, Right, config.HopTicks/2)
	if slices.Contains(ev, Splashed) {
		t.Fatalf("splashed mid-hop: %v", ev)
	}
	if ev = run(w, None, config.HopTicks); !slices.Contains(ev, Splashed) {
		t.Errorf("events %v: want a splash on landing in the flood", ev)
	}
}

// TestFloodSurge: the river runs 20% faster in a flood, easing in over a
// second as it comes and out as it drains, and its current with it.
func TestFloodSurge(t *testing.T) {
	w := floodWorld(0)
	w.Lanes[riverRow] = Lane{Kind: config.River, Dir: -1, Speed: 50, Current: 10}
	run(w, None, config.FloodSurgeTicks/2)
	if got := w.Lanes[riverRow].Surge; got <= 0 || got >= config.FloodSurgePct {
		t.Errorf("half a second in: surge %d%%, want it easing in", got)
	}
	run(w, None, config.FloodSurgeTicks)
	l := &w.Lanes[riverRow]
	if l.Surge != config.FloodSurgePct || l.Step() != -60 || l.Slide() != -12 {
		t.Errorf("surge %d%%, step %d, slide %d: want 20%%, -60, -12", l.Surge, l.Step(), l.Slide())
	}
	if w.Lanes[config.StartRow].Surge != 0 {
		t.Error("the flood sped up a lane that isn't river")
	}
	run(w, None, w.Spells[0].Left+1)
	if l.Surge != 0 {
		t.Errorf("after the flood: surge %d%%", l.Surge)
	}
}
