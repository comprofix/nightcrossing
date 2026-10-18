package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The heron's frames (D34, D67).
const (
	heronStand = 0
	heronStep  = 1 // a stalking heron walking into the next cell
	heronCock  = 2 // cocking its head before a strike
	heronReach = 3 // its neck out into the next cell, where heron_jab draws its head
)

// The predators' lights (D67, D68): a heron flashing its warning glows
// like a sinking ship's, and a fox's eyes and the fox glow in the nook.
const (
	foxEyesR     = 12
	foxEyesLight = 0.5
	foxR         = 20
	foxLight     = 0.45
)

// heronFlashing reports whether heron e is lit up this tick: it flashes
// while it cocks its head, the warning before it strikes (D67).
func heronFlashing(e world.Entity) bool {
	return e.Cock > 0 && (e.Cock/config.DiveFlashTicks)%2 == 0
}

// heronSprite is how heron e is drawn: its sprite, frame and facing
// (DESIGN §5.2, D34, D67).
func heronSprite(ctx *scene.Context, e world.Entity) (*gfx.Sprite, int, bool) {
	sheet := ctx.Sheet(Sheet)
	sp, frame := sheet.Sprite("heron"), heronStand
	switch {
	case e.Jab > 0:
		frame = heronReach
	case e.Cock > 0:
		frame = heronCock
		if heronFlashing(e) {
			sp = sheet.Sprite("heron_flash")
		}
	case e.Stride > 0:
		frame = heronStep
	}
	return sp, frame, e.HeronDir < 0
}

// eachHeron calls draw with the frame, place and facing of each heron on
// row r, and of the head of each one jabbing, in the cell it faces.
func (p *Play) eachHeron(ctx *scene.Context, r int, draw func(img *ebiten.Image, x, y float64, flip bool)) {
	lane := &p.world.Lanes[r]
	jab := ctx.Sheet(Sheet).Sprite("heron_jab")
	for _, e := range lane.Entities {
		if !e.Heron {
			continue
		}
		sp, frame, flip := heronSprite(ctx, e)
		x := float64(config.FieldX) + math.Floor(float64(e.X()+e.HeronX())/config.SubPx)
		draw(sp.Frame(frame), x, cellY(r), flip)
		if e.Jab > 0 {
			draw(jab.Frame(0), x+float64(e.HeronDir*config.Cell), cellY(r), flip)
		}
	}
}

// drawFox draws the fox in its nook, or only its eyes as it comes (D68).
func (p *Play) drawFox(ctx *scene.Context, dst *ebiten.Image) {
	f := p.world.Fox
	if f.Phase == world.FoxAway {
		return
	}
	name := "fox"
	if f.Phase == world.FoxEyes {
		name = "fox_eyes"
	}
	sp := ctx.Sheet(Sheet).Sprite(name)
	x := cellX(config.Nooks[f.Nook]) + float64(config.NookWidth*config.Cell-sp.W)/2
	sp.Draw(dst, 0, x, cellY(config.GoalRow), false)
}

// addFoxLight adds the glow of a fox's eyes in a nook, and of the fox, so
// night never hides it (D68).
func (p *Play) addFoxLight(l *fx.Lighting) {
	f := p.world.Fox
	if f.Phase == world.FoxAway {
		return
	}
	x, y := cellX(config.Nooks[f.Nook])+float64(config.NookWidth*config.Cell)/2, cellY(config.GoalRow)+float64(config.Cell)/2
	r, s := foxR, foxLight
	if f.Phase == world.FoxEyes {
		r, s = foxEyesR, foxEyesLight
	}
	l.Add(fx.Light{X: x, Y: y, Radius: float64(r), Colour: gfx.RGBA(resurrect64.Amber), Strength: s})
}
