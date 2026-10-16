package play

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// Demo is the title's night scene (DESIGN §7): the autopilot plays levels
// 1 to config.DemoLevels in turn, with no sound, shake or status bar. It
// has its own seed and session, so it leaves the game's RNG, the game in
// progress and the scores alone, and plays the same every time.
type Demo struct {
	play  *Play
	level int // the level being played
	ended int // ticks since the level was cleared or lost; 0 until then
	pilot world.Autopilot
}

// NewDemo starts the demo on level 1.
func NewDemo(*scene.Context) scene.Scene {
	d := &Demo{}
	d.start(1)
	return d
}

// start plays level n with a fresh session, seeded for that level.
func (d *Demo) start(n int) {
	s := session.New(config.Stroll, config.StartLives)
	d.play = build(s, config.StrollLevels, config.StrollLevels.Level(n), rand.New(rand.NewPCG(config.DemoSeed, uint64(n))))
	d.level, d.ended = n, 0
}

// World is the demo's game, for tests.
func (d *Demo) World() *world.World { return d.play.world }

// Update implements scene.Scene.
func (d *Demo) Update(*scene.Context) error {
	p, w := d.play, d.play.world
	p.ticks++
	p.updateEffects()
	if w.Cleared || w.Over {
		if d.ended++; d.ended > clearTicks {
			d.start(d.level%config.DemoLevels + 1)
		}
		return nil
	}
	for _, e := range w.Update(d.pilot.Next(w)) {
		p.effects(e)
	}
	return nil
}

// Draw implements scene.Scene: the playfield alone.
func (d *Demo) Draw(ctx *scene.Context, dst *ebiten.Image) {
	dst.Fill(gfx.RGBA(ctx.Scheme.Backdrop))
	d.play.drawField(ctx, dst)
}
