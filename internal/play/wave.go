package play

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The traffic lights' frames (DESIGN §6.3, D43).
const (
	lightGreen = 0
	lightAmber = 1
	lightRed   = 2
)

// The traffic lights' glow on the edge of their lane, while amber or red.
const (
	trafficLightR = 44
	trafficLight  = 1.3
)

// trafficFrame is road row r's traffic light: amber while a wave is
// announced on it, red through the wave, green otherwise.
func (p *Play) trafficFrame(r int) int {
	s := p.world.Spell(config.Wave)
	if s == nil || s.Lane != r {
		return lightGreen
	}
	switch s.Phase {
	case world.SpellWarning:
		return lightAmber
	case world.SpellOn:
		return lightRed
	}
	return lightGreen
}

// fromLeft reports whether road row r's traffic comes from the left.
func (p *Play) fromLeft(r int) bool { return p.world.Lanes[r].Dir > 0 }

// drawTrafficLights draws a traffic light in the margin beside each road
// lane, on the side its traffic comes from, in a level with rush hour
// waves (D43). They stand outside the field, so they never hide a
// vehicle.
func (p *Play) drawTrafficLights(ctx *scene.Context, dst *ebiten.Image) {
	if p.world.Spell(config.Wave) == nil {
		return
	}
	sp := ctx.Sheet(Sheet).Sprite("traffic_light")
	for r := range p.world.Lanes {
		if p.world.Lanes[r].Kind != config.Road {
			continue
		}
		x := float64(config.FieldX + config.Cols*config.Cell)
		if p.fromLeft(r) {
			x = float64(config.FieldX - sp.W)
		}
		sp.Draw(dst, p.trafficFrame(r), x, cellY(r)+float64(config.Cell-sp.H)/2, false)
	}
}

// addWaveLights washes the edge of a lane whose light is amber or red with
// its colour, so the cue shows on the field too.
func (p *Play) addWaveLights(l *fx.Lighting) {
	s := p.world.Spell(config.Wave)
	if s == nil || s.Phase == world.SpellQuiet {
		return
	}
	c := resurrect64.Amber
	if p.trafficFrame(s.Lane) == lightRed {
		c = resurrect64.Red
	}
	x := float64(config.FieldX + config.Cols*config.Cell)
	if p.fromLeft(s.Lane) {
		x = float64(config.FieldX)
	}
	l.Add(fx.Light{X: x, Y: cellY(s.Lane) + float64(config.Cell)/2, Radius: trafficLightR,
		Colour: gfx.RGBA(c), Strength: trafficLight})
}

// eventMessage is the hint strip's warning for a random event being
// announced, or "" (DESIGN §6.1).
func (p *Play) eventMessage() string {
	for _, s := range p.world.Spells {
		if s.Phase == world.SpellWarning {
			if s.Kind == config.Gust {
				return config.GustMessage(s.Dir) // which way it blows (D47)
			}
			return config.Events[s.Kind].Message
		}
	}
	return ""
}
