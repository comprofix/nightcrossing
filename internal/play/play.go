// Package play is Night Crossing's play scene. It runs the world one tick at
// a time, draws its state with the status bar and hint strip, and turns the
// world's events into sound and effects. The rules live in package world.
package play

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/debug"
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

// New starts a game at a skill.
func New(ctx *scene.Context, skill config.Skill) scene.Scene {
	ctx.Session = session.New(skill, config.StartLives)
	return &Play{
		world: world.New(config.Cols, config.Rows, skill, ctx.RNG),
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
		}
	}
	return nil
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

	p.drawHero(ctx, dst)
}

// drawHero draws the hero turned to face its way, on its leap frame while
// hopping, rising and falling over the hop, then squashed for a moment on
// landing (DESIGN §3.3).
func (p *Play) drawHero(ctx *scene.Context, dst *ebiten.Image) {
	pl := &p.world.Player
	sp := ctx.Sheet(Sheet).Sprite(p.hero.Sprite)

	frame, lift := frameRest, 0.0
	switch {
	case pl.Hopping():
		frame = frameLeap
		t := float64(pl.HopProgress()) / config.HopTicks
		lift = math.Round(config.HopLiftPx * math.Sin(math.Pi*t))
	case pl.Still < config.LandTicks:
		frame = frameLand
	}

	x := float64(config.FieldX) + float64(pl.X)/config.SubPx
	y := float64(config.FieldY) + float64(pl.Y)/config.SubPx - lift
	var op ebiten.DrawImageOptions
	op.GeoM = facing(pl.Facing, sp.W, sp.H)
	op.GeoM.Translate(x, y)
	dst.DrawImage(sp.Frame(frame), &op)

	if debug.Enabled {
		debug.Box(dst, cellX(pl.Col), cellY(pl.Row), config.Cell, config.Cell, ctx.Scheme.Highlight)
	}
}

// facing turns a w×h sprite drawn facing up to face d, keeping it in the
// same w×h box. The turns are exact quarter turns, so pixels stay whole.
func facing(d world.Dir, w, h int) ebiten.GeoM {
	var g ebiten.GeoM
	fw, fh := float64(w), float64(h)
	switch d {
	case world.Right: // x' = h - y, y' = x
		g.SetElement(0, 0, 0)
		g.SetElement(0, 1, -1)
		g.SetElement(0, 2, fh)
		g.SetElement(1, 0, 1)
		g.SetElement(1, 1, 0)
	case world.Down: // x' = w - x, y' = h - y
		g.SetElement(0, 0, -1)
		g.SetElement(0, 2, fw)
		g.SetElement(1, 1, -1)
		g.SetElement(1, 2, fh)
	case world.Left: // x' = y, y' = w - x
		g.SetElement(0, 0, 0)
		g.SetElement(0, 1, 1)
		g.SetElement(1, 0, -1)
		g.SetElement(1, 1, 0)
		g.SetElement(1, 2, fw)
	}
	return g
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
