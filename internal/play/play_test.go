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
	return b.String()
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
