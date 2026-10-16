package play

import (
	"image/color"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/world"
)

// colours is palette colours by name, as RGBA.
func colours(cs ...int) []color.RGBA {
	out := make([]color.RGBA, len(cs))
	for i, c := range cs {
		out[i] = gfx.RGBA(c)
	}
	return out
}

// The particle bursts (DESIGN §3.3: splashes, sparks and dust puffs).
var (
	hopDust = fx.Burst{Count: 4, Speed: 0.6, Drag: 0.9, Life: 14, Size: 1,
		Colours: colours(resurrect64.Khaki, resurrect64.Fern)}
	splashDrops = fx.Burst{Count: 16, Speed: 1.4, Up: 1.6, Gravity: 0.14, Life: 32, Size: 2,
		Colours: colours(resurrect64.Sky, resurrect64.Seafoam, resurrect64.Mist)}
	crashSparks = fx.Burst{Count: 12, Speed: 2.4, Up: 0.6, Gravity: 0.08, Drag: 0.92, Life: 22, Size: 1,
		Colours: colours(resurrect64.Gold, resurrect64.Lemon, resurrect64.Amber)}
	bumpDust = fx.Burst{Count: 8, Speed: 1, Drag: 0.88, Life: 24, Size: 2,
		Colours: colours(resurrect64.Fog, resurrect64.Mist)}
	trainSparks = fx.Burst{Count: 28, Speed: 3.2, Up: 0.8, Gravity: 0.08, Drag: 0.92, Life: 30, Size: 2,
		Colours: colours(resurrect64.Gold, resurrect64.Lemon, resurrect64.Amber, resurrect64.White)}
	feathers = fx.Burst{Count: 8, Speed: 1.2, Up: 0.4, Gravity: 0.03, Drag: 0.9, Life: 40, Size: 2,
		Colours: colours(resurrect64.White, resurrect64.Fog)}
	goalSparkles = fx.Burst{Count: 18, Speed: 1.3, Up: 0.8, Gravity: 0.02, Drag: 0.95, Life: 42, Size: 1,
		Colours: colours(resurrect64.Gold, resurrect64.Lemon, resurrect64.Honey)}
	oneUpSparkles = fx.Burst{Count: 14, Speed: 1.1, Up: 1, Gravity: 0.02, Drag: 0.95, Life: 40, Size: 1,
		Colours: colours(resurrect64.Leaf, resurrect64.Lime, resurrect64.White)}
)

// at is burst b from the given point.
func at(b fx.Burst, x, y float64) fx.Burst {
	b.X, b.Y = x, y
	return b
}

// effects starts the particles and glows for this tick's events.
func (p *Play) effects(e world.Event) {
	w := p.world
	pl := &w.Player
	x, y := p.heroCentre()
	switch e {
	case world.Hopped:
		if w.Lanes[pl.FromRow].Kind != config.River {
			p.parts.Emit(at(hopDust, x, cellY(pl.FromRow)+config.Cell-4))
		}
	case world.Splashed, world.Swept:
		p.parts.Emit(at(splashDrops, x, cellY(pl.LaneRow())+config.Cell/2))
	case world.Squashed:
		p.parts.Emit(at(crashSparks, x, y))
	case world.Struck:
		p.parts.Emit(at(trainSparks, x, y))
	case world.Bumped:
		p.parts.Emit(at(bumpDust, x, y))
	case world.Caught:
		p.parts.Emit(at(feathers, x, y))
	case world.Goal:
		if n := world.NookAt(pl.Col); n >= 0 {
			nx := cellX(config.Nooks[n]) + float64(config.NookWidth*config.Cell)/2
			p.parts.Emit(at(goalSparkles, nx, cellY(config.GoalRow)+config.Cell/2))
			p.nookGlow[n] = goalGlowTick
		}
	case world.ExtraLife:
		p.parts.Emit(at(oneUpSparkles, x, y))
	}
}

// updateEffects runs the particles and fades the nook glows one tick.
func (p *Play) updateEffects() {
	p.parts.Update()
	p.updatePops()
	p.updateSky()
	for i := range p.nookGlow {
		p.nookGlow[i] = max(p.nookGlow[i]-1, 0)
	}
}
