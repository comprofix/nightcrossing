package play

import (
	"image/color"
	"math"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
)

// skyLook is how the field is lit at one time of day (DESIGN §6.2, D39).
type skyLook struct {
	ambient  int     // the light everywhere, a palette colour
	level    float64 // how much of it
	lamps    float64 // how strong lamps, headlights and the moon are (0–1)
	glowFrom float64 // the brightness the glow starts at, higher by day
}

// skyLooks are the times of day. Dawn is cool pink and blue, day is plain
// daylight with the lamps faded, dusk is warm, and night is a little
// darker than Stroll's (D39).
var skyLooks = [...]skyLook{
	config.Dawn:  {ambient: resurrect64.Orchid, level: 0.85, lamps: 0.5, glowFrom: 0.75},
	config.Day:   {ambient: resurrect64.White, level: 1, lamps: 0.15, glowFrom: 0.95},
	config.Dusk:  {ambient: resurrect64.Apricot, level: 0.8, lamps: 0.7, glowFrom: 0.7},
	config.Night: {ambient: resurrect64.Sky, level: 0.5, lamps: 1, glowFrom: night.GlowFrom},
}

// The clock counts in ticks, config.DayEaseTicks to a stage, so it moves
// the same every play. skyBefore is where it stands as a level with the
// cycle starts: the night before its dawn, so each level opens with the
// sky brightening.
const skyBefore = -config.DayEaseTicks

// updateSky eases the clock toward the time of day for the nooks filled,
// one stage each config.DayEaseTicks. It only ever moves forward.
func (p *Play) updateSky() {
	if !p.world.Level.DayCycle {
		return
	}
	filled := 0
	for _, f := range p.world.Filled {
		if f {
			filled++
		}
	}
	p.sky = min(p.sky+1, int(skyFor(p.world.Level, filled))*config.DayEaseTicks)
}

// sky is the light at a point on the clock, mixed between the stages
// either side of it.
type sky struct {
	ambient                color.RGBA
	level, lamps, glowFrom float64
}

// skyAt is the light at clock position at, in stages: 1.5 is halfway
// from day to dusk, and -1 is the night before dawn. The
// ambient colour is mixed at full strength and scaled by the mixed level,
// so a dark stage's colour doesn't wash the next one grey.
func skyAt(at float64) sky {
	stage := func(i int) skyLook {
		if i < 0 {
			return skyLooks[config.Night]
		}
		return skyLooks[min(i, len(skyLooks)-1)]
	}
	lo := math.Floor(at)
	a, b, k := stage(int(lo)), stage(int(lo)+1), at-lo
	mix := func(x, y float64) float64 { return x + (y-x)*k }
	ca, cb := gfx.RGBA(a.ambient), gfx.RGBA(b.ambient)
	m := func(x, y uint8) uint8 { return uint8(math.Round(mix(float64(x), float64(y)))) }
	return sky{
		ambient:  color.RGBA{m(ca.R, cb.R), m(ca.G, cb.G), m(ca.B, cb.B), 255},
		level:    mix(a.level, b.level),
		lamps:    mix(a.lamps, b.lamps),
		glowFrom: mix(a.glowFrom, b.glowFrom),
	}
}

// look is how the field is lit now: Stroll's fixed night, or the day and
// night cycle where the clock stands.
func (p *Play) look() fx.Look {
	l := night
	if p.world.Level.DayCycle {
		s := skyAt(p.stage())
		l.Ambient, l.Level, l.GlowFrom = s.ambient, s.level, s.glowFrom
	}
	return overcast(l, p.overcastK())
}

// lampK is how strong lamps, headlights, the moon and the hero's own light
// are now: full at night and in fog or rain, faded by day. Warnings, herons and the crossing
// lights keep their full strength at every time of day (D39).
func (p *Play) lampK() float64 {
	k := 1.0
	if p.world.Level.DayCycle {
		k = skyAt(p.stage()).lamps
	}
	return k + (1-k)*p.overcastK() // fog and rain turn them on (D44)
}

// stage is the clock in stages, for skyAt.
func (p *Play) stage() float64 { return float64(p.sky) / config.DayEaseTicks }

// skyFor is the time of day level is lit for with filled nooks filled: as
// the cycle has it, or the late afternoon all through on a bonus level
// lit for dusk (DESIGN §6.7).
func skyFor(level config.Level, filled int) config.TimeOfDay {
	if b := level.Bonus; b != nil && b.Dusk {
		return config.Dusk
	}
	return config.DayCycle[filled]
}

// skyStart is where the clock stands as level starts: the night before
// its dawn, so each level opens with the sky brightening, or already dusk
// on a bonus level lit for it.
func skyStart(level config.Level) int {
	if b := level.Bonus; b != nil && b.Dusk {
		return int(config.Dusk) * config.DayEaseTicks
	}
	return skyBefore
}
