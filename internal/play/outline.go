package play

import (
	"math"

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
		pr := config.Project
		p.outline = ebiten.NewImage(pr.ScreenWidth, pr.ScreenHeight)
	}
	p.outline.Clear()
	var solid colorm.ColorM
	c := gfx.RGBA(outlineColour)
	solid.Scale(0, 0, 0, 1)
	solid.Translate(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0)
	// Each hazard in the solid colour a pixel to each side, then cut its own
	// shape out, which leaves the ring.
	p.eachHazard(ctx, func(img *ebiten.Image, x, y float64, flip bool) {
		for _, d := range [...][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			op := &colorm.DrawImageOptions{GeoM: spriteGeoM(img, x+d[0], y+d[1], flip)}
			colorm.DrawImage(p.outline, img, solid, op)
		}
	})
	p.eachHazard(ctx, func(img *ebiten.Image, x, y float64, flip bool) {
		op := &ebiten.DrawImageOptions{GeoM: spriteGeoM(img, x, y, flip), Blend: ebiten.BlendDestinationOut}
		p.outline.DrawImage(img, op)
	})
	dst.DrawImage(p.outline, nil)
}

// eachHazard calls draw with the frame, place and facing of every sprite
// that can kill the hero by touch: each piece of a vehicle and of the
// train, each heron, and each water rat and snake.
func (p *Play) eachHazard(ctx *scene.Context, draw func(img *ebiten.Image, x, y float64, flip bool)) {
	p.eachTrainPiece(ctx, draw)
	p.eachProwler(ctx, draw)
	sheet := ctx.Sheet(Sheet)
	for r := range p.world.Lanes {
		lane := &p.world.Lanes[r]
		flip := lane.Dir < 0
		p.eachHeron(ctx, r, draw)
		for _, e := range lane.Entities {
			if e.Out {
				continue
			}
			x := float64(config.FieldX) + math.Floor(float64(e.X())/config.SubPx)
			if lane.Kind != config.Road {
				continue
			}
			names := entitySprites(e.Kind)
			n := len(names)
			for i := range names {
				name := names[i]
				if flip {
					name = names[n-1-i]
				}
				sp := sheet.Sprite(name)
				draw(sp.Frame(min(wheelFrame(e), sp.Frames()-1)), x, cellY(r), flip)
				x += float64(sp.W)
			}
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
