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
// platform's end, reared up with its head and paws over it, by the
// sprite's name; and where its sprite goes on the screen, mirrored to face
// left. Its body is the cell behind its nose, in the water (D69).
func (p *Play) ratSprite(ctx *scene.Context, r world.Rat) (name string, sp *gfx.Sprite, x float64, flip bool) {
	name = "rat"
	if r.Phase == world.RatGrab {
		name = "rat_grab"
	}
	sp = ctx.Sheet(Sheet).Sprite(name)
	x, flip = p.ratLeft(r, sp.W)
	return name, sp, x, flip
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

// eachProwler calls draw with each water rat swimming or grabbing and the
// snake out on the verge: the ones that can kill by touch.
func (p *Play) eachProwler(ctx *scene.Context, draw func(pc piece)) {
	w := p.world
	sheet := ctx.Sheet(Sheet)
	for _, r := range w.Rats {
		if r.Phase == world.RatSwim || r.Phase == world.RatGrab {
			name, sp, x, flip := p.ratSprite(ctx, r)
			draw(piece{name: name, frame: int(w.Tick/ratPaddleTicks) % sp.Frames(), x: x, y: cellY(r.Row), flip: flip})
		}
	}
	if s := w.Snake; s.Phase == world.SnakeOut {
		sp := sheet.Sprite("snake")
		draw(piece{name: "snake", frame: int(w.Tick/snakeWaveTicks) % sp.Frames(), x: p.snakeLeft(), y: cellY(config.VergeRow), flip: s.Heading() < 0})
	}
}

// eachProwl calls ripple and wake with each water rat's marks on the
// water: rings spreading as it comes up, about cx, cy, i staggering each
// rat's; and the V of its wake as it swims, from its tail at x, y, going
// back the way back says. Then draw with the verge grass rustling where
// the snake will come out, and with the rats and the snake (D69, D70).
func (p *Play) eachProwl(ctx *scene.Context, ripple func(cx, cy float64, i int), wake func(x, y, back float64), draw func(pc piece)) {
	w := p.world
	for i, r := range w.Rats {
		if r.Phase == world.RatAway {
			continue
		}
		x, flip := p.ratLeft(r, config.Cell)
		y := cellY(r.Row) + ratWaterY
		switch r.Phase {
		case world.RatRipple:
			ripple(x+config.Cell/2, y, i)
		case world.RatSwim:
			if flip {
				wake(x+config.Cell, y, 1)
			} else {
				wake(x, y, -1)
			}
		}
	}
	if w.Snake.Phase == world.SnakeRustle {
		sp := ctx.Sheet(Sheet).Sprite("rustle")
		draw(piece{name: "rustle", frame: int(w.Tick/rustleSwayTicks) % sp.Frames(), x: p.snakeLeft(), y: cellY(config.VergeRow)})
	}
	p.eachProwler(ctx, draw)
}

// drawProwl draws the water rats and the snake, with their marks, in the
// pixel art.
func (p *Play) drawProwl(ctx *scene.Context, dst *ebiten.Image) {
	p.eachProwl(ctx, func(cx, cy float64, i int) {
		p.eachRipple(i, func(rx, ry float64, c int) {
			for d := range rippleDots {
				a := 2 * math.Pi * float64(d) / rippleDots
				vector.FillRect(dst, float32(math.Round(cx+rx*math.Cos(a))), float32(math.Round(cy+ry*math.Sin(a))), 1, 1, gfx.RGBA(c), false)
			}
		})
	}, func(x, y, back float64) {
		eachWakeDot(func(dx, dy float64, c int) {
			vector.FillRect(dst, float32(x+back*dx), float32(y+dy), 1, 1, gfx.RGBA(c), false)
		})
	}, func(pc piece) { drawPiece(ctx, dst, pc) })
}

// eachRipple calls ring with the size and colour of each ring spreading
// where water rat i is coming up.
func (p *Play) eachRipple(i int, ring func(rx, ry float64, c int)) {
	for k := range rippleRings {
		t := int((p.world.Tick + int64(k*rippleGrowTicks/rippleRings+i*7)) % rippleGrowTicks)
		c := resurrect64.Mist
		if t > rippleGrowTicks/2 {
			c = resurrect64.Sky
		}
		ring(2+float64((rippleW-2)*t)/rippleGrowTicks, 1+float64((rippleH-1)*t)/rippleGrowTicks, c)
	}
}

// eachWakeDot calls dot with each dot of a swimming rat's wake, how far
// back from its tail and across from its waterline, and its colour.
func eachWakeDot(dot func(dx, dy float64, c int)) {
	for k := 1; k <= ratWake; k++ {
		c := resurrect64.Cobalt
		if k%2 == 0 {
			c = resurrect64.Sky
		}
		for _, dy := range []float64{-float64(k) / 2, float64(k) / 2} {
			dot(float64(2*k), dy, c)
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
			_, sp, x, flip := p.ratSprite(ctx, r)
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
