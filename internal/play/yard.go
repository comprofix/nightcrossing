package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// A bonus level's goods yard and fruit (DESIGN §6.7): the rakes drawn as
// the train is, the fruit on them, and the last train pulling out when the
// clock runs out.
const (
	departTicks = 2 * config.TPS // TIME UP shows this long with the last train, as it pulls out
	departPull  = 0.1            // how fast it gathers speed, in pixels a tick, each tick
)

// rakePieces are rake k's sprites left to right as it faces right: its
// carriages, then its engine if it has one.
func rakePieces(k *world.Rake) []string {
	var ps []string
	for range k.Cars {
		ps = append(ps, "car_back")
		for range carMids {
			ps = append(ps, "car_mid")
		}
		ps = append(ps, "car_front")
	}
	if k.Engine {
		ps = append(ps, "engine_back", "engine_mid", "engine_nose")
	}
	return ps
}

// departing reports whether the last train is pulling out: the clock ran
// out on a level with one.
func (p *Play) departing() bool {
	return p.clear > 0 && p.ending == world.BonusOver && p.world.LastTrain() != nil
}

// clearFor is how long the end of the level shows before the tally: long
// enough to watch the last train go, on a level with one.
func (p *Play) clearFor() int {
	if p.departing() {
		return departTicks
	}
	return clearTicks
}

// rakeX is rake k's left end on the screen, in pixels: the last train
// gathering speed the way it faces as it pulls out.
func (p *Play) rakeX(k *world.Rake) float64 {
	x := float64(config.FieldX) + math.Floor(float64(k.X)/config.SubPx)
	if k.Last && p.departing() {
		t := float64(p.clear)
		x += float64(k.Facing) * math.Floor(departPull*t*t/2)
	}
	return x
}

// drawYard draws the rakes standing and shunting in the yard, mirrored
// when they face left.
func (p *Play) drawYard(ctx *scene.Context, dst *ebiten.Image) {
	sheet := ctx.Sheet(Sheet)
	for i := range p.world.Yard {
		k := &p.world.Yard[i]
		names := rakePieces(k)
		flip := k.Facing < 0
		x, y := p.rakeX(k), cellY(k.Row)
		for j := range names {
			name := names[j]
			if flip {
				name = names[len(names)-1-j]
			}
			sheet.Sprite(name).Draw(dst, 0, x+float64(j*config.Cell), y, flip)
		}
	}
}

// pickXY is where fruit i's sprite's top-left is drawn, centred in its
// cell; on the last train, it leaves with it.
func (p *Play) pickXY(sp *gfx.Sprite, i int) (x, y float64) {
	w := p.world
	x = float64(config.FieldX) + math.Floor(float64(w.PickX(i))/config.SubPx) + float64(config.Cell-sp.W)/2
	if k := w.LastTrain(); k != nil && w.Picks[i].Row == k.Row {
		x += p.rakeX(k) - (float64(config.FieldX) + math.Floor(float64(k.X)/config.SubPx))
	}
	return x, cellY(w.Picks[i].Row) + float64(config.Cell-sp.H)/2
}

// drawPicks draws the fruit still to be collected.
func (p *Play) drawPicks(ctx *scene.Context, dst *ebiten.Image) {
	sp := ctx.Sheet(Sheet).Sprite("fruit")
	for i, pk := range p.world.Picks {
		if p.world.Showing(i) {
			x, y := p.pickXY(sp, i)
			sp.Draw(dst, min(pk.Fruit, sp.Frames()-1), x, y, false)
		}
	}
}

// addPickLights adds the soft light on each fruit still to be collected,
// as on fruit in a nook (D59).
func (p *Play) addPickLights(ctx *scene.Context, l *fx.Lighting) {
	sp := ctx.Sheet(Sheet).Sprite("fruit")
	for i := range p.world.Picks {
		if p.world.Showing(i) {
			x, y := p.pickXY(sp, i)
			l.Add(fx.Light{X: x + float64(sp.W)/2, Y: y + float64(sp.H)/2, Radius: fruitR,
				Colour: gfx.RGBA(resurrect64.Gold), Strength: fruitLight})
		}
	}
}
