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

// The street sweepers on empty roads at dawn (DESIGN §6.7): their brushes
// turn and their beacons flash, a frame every sweeperFrameTicks; a
// beacon's amber light, and the shine left on the road by the night's
// rain.
const (
	sweeperFrameTicks = 6
	beaconR           = 40
	beaconLight       = 0.8
	beaconX           = 38 // the beacon in the sprite, facing right, in pixels
	wetRoad           = 0.7
)

// sweeperX is sweeper s's left end on the screen, in pixels.
func sweeperX(s *world.Sweeper) float64 {
	return float64(config.FieldX) + math.Floor(float64(s.X)/config.SubPx)
}

// eachSweeper calls draw with each street sweeper still sweeping, facing
// the way it goes.
func (p *Play) eachSweeper(ctx *scene.Context, draw func(pc piece)) {
	sp := ctx.Sheet(Sheet).Sprite("sweeper")
	frame := int(p.world.Tick/sweeperFrameTicks) % sp.Frames()
	for i := range p.world.Sweepers {
		if s := &p.world.Sweepers[i]; s.On {
			draw(piece{name: "sweeper", frame: frame, x: sweeperX(s), y: cellY(s.Row), flip: s.Dir < 0})
		}
	}
}

// addSweeperLights adds each sweeper's beacon, lit on alternate frames,
// the two out of step. A sweeper waiting off the field shows its beacon
// at the edge before it comes.
func (p *Play) addSweeperLights(l *fx.Lighting) {
	for i := range p.world.Sweepers {
		s := &p.world.Sweepers[i]
		if !s.On || (int(p.world.Tick/sweeperFrameTicks)+i)%2 != 0 {
			continue
		}
		x := sweeperX(s) + beaconX
		if s.Dir < 0 {
			x = sweeperX(s) + float64(config.SweeperCells*config.Cell-beaconX)
		}
		l.Add(fx.Light{X: x, Y: cellY(s.Row) + float64(config.Cell)/2, Radius: beaconR,
			Colour: gfx.RGBA(resurrect64.Amber), Strength: beaconLight})
	}
}

// wetK is how much the road shines from the night's rain on a bonus level
// with a wet road: only the look (DESIGN §6.7).
func (p *Play) wetK() float64 {
	if b := p.world.Level.Bonus; b != nil && b.Wet {
		return wetRoad
	}
	return 0
}
