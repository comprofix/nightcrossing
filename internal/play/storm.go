package play

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/world"
)

// Storm's look (DESIGN §6.4): arrows on the water of a current row, and
// wind streaks through a gust spell, building before each push.
const (
	arrowGapMin   = 36   // between a current's arrows, at its strongest
	arrowGapMax   = 72   // and at its weakest
	arrowArm      = 4    // an arrow's arms, in pixels
	arrowPair     = 5    // the second chevron, behind the first
	arrowWeakest  = 3    // the current strength the gap and brightness run from (D50)
	arrowStrong   = 14   // and to: level 8's at the ceiling
	arrowDimAlpha = 0.75 // a weak current's arrows
	arrowY        = 12   // the arrows' line, down from the row's top

	windStreaks = 120 // streaks across the field at the height of a gust
	windCalm    = 0.4 // the share of them through the spell between pushes
	windLen     = 24  // a streak's length, in pixels
	windFall    = 10  // pixels a tick
	windAlpha   = 0.8
)

// drawCurrent draws river row r's current, if it has one: a line of
// arrows on the water pointing and moving the way it slides the hero,
// closer together and brighter the stronger it is (D46).
func (p *Play) drawCurrent(dst *ebiten.Image, r int) {
	w := p.world
	lane := &w.Lanes[r]
	slide := lane.Slide()
	if slide == 0 {
		return
	}
	k := math.Min(1, math.Max(0, float64(abs(slide)-arrowWeakest)/float64(arrowStrong-arrowWeakest)))
	gap := arrowGapMax - int(k*float64(arrowGapMax-arrowGapMin))
	c := gfx.RGBA(resurrect64.Seafoam)
	a := float32(arrowDimAlpha + (1-arrowDimAlpha)*k)
	col := color.RGBA{uint8(float32(c.R) * a), uint8(float32(c.G) * a), uint8(float32(c.B) * a), uint8(255 * a)}

	dir := float32(1)
	if slide < 0 {
		dir = -1
	}
	drift := int(w.Tick * int64(slide) / config.SubPx) // at the slide's own speed
	off := ((drift+r*13)%gap+gap)%gap - gap
	x0, y := float32(config.FieldX), float32(cellY(r)+arrowY)
	for dx := off; dx < w.Cols*config.Cell+gap; dx += gap {
		for _, back := range []float32{0, arrowPair} { // a double chevron, 2px thick
			tip := x0 + float32(dx) - dir*back
			for i := float32(0); i <= arrowArm; i++ {
				vector.FillRect(dst, tip-dir*i-1, y-i, 2, 1, col, false)
				vector.FillRect(dst, tip-dir*i-1, y+i, 2, 1, col, false)
			}
		}
	}
}

// updateWind eases the wind in through a gust spell's warning and out
// after it, one step a tick, like the weather.
func (p *Play) updateWind() {
	s := p.world.Spell(config.Gust)
	if s != nil && s.Phase != world.SpellQuiet {
		p.windIn = min(p.windIn+1, config.OvercastTicks)
	} else {
		p.windIn = max(p.windIn-1, 0)
	}
}

// windK is how hard the wind is blowing, 0–1, and which way: easing in
// with the spell to a steady share, and building to full through the tell
// before each push (D47).
func (p *Play) windK() (k float64, dir int) {
	s := p.world.Spell(config.Gust)
	if s == nil || p.windIn == 0 {
		return 0, 0
	}
	k = windCalm * float64(p.windIn) / config.OvercastTicks
	if n := s.NextPush(); n >= 0 && n <= config.GustTellTicks {
		k = math.Max(k, 1-float64(n)/config.GustTellTicks*(1-windCalm))
	}
	return k, s.Dir
}

// drawWind draws the streaks of a gust spell blowing across the field,
// over the light so they show at any time of day. Each streak runs at a
// steady pace from its own place, the same every game.
func (p *Play) drawWind(dst *ebiten.Image) {
	k, dir := p.windK()
	if k <= 0 {
		return
	}
	w, h := float64(config.Cols*config.Cell+windLen), float64(config.Rows*config.Cell)
	c := gfx.RGBA(resurrect64.White)
	// Each streak fades from its head, the way the wind goes, to its tail.
	var cols [3]color.RGBA
	for j, f := range []float64{1, 0.55, 0.25} {
		a := windAlpha * k * f
		cols[j] = color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
	}
	const part = windLen / 3
	for i := range int(windStreaks * k) {
		y := float64(noise(int64(i), 3, 0)%1000) / 1000 * h
		x := math.Mod(float64(noise(int64(i), 4, 0)%1000)/1000*w+float64(dir*p.ticks*windFall), w)
		if x < 0 {
			x += w
		}
		head := float32(float64(config.FieldX) + x - windLen/2)
		y0 := float32(float64(config.FieldY) + math.Floor(y))
		for j, col := range cols {
			x0 := head - float32(dir*j*part)
			if dir > 0 {
				x0 -= part
			}
			vector.FillRect(dst, x0, y0, part, 1, col, false)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
