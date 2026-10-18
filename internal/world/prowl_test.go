package world

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

// ratWorld is riverWorld with two still long logs flowing dir, one from
// cell 4 and one from cell 15, and water rats on.
func ratWorld(dir int) *World {
	w := riverWorld(config.LogLong, false, 4*cellUnits, dir, 0)
	w.Level.Rats = true
	lane := &w.Lanes[riverRow]
	lane.Entities = append(lane.Entities, Entity{Kind: config.LogLong, U: 15*cellUnits + Buffer})
	for i := range lane.Entities {
		lane.Entities[i].log = true
	}
	return w
}

// TestRatSwims: a water rat comes up with a ripple off its log's front
// end, harmless, then swims smoothly for it against the flow, harmless
// in the water, until its nose touches the end; it grabs the end for
// config.RatGrabTicks, and dives (D69).
func TestRatSwims(t *testing.T) {
	for _, dir := range []int{1, -1} {
		w := ratWorld(dir)
		w.Rats[0] = Rat{Phase: RatRipple, Row: riverRow, Target: 0, ticks: config.RippleTicks}
		e := w.Lanes[riverRow].Entities[0]
		front := e.X()
		if dir > 0 {
			front += e.Kind.Length * config.SubPx
		}
		nose, facing := w.RatNose(w.Rats[0])
		if nose != front+dir*config.RatStartCells*cellUnits || facing != -dir {
			t.Fatalf("dir %d: nose at %d facing %d, want %d facing %d", dir, nose, facing, front+dir*config.RatStartCells*cellUnits, -dir)
		}
		for range config.RippleTicks - 1 {
			w.Update(None)
			if w.Rats[0].Phase != RatRipple || w.ratBites(riverRow, 0, fieldWidth) {
				t.Fatalf("dir %d: phase %v, bites %v: want a harmless ripple", dir, w.Rats[0].Phase, w.ratBites(riverRow, 0, fieldWidth))
			}
		}
		step := (config.RatStartCells*cellUnits + config.RatSwimTicks - 1) / config.RatSwimTicks
		var ev []Event
		for range config.RatSwimTicks + 1 {
			ev = append(ev, w.Update(None)...)
			n, _ := w.RatNose(w.Rats[0])
			if d := (nose - n) * dir; d < 0 || d > step {
				t.Fatalf("dir %d: the nose moved %d units in a tick, want 0–%d", dir, d, step)
			}
			if w.Rats[0].Phase == RatSwim && w.ratBites(riverRow, 0, fieldWidth) {
				t.Fatalf("dir %d: a swimming rat bites", dir)
			}
			nose = n
		}
		if !slices.Contains(ev, RatGrabs) || w.Rats[0].Phase != RatGrab || nose != front {
			t.Fatalf("dir %d: events %v, rat %+v, nose %d: want it grabbing the log's front at %d", dir, ev, w.Rats[0], nose, front)
		}
		over := config.RatOverPx * config.SubPx
		if hl, hr := w.RatHead(w.Rats[0]); min(hl, hr) != min(front, front-dir*over) || hr-hl != over {
			t.Errorf("dir %d: head from %d to %d, want %d units onto the log from %d", dir, hl, hr, over, front)
		}
		if ev := run(w, None, config.RatGrabTicks); !slices.Contains(ev, RatDives) || w.Rats[0].Phase != RatAway {
			t.Fatalf("dir %d: events %v, rat %+v: want it dived", dir, ev, w.Rats[0])
		}
	}
}

// TestRatNipsAtTheEnd: a water rat catches a hero waiting on its log's
// front cell, only once it grabs the log's end, and not one a cell
// further in (D69).
func TestRatNipsAtTheEnd(t *testing.T) {
	for _, c := range []struct {
		dir, cell int
		nipped    bool
	}{{1, 8, true}, {1, 7, false}, {-1, 4, true}, {-1, 5, false}} {
		w := ratWorld(c.dir)
		standOn(w, c.cell*cellUnits)
		w.Rats[0] = Rat{Phase: RatRipple, Row: riverRow, Target: 0, ticks: config.RippleTicks}
		var ev []Event
		for range config.RippleTicks + config.RatSwimTicks + config.RatGrabTicks {
			now := w.Update(None)
			if slices.Contains(now, Nipped) && w.Rats[0].Phase != RatGrab {
				t.Errorf("dir %d, cell %d: nipped by a rat %+v, not grabbing the log", c.dir, c.cell, w.Rats[0])
			}
			ev = append(ev, now...)
		}
		if got := slices.Contains(ev, Nipped); got != c.nipped {
			t.Errorf("dir %d, cell %d: nipped %v, want %v (events %v)", c.dir, c.cell, got, c.nipped, ev)
		}
	}
}

// TestRatAims: a water rat swims only for a log with no heron or a ship
// that never sinks, at least three cells long, with open water off its
// front end, on the field from its ripple to its dive, and never for the
// only platform on the field (D69).
func TestRatAims(t *testing.T) {
	for _, c := range []struct {
		name string
		set  func(l *Lane)
		want bool
	}{
		{"a log", func(l *Lane) {}, true},
		{"a heron's log", func(l *Lane) { l.Entities[0].Heron = true }, false},
		{"lily pads", func(l *Lane) { l.Entities[0].Kind, l.Entities[0].Dives = config.Pads3, true }, false},
		{"lily pads that never dive", func(l *Lane) { l.Entities[0].Kind, l.Entities[0].log = config.Pads3, false }, false},
		{"a steady barge", func(l *Lane) { l.Entities[0].Kind = config.Barge }, true},
		{"a sinking barge", func(l *Lane) { l.Entities[0].Kind, l.Entities[0].Sinks = config.Barge, true }, false},
		{"a short platform", func(l *Lane) { l.Entities[0].Kind = config.Pads2 }, false},
		{"no water ahead", func(l *Lane) { l.Entities[1].U = 10*cellUnits + Buffer }, false},
		{"the only one on the field", func(l *Lane) { l.Entities[1].U = l.Loop - Buffer/2 }, false},
		{"near the edge", func(l *Lane) { l.Entities[0].U = 19*cellUnits + Buffer; l.Entities[1].U = 2*cellUnits + Buffer }, false},
	} {
		w := ratWorld(1)
		c.set(&w.Lanes[riverRow])
		if got := w.ratCanAim(riverRow, 0); got != c.want {
			t.Errorf("%s: can aim %v, want %v", c.name, got, c.want)
		}
	}
}

// keepAlive puts the hero back on its feet with time on the clock, so a
// level runs on.
func keepAlive(w *World) {
	if w.Dying > 0 || w.Home > 0 {
		w.newAttempt()
		w.Dying, w.Home = 0, 0
	}
	w.Player.Death, w.TicksLeft = 0, 1000
	w.Session.Lives = config.StartLives
}

// TestProwlOnTheLevels: on Last Train's levels 4 to 6, water rats come,
// the snake from level 5, one at a time and two at once from level 6,
// never with a rat for a heron's log, and none on level 3 (D66).
func TestProwlOnTheLevels(t *testing.T) {
	for n := 3; n <= 6; n++ {
		lv := config.LastTrain.Levels.Level(n)
		w := New(session.New(config.LastTrain, config.StartLives), lv, rand.New(rand.NewPCG(uint64(n), 7)))
		rats, snakes, most := 0, 0, 0
		for range 20 * 60 * config.TPS {
			for _, e := range w.Update(None) {
				switch e {
				case Ripple:
					rats++
				case GrassRustles:
					snakes++
				}
			}
			most = max(most, w.prowling())
			for _, r := range w.Rats {
				if r.Phase != RatAway && w.Lanes[r.Row].Entities[r.Target].Heron {
					t.Fatalf("level %d: a rat swims for a heron's log", n)
				}
			}
			keepAlive(w)
		}
		want := map[int]int{3: 0, 4: 1, 5: 1, 6: 2}[n]
		if most != want || rats == 0 != (n < config.RatFrom) || snakes == 0 != (n < config.SnakeFrom) {
			t.Errorf("level %d: %d rats, %d snakes, at most %d out at once, want %d", n, rats, snakes, most, want)
		}
	}
}

// snakeWorld is a test level with the snake on, and a quiet that is over.
func snakeWorld() *World {
	w := newWorld(1)
	w.Level.Rats, w.Level.Snake, w.Level.ProwlMin, w.Level.ProwlMax = true, true, 600, 600
	w.Tick, w.prowlIn = config.LevelCalmTicks, 1
	return w
}

// TestSnakeSlides: the grass rustles, harmless; then the snake slides
// smoothly back and forth along its stretch, turning at its ends with
// SnakeTurns, and slips away after its time (D70).
func TestSnakeSlides(t *testing.T) {
	w := snakeWorld()
	if ev := w.Update(None); !slices.Contains(ev, GrassRustles) || w.Snake.Phase != SnakeRustle {
		t.Fatalf("events %v, snake %+v: want the grass rustling", ev, w.Snake)
	}
	if w.snakeBites(0, fieldWidth) {
		t.Fatal("a rustle bites")
	}
	if ev := run(w, None, config.RustleTicks); !slices.Contains(ev, SnakeSlides) || !w.snakeBites(0, fieldWidth) {
		t.Fatalf("events %v, snake %+v: want it out", ev, w.Snake)
	}
	lo, hi := w.Snake.Col*cellUnits, (w.Snake.Col+config.SnakeStretch-config.SnakeCells)*cellUnits
	step := (cellUnits + config.SnakeCellTicks - 1) / config.SnakeCellTicks
	x, turns := w.Snake.X(), 0
	heading := w.Snake.Heading()
	out := 0
	for w.Snake.Phase == SnakeOut {
		out = w.Snake.T
		hissed := slices.Contains(w.Update(None), SnakeTurns)
		if w.Snake.Phase != SnakeOut {
			break
		}
		nx := w.Snake.X()
		if nx < lo || nx > hi || abs(nx-x) > step {
			t.Fatalf("snake at %d, from %d: want %d–%d, moving at most %d a tick", nx, x, lo, hi, step)
		}
		if h := w.Snake.Heading(); h != heading {
			turns++
			heading = h
			if !hissed {
				t.Errorf("the snake turned at %d without SnakeTurns", nx)
			}
		} else if hissed {
			t.Errorf("SnakeTurns at %d with no turn", nx)
		}
		x = nx
	}
	if out < config.SnakeOutMinTicks-1 || w.Snake.Phase != SnakeAway || turns < 1 {
		t.Errorf("snake %+v after %d turns: want it gone after its time", w.Snake, turns)
	}
}

// TestSnakeBites: the snake out on the verge kills a hero there, and not
// one beside its stretch (D70).
func TestSnakeBites(t *testing.T) {
	for _, c := range []struct {
		col    int
		bitten bool
	}{{10, true}, {13, true}, {16, false}, {9, false}} {
		w := snakeWorld()
		w.Snake = Snake{Phase: SnakeOut, Col: 10, Dir: 1, ticks: config.SnakeOutMinTicks}
		w.Player = newPlayer(c.col, config.VergeRow)
		ev := run(w, None, config.SnakeOutMinTicks-1)
		if got := slices.Contains(ev, Bitten); got != c.bitten {
			t.Errorf("column %d: bitten %v, want %v", c.col, got, c.bitten)
		}
	}
}

// TestSnakeAndFlood: no snake comes while a storm with a flood is warned
// or on, and a storm warned with the snake out brings no flood (D70).
func TestSnakeAndFlood(t *testing.T) {
	storm := Spell{Kind: config.StormFront, Wet: true, Phase: SpellWarning, Left: 1000}
	storm.Timing.Size = 6
	storm.Cells[3] = true

	w := snakeWorld()
	w.Spells = []Spell{storm}
	w.Update(None)
	if w.Snake.Phase != SnakeAway {
		t.Errorf("snake %+v with a flood coming", w.Snake)
	}

	w = snakeWorld()
	w.Update(None)
	if w.Snake.Phase != SnakeRustle {
		t.Fatalf("snake %+v: want it coming, with no flood", w.Snake)
	}
	s := storm
	w.warnStorm(&s)
	if s.Floods() {
		t.Error("a storm warned with the snake out floods the verge")
	}
}

// TestNoProwlWithoutTheLevel: Stroll's level 8 never has a rat or snake.
func TestNoProwlWithoutTheLevel(t *testing.T) {
	w := New(session.New(config.Stroll, 3), config.Stroll.Levels.Level(8), rand.New(rand.NewPCG(4, 4)))
	for range 5 * 60 * config.TPS {
		w.Update(None)
		if w.prowling() > 0 {
			t.Fatalf("rats %+v, snake %+v on Stroll", w.Rats, w.Snake)
		}
		keepAlive(w)
	}
}
