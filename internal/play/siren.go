package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// An emergency vehicle's look and sound (DESIGN §6.9, D73): the two
// colours of its lights, which take turns, and its siren, heard quietly
// at first and then close by.
type siren struct {
	colours   [2]int
	bar       int // the middle of its light bar, in pixels from its back
	far, near *audio.Effect
	horn      *audio.Effect // a blast as it is announced, or nil
}

// sirens are the vehicles' looks and sounds, in config.Responders' order.
var sirens = [...]siren{
	{colours: [2]int{resurrect64.Red, resurrect64.Blue}, bar: 23, far: PoliceFar, near: PoliceNear},
	{colours: [2]int{resurrect64.Red, resurrect64.White}, bar: 46, far: AmbulanceFar, near: AmbulanceNear},
	{colours: [2]int{resurrect64.Red, resurrect64.Amber}, bar: 67, far: FireFar, near: FireNear, horn: FireHorn},
}

// The lights' flashing, and their glow at the lane's edge and on the
// vehicle; and how long the siren is heard from afar before close by.
const (
	sirenFlashTicks = 8
	sirenEdgeR      = 72
	sirenEdge       = 2.4
	sirenBarR       = 28
	sirenBar        = 1.2
	sirenFarTicks   = config.TPS
	brakeR          = 14 // a queue's brake lights, brighter than its tail lights
	brakeLight      = 1.2
	dippedLight     = 0.4 // and its headlamps, dipped: no beams
)

// sirenLit is which of the vehicle's two lights is lit this tick, the
// same as its sprite's frame.
func (p *Play) sirenLit() int { return int(p.world.Tick/sirenFlashTicks) % 2 }

// eachSiren calls draw with the emergency vehicle's frame, place and
// facing, while it comes through its lane.
func (p *Play) eachSiren(ctx *scene.Context, draw func(img *ebiten.Image, x, y float64, flip bool)) {
	s := p.world.Siren
	if s.Phase != world.SirenRun {
		return
	}
	sp := ctx.Sheet(Sheet).Sprite(s.Responder().ID)
	x := float64(config.FieldX) + math.Floor(float64(s.X)/config.SubPx)
	draw(sp.Frame(p.sirenLit()), x, cellY(s.Row), p.world.Lanes[s.Row].Dir < 0)
}

// drawSiren draws the emergency vehicle coming through.
func (p *Play) drawSiren(ctx *scene.Context, dst *ebiten.Image) {
	p.eachSiren(ctx, func(img *ebiten.Image, x, y float64, flip bool) {
		dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: spriteGeoM(img, x, y, flip)})
	})
}

// addSirenLights adds the emergency's lights (D72): through the warning
// and the run, its two colours flash in turn at the field's edge the
// vehicle comes in by, and on its light bar as it comes through, with its
// headlights; and a queue's cars show their brake lights while they stand
// (see addVehicleLights).
func (p *Play) addSirenLights(l *fx.Lighting) {
	s := p.world.Siren
	if !s.Lights() {
		return
	}
	look := sirens[s.Vehicle]
	c := gfx.RGBA(look.colours[p.sirenLit()])
	y := cellY(s.Row) + float64(config.Cell)/2
	right := p.world.Lanes[s.Row].Dir < 0
	edge := float64(config.FieldX + config.Cell/2) // just inside the field, where it shows
	if right {
		edge = float64(config.FieldX + config.Cols*config.Cell - config.Cell/2)
	}
	l.Add(fx.Light{X: edge, Y: y, Radius: sirenEdgeR, Colour: c, Strength: sirenEdge})
	if s.Phase != world.SirenRun {
		return
	}
	n := float64(s.Responder().Length)
	x := float64(config.FieldX) + math.Floor(float64(s.X)/config.SubPx)
	bar, front := x+float64(look.bar), x+n
	if right {
		bar, front = x+n-float64(look.bar), x
	}
	l.Add(fx.Light{X: bar, Y: y, Radius: sirenBarR, Colour: c, Strength: sirenBar})
	l.Add(fx.Light{Shape: fx.Beam, X: front, Y: y, Radius: headlightR, Flip: right,
		Colour: gfx.RGBA(resurrect64.Lemon), Strength: headlight})
}

// sirenEffect is the siren's sound this tick, or nil: again and again
// through the warning, quietly at first, then close by as the vehicle
// comes, until it has gone or has run the hero down (D72). It starts with
// its event (soundsFor).
func (p *Play) sirenEffect() *audio.Effect {
	s := p.world.Siren
	if !s.Lights() || p.sirenCut || s.T == 0 {
		return nil
	}
	look := sirens[s.Vehicle]
	if s.T%int(audio.Length(look.near.Part.Notes)/audio.Ticks(1)) != 0 {
		return nil
	}
	if s.Phase == world.SirenWarning && s.T < sirenFarTicks {
		return look.far
	}
	return look.near
}

// sirenSounds are the effects that announce the vehicle: its siren from
// afar, and the fire truck's horn.
func (p *Play) sirenSounds() []*audio.Effect {
	look := sirens[p.world.Siren.Vehicle]
	if look.horn != nil {
		return []*audio.Effect{look.far, look.horn}
	}
	return []*audio.Effect{look.far}
}
