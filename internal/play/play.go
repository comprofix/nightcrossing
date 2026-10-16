// Package play is Night Crossing's play scene. It runs the world one tick at
// a time, draws its state with the status bar and hint strip, and turns the
// world's events into sound and effects. The rules live in package world.
package play

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/ui"
	"github.com/comprofix/nightcrossing/internal/world"
)

// Sheet is the name of the game's sprite sheet.
const Sheet = "nightcrossing"

// Status bar layout, in screen pixels from the left (DESIGN §3.2).
const (
	hudHiX     = 136
	hudLevelX  = 240
	hudLivesX  = 304
	hudLifeGap = 2
	timeBarW   = 120
	timeBarH   = 8
	labelGap   = 8 // between a label and what it labels
)

// hint is the help shown in the strip below the playfield.
const hint = "ARROWS/WASD: HOP   ESC: PAUSE   CTRL+R: RESTART"

// Play is a game in progress.
type Play struct {
	world *world.World
	hero  Hero
	level int
}

// Effects, in ticks and pixels.
const (
	shakeTicks    = 12
	shakePx       = 3
	gameOverTicks = 2 * config.TPS // the game over message shows this long
)

// New starts a game at a skill.
func New(ctx *scene.Context, skill config.Skill) scene.Scene {
	return newPlay(ctx, skill, config.Levels[0])
}

// newPlay starts a game at a skill on a given level, so tests can use
// levels of their own.
func newPlay(ctx *scene.Context, skill config.Skill, level config.Level) *Play {
	ctx.Session = session.New(skill, config.StartLives)
	return &Play{
		world: world.New(ctx.Session, level, ctx.RNG),
		hero:  Toad,
		level: 1,
	}
}

// World is the game's state, for tests.
func (p *Play) World() *world.World { return p.world }

// Update implements scene.Scene.
func (p *Play) Update(ctx *scene.Context) error {
	in := ctx.Input
	switch {
	case in.JustPressed(input.Pause):
		ctx.Cut(scene.NewPause(p))
		return nil
	case in.JustPressed(input.Menu):
		ctx.Switch(scene.NewMainMenu())
		return nil
	case in.JustPressed(input.Restart):
		ctx.Switch(New(ctx, ctx.Session.Skill))
		return nil
	}
	for _, e := range p.world.Update(pressed(in)) {
		switch e {
		case world.Hopped:
			ctx.Audio.PlayEffect(p.hero.Hop)
		case world.Squashed:
			ctx.Audio.PlayEffect(Crash)
			ctx.Shake.Start(shakeTicks, shakePx)
		case world.GameOver:
			ctx.Audio.PlayEffect(GameOver)
			ctx.Cut(newEnded(p))
		}
	}
	return nil
}

// Ended holds the last moment of the game on screen under a GAME OVER
// message, then goes on to the high scores. Confirm moves on sooner.
type Ended struct {
	play  *Play
	ticks int
}

func newEnded(p *Play) *Ended { return &Ended{play: p} }

// Update implements scene.Scene.
func (e *Ended) Update(ctx *scene.Context) error {
	e.ticks++
	if e.ticks >= gameOverTicks || (e.ticks > config.TPS/2 && ctx.Input.JustPressed(input.Confirm)) {
		s := ctx.Session
		ctx.Switch(scene.AfterGame(ctx, s.Skill, s.Score))
	}
	return nil
}

// Draw implements scene.Scene.
func (e *Ended) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	e.play.Draw(ctx, dst)
	scene.Dim(dst, sc.Shadow, 0.5)
	ui.TextCentred(dst, f.Large, "GAME OVER", (config.Project.ScreenHeight-f.Large.Size)/2, sc.Title, sc.Shadow)
}

// pressed is the direction pressed this tick: the first of up, down, left
// and right. Each press is one hop; holding a key does not repeat it
// (DESIGN §5.1).
func pressed(in *input.Input) world.Dir {
	switch {
	case in.JustPressed(input.Up):
		return world.Up
	case in.JustPressed(input.Down):
		return world.Down
	case in.JustPressed(input.Left):
		return world.Left
	case in.JustPressed(input.Right):
		return world.Right
	}
	return world.None
}

// Draw implements scene.Scene.
func (p *Play) Draw(ctx *scene.Context, dst *ebiten.Image) {
	dst.Fill(gfx.RGBA(ctx.Scheme.Backdrop))
	p.drawField(ctx, dst)
	p.drawHUD(ctx, dst)
	p.drawHint(ctx, dst)
}

// drawHUD draws the status bar: score, best score, level, lives and time.
func (p *Play) drawHUD(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	s := ctx.Session
	ty := (config.HUDHeight - f.Small.Size) / 2

	f.Small.DrawShadowed(dst, fmt.Sprintf("SCORE %06d", s.Score), config.FieldMargin, ty, sc.Text, sc.Shadow)
	f.Small.DrawShadowed(dst, fmt.Sprintf("HI %06d", best(ctx)), hudHiX, ty, sc.Text, sc.Shadow)
	f.Small.DrawShadowed(dst, fmt.Sprintf("LV %d", p.level), hudLevelX, ty, sc.Text, sc.Shadow)

	life := ctx.Sheet(Sheet).Sprite("life")
	ly := float64((config.HUDHeight - life.H) / 2)
	for i := range s.Lives {
		life.Draw(dst, 0, float64(hudLivesX+i*(life.W+hudLifeGap)), ly, false)
	}

	sw := config.Project.ScreenWidth
	barX := sw - config.FieldMargin - timeBarW
	barY := (config.HUDHeight - timeBarH) / 2
	label := "TIME"
	f.Small.DrawShadowed(dst, label, barX-labelGap-len(label)*f.Small.Size, ty, sc.Text, sc.Shadow)
	full := world.AttemptTicks(p.world.Skill)
	filled := timeBarW * p.world.TicksLeft / full
	vector.FillRect(dst, float32(barX), float32(barY), timeBarW, timeBarH, gfx.RGBA(sc.Panel), false)
	vector.FillRect(dst, float32(barX), float32(barY), float32(filled), timeBarH, gfx.RGBA(resurrect64.Emerald), false)
}

// drawHint draws the strip below the playfield.
func (p *Play) drawHint(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	top := config.FieldY + config.Rows*config.Cell
	ui.TextCentred(dst, f.Small, hint, top+(config.HintHeight-f.Small.Size)/2, sc.Dim, sc.Shadow)
}

// best is the top score in the current skill's table, or 0.
func best(ctx *scene.Context) int {
	if es := ctx.Scores.Table(ctx.Session.Skill.ID); len(es) > 0 {
		return es[0].Score
	}
	return 0
}
