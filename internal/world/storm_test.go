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

// windy starts a dry storm on w pushing dir, its next push in one tick,
// then one every beat ticks.
func windy(w *World, dir, beat int) {
	w.Spells = []Spell{{Kind: config.StormFront, Timing: config.EventTiming{QuietMin: 1 << 20, QuietMax: 1 << 20, Last: 1 << 20, Beat: beat},
		Phase: SpellOn, Length: 1 << 20, Left: 1<<20 - beat + 1, Dir: dir}}
}

// TestWindPushesOneCellOnLand: on land the push is a forced sideways hop
// of one cell on the grid, keeping the facing (D47).
func TestWindPushesOneCellOnLand(t *testing.T) {
	w := newWorld(1)
	windy(w, 1, 90)
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

// TestNoWindPushOffTheField: at the field's edge, on land, the push doesn't
// happen.
func TestNoWindPushOffTheField(t *testing.T) {
	w := newWorld(1)
	w.Player = newPlayer(config.Cols-1, config.StartRow)
	windy(w, 1, 90)
	if ev := run(w, None, config.HopTicks+1); slices.Contains(ev, Pushed) {
		t.Errorf("pushed off the field: %v", ev)
	}
	at(t, w, config.Cols-1, config.StartRow)
}

// TestWindWaitsForTheLanding: a push that comes mid-hop waits for the
// landing, then goes before a queued press; it is delayed, never dodged.
func TestWindWaitsForTheLanding(t *testing.T) {
	w := newWorld(1)
	run(w, Up, 3)
	windy(w, -1, 1000)
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

// TestWindOnAPlatformDrifts: on a platform the push drifts the hero a cell
// along it; past its end is a splash.
func TestWindOnAPlatformDrifts(t *testing.T) {
	w := currentWorld(0, 8)
	windy(w, 1, 1000)
	x0 := w.Player.X
	ev := run(w, None, config.HopTicks+1)
	if count(ev, Pushed) != 1 || w.Player.X != x0+cellUnits || w.Dying > 0 {
		t.Fatalf("events %v, at %d: want a drift of a cell from %d", ev, w.Player.X, x0)
	}

	w = currentWorld(0, 10) // the log's last cell
	windy(w, 1, 1000)
	if ev := run(w, None, config.HopTicks+1); !slices.Contains(ev, Splashed) {
		t.Errorf("events %v: want a splash off the log's end", ev)
	}
}

// TestWindAddsToTheCurrent: while a push drifts the hero, the current
// still slides it.
func TestWindAddsToTheCurrent(t *testing.T) {
	w := currentWorld(4, 7)
	windy(w, 1, 1000)
	x0 := w.Player.X
	run(w, None, 1) // the push begins: the hero is still before it
	run(w, None, config.HopTicks)
	if got, want := w.Player.X-x0, cellUnits+(config.HopTicks+1)*4; got != want {
		t.Errorf("moved %d units, want a cell and the slide: %d", got, want)
	}
}

// TestWindNeverSlides: in the rain a push onto the road doesn't slide, and
// the push is neither a hop for the combo nor breaks it.
func TestWindNeverSlides(t *testing.T) {
	w := wetWorld()
	w.Level.Combo = true
	w.Player = newPlayer(config.StartCol, 10)
	w.furthest = 10
	w.Combo = Combo{Hops: 2, Left: config.ComboTicks}
	w.Spells = append(w.Spells, Spell{Kind: config.StormFront, Timing: config.EventTiming{QuietMin: 1, QuietMax: 1, Last: 1 << 20, Beat: 1000},
		Phase: SpellOn, Length: 1 << 20, Left: 1<<20 - 1000 + 1, Dir: 1})
	ev := run(w, None, config.HopTicks+2)
	if count(ev, Pushed) != 1 || slices.Contains(ev, Slid) {
		t.Errorf("events %v: want a push and no slide", ev)
	}
	if w.Combo.Hops != 2 {
		t.Errorf("combo %d hops after a push, want 2", w.Combo.Hops)
	}
}

// stormWorld is level n of Storm from seed.
func stormWorld(n int, seed uint64) *World {
	return New(session.New(config.Storm, config.StartLives), config.Storm.Levels.Level(n), rand.New(rand.NewPCG(seed, seed)))
}

// keep keeps the hero going on w for a long run: a full clock and lives.
func keep(w *World) {
	w.TicksLeft = w.Level.TimerTicks
	w.Session.Lives = config.StartLives
}

// stormLog is one storm as it ran: when it was announced, began and
// rolled away, its kind, and its pushes and flood.
type stormLog struct {
	warned, started, ended int64
	wet                    bool
	pushes                 int
	floodAt                int64 // when its flood came in; 0 for none
	floodEnds              int
}

// runStorms runs w for ticks, keeping the hero going, and logs each storm.
// Rain must fall all through each wet storm.
func runStorms(t *testing.T, w *World, ticks int) []stormLog {
	t.Helper()
	var out []stormLog
	for range ticks {
		ev := w.Update(None)
		keep(w)
		for _, e := range ev {
			switch e {
			case WetStormWarning, DryStormWarning:
				out = append(out, stormLog{warned: w.Tick, wet: e == WetStormWarning})
			case StormStart:
				out[len(out)-1].started = w.Tick
			case StormEnd:
				out[len(out)-1].ended = w.Tick
			case Pushed:
				if len(out) > 0 {
					out[len(out)-1].pushes++
				}
			case FloodStart:
				out[len(out)-1].floodAt = w.Tick
			case FloodEnd:
				out[len(out)-1].floodEnds++
			}
		}
		if s := w.Spell(config.StormFront); s.Phase == SpellOn && s.Wet && !w.Raining() {
			t.Fatalf("tick %d: a wet storm with no rain", w.Tick)
		}
	}
	return out
}

// TestStormFrontRuns: storms come after a clear spell, are announced with
// their kind, hold for their seeded length and roll away. A wet storm
// rains all through, and from level 3 floods the verge once, a little
// after its rain sets in; a dry storm pushes on its beat and floods
// nothing. The same seed plays the same storms (D51).
func TestStormFrontRuns(t *testing.T) {
	for _, n := range []int{2, 3, config.EventTopLevel} {
		st := config.Storm.Levels.Level(n).Events[config.StormFront]
		for seed := uint64(1); seed <= 3; seed++ {
			logs := runStorms(t, stormWorld(n, seed), 6*60*config.TPS)
			if len(logs) < 4 {
				t.Fatalf("level %d seed %d: %d storms in six minutes", n, seed, len(logs))
			}
			if int(logs[0].warned) < st.QuietMin {
				t.Errorf("level %d seed %d: the first storm warned at %d, inside the first clear spell", n, seed, logs[0].warned)
			}
			for i, l := range logs {
				if l.ended == 0 {
					continue // still on when the run ended
				}
				if got := l.started - l.warned; got != int64(config.Events[config.StormFront].WarnTicks) {
					t.Errorf("level %d seed %d storm %d: warned %d ticks ahead", n, seed, i, got)
				}
				if got := int(l.ended - l.started); got < st.Last || got > st.LastMax {
					t.Errorf("level %d seed %d storm %d: held %d ticks, want %d–%d", n, seed, i, got, st.Last, st.LastMax)
				}
				switch {
				case l.wet && l.pushes > 0, !l.wet && l.floodAt > 0:
					t.Errorf("level %d seed %d storm %d: wet %v with %d pushes and a flood at %d", n, seed, i, l.wet, l.pushes, l.floodAt)
				case l.wet && n >= 3 && (l.floodAt-l.started != config.StormFloodDelay || l.floodEnds != 1):
					t.Errorf("level %d seed %d storm %d: a wet storm flooding %d ticks in, draining %d times", n, seed, i, l.floodAt-l.started, l.floodEnds)
				case l.wet && n < 3 && l.floodAt > 0:
					t.Errorf("level %d seed %d storm %d: a flood before level 3", n, seed, i)
				case !l.wet && l.pushes == 0:
					t.Errorf("level %d seed %d storm %d: a dry storm that never pushed", n, seed, i)
				}
			}
			if again := runStorms(t, stormWorld(n, seed), 6*60*config.TPS); !slices.Equal(logs, again) {
				t.Errorf("level %d seed %d: the same seed played different storms", n, seed)
			}
		}
	}
}

// TestStormKinds: level 1's storms are all wet; level 2's first is dry and
// level 3's wet; then a seeded mix of both, never three of a kind in a
// row, with dry storms blowing both ways. A testing kind makes every storm
// that kind (D51, D52).
func TestStormKinds(t *testing.T) {
	kinds := func(lv config.Level, seed uint64) (wet []bool, dirs map[int]bool) {
		w := New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
		s := w.Spell(config.StormFront)
		dirs = map[int]bool{}
		for range 60 {
			wet = append(wet, s.Wet)
			if !s.Wet {
				dirs[s.Dir] = true
			}
			w.nextStorm(s)
		}
		return wet, dirs
	}
	for seed := uint64(1); seed <= 8; seed++ {
		if wet, _ := kinds(config.Storm.Levels.Level(1), seed); slices.Contains(wet, false) {
			t.Errorf("seed %d: a dry storm on level 1", seed)
		}
		for n, first := range map[int]bool{2: false, 3: true} {
			if wet, _ := kinds(config.Storm.Levels.Level(n), seed); wet[0] != first {
				t.Errorf("seed %d: level %d's first storm wet %v, want %v", seed, n, wet[0], first)
			}
		}
		wet, dirs := kinds(config.Storm.Levels.Level(config.EventTopLevel), seed)
		for i := 2; i < len(wet); i++ {
			if wet[i] == wet[i-1] && wet[i] == wet[i-2] {
				t.Fatalf("seed %d: three storms in a row wet %v: %v", seed, wet[i], wet)
			}
		}
		if !slices.Contains(wet, true) || !slices.Contains(wet, false) || len(dirs) != 2 {
			t.Errorf("seed %d: storms %v, dry ones blowing %v", seed, wet, dirs)
		}
		for _, k := range []config.StormKind{config.WetStorm, config.DryStorm} {
			for _, n := range []int{1, 2, config.EventTopLevel} {
				lv := config.Storm.Levels.Level(n)
				lv.Storms = k
				if wet, _ := kinds(lv, seed); slices.Contains(wet, k == config.DryStorm) {
					t.Errorf("seed %d level %d: storms %v, want them all kind %d", seed, n, wet, k)
				}
			}
		}
	}
}

// TestWeatherAroundStorms: plain rain never comes in a storm and fog never
// in a wet one; fog in a dry storm is only announced once the storm is on;
// no weather is announced in the clear stretch after a storm; and the
// weather still comes (D51).
func TestWeatherAroundStorms(t *testing.T) {
	lv := config.Storm.Levels.Level(config.EventTopLevel)
	seen := map[config.EventKind]bool{}
	for seed := uint64(1); seed <= 4; seed++ {
		w := New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
		st := w.Spell(config.StormFront)
		cleared := int64(-1 << 40)
		for range 8 * 60 * config.TPS {
			ev := w.Update(None)
			keep(w)
			if slices.Contains(ev, StormEnd) {
				cleared = w.Tick
			}
			for _, e := range ev {
				if e != FogWarning && e != RainWarning {
					continue
				}
				if w.Tick-cleared < config.StormClearMin {
					t.Fatalf("seed %d tick %d: weather %d announced %d ticks after a storm", seed, w.Tick, e, w.Tick-cleared)
				}
				if st.Phase != SpellQuiet && (e == RainWarning || st.Wet || st.Phase != SpellOn) {
					t.Fatalf("seed %d tick %d: weather %d announced in a storm, wet %v, phase %d", seed, w.Tick, e, st.Wet, st.Phase)
				}
			}
			rain, fog := w.Spell(config.Rain), w.Spell(config.Fog)
			if st.Phase != SpellQuiet && (!st.Wet && rain.Phase != SpellQuiet || st.Wet && fog.Phase != SpellQuiet) {
				t.Fatalf("seed %d tick %d: rain %d and fog %d in a storm, wet %v", seed, w.Tick, rain.Phase, fog.Phase, st.Wet)
			}
			for _, s := range []*Spell{rain, fog} {
				if s.Phase == SpellOn {
					seen[s.Kind] = true
				}
			}
		}
	}
	if !seen[config.Rain] || !seen[config.Fog] {
		t.Errorf("weathers seen: %v", seen)
	}
}

// TestNextPush: the tell knows when the next push comes.
func TestNextPush(t *testing.T) {
	s := Spell{Kind: config.StormFront, Timing: config.EventTiming{Beat: 180}, Length: 600, Phase: SpellWarning, Left: 120}
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
	if s.Wet = true; s.NextPush() != -1 {
		t.Error("a wet storm pushes")
	}
}

// TestOneBigEventAtATime: a rush hour wave and a storm never overlap, and
// each still comes (D48, D51).
func TestOneBigEventAtATime(t *testing.T) {
	lv := config.Storm.Levels.Level(config.EventTopLevel)
	for seed := uint64(1); seed <= 4; seed++ {
		w := New(session.New(config.Storm, config.StartLives), lv, rand.New(rand.NewPCG(seed, seed)))
		came := map[Event]int{}
		for range 4 * 60 * config.TPS {
			for _, e := range w.Update(None) {
				came[e]++
			}
			keep(w)
			if w.Spell(config.Wave).Phase != SpellQuiet && w.Spell(config.StormFront).Phase != SpellQuiet {
				t.Fatalf("seed %d tick %d: a wave in a storm", seed, w.Tick)
			}
		}
		for _, e := range []Event{WaveStart, StormStart} {
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

// floodWorld is an all-safe level with a wet storm forced on, its flood
// over verge columns 10–12 just coming in, with the hero on the verge at
// col.
func floodWorld(col int) *World {
	w := newWorld(1)
	s := Spell{Kind: config.StormFront, Wet: true, Timing: config.Storm.Levels.Level(3).Events[config.StormFront], Phase: SpellOn}
	s.Length = s.Timing.Last
	s.Left = s.Length - config.StormFloodDelay
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
	ev := run(w, None, w.Spells[0].Left+1)
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
