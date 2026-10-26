package play

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
)

// After the storm (DESIGN §6.7): a rainbow arching over the river, its top
// over the far bank and its ends meeting the field's sides lower down, and
// patches of mist drifting on the water. Only the look. In pixels.
const (
	rainbowTop    = 6  // the arc's top, below the field's top
	rainbowEdge   = 96 // where it meets the field's sides, below the field's top
	rainbowBand   = 3  // each colour's width
	rainbowAlpha  = 0.3
	mistPatches   = 6
	mistAlpha     = 0.3
	mistSpeedBase = 0.1 // pixels a tick
)

// rainbowBands are the rainbow's colours, from the outside in.
var rainbowBands = [...]int{resurrect64.Red, resurrect64.Orange, resurrect64.Lemon, resurrect64.Lime,
	resurrect64.Sky, resurrect64.Violet}

// rainbow reports whether the level hangs a rainbow and mist over the
// river.
func (p *Play) rainbow() bool {
	b := p.world.Level.Bonus
	return b != nil && b.Rainbow
}

// drawRainbow draws the rainbow over the river, s times the size: an arc
// of a circle whose centre is far below the field, so only its top shows.
func (p *Play) drawRainbow(dst *ebiten.Image, s int) {
	if !p.rainbow() {
		return
	}
	half := float64(p.world.Cols*config.Cell) / 2
	d := float64(rainbowEdge - rainbowTop)
	r := (half*half + d*d) / (2 * d) // the circle through the top and both edges
	cx := float64(config.FieldX) + half
	cy := float64(config.FieldY+rainbowTop) + r
	sf := float32(s)
	for i, c := range rainbowBands {
		rr := r - float64(i*rainbowBand) - rainbowBand/2.0
		vector.StrokeCircle(dst, sf*float32(cx), sf*float32(cy), sf*float32(rr), sf*rainbowBand, faded(c, rainbowAlpha), true)
	}
}

// faded is palette colour c at alpha a, premultiplied.
func faded(c int, a float64) color.RGBA {
	k := gfx.RGBA(c)
	return color.RGBA{uint8(float64(k.R) * a), uint8(float64(k.G) * a), uint8(float64(k.B) * a), uint8(255 * a)}
}

// mistPatch is where mist patch i is this tick: drifting slowly along the
// river, each at its own speed and height, wrapping round. The same every
// game.
func (p *Play) mistPatch(i int) (x, y float64) {
	top, h := cellY(1)-fogPatchH/4, float64(5*config.Cell)
	span := float64(config.Cols*config.Cell + fogPatchW)
	speed := mistSpeedBase + 0.04*float64(i%3)
	if i%2 == 1 {
		speed = -speed
	}
	x = math.Mod(float64(i)*span/mistPatches+float64(p.ticks)*speed, span)
	if x < 0 {
		x += span
	}
	y = top + math.Mod(float64(i)*h*0.41, h-fogPatchH/2)
	return float64(config.FieldX) + x - fogPatchW, y
}

// drawMist draws the mist on the water, thinner than fog, before the
// light, s times the size.
func (p *Play) drawMist(dst *ebiten.Image, s int) {
	if !p.rainbow() {
		return
	}
	for i := range mistPatches {
		x, y := p.mistPatch(i)
		p.layPatch(dst, s, x, y, mistAlpha)
	}
}
