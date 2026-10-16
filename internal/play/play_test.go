package play

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
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
	for _, s := range sprites {
		size[s.Name] = [2]int{s.W, s.H}
	}
	want := map[string][2]int{
		"toad":  {config.Cell, config.Cell},
		"verge": {config.Cell, config.Cell},
		"life":  {12, 12},
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
}

func TestNewGame(t *testing.T) {
	for _, sk := range config.Skills {
		h := scenetest.New(t, 1)
		p := New(h.Ctx, sk).(*Play)
		if s := h.Ctx.Session; s.Skill.ID != sk.ID || s.Lives != config.StartLives || s.Score != 0 {
			t.Errorf("%s: session %+v", sk.ID, s)
		}
		if got, want := p.World().TicksLeft, world.AttemptTicks(sk); got != want {
			t.Errorf("%s: %d ticks on the clock, want %d", sk.ID, got, want)
		}
	}
}

// summary is the game's state as text, for the golden file.
func summary(p *Play, ctx *scene.Context) string {
	w, s := p.World(), ctx.Session
	var b strings.Builder
	fmt.Fprintf(&b, "grid %dx%d  skill %s  level %d\n", w.Cols, w.Rows, w.Skill.ID, p.level)
	fmt.Fprintf(&b, "tick %d  ticks left %d\n", w.Tick, w.TicksLeft)
	fmt.Fprintf(&b, "score %d  lives %d\n", s.Score, s.Lives)
	pl := w.Player
	fmt.Fprintf(&b, "hero cell %d,%d  at %d,%d  facing %v  hopping %v\n", pl.Col, pl.Row, pl.X, pl.Y, pl.Facing, pl.Hopping())
	return b.String()
}

// TestHopRoute plays a route of single presses through the real input,
// some spaced out and some pressed just before landing, and checks where
// the hero ends up against the golden file.
func TestHopRoute(t *testing.T) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay = New
	p := New(h.Ctx, config.Skills[1]).(*Play)
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

// TestMenuToPlayAndPause goes from the main menu to the second skill, plays
// a little, pauses, waits and resumes. The world must not run while paused.
func TestMenuToPlayAndPause(t *testing.T) {
	h := scenetest.New(t, 42)
	h.Ctx.NewPlay = New
	h.Start(scene.NewMainMenu())
	h.Ticks(scene.FadeTicks)

	h.Press(ebiten.KeyEnter) // Play
	scenetest.Current[*scene.SkillSelect](h)
	h.Press(ebiten.KeyArrowDown)
	h.Press(ebiten.KeyEnter) // Dash
	p := scenetest.Current[*Play](h)
	if p.World().Skill.ID != config.Skills[1].ID {
		t.Fatalf("started at %s, want %s", p.World().Skill.ID, config.Skills[1].ID)
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

// TestRestart starts a fresh game at the same skill.
func TestRestart(t *testing.T) {
	h := scenetest.New(t, 7)
	h.Ctx.NewPlay = New
	first := New(h.Ctx, config.Skills[2]).(*Play)
	h.Start(first)
	h.Ticks(scene.FadeTicks + 30)
	h.Press(ebiten.KeyControl, ebiten.KeyR)
	p := scenetest.Current[*Play](h)
	if p == first {
		t.Fatal("restart kept the old game")
	}
	if p.World().Skill.ID != config.Skills[2].ID {
		t.Errorf("restarted at %s, want %s", p.World().Skill.ID, config.Skills[2].ID)
	}
}
