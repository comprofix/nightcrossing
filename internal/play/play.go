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
	level int
}

// New starts a game at a skill.
func New(ctx *scene.Context, skill config.Skill) scene.Scene {
	ctx.Session = session.New(skill, config.StartLives)
	return &Play{
		world: world.New(config.Cols, config.Rows, skill, ctx.RNG),
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
	p.world.Update()
	return nil
}

// Draw implements scene.Scene.
func (p *Play) Draw(ctx *scene.Context, dst *ebiten.Image) {
	dst.Fill(gfx.RGBA(ctx.Scheme.Backdrop))
	p.drawField(ctx, dst)
	p.drawHUD(ctx, dst)
	p.drawHint(ctx, dst)
}

// drawField draws the grid: grass on the goal bank and the start row, open
// ground between with faint cell lines, and the hero on the start row.
func (p *Play) drawField(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	sheet := ctx.Sheet(Sheet)
	fw, fh := float32(w.Cols*config.Cell), float32(w.Rows*config.Cell)
	vector.FillRect(dst, config.FieldX, config.FieldY, fw, fh, gfx.RGBA(resurrect64.Ink), false)

	line := gfx.RGBA(resurrect64.Shadow)
	for c := 1; c < w.Cols; c++ {
		x := float32(config.FieldX + c*config.Cell)
		vector.FillRect(dst, x, config.FieldY, 1, fh, line, false)
	}
	for r := 1; r < w.Rows; r++ {
		y := float32(config.FieldY + r*config.Cell)
		vector.FillRect(dst, config.FieldX, y, fw, 1, line, false)
	}

	verge := sheet.Sprite("verge")
	for _, r := range []int{config.GoalRow, config.StartRow} {
		for c := range w.Cols {
			verge.Draw(dst, 0, cellX(c), cellY(r), false)
		}
	}

	sheet.Sprite("toad").Draw(dst, 0, cellX(w.Cols/2), cellY(config.StartRow), false)
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

// cellX and cellY are the screen position of a grid cell's top-left corner.
func cellX(c int) float64 { return float64(config.FieldX + c*config.Cell) }
func cellY(r int) float64 { return float64(config.FieldY + r*config.Cell) }
