package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/debug"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The water rat's and the snake's look (D69, D70), in pixels and ticks.
const (
	ratPaddleTicks  = 12 // the rat's paddle frames change this often
	snakeWaveTicks  = 15 // the snake's body waves this often
	rustleSwayTicks = 8  // the rustling grass sways this often

	rippleRings     = 3  // rings spreading where a water rat comes up
	rippleGrowTicks = 30 // a ring grows to its widest in this long
	rippleW         = 11 // a ring's widest half-width
	rippleH         = 4  // and half-height
	rippleDots      = 20 // the dots round a ring
	ratWaterY       = 16 // the waterline in the rat's sprite, from its top
	ratWake         = 6  // the V of wake behind a swimming rat, in 2px steps
	ratEyeBack      = 6  // a rat's eye, from the front of its sprite
	grabEyeBack     = 10 // and of its sprite grabbing a platform's end

	ratR        = 12 // the glint of a rat's eye
	ratLight    = 0.4
	rippleR     = 14
	rippleLight = 0.3
	snakeR      = 22
	snakeLight  = 0.35
	rustleR     = 16
	rustleLight = 0.3
)

// ratSprite is how water rat r is drawn: swimming, or grabbing its
// platform's end, reared up with its head and paws over it; and where its
// sprite goes on the screen, mirrored to face left. Its body is the cell
// behind its nose, in the water (D69).
func (p *Play) ratSprite(ctx *scene.Context, r world.Rat) (sp *gfx.Sprite, x float64, flip bool) {
	sp = ctx.Sheet(Sheet).Sprite("rat")
	if r.Phase == world.RatGrab {
		sp = ctx.Sheet(Sheet).Sprite("rat_grab")
	}
	x, flip = p.ratLeft(r, sp.W)
	return sp, x, flip
}

// ratLeft is where a w-wide sprite of water rat r goes on the screen, and
// whether it is mirrored to face left: its body the cell behind its nose,
// and anything wider reaching past its nose.
func (p *Play) ratLeft(r world.Rat, w int) (x float64, flip bool) {
	nose, facing := p.world.RatNose(r)
	left := nose - config.Cell*config.SubPx
	if facing < 0 {
		left = nose - (w-config.Cell)*config.SubPx
	}
	return float64(config.FieldX) + math.Floor(float64(left)/config.SubPx), facing < 0
}

// snakeLeft is where the snake's sprite goes on the screen.
func (p *Play) snakeLeft() float64 {
	return float64(config.FieldX) + math.Floor(float64(p.world.Snake.X())/config.SubPx)
}

// eachProwler calls draw with the frame, place and facing of each water
// rat swimming or grabbing and of the snake out on the verge: the ones
// that can kill by touch.
func (p *Play) eachProwler(ctx *scene.Context, draw func(img *ebiten.Image, x, y float64, flip bool)) {
	w := p.world
	sheet := ctx.Sheet(Sheet)
	for _, r := range w.Rats {
		if r.Phase == world.RatSwim || r.Phase == world.RatGrab {
			sp, x, flip := p.ratSprite(ctx, r)
			draw(sp.Frame(int(w.Tick/ratPaddleTicks)), x, cellY(r.Row), flip)
		}
	}
	if s := w.Snake; s.Phase == world.SnakeOut {
		draw(sheet.Sprite("snake").Frame(int(w.Tick/snakeWaveTicks)), p.snakeLeft(), cellY(config.VergeRow), s.Heading() < 0)
	}
}

// drawProwl draws the water rats and the snake: a rat's ripple as it comes
// up and the V of its wake as it swims, and the verge grass rustling
// where the snake will come out (D69, D70).
func (p *Play) drawProwl(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	for i, r := range w.Rats {
		if r.Phase == world.RatAway {
			continue
		}
		x, flip := p.ratLeft(r, config.Cell)
		y := cellY(r.Row) + ratWaterY
		switch r.Phase {
		case world.RatRipple:
			p.drawRipple(dst, x+config.Cell/2, y, i)
		case world.RatSwim:
			tail, back := x, -1.0
			if flip {
				tail, back = x+config.Cell, 1
			}
			for k := 1; k <= ratWake; k++ {
				c := resurrect64.Cobalt
				if k%2 == 0 {
					c = resurrect64.Sky
				}
				wx := float32(tail + back*float64(2*k))
				for _, dy := range []float64{-float64(k) / 2, float64(k) / 2} {
					vector.FillRect(dst, wx, float32(y+dy), 1, 1, gfx.RGBA(c), false)
				}
			}
		}
	}
	if w.Snake.Phase == world.SnakeRustle {
		ctx.Sheet(Sheet).Sprite("rustle").Draw(dst, int(w.Tick/rustleSwayTicks), p.snakeLeft(), cellY(config.VergeRow), false)
	}
	p.eachProwler(ctx, func(img *ebiten.Image, x, y float64, flip bool) {
		dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: spriteGeoM(img, x, y, flip)})
	})
}

// drawRipple draws rings spreading on the water around cx, cy, the warning
// that a water rat is coming up; i staggers each rat's rings.
func (p *Play) drawRipple(dst *ebiten.Image, cx, cy float64, i int) {
	for k := range rippleRings {
		t := int((p.world.Tick + int64(k*rippleGrowTicks/rippleRings+i*7)) % rippleGrowTicks)
		rx := 2 + float64((rippleW-2)*t)/rippleGrowTicks
		ry := 1 + float64((rippleH-1)*t)/rippleGrowTicks
		c := resurrect64.Mist
		if t > rippleGrowTicks/2 {
			c = resurrect64.Sky
		}
		for d := range rippleDots {
			a := 2 * math.Pi * float64(d) / rippleDots
			vector.FillRect(dst, float32(math.Round(cx+rx*math.Cos(a))), float32(math.Round(cy+ry*math.Sin(a))), 1, 1, gfx.RGBA(c), false)
		}
	}
}

// addProwlLights lights the water rats and the snake, so night never hides
// them or their warnings (D69, D70).
func (p *Play) addProwlLights(ctx *scene.Context, l *fx.Lighting) {
	w := p.world
	cell := float64(config.Cell)
	for _, r := range w.Rats {
		y := cellY(r.Row) + cell/2
		switch r.Phase {
		case world.RatRipple:
			x, _ := p.ratLeft(r, config.Cell)
			l.Add(fx.Light{X: x + cell/2, Y: y, Radius: rippleR, Colour: gfx.RGBA(resurrect64.Sky), Strength: rippleLight})
		case world.RatSwim, world.RatGrab:
			sp, x, flip := p.ratSprite(ctx, r)
			back := float64(ratEyeBack) // its eye, near the front
			if r.Phase == world.RatGrab {
				back = grabEyeBack
			}
			eye := x + float64(sp.W) - back
			if flip {
				eye = x + back
			}
			l.Add(fx.Light{X: eye, Y: y, Radius: ratR, Colour: gfx.RGBA(resurrect64.Lemon), Strength: ratLight})
		}
	}
	y := cellY(config.VergeRow) + cell/2
	switch w.Snake.Phase {
	case world.SnakeRustle:
		l.Add(fx.Light{X: p.snakeLeft() + cell, Y: y, Radius: rustleR, Colour: gfx.RGBA(resurrect64.Lime), Strength: rustleLight})
	case world.SnakeOut:
		l.Add(fx.Light{X: p.snakeLeft() + cell, Y: y, Radius: snakeR, Colour: gfx.RGBA(resurrect64.Lime), Strength: snakeLight})
	}
}

// drawProwlBoxes outlines, in debug builds, the head and paws of each
// water rat grabbing a platform's end, and the snake out on the verge:
// what kills.
func (p *Play) drawProwlBoxes(dst *ebiten.Image, c int) {
	w := p.world
	m := float64(config.HitMargin)
	for _, r := range w.Rats {
		if r.Phase != world.RatGrab {
			continue
		}
		hl, hr := w.RatHead(r)
		debug.Box(dst, float64(config.FieldX)+float64(hl)/config.SubPx, cellY(r.Row), float64(hr-hl)/config.SubPx, config.Cell, c)
	}
	if w.Snake.Phase == world.SnakeOut {
		debug.Box(dst, p.snakeLeft()+m, cellY(config.VergeRow), config.SnakeCells*config.Cell-2*m, config.Cell, c)
	}
}
