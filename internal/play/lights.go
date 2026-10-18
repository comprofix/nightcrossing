package play

import (
	"math"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// night is how Stroll's fixed night looks (DESIGN §3.3: the whole scene is
// darkened to a fixed night level, and light sources are drawn on top).
var night = fx.Look{
	Ambient:  gfx.RGBA(resurrect64.Sky),
	Level:    0.6,
	Max:      1.15,
	Bands:    6,
	Glow:     1.5, // the Glow option turns it off (DESIGN §8)
	GlowFrom: 0.5,
}

// highContrastLevel is the night's light in high contrast: the darkness
// turned down (DESIGN §8).
const highContrastLevel = 0.85

// lamp is where a street lamp stands.
type lamp struct{ col, row int }

// Lamps along the verge and the start row: scenery that lights the edges of
// the road and river (DESIGN §3.3: street lamps).
var lamps = []lamp{
	{3, 6}, {12, 6}, {21, 6},
	{7, config.StartRow}, {17, config.StartRow},
}

// bonusLamps are the fewer lamps of a bonus level's calmer night, so its
// fireflies glow over the field (DESIGN §6.7).
var bonusLamps = []lamp{{12, 6}, {7, config.StartRow}, {17, config.StartRow}}

// streetLamps are the level's street lamps.
func (p *Play) streetLamps() []lamp {
	if p.world.Bonus() {
		return bonusLamps
	}
	return lamps
}

// Light sizes in pixels, and strengths (0–1).
const (
	heroLightR     = 40 // the hero's own small light, so it can always be seen
	heroLight      = 0.55
	lampR          = 64
	lampLight      = 0.55
	headlightR     = 76 // a vehicle's beam
	bikeLightR     = 48
	headlight      = 0.75
	lensR          = 8 // the glow at a headlamp itself
	tailR          = 10
	tailLight      = 0.7
	heronR         = 20 // every hazard has its own light, herons too
	heronLight     = 0.35
	warnR          = 18 // a diving pad or sinking ship's warning
	warnLight      = 0.7
	cabinR         = 14 // the ferry's cabin windows and a barge's lantern
	cabinLight     = 0.45
	nookR          = 22 // a filled nook glows (DESIGN §3.3)
	nookLight      = 0.3
	goalGlowR      = 48 // and flares as the toad arrives
	goalGlow       = 0.9
	goalGlowTick   = 45
	flickerTicks   = 4 // lights flicker in steps this long (DESIGN §3.3)
	fruitR         = 16
	fruitLight     = 0.35
	fireflyR       = 20 // the firefly's tail, pulsing (D58)
	fireflyLight   = 0.4
	fireflyPulse   = 50 // ticks for each pulse
	fireflyStagger = 17 // ticks out of step from one firefly to the next (D61)
	fireflyTailX   = 3  // the middle of its tail in its sprite, flying right
	fireflyTailY   = 8
	sunGlintR      = 12 // the sun on a dragonfly's wings, now and then
	sunGlint       = 0.5
	sunGlintTicks  = 40 // a glint every so many ticks
	sunGlintFor    = 6  // for so many
)

// addFlyerLight adds the light of firefly f, the i'th of the flyers, its
// tail glowing and pulsing, each a little out of step with the others; or,
// by day, the glint of the sun on a dragonfly's wings (D58, D61).
func (p *Play) addFlyerLight(ctx *scene.Context, l *fx.Lighting, f *world.Flyer, i int) {
	sp, _, x, y, flip := p.flyer(ctx, f)
	if sp == nil {
		return
	}
	t := int(p.world.Tick)
	if f.Day {
		if t%sunGlintTicks < sunGlintFor {
			l.Add(fx.Light{X: x + float64(sp.W)/2, Y: y + float64(sp.H)/2, Radius: sunGlintR,
				Colour: gfx.RGBA(resurrect64.Seafoam), Strength: sunGlint})
		}
		return
	}
	tx := float64(fireflyTailX)
	if flip {
		tx = float64(sp.W - fireflyTailX)
	}
	k := 0.6 + 0.4*math.Sin(2*math.Pi*float64(t+i*fireflyStagger)/fireflyPulse)
	l.Add(fx.Light{X: x + tx, Y: y + fireflyTailY, Radius: fireflyR,
		Colour: gfx.RGBA(resurrect64.Lemon), Strength: fireflyLight * k})
}

// addLights adds every light on the field: the hero's, the lamps, the
// moon on the water, each vehicle's headlight and tail light, the ferry
// and barges, warnings and herons, the crossing signals and the train,
// and the nooks.
func (p *Play) addLights(ctx *scene.Context, l *fx.Lighting) {
	w := p.world
	cell := float64(config.Cell)
	fade := p.lampK() // faded by day (D39)
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.River { // the moon's path on the water
			l.Add(fx.Light{X: float64(config.FieldX + moonX), Y: cellY(r) + cell/2,
				Radius: moonR, Colour: gfx.RGBA(resurrect64.Sky), Strength: moonLight * fade})
		}
	}
	for i, lp := range p.streetLamps() {
		l.Add(fx.Light{X: cellX(lp.col) + cell/2, Y: cellY(lp.row) + lampHeadY,
			Radius: lampR, Colour: gfx.RGBA(resurrect64.Honey), Strength: lampLight * p.streetLampK(i)})
	}
	for r := range w.Lanes {
		lane := &w.Lanes[r]
		switch lane.Kind {
		case config.Road:
			p.addVehicleLights(ctx, l, r)
		case config.River:
			p.addRiverLights(ctx, l, r)
		}
	}
	p.addRailLights(l)
	p.addWaveLights(l)
	for i, c := range config.Nooks {
		x, y := cellX(c)+float64(config.NookWidth*config.Cell)/2, cellY(config.GoalRow)+cell/2
		if w.Filled[i] {
			l.Add(fx.Light{X: x, Y: y, Radius: nookR, Colour: gfx.RGBA(resurrect64.Gold), Strength: nookLight * fade})
		}
		if g := p.nookGlow[i]; g > 0 {
			l.Add(fx.Light{X: x, Y: y, Radius: goalGlowR, Colour: gfx.RGBA(resurrect64.Gold),
				Strength: goalGlow * float64(g) / goalGlowTick * fade})
		}
	}
	if n := w.FruitNook; n >= 0 { // a soft light on the fruit (D59)
		x, y := cellX(config.Nooks[n])+float64(config.NookWidth*config.Cell)/2, cellY(config.GoalRow)+cell/2
		l.Add(fx.Light{X: x, Y: y, Radius: fruitR, Colour: gfx.RGBA(resurrect64.Gold), Strength: fruitLight * fade})
	}
	p.addFoxLight(l)
	p.addPickLights(ctx, l)
	p.addSweeperLights(l)
	for i, fl := range p.flyers() {
		p.addFlyerLight(ctx, l, fl, i)
	}
	if pl := &w.Player; w.Home == 0 && pl.Death == 0 {
		x, y := p.heroCentre()
		l.Add(fx.Light{X: x, Y: y, Radius: heroLightR, Colour: gfx.RGBA(resurrect64.Honey), Strength: heroLight * fade})
	}
}

// addVehicleLights adds a headlight beam ahead of each vehicle in road
// row r and a tail light behind it.
func (p *Play) addVehicleLights(ctx *scene.Context, l *fx.Lighting, r int) {
	lane := &p.world.Lanes[r]
	y := cellY(r) + float64(config.Cell)/2
	for i, e := range lane.Entities {
		if e.Out {
			continue
		}
		x := float64(config.FieldX) + math.Floor(float64(e.X())/config.SubPx)
		front, back := x+float64(e.Kind.Length), x
		if lane.Dir < 0 {
			front, back = x, x+float64(e.Kind.Length)
		}
		reach := float64(headlightR)
		if e.Kind.ID == config.Bike.ID {
			reach = bikeLightR
		}
		k := flicker(ctx.Tick, r, i) * p.lampK()
		l.Add(fx.Light{Shape: fx.Beam, X: front, Y: y, Radius: reach, Flip: lane.Dir < 0,
			Colour: gfx.RGBA(resurrect64.Lemon), Strength: headlight * k})
		l.Add(fx.Light{X: front, Y: y, Radius: lensR, Colour: gfx.RGBA(resurrect64.Lemon), Strength: k})
		l.Add(fx.Light{X: back, Y: y, Radius: tailR, Colour: gfx.RGBA(resurrect64.Red), Strength: tailLight * p.lampK()})
	}
}

// addRiverLights adds the lights on river row r: a warning glow on each
// flashing pad or ship piece, the ferry's cabin and a barge's lantern, and
// a cold glow on each heron, so night never hides a danger.
func (p *Play) addRiverLights(ctx *scene.Context, l *fx.Lighting, r int) {
	lane := &p.world.Lanes[r]
	cell := float64(config.Cell)
	y := cellY(r) + cell/2
	for _, e := range lane.Entities {
		x := float64(config.FieldX) + math.Floor(float64(e.X())/config.SubPx)
		names := entitySprites(e.Kind)
		flash := e.Stage() == world.Warning && stageFrame(world.Warning, e.FlashPhase()) == frameFlash
		for i, name := range names {
			if lane.Dir < 0 {
				name = names[len(names)-1-i]
			}
			cx := x + float64(i)*cell + cell/2
			if flash {
				l.Add(fx.Light{X: cx, Y: y, Radius: warnR, Colour: gfx.RGBA(resurrect64.Coral), Strength: warnLight})
			}
			if st := e.Stage(); st == world.Afloat || st == world.Warning {
				switch name {
				case "ferry_cabin", "barge_back":
					l.Add(fx.Light{X: cx, Y: y, Radius: cabinR, Colour: gfx.RGBA(resurrect64.Gold), Strength: cabinLight * p.lampK()})
				}
			}
		}
		if e.Heron {
			hx := x + float64(e.HeronX())/config.SubPx + cell/2
			l.Add(fx.Light{X: hx, Y: y, Radius: heronR, Colour: gfx.RGBA(resurrect64.Seafoam), Strength: heronLight})
			if heronFlashing(e) { // its warning, like a sinking ship's (D67)
				l.Add(fx.Light{X: hx, Y: y, Radius: warnR, Colour: gfx.RGBA(resurrect64.Coral), Strength: warnLight})
			}
		}
	}
}

// flicker is a light's strength this tick, from 0.85 to 1, changing every
// few ticks. Each light flickers in its own way, the same every game.
func flicker(tick int64, row, i int) float64 {
	return 0.85 + 0.15*float64(noise(tick/flickerTicks, int64(row), int64(i))%1000)/999
}
