package play

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The weather's look (DESIGN §6.3, D29, D31, D44). Fog and rain are an
// overcast layer on the time of day: the light dims and greys, and every
// lamp and headlight comes on.
const (
	overcastDim  = 0.3 // how much of the light overcast takes away
	overcastGrey = 0.5 // how far it greys the light toward fogGrey
	fogGrey      = resurrect64.Fog

	fogPatches = 7    // drifting patches over the road
	fogPatchW  = 128  // a patch's size, in pixels
	fogPatchH  = 40   //
	fogAlpha   = 0.88 // a patch's middle, at full fog

	rainDrops  = 140 // streaks falling over the field
	rainLen    = 6   // a streak's length, in pixels
	rainSlant  = 2   // how far it leans
	rainFall   = 7   // pixels a tick
	rainAlpha  = 0.5
	rainFine   = 0.6  // a streak's width in the smooth look, in the game's pixels
	rainSheen  = 0.18 // the wet road's sheen
	sheenColor = resurrect64.Blue
)

// updateWeather eases the overcast, the fog and the rain toward whether
// they are announced or on, one step a tick, over config.OvercastTicks:
// they come in through the warning and lift after. A wet storm's rain
// comes in the same way. A storm's own sky rolls in once it begins and
// away after it ends, over config.StormSkyTicks (D49, D51).
func (p *Play) updateWeather() {
	ease := func(v *int, on bool, most int) {
		if on {
			*v = min(*v+1, most)
		} else {
			*v = max(*v-1, 0)
		}
	}
	fog, rain := p.weatherOn(config.Fog), p.weatherOn(config.Rain)
	ease(&p.overcast, fog || rain, config.OvercastTicks)
	ease(&p.fogIn, fog, config.OvercastTicks)
	ease(&p.rainIn, rain || p.wetStorm(), config.OvercastTicks)
	ease(&p.stormSky, p.stormOn(), config.StormSkyTicks)
}

// stormOn reports whether a storm is on: past its warning, before it rolls
// away.
func (p *Play) stormOn() bool {
	s := p.world.Spell(config.StormFront)
	return s != nil && s.Phase == world.SpellOn
}

// weatherOn reports whether weather k is announced or on.
func (p *Play) weatherOn(k config.EventKind) bool {
	s := p.world.Spell(k)
	return s != nil && s.Phase != world.SpellQuiet
}

// wetStorm reports whether a wet storm is announced or on: it rains all
// through (D51).
func (p *Play) wetStorm() bool {
	s := p.world.Spell(config.StormFront)
	return s != nil && s.Phase != world.SpellQuiet && s.Wet
}

// overcastK, fogK and rainK are how far in each is, 0–1. The overcast is
// the weather's or the storm's sky, whichever is further in. Fog thins to
// config.StormFogDensityPct as a dry storm's wind comes in (D51).
func (p *Play) overcastK() float64 {
	return math.Max(float64(p.overcast)/config.OvercastTicks, float64(p.stormSky)/config.StormSkyTicks)
}
func (p *Play) rainK() float64 { return float64(p.rainIn) / config.OvercastTicks }
func (p *Play) fogK() float64 {
	thin := 1 - (1-config.StormFogDensityPct/100.0)*float64(p.windIn)/config.OvercastTicks
	return float64(p.fogIn) / config.OvercastTicks * thin
}

// overcast is look l dimmed and greyed k of the way to overcast (D44).
func overcast(l fx.Look, k float64) fx.Look {
	if k <= 0 {
		return l
	}
	g := gfx.RGBA(fogGrey)
	m := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*k*overcastGrey)) }
	l.Ambient = color.RGBA{m(l.Ambient.R, g.R), m(l.Ambient.G, g.G), m(l.Ambient.B, g.B), 255}
	l.Level *= 1 - overcastDim*k
	return l
}

// streetLampK is how strong street lamp i is: as the time of day has it,
// or on at full once the overcast passes its turn, so the lamps flick on
// one by one as fog or rain comes in (D44).
func (p *Play) streetLampK(i int) float64 {
	if p.overcastK() >= float64(i+1)/float64(len(lamps)+1) {
		return 1
	}
	if p.world.Level.DayCycle {
		return skyAt(p.stage()).lamps
	}
	return 1
}

// fogPatch is where fog patch i is this tick: drifting slowly along the
// road, each at its own speed and height, wrapping round. The same every
// game.
func (p *Play) fogPatch(i int) (x, y float64) {
	top, h := cellY(8)-fogPatchH/4, float64(4*config.Cell)
	span := float64(config.Cols*config.Cell + fogPatchW)
	speed := 0.15 + 0.05*float64(i%4)
	if i%2 == 1 {
		speed = -speed
	}
	x = math.Mod(float64(i)*span/fogPatches+float64(p.ticks)*speed, span)
	if x < 0 {
		x += span
	}
	y = top + math.Mod(float64(i)*h*0.37, h-fogPatchH/2)
	return float64(config.FieldX) + x - fogPatchW, y
}

// drawFog draws the fog patches over the road, faded in and out with the
// fog, s times the size. It is drawn before the light, so headlights still
// shine through it (D29).
func (p *Play) drawFog(dst *ebiten.Image, s int) {
	k := p.fogK()
	if k <= 0 {
		return
	}
	for i := range fogPatches {
		x, y := p.fogPatch(i)
		p.layPatch(dst, s, x, y, fogAlpha*k)
	}
}

// layPatch draws a fog patch with its top-left at x, y in the game's
// coordinates, s times the size, a times as opaque: its texture drawn at
// that size, so its edges stay soft.
func (p *Play) layPatch(dst *ebiten.Image, s int, x, y, a float64) {
	if p.fogImg == nil || p.fogImg.Bounds().Dx() != fogPatchW*s {
		p.fogImg = fogTexture(s)
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(x, y)
	op.GeoM.Scale(float64(s), float64(s))
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(p.fogImg, &op)
}

// fogTexture is one fog patch s times the size: a soft grey oval,
// thickest in the middle.
func fogTexture(s int) *ebiten.Image {
	w, h := fogPatchW*s, fogPatchH*s
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := gfx.RGBA(fogGrey)
	for y := range h {
		for x := range w {
			dx := (float64(x)+0.5)/float64(w)*2 - 1
			dy := (float64(y)+0.5)/float64(h)*2 - 1
			d := math.Sqrt(dx*dx + dy*dy)
			a := math.Max(0, math.Min(1, (1-d)*2.2))
			img.SetRGBA(x, y, color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)})
		}
	}
	return ebiten.NewImageFromImage(img)
}

// drawRoadSheen gives the road rows a wet sheen in the rain, or on a
// bonus level left wet by the night's rain (DESIGN §6.7), under the
// light, so headlights gleam on it (D31).
func (p *Play) drawRoadSheen(dst *ebiten.Image) {
	k := max(p.rainK(), p.wetK())
	if k <= 0 {
		return
	}
	c := gfx.RGBA(sheenColor)
	for r := range p.world.Lanes {
		if p.world.Lanes[r].Kind != config.Road {
			continue
		}
		a := float32(rainSheen * k)
		vector.FillRect(dst, float32(config.FieldX), float32(cellY(r)), float32(config.Cols*config.Cell), config.Cell,
			color.RGBA{uint8(float32(c.R) * a), uint8(float32(c.G) * a), uint8(float32(c.B) * a), uint8(255 * a)}, false)
	}
}

// drawRain draws the rain falling over the field, on top of the light so
// it shows at any time of day. Each streak falls at a steady pace from its
// own place, the same every game. It is drawn s times the size, finer and
// smoothed when larger than the game's own pixels.
func (p *Play) drawRain(dst *ebiten.Image, s int) {
	k := p.rainK()
	if k <= 0 {
		return
	}
	w, h := float64(config.Cols*config.Cell), float64(config.Rows*config.Cell)
	c := gfx.RGBA(resurrect64.Mist)
	a := rainAlpha * k
	col := color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
	width := float32(s)
	if s > 1 {
		width *= rainFine
	}
	for i := range rainDrops {
		x := float64(noise(int64(i), 1, 0)%1000) / 1000 * w
		y := math.Mod(float64(noise(int64(i), 2, 0)%1000)/1000*h+float64(p.ticks*rainFall), h)
		sf := float32(s)
		x0, y0 := sf*float32(float64(config.FieldX)+x), sf*float32(float64(config.FieldY)+y)
		vector.StrokeLine(dst, x0, y0, x0-sf*rainSlant, y0+sf*rainLen, width, col, s > 1)
	}
}
