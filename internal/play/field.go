package play

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/debug"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// Road markings, in pixels.
const (
	dashLen = 12 // a lane dash and the gap after it
	dashGap = 12
)

// pieces are the sprites a vehicle is built from, left to right as it
// faces right. Vehicles not listed are one sprite named by their ID
// (DESIGN §3.3: long vehicles are built from end, middle and end pieces).
var pieces = map[string][]string{
	config.Truck.ID: {"truck_back", "truck_mid", "truck_front"},
}

// vehicleSprites is the sprites for vehicle v, left to right facing right.
func vehicleSprites(v config.Vehicle) []string {
	if ps, ok := pieces[v.ID]; ok {
		return ps
	}
	return []string{v.ID}
}

// drawField draws the rows (grass where it is safe, road where it is
// not), the traffic, and the hero. Everything on the field is clipped to
// it, so vehicles slide in and out at its edges.
func (p *Play) drawField(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	field := dst.SubImage(image.Rect(config.FieldX, config.FieldY,
		config.FieldX+w.Cols*config.Cell, config.FieldY+w.Rows*config.Cell)).(*ebiten.Image)

	verge := ctx.Sheet(Sheet).Sprite("verge")
	for r := range w.Lanes {
		switch w.Lanes[r].Kind {
		case config.Safe:
			for c := range w.Cols {
				verge.Draw(field, 0, cellX(c), cellY(r), false)
			}
		case config.Road:
			p.drawRoad(field, r)
		}
	}
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.Road {
			p.drawTraffic(ctx, field, r)
		}
	}
	p.drawHero(ctx, field)

	if debug.Enabled {
		p.drawHitBoxes(ctx, field)
	}
}

// drawRoad draws row r as asphalt, with dashes where it meets another road
// lane and a solid edge where it meets anything else.
func (p *Play) drawRoad(dst *ebiten.Image, r int) {
	w := p.world
	x, y := float32(config.FieldX), float32(cellY(r))
	width := float32(w.Cols * config.Cell)
	vector.FillRect(dst, x, y, width, config.Cell, gfx.RGBA(resurrect64.Shadow), false)

	line := gfx.RGBA(resurrect64.Dusk)
	edge := func(other int, ly float32) {
		if other >= 0 && other < w.Rows && w.Lanes[other].Kind == config.Road {
			for dx := float32(0); dx < width; dx += dashLen + dashGap {
				vector.FillRect(dst, x+dx, ly, min(dashLen, width-dx), 1, line, false)
			}
			return
		}
		vector.FillRect(dst, x, ly, width, 1, line, false)
	}
	edge(r-1, y)
	if r+1 >= w.Rows || w.Lanes[r+1].Kind != config.Road {
		edge(r+1, y+config.Cell-1) // a shared edge is drawn once, by the lower lane
	}
}

// drawTraffic draws row r's vehicles, mirrored when the lane moves left.
func (p *Play) drawTraffic(ctx *scene.Context, dst *ebiten.Image, r int) {
	lane := &p.world.Lanes[r]
	sheet := ctx.Sheet(Sheet)
	flip := lane.Dir < 0
	y := cellY(r)
	for _, v := range lane.Vehicles {
		x := float64(config.FieldX) + math.Floor(float64(v.X())/config.SubPx)
		names := vehicleSprites(v.Kind)
		for i := range names {
			name := names[i]
			if flip {
				name = names[len(names)-1-i]
			}
			sp := sheet.Sprite(name)
			sp.Draw(dst, 0, x, y, flip)
			x += float64(sp.W)
		}
	}
}

// drawHero draws the hero turned to face its way, on its leap frame while
// hopping, rising and falling over the hop, then squashed for a moment on
// landing (DESIGN §3.3). A dead hero lies flat where it was hit.
func (p *Play) drawHero(ctx *scene.Context, dst *ebiten.Image) {
	pl := &p.world.Player
	sp := ctx.Sheet(Sheet).Sprite(p.hero.Sprite)

	frame, lift := frameRest, 0.0
	switch {
	case pl.Death != 0:
		frame = frameSquashed
	case pl.Hopping():
		frame = frameLeap
		t := float64(pl.HopProgress()) / config.HopTicks
		lift = math.Round(config.HopLiftPx * math.Sin(math.Pi*t))
	case pl.Still < config.LandTicks:
		frame = frameLand
	}

	x := float64(config.FieldX) + float64(pl.X)/config.SubPx
	y := float64(config.FieldY) + float64(pl.Y)/config.SubPx - lift
	var op ebiten.DrawImageOptions
	op.GeoM = facing(pl.Facing, sp.W, sp.H)
	op.GeoM.Translate(x, y)
	dst.DrawImage(sp.Frame(frame), &op)
}

// drawHitBoxes outlines what the rules see, in debug builds: the hero's
// collision box in the row it counts as being in, and each vehicle's.
func (p *Play) drawHitBoxes(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	pl := &w.Player
	c := ctx.Scheme.Highlight
	m := config.HeroMargin
	debug.Box(dst, float64(config.FieldX)+float64(pl.X)/config.SubPx+float64(m), cellY(pl.LaneRow()),
		float64(config.Cell-2*m), config.Cell, c)
	for r := range w.Lanes {
		lane := &w.Lanes[r]
		for _, v := range lane.Vehicles {
			hm := w.Skill.HitMargin
			x := float64(config.FieldX) + float64(v.X())/config.SubPx + float64(hm)
			debug.Box(dst, x, cellY(r), float64(v.Kind.Length-2*hm), config.Cell, c)
		}
	}
}

// facing turns a w×h sprite drawn facing up to face d, keeping it in the
// same w×h box. The turns are exact quarter turns, so pixels stay whole.
func facing(d world.Dir, w, h int) ebiten.GeoM {
	var g ebiten.GeoM
	fw, fh := float64(w), float64(h)
	switch d {
	case world.Right: // x' = h - y, y' = x
		g.SetElement(0, 0, 0)
		g.SetElement(0, 1, -1)
		g.SetElement(0, 2, fh)
		g.SetElement(1, 0, 1)
		g.SetElement(1, 1, 0)
	case world.Down: // x' = w - x, y' = h - y
		g.SetElement(0, 0, -1)
		g.SetElement(0, 2, fw)
		g.SetElement(1, 1, -1)
		g.SetElement(1, 2, fh)
	case world.Left: // x' = y, y' = w - x
		g.SetElement(0, 0, 0)
		g.SetElement(0, 1, 1)
		g.SetElement(1, 0, -1)
		g.SetElement(1, 1, 0)
		g.SetElement(1, 2, fw)
	}
	return g
}

// cellX and cellY are the screen position of a grid cell's top-left corner.
func cellX(c int) float64 { return float64(config.FieldX + c*config.Cell) }
func cellY(r int) float64 { return float64(config.FieldY + r*config.Cell) }
