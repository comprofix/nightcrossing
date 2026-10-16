package play

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/spritegrid"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestSprites checks the sprite source parses and packs, and has every
// sprite the play scene draws, at the size it is drawn.
func TestSprites(t *testing.T) {
	f, err := os.Open("../../assets/sprites/" + Sheet + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sprites, err := spritegrid.Parse(f, resurrect64.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := spritegrid.Pack(sprites, resurrect64.Palette); err != nil {
		t.Fatal(err)
	}
	size := map[string][2]int{}
	frames := map[string]int{}
	for _, s := range sprites {
		size[s.Name] = [2]int{s.W, s.H}
		frames[s.Name] = len(s.Frames)
	}
	want := map[string][2]int{
		Toad.Sprite: {config.Cell, config.Cell},
		"verge":     {config.Cell, config.Cell},
		"life":      {12, 12},
		"track":     {config.Cell, config.Cell},
		Toad.Splash: {config.Cell, config.Cell},
		"pad":       {config.Cell, config.Cell},
	}
	for name, wh := range want {
		got, ok := size[name]
		switch {
		case !ok:
			t.Errorf("no sprite named %q", name)
		case got != wh:
			t.Errorf("sprite %q is %d×%d, want %d×%d", name, got[0], got[1], wh[0], wh[1])
		}
	}
	if n := frames[Toad.Sprite]; n != frameTuck+1 {
		t.Errorf("the hero sprite has %d frames, want %d", n, frameTuck+1)
	}
	if n := frames["pad"]; n != frameRing+1 {
		t.Errorf("the lily pad has %d frames, want %d", n, frameRing+1)
	}
	// A sinking ship needs every piece in each of its sinking looks.
	for _, s := range config.Ships() {
		for _, name := range entitySprites(s) {
			for _, look := range []string{"_flash", "_low", "_deep"} {
				if size[name+look] != size[name] {
					t.Errorf("%s: no sprite %q the size of %q", s.ID, name+look, name)
				}
			}
		}
	}

	// Each entity's pieces are a lane high and add up to its length.
	for _, v := range config.Entities() {
		width := 0
		for _, name := range entitySprites(v) {
			wh, ok := size[name]
			if !ok {
				t.Errorf("%s: no sprite named %q", v.ID, name)
				continue
			}
			if wh[1] != config.Cell {
				t.Errorf("%s: sprite %q is %dpx high, want %d", v.ID, name, wh[1], config.Cell)
			}
			width += wh[0]
		}
		if width != v.Length {
			t.Errorf("%s: sprites are %dpx long, the entity %d", v.ID, width, v.Length)
		}
	}
}

func TestNewGame(t *testing.T) {
	for _, sk := range config.Modes {
		h := scenetest.New(t, 1)
		p := New(h.Ctx, sk).(*Play)
		if s := h.Ctx.Session; s.Mode.ID != sk.ID || s.Lives != config.StartLives || s.Score != 0 {
			t.Errorf("%s: session %+v", sk.ID, s)
		}
		if got, want := p.World().TicksLeft, config.TimerTicks(1); got != want {
			t.Errorf("%s: %d ticks on the clock, want %d", sk.ID, got, want)
		}
	}
}

// summary is the game's state as text, for the golden file.
func summary(p *Play, ctx *scene.Context) string {
	w, s := p.World(), ctx.Session
	var b strings.Builder
	fmt.Fprintf(&b, "grid %dx%d  mode %s  level %d\n", w.Cols, w.Rows, s.Mode.ID, w.Level.Number)
	fmt.Fprintf(&b, "tick %d  ticks left %d\n", w.Tick, w.TicksLeft)
	fmt.Fprintf(&b, "score %d  lives %d\n", s.Score, s.Lives)
	pl := w.Player
	fmt.Fprintf(&b, "hero cell %d,%d  at %d,%d  facing %v  hopping %v\n", pl.Col, pl.Row, pl.X, pl.Y, pl.Facing, pl.Hopping())
	return b.String()
}

// TestHopRoute plays a route of single presses through the real input,
// some spaced out and some pressed just before landing, and checks where
// the hero ends up against the golden file. The level is all safe, so
// only the input and hopping count.
func TestHopRoute(t *testing.T) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.Modes[0], config.StartLives)
	p := newPlay(h.Ctx, config.StrollLevels, config.Level{Number: 1, TimerTicks: config.TimerTicks(1)})
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	hop := func(k ebiten.Key, gap int) {
		h.Run(scenetest.Hold(1, k), scenetest.Hold(gap))
	}
	// Spaced out: each hop lands before the next press.
	hop(ebiten.KeyArrowUp, config.HopTicks+2)
	hop(ebiten.KeyW, config.HopTicks+2)
	hop(ebiten.KeyArrowLeft, config.HopTicks+2)
	hop(ebiten.KeyArrowLeft, config.HopTicks+2)
	// Quick: each press lands in the last ticks of the hop before it.
	for range 3 {
		hop(ebiten.KeyArrowUp, config.HopTicks-2)
	}
	hop(ebiten.KeyD, config.HopTicks+2)
	// And one back down. Route: up 2, left 2, up 3, right 1, down 1.
	hop(ebiten.KeyArrowDown, 2*config.HopTicks)

	scenetest.Golden(t, "hop_route", summary(p, h.Ctx))
}

func TestFacingKeepsTheSpriteInItsBox(t *testing.T) {
	const w, h = 24, 24
	// The sprite's top middle is the way it faces.
	cases := []struct {
		d      world.Dir
		tx, ty float64 // where the top middle should end up
	}{
		{world.Up, w / 2, 0},
		{world.Right, w, h / 2},
		{world.Down, w / 2, h},
		{world.Left, 0, h / 2},
	}
	for _, c := range cases {
		g := facing(c.d, w, h)
		if x, y := g.Apply(w/2, 0); x != c.tx || y != c.ty {
			t.Errorf("%v: top middle goes to %v,%v, want %v,%v", c.d, x, y, c.tx, c.ty)
		}
		for _, corner := range [][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
			x, y := g.Apply(corner[0], corner[1])
			if x < 0 || x > w || y < 0 || y > h || x != float64(int(x)) || y != float64(int(y)) {
				t.Errorf("%v: corner %v goes to %v,%v, outside the box or between pixels", c.d, corner, x, y)
			}
		}
	}
}

// TestMenuToPlayAndPause goes from the main menu to the mode select and
// the intro, plays a little, pauses, waits and resumes. The world must
// not run while paused.
func TestMenuToPlayAndPause(t *testing.T) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay = New
	h.Start(scene.NewMainMenu())
	h.Ticks(scene.FadeTicks)

	h.Press(ebiten.KeyEnter) // Play
	scenetest.Current[*scene.ModeSelect](h)
	h.Press(ebiten.KeyEnter) // Stroll, the only mode so far
	scenetest.Current[*scene.Intro](h)
	h.Press(ebiten.KeyEnter) // play it
	p := scenetest.Current[*Play](h)
	if h.Ctx.Session.Mode.ID != config.Modes[0].ID {
		t.Fatalf("started at %s, want %s", h.Ctx.Session.Mode.ID, config.Modes[0].ID)
	}

	h.Ticks(60)
	h.Press(ebiten.KeyEscape)
	scenetest.Current[*scene.Pause](h)
	before := p.World().Tick
	h.Ticks(120)
	if p.World().Tick != before {
		t.Errorf("the world ran %d ticks while paused", p.World().Tick-before)
	}
	h.Press(ebiten.KeyEscape)
	scenetest.Current[*Play](h)
	h.Ticks(30)

	scenetest.Golden(t, "menu_to_play", summary(p, h.Ctx))
}

// TestRestart starts a fresh game in the same mode.
func TestRestart(t *testing.T) {
	h := scenetest.New(t, 7)
	h.Ctx.NewPlay = New
	first := New(h.Ctx, config.Modes[0]).(*Play)
	h.Start(first)
	h.Ticks(scene.FadeTicks + 30)
	h.Press(ebiten.KeyControl, ebiten.KeyR)
	p := scenetest.Current[*Play](h)
	if p == first {
		t.Fatal("restart kept the old game")
	}
	if h.Ctx.Session.Mode.ID != config.Modes[0].ID || p.World().Level.Number != 1 {
		t.Errorf("restarted at %s level %d, want %s level 1", h.Ctx.Session.Mode.ID, p.World().Level.Number, config.Modes[0].ID)
	}
}

// TestRoadReplay plays level 1 from a fixed seed as a
// reckless player who crosses the road to the verge and back, again and
// again, pressing every few ticks and never looking. It records every
// crossing and death, then checks the game ends, shows GAME OVER and goes
// on to the high scores, and that it all matches the golden file. A
// change to the lanes, collisions or deaths shows up here; if it is
// intended, run the test with -update.
func TestRoadReplay(t *testing.T) {
	const (
		pressEvery = 12
		verge      = 6 // the safe row above level 1's road
	)
	for _, sk := range config.Modes {
		t.Run(sk.ID, func(t *testing.T) {
			h := scenetest.New(t, 42)
			h.Ctx.NewPlay = New
			p := New(h.Ctx, sk).(*Play)
			h.Start(p)
			h.Ticks(scene.FadeTicks)

			var log strings.Builder
			w := p.World()
			key := ebiten.KeyArrowUp
			for i := 0; !w.Over; i++ {
				if i > 300*config.TPS {
					t.Fatal("the game did not end")
				}
				row, lives := w.Player.LaneRow(), h.Ctx.Session.Lives
				switch {
				case key == ebiten.KeyArrowUp && w.Player.Row == verge && !w.Player.Hopping():
					key = ebiten.KeyArrowDown
					fmt.Fprintf(&log, "tick %d: crossed up\n", w.Tick)
				case key == ebiten.KeyArrowDown && w.Player.Row == config.StartRow && !w.Player.Hopping():
					key = ebiten.KeyArrowUp
					fmt.Fprintf(&log, "tick %d: crossed down\n", w.Tick)
				}
				if i%pressEvery == 0 {
					h.Run(scenetest.Hold(1, key))
				} else {
					h.Ticks(1)
				}
				if h.Ctx.Session.Lives < lives {
					key = ebiten.KeyArrowUp // back to the start row
					fmt.Fprintf(&log, "tick %d: died in row %d (was in %d), %d lives left\n", w.Tick, w.Player.LaneRow(), row, h.Ctx.Session.Lives)
				}
			}
			scenetest.Current[*Ended](h)
			h.Ticks(gameOverTicks + 2*scene.FadeTicks)
			if h.Ctx.Session.Score > 0 {
				scenetest.Current[*scene.HighScoreEntry](h) // the first score on an empty table
			} else {
				scenetest.Current[*scene.HighScoreTable](h)
			}

			scenetest.Golden(t, "road_"+sk.ID, log.String()+summary(p, h.Ctx))
		})
	}
}

// TestLevelClear fills the last nook of an all-safe level with a bank at
// the top, then checks the play scene shows LEVEL CLEAR, goes to the tally
// with the level's points, and on to the next level of the mode.
func TestLevelClear(t *testing.T) {
	h := scenetest.New(t, 3)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.Modes[0], config.StartLives)
	lv := config.Level{Number: 1, TimerTicks: config.TimerTicks(1)}
	lv.Lanes[config.GoalRow].Kind = config.Bank
	p := newPlay(h.Ctx, config.StrollLevels, lv)
	n := world.NookAt(config.StartCol)
	if n < 0 {
		t.Fatalf("the start column %d is not under a nook", config.StartCol)
	}
	for i := range p.World().Filled {
		p.World().Filled[i] = i != n
	}
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	for range config.StartRow - config.GoalRow {
		h.Run(scenetest.Hold(1, ebiten.KeyArrowUp), scenetest.Hold(config.HopTicks+1))
	}
	if !p.World().Cleared {
		t.Fatalf("not cleared: hero at row %d, filled %v", p.World().Player.Row, p.World().Filled)
	}
	h.Ticks(clearTicks + 1 + scene.FadeTicks)
	scenetest.Current[*scene.Tally](h)
	st := p.World().Stats
	want := st.Hops + st.Goals*config.PointsGoal + st.TimeBonus + st.Clear
	if st.Goals != 1 || st.Clear != config.PointsAllGoals || h.Ctx.Session.Score != want {
		t.Errorf("stats %+v, score %d, want %d", st, h.Ctx.Session.Score, want)
	}

	h.Ticks(scene.FadeTicks)
	h.Press(ebiten.KeyEnter) // finish the count
	h.Press(ebiten.KeyEnter) // go on
	h.Ticks(scene.FadeTicks)
	next := scenetest.Current[*Play](h)
	if got := next.World().Level.Number; got != 2 {
		t.Errorf("went on to level %d, want 2", got)
	}
	if next.World().Level.TimerTicks != config.TimerTicks(2) {
		t.Errorf("level 2 timer %d, want %d", next.World().Level.TimerTicks, config.TimerTicks(2))
	}
}

// TestClearingLevel8ClearsTheMode checks clearing a mode's last level marks
// the mode cleared and asks for the scores to be saved, and an earlier
// level does not.
func TestClearingLevel8ClearsTheMode(t *testing.T) {
	for _, n := range []int{7, 8} {
		h := scenetest.New(t, 3)
		h.Ctx.NewPlay = New
		h.Ctx.Session = session.New(config.Modes[0], config.StartLives)
		lv := config.Level{Number: n, TimerTicks: config.TimerTicks(n)}
		lv.Lanes[config.GoalRow].Kind = config.Bank
		p := newPlay(h.Ctx, config.Modes[0].Levels, lv)
		nook := world.NookAt(config.StartCol)
		for i := range p.World().Filled {
			p.World().Filled[i] = i != nook
		}
		h.Start(p)
		h.Ticks(scene.FadeTicks)
		for range config.StartRow - config.GoalRow {
			h.Run(scenetest.Hold(1, ebiten.KeyArrowUp), scenetest.Hold(config.HopTicks+1))
		}
		want := n == 8
		if got := h.Ctx.Scores.IsCleared(config.Modes[0].ID); got != want || h.Ctx.TakeScoresChanged() != want {
			t.Errorf("level %d cleared: mode cleared %v, want %v", n, got, want)
		}
	}
}

// TestTitleDemo runs the title over the demo: the demo must leave the
// game's RNG and the game in progress alone, and play the same every time.
func TestTitleDemo(t *testing.T) {
	run := func(demo bool) (uint64, *Demo) {
		h := scenetest.New(t, 42)
		if demo {
			h.Ctx.NewDemo = NewDemo
		}
		title := scene.NewTitle()
		h.Start(title)
		h.Ticks(20 * config.TPS)
		if h.Ctx.Session != nil {
			t.Errorf("the demo started a game")
		}
		d, _ := title.Demo().(*Demo)
		return h.Ctx.RNG.Uint64(), d
	}
	r1, d1 := run(true)
	_, d2 := run(true)
	r0, _ := run(false)
	if r1 != r0 {
		t.Errorf("the demo used the game's RNG")
	}
	if d1 == nil || d2 == nil {
		t.Fatalf("no demo ran")
	}
	w1, w2 := d1.World(), d2.World()
	if w1.Tick != w2.Tick || w1.Player != w2.Player || w1.Filled != w2.Filled {
		t.Errorf("two demos differ")
	}
	if w1.Filled == [len(config.Nooks)]bool{} {
		t.Errorf("no toad home after 20 s of the demo")
	}
}

// TestGamepadOnly goes through every screen of a game with a gamepad alone:
// title, menus, the intro and back, play, pause, options and the high
// scores.
func TestGamepadOnly(t *testing.T) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay, h.Ctx.NewDemo = New, NewDemo
	h.Start(scene.NewTitle())
	h.Ticks(scene.FadeTicks)
	pad := func(b input.Button) {
		t.Helper()
		h.Run(input.Step{Buttons: []input.Button{b}, Ticks: 1}, input.Idle(1))
		h.Ticks(2 * scene.FadeTicks)
	}

	pad(input.PadA) // any button
	scenetest.Current[*scene.MainMenu](h)
	pad(input.PadA) // Play
	scenetest.Current[*scene.ModeSelect](h)
	pad(input.PadA)
	scenetest.Current[*scene.Intro](h)
	pad(input.PadB)
	scenetest.Current[*scene.ModeSelect](h)
	pad(input.PadA)
	pad(input.PadA) // play
	p := scenetest.Current[*Play](h)
	h.Ticks(bannerTicks)
	pad(input.PadUp)
	if row := p.World().Player.Row; row != config.StartRow-1 {
		t.Fatalf("hero on row %d after Up, want %d", row, config.StartRow-1)
	}

	pad(input.PadStart)
	scenetest.Current[*scene.Pause](h)
	pad(input.PadB) // resume
	scenetest.Current[*Play](h)
	pad(input.PadStart)
	pad(input.PadDown) // Options
	pad(input.PadA)
	scenetest.Current[*scene.Options](h)
	pad(input.PadB)
	scenetest.Current[*scene.Pause](h)
	pad(input.PadDown) // the cursor is still on Options: Quit to menu
	pad(input.PadA)
	scenetest.Current[*scene.MainMenu](h)

	pad(input.PadDown) // High scores
	pad(input.PadDown)
	pad(input.PadA)
	scenetest.Current[*scene.HighScoreTable](h)
	pad(input.PadB)
	scenetest.Current[*scene.MainMenu](h)
	pad(input.PadB)
	scenetest.Current[*scene.Title](h)
}

// TestStartLevel starts a test game part way (-level in debug builds): it
// starts on that level and clears no mode.
func TestStartLevel(t *testing.T) {
	h := scenetest.New(t, 42)
	h.Ctx.StartLevel = 38
	p := New(h.Ctx, config.Stroll).(*Play)
	if n := p.World().Level.Number; n != 38 {
		t.Fatalf("started on level %d, want 38", n)
	}
	p.markCleared(h.Ctx)
	if h.Ctx.Scores.IsCleared(config.Stroll.ID) {
		t.Error("a test game cleared the mode")
	}
}
