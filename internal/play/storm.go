package play

import (
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

// Storm's look (DESIGN §6.4): arrows on the water of a current row, and
// wind streaks through a dry storm, building before each push.
const (
	arrowGapMin   = 36   // between a current's arrows, at its strongest
	arrowGapMax   = 72   // and at its weakest
	arrowArm      = 4    // an arrow's arms, in pixels
	arrowPair     = 5    // the second chevron, behind the first
	arrowWeakest  = 3    // the current strength the gap and brightness run from (D50)
	arrowStrong   = 14   // and to: level 8's at the ceiling
	arrowDimAlpha = 0.75 // a weak current's arrows
	arrowY        = 12   // the arrows' line, down from the row's top

	windStreaks = 120 // streaks across the field at the height of a push
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

// dryStorm is the storm, if a dry one is announced or on.
func (p *Play) dryStorm() *world.Spell {
	s := p.world.Spell(config.StormFront)
	if s == nil || s.Phase == world.SpellQuiet || s.Wet {
		return nil
	}
	return s
}

// updateWind eases the wind in through a dry storm's warning and out
// after it rolls away, one step a tick, like the weather. Its direction is
// kept as it dies away.
func (p *Play) updateWind() {
	if s := p.dryStorm(); s != nil {
		p.windIn = min(p.windIn+1, config.OvercastTicks)
		p.windDir = s.Dir
	} else {
		p.windIn = max(p.windIn-1, 0)
	}
}

// windK is how hard the wind is blowing, 0–1, and which way: easing in
// with the storm to a steady share, and building to full through the tell
// before each push (D47).
func (p *Play) windK() (k float64, dir int) {
	if p.windIn == 0 {
		return 0, 0
	}
	k = windCalm * float64(p.windIn) / config.OvercastTicks
	if s := p.dryStorm(); s != nil {
		if n := s.NextPush(); n >= 0 && n <= config.WindTellTicks {
			k = math.Max(k, 1-float64(n)/config.WindTellTicks*(1-windCalm))
		}
	}
	return k, p.windDir
}

// drawWind draws the streaks of a dry storm's wind blowing across the field,
// over the light so they show at any time of day. Each streak runs at a
// steady pace from its own place, the same every game. They are drawn s
// times the size.
func (p *Play) drawWind(dst *ebiten.Image, s int) {
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
			sf := float32(s)
			vector.FillRect(dst, sf*x0, sf*y0, sf*part, sf, col, false)
		}
	}
}

// The flood's look on the verge, and the storm sky (D48, D49).
const (
	lapDepth   = 3 // how far the water laps up onto the verge, in pixels
	lapStep    = 2 // the width of each lap
	lapTicks   = 7 // the laps change this often
	puddleW    = 7 // a puddle, in pixels
	puddleH    = 3
	chopFlecks = 6 // the wind's chop on each river row
	chopTicks  = 5 // it changes this often

	lightningFirst = 2 * config.TPS // a storm's first flash comes within this of it beginning
	lightningMin   = 4 * config.TPS // between flashes, at least
	lightningMax   = 8 * config.TPS // and at most
	lightningRetry = config.TPS / 4 // a flash held back by a warning or a tell tries again after this
	flashHold      = 4              // ticks a flash holds at its brightest
	flashFade      = 18             // ticks it fades over
	flashMix       = 0.6            // how far it lifts the light toward daylight
	flashSoft      = 0.25           // and with reduced motion, a soft lift instead
	thunderDelay   = 20             // ticks from the flash to its thunder
	noFlash        = 1 << 30        // flash before the first one
)

// drawFlood draws a wet storm's flood on the verge: the water lapping up
// its river edge from the storm's warning on, so the flood is seen coming,
// puddles on the cells about to go under or draining, and open water on
// those that are under (D48).
func (p *Play) drawFlood(dst *ebiten.Image) {
	w := p.world
	s := w.Spell(config.StormFront)
	if s == nil || s.Phase == world.SpellQuiet || !s.Floods() || w.Lanes[config.VergeRow].Kind != config.Safe {
		return
	}
	y := float32(cellY(config.VergeRow))
	water, crest := gfx.RGBA(resurrect64.Navy), gfx.RGBA(resurrect64.Cobalt)
	for x := 0; x < w.Cols*config.Cell; x += lapStep {
		d := float32(1 + noise(w.Tick/lapTicks, int64(x), 9)%lapDepth)
		vector.FillRect(dst, float32(config.FieldX+x), y, lapStep, d, water, false)
	}
	for c := range w.Cols {
		x := float32(cellX(c))
		switch {
		case w.Flooded(c):
			vector.FillRect(dst, x, y, config.Cell, config.Cell, water, false)
			for i, wr := range waveRows {
				off := float32((int(w.Tick)/(waveBob/2) + i*7 + c*3) % (config.Cell - waveLen))
				vector.FillRect(dst, x+off, y+float32(wr.y), waveLen, 1, crest, false)
			}
		case w.Puddled(c):
			for _, pd := range [][2]float32{{3, 5}, {12, 14}} {
				vector.FillRect(dst, x+pd[0], y+pd[1], puddleW, puddleH, water, false)
				vector.FillRect(dst, x+pd[0]+1, y+pd[1], puddleW-3, 1, crest, false)
			}
		}
	}
}

// drawChop draws the wind's chop on river row r all through Storm, storm
// or clear: white flecks that come and go (D49).
func (p *Play) drawChop(dst *ebiten.Image, r int) {
	w := p.world
	if w.Spell(config.StormFront) == nil {
		return
	}
	c := gfx.RGBA(resurrect64.Mist)
	for k := range chopFlecks {
		h := noise(w.Tick/chopTicks, int64(r+50), int64(k))
		x := float32(config.FieldX) + float32(h%uint32(w.Cols*config.Cell))
		y := float32(cellY(r)) + float32(3+h>>12%(config.Cell-6))
		vector.FillRect(dst, x, y, 2, 1, c, false)
	}
}

// updateLightning runs a storm's lightning one tick: only while a storm is
// on, the first flash within lightningFirst ticks of it beginning, then one
// every lightningMin to lightningMax ticks, held back while an event is
// being announced or the wind's push is coming, so it never hides a cue
// (D49). A flash already showing fades out after the storm. Its timing
// comes from the level and the count of flashes, not the game's random
// numbers, so the world plays the same with or without it.
func (p *Play) updateLightning() {
	p.flash = min(p.flash+1, noFlash)
	on := p.stormOn()
	if on && !p.storming {
		p.nextFlash = 1 + p.flashRoll(lightningFirst)
	}
	if p.storming = on; !on {
		return
	}
	if p.nextFlash--; p.nextFlash > 0 {
		return
	}
	if p.cueShowing() {
		p.nextFlash = lightningRetry
		return
	}
	p.flash, p.flashes = 0, p.flashes+1
	p.nextFlash = lightningMin + p.flashRoll(lightningMax-lightningMin+1)
}

// flashRoll is a number from 0 to n-1 for the next flash's timing.
func (p *Play) flashRoll(n int) int {
	h := noise(int64(p.world.Level.Number), int64(p.flashes), 11)
	return int(h % uint32(n))
}

// cueShowing reports whether an event is being announced or the wind's
// push is coming.
func (p *Play) cueShowing() bool {
	for i := range p.world.Spells {
		s := &p.world.Spells[i]
		if s.Phase == world.SpellWarning {
			return true
		}
		if n := s.NextPush(); n >= 0 && n <= config.WindTellTicks+flashHold+flashFade {
			return true
		}
	}
	return false
}

// thunderNow reports whether the last flash's thunder sounds this tick.
func (p *Play) thunderNow() bool { return p.flash == thunderDelay }

// lightning is look l lit by a flash, if one is showing: lifted toward
// daylight at once and fading, or with reduced motion a soft lift that
// rises and falls (D49).
func (p *Play) lightning(l fx.Look, reduced bool) fx.Look {
	t := p.flash
	if t >= flashHold+flashFade {
		return l
	}
	var k float64
	switch {
	case reduced:
		k = flashSoft * math.Sin(math.Pi*float64(t)/(flashHold+flashFade))
	case t < flashHold:
		k = flashMix
	default:
		k = flashMix * (1 - float64(t-flashHold)/flashFade)
	}
	day := gfx.RGBA(resurrect64.White)
	m := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*k)) }
	l.Ambient = color.RGBA{m(l.Ambient.R, day.R), m(l.Ambient.G, day.G), m(l.Ambient.B, day.B), 255}
	l.Level += (1 - l.Level) * k
	return l
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
