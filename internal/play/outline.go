package play

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
)

// outlineColour rings the hazards in high contrast (DESIGN §8).
const outlineColour = resurrect64.White

// drawOutlines rings every vehicle, heron, water rat and snake with a one-pixel line in
// outlineColour (DESIGN §8: high contrast). It draws over the lit field,
// so night does not dim the line.
func (p *Play) drawOutlines(ctx *scene.Context, dst *ebiten.Image) {
	if p.outline == nil {
		p.outline = ebiten.NewImage(p.size.X, p.size.Y) // in the game's coordinates, as the sprites are
	}
	p.outline.Clear()
	var solid colorm.ColorM
	c := gfx.RGBA(outlineColour)
	solid.Scale(0, 0, 0, 1)
	solid.Translate(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0)
	// Each hazard in the solid colour a pixel to each side, then cut its own
	// shape out, which leaves the ring.
	p.eachHazard(ctx, func(pc piece) {
		img := pc.image(ctx)
		for _, d := range [...][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			op := &colorm.DrawImageOptions{GeoM: spriteGeoM(img, pc.x+d[0], pc.y+d[1], pc.flip)}
			colorm.DrawImage(p.outline, img, solid, op)
		}
	})
	p.eachHazard(ctx, func(pc piece) {
		img := pc.image(ctx)
		op := &ebiten.DrawImageOptions{GeoM: spriteGeoM(img, pc.x, pc.y, pc.flip), Blend: ebiten.BlendDestinationOut}
		p.outline.DrawImage(img, op)
	})
	dst.DrawImage(p.outline, nil)
}

// drawSmoothOutlines rings the hazards as drawOutlines does, in the smooth
// look: a game pixel wide, round each one's smooth art, or its sprite
// enlarged until it has some.
func (p *Play) drawSmoothOutlines(ctx *scene.Context, dst *ebiten.Image, k int) {
	if p.outlineK == nil {
		p.outlineK = ebiten.NewImage(p.size.X*k, p.size.Y*k)
	}
	p.outlineK.Clear()
	var solid colorm.ColorM
	c := gfx.RGBA(outlineColour)
	solid.Scale(0, 0, 0, 1)
	solid.Translate(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0)
	p.eachHazard(ctx, func(pc piece) {
		img, g := p.smoothPiece(ctx, pc, k)
		for _, d := range [...][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			op := &colorm.DrawImageOptions{GeoM: g}
			op.GeoM.Translate(d[0]*float64(k), d[1]*float64(k))
			colorm.DrawImage(p.outlineK, img, solid, op)
		}
	})
	p.eachHazard(ctx, func(pc piece) {
		img, g := p.smoothPiece(ctx, pc, k)
		p.outlineK.DrawImage(img, &ebiten.DrawImageOptions{GeoM: g, Blend: ebiten.BlendDestinationOut})
	})
	dst.DrawImage(p.outlineK, nil)
}

// eachHazard calls draw with every piece that can kill the hero by touch:
// each piece of a vehicle and of the train, each heron, and each water rat
// and snake.
func (p *Play) eachHazard(ctx *scene.Context, draw func(pc piece)) {
	p.eachTrainPiece(ctx, draw)
	p.eachProwler(ctx, draw)
	p.eachSiren(ctx, draw)
	for r := range p.world.Lanes {
		p.eachHeron(r, draw)
		if p.world.Lanes[r].Kind == config.Road {
			p.eachEntityPiece(ctx, r, draw)
		}
	}
}

// spriteGeoM places a sprite frame's top-left corner at x, y, mirrored with
// flip, as gfx.Sprite.Draw does.
func spriteGeoM(img *ebiten.Image, x, y float64, flip bool) ebiten.GeoM {
	var g ebiten.GeoM
	if flip {
		g.Scale(-1, 1)
		g.Translate(float64(img.Bounds().Dx()), 0)
	}
	g.Translate(x, y)
	return g
}
