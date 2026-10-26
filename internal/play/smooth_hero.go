package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The hero in the smooth look (DESIGN §3.6): the toad in each of its frames,
// the toadlets with and without their school caps, and the splash, each
// drawn from shapes in its sprite's place, size and colours.

// toadPose is a frame of the toad sprite, facing up in its 24px box: where
// its eyes are, its body, the patch on its back, and its left legs, each a
// line from the body out to the toe, with how thick it is. The right legs
// are the left ones mirrored.
type toadPose struct {
	eyeY   float64    // the top of the eyes
	closed bool       // eyes shut: squashed flat
	body   [4]float64 // left, top, right, bottom
	rad    float64    // how round the body's corners are
	shade  float64    // how deep the darker band round its rump is
	patch  [4]float64 // left, top, right, bottom of the patch on its back
	point  bool       // whether the patch comes to a point below
	legs   [][][2]float64
	widths []float64 // each leg's thickness
}

// toadPoses are the toad sprite's frames, in its order: rest, leap, land,
// squashed, crouch, stretch, tuck.
var toadPoses = [...]toadPose{
	frameRest: {eyeY: 2, body: [4]float64{4.2, 4.6, 19.8, 19}, rad: 5, shade: 3,
		patch: [4]float64{9, 8, 15, 13}, point: true,
		legs: [][][2]float64{
			{{5.4, 10.2}, {3.4, 8.4}, {2.6, 7.6}},
			{{5.5, 16}, {3, 16.8}, {2.4, 18.2}, {4, 19.6}, {6.2, 19.6}},
		}, widths: []float64{1.2, 2}},
	frameLeap: {eyeY: 0, body: [4]float64{4.2, 2.6, 19.8, 16}, rad: 5.5, shade: 2.5,
		patch: [4]float64{9, 6, 15, 10}, point: true,
		legs: [][][2]float64{
			{{5.4, 6.2}, {3.4, 4.4}, {2.6, 3.6}},
			{{6.6, 14.5}, {6.2, 17}, {5, 19.6}, {5, 21.3}},
			{{3.8, 21.4}, {6.9, 21.4}},
		}, widths: []float64{1.2, 1.3, 1.2}},
	frameLand: {eyeY: 4, body: [4]float64{3.2, 6.6, 20.8, 16}, rad: 4, shade: 2.5,
		patch: [4]float64{8, 8, 16, 11}, point: true,
		legs: [][][2]float64{
			{{4, 10}, {2, 9}, {1.6, 8.2}},
			{{5, 14.3}, {2.2, 15.4}, {1, 16.6}},
			{{2.2, 15.4}, {3, 16.6}},
		}, widths: []float64{1.2, 1.4, 1}},
	frameSquashed: {eyeY: 7, closed: true, body: [4]float64{4.2, 8.8, 19.8, 17}, rad: 3, shade: 2.4,
		patch: [4]float64{9, 12, 15, 13.2},
		legs: [][][2]float64{
			{{5, 12}, {2, 10.8}, {1, 10.2}},
			{{5, 13}, {2, 14.2}, {1, 14.8}},
		}, widths: []float64{1.6, 1.6}},
	frameCrouch: {eyeY: 3, body: [4]float64{4.2, 5.6, 19.8, 17}, rad: 5, shade: 3,
		patch: [4]float64{9, 8, 15, 12}, point: true,
		legs: [][][2]float64{
			{{5.4, 11.2}, {3.4, 9.4}, {2.6, 8.6}},
			{{5.6, 14.8}, {2.6, 15.8}, {3, 17.4}, {6.4, 17.6}},
		}, widths: []float64{1.2, 2.2}},
	frameStretch: {eyeY: 1, body: [4]float64{4.2, 3.6, 19.8, 15}, rad: 5.5, shade: 2.5,
		patch: [4]float64{9, 6, 15, 10}, point: true,
		legs: [][][2]float64{
			{{5.4, 7.2}, {3.4, 5.4}, {2.6, 4.6}},
			{{6.2, 13.6}, {5.6, 15.6}, {4.7, 17.4}, {4.4, 18.4}},
			{{2.8, 18.5}, {6, 18.5}},
		}, widths: []float64{1.2, 1.3, 1.2}},
	frameTuck: {eyeY: 2, body: [4]float64{4.2, 4.6, 19.8, 16}, rad: 5, shade: 2.5,
		patch: [4]float64{9, 7, 15, 11}, point: true,
		legs: [][][2]float64{
			{{5, 8.4}, {2.6, 6.6}, {1.6, 5.6}},
			{{5.6, 14.4}, {3.2, 15}, {3.6, 16.4}, {6, 16.6}},
		}, widths: []float64{1.2, 2}},
}

// toadColours are a toad's colours: its body, the darker band round its
// rump, the patch on its back and its legs.
type toadColours struct{ body, shade, patch, legs int }

var (
	toadColour    = toadColours{resurrect64.Emerald, resurrect64.Forest, resurrect64.Leaf, resurrect64.Emerald}
	toadletColour = toadColours{resurrect64.Leaf, resurrect64.Emerald, resurrect64.Lime, resurrect64.Emerald}
)

// outlinePx is how far a smooth thing's ink outline reaches past its
// colour, in the game's pixels.
const outlinePx = 0.85

// smoothToad draws a toad in pose ps and colours c in a 24px box at the
// pen's origin, and its school cap if it wears one.
func smoothToad(pn pen, ps toadPose, c toadColours, cap bool) {
	mirror := func(pts [][2]float64) []float64 {
		var xy []float64
		for _, p := range pts {
			xy = append(xy, config.Cell-p[0], p[1])
		}
		return xy
	}
	flat := func(pts [][2]float64) []float64 {
		var xy []float64
		for _, p := range pts {
			xy = append(xy, p[0], p[1])
		}
		return xy
	}
	b := ps.body
	eyes := []float64{7.5, 16.5}
	eyeY := ps.eyeY + 1.9

	// The ink first, under everything, so it rings each part.
	for i, l := range ps.legs {
		pn.stroke(resurrect64.Ink, ps.widths[i]+2*outlinePx, flat(l)...)
		pn.stroke(resurrect64.Ink, ps.widths[i]+2*outlinePx, mirror(l)...)
	}
	pn.round(b[0]-outlinePx, b[1]-outlinePx, b[2]-b[0]+2*outlinePx, b[3]-b[1]+2*outlinePx, ps.rad+outlinePx, resurrect64.Ink)
	for _, ex := range eyes {
		if ps.closed {
			pn.ellipse(ex, eyeY-0.9, 2.4, 1.2, resurrect64.Ink)
		} else {
			pn.circle(ex, eyeY, 1.6+outlinePx, resurrect64.Ink)
		}
	}

	for i, l := range ps.legs {
		pn.stroke(c.legs, ps.widths[i], flat(l)...)
		pn.stroke(c.legs, ps.widths[i], mirror(l)...)
	}
	pn.round(b[0], b[1], b[2]-b[0], b[3]-b[1], ps.rad, c.shade)
	pn.round(b[0]+1.4, b[1], b[2]-b[0]-2.8, b[3]-b[1]-ps.shade, ps.rad-1, c.body)
	pt := ps.patch
	if ps.point {
		mid := float64(config.Cell) / 2
		pn.poly(c.patch, pt[0]+0.2, pt[1], pt[2]-0.2, pt[1], pt[2]-0.2, pt[3]-0.6, mid+0.9, pt[3]+0.9, mid-0.9, pt[3]+0.9, pt[0]+0.2, pt[3]-0.6)
	} else {
		pn.round(pt[0], pt[1], pt[2]-pt[0], pt[3]-pt[1], 0.6, c.patch)
	}
	if !ps.closed {
		for _, ex := range eyes {
			pn.circle(ex, eyeY, 1.6, resurrect64.Gold)
			pn.circle(ex, eyeY+0.6, 0.6, resurrect64.Ink)
		}
	}
	if cap {
		y := ps.eyeY
		pn.round(8.8, y-1.6, 6.4, 4.8, 2.4, resurrect64.Ink)
		pn.round(9.8, y-0.7, 4.4, 3, 1.5, resurrect64.Red)
		pn.ellipse(12, y-0.5, 1.6, 0.8, resurrect64.Brick)
	}
}

// smoothSplash draws frame f of the splash in a 24px box at the pen's
// origin: a burst of drops, then rings spreading out and darkening.
func smoothSplash(pn pen, f int) {
	const cx, cy = 11.5, 11.5
	switch f {
	case 0:
		for i := range 8 {
			a := float64(i) * math.Pi / 4
			r := 5.0
			if i%2 == 1 {
				r = 5.7
			}
			pn.circle(cx+r*math.Cos(a), cy+r*math.Sin(a), 0.6, resurrect64.White)
		}
		pn.circle(cx, cy, 3, resurrect64.Sky)
		pn.circle(cx, cy, 1.9, resurrect64.White)
	case 1:
		ring(pn, cx, cy+0.5, 6, 5, 0.9, resurrect64.Sky)
		pn.circle(cx, cy+0.5, 0.6, resurrect64.White)
		for _, d := range [][2]float64{{2.5, 3.5}, {20.5, 3.5}, {2.5, 19.5}, {20.5, 19.5}} {
			pn.circle(d[0], d[1], 0.6, resurrect64.White)
		}
	default:
		c := resurrect64.Blue
		if f == 3 {
			c = resurrect64.Indigo
		}
		ring(pn, cx, cy+0.5, 9, 7.5, 0.9, c)
	}
}

// ring draws an ellipse's outline about x, y, w wide.
func ring(pn pen, x, y, rx, ry, w float64, c int) {
	const n = 32
	xy := make([]float64, 0, 2*n+2)
	for i := range n + 1 {
		t := 2 * math.Pi * float64(i) / n
		xy = append(xy, x+rx*math.Cos(t), y+ry*math.Sin(t))
	}
	pn.stroke(c, w, xy...)
}

func init() {
	smoothArts["toad"] = func(pn pen, f int) { smoothToad(pn, toadPoses[f], toadColour, false) }
	smoothArts["toadlet"] = func(pn pen, f int) { smoothToadlet(pn, f, false) }
	smoothArts["toadlet_cap"] = func(pn pen, f int) { smoothToadlet(pn, f, true) }
	smoothArts["splash"] = smoothSplash
}

// smoothToadlet draws frame f of a toadlet, the toad two thirds the size in
// its own colours, in its school cap if cap.
func smoothToadlet(pn pen, f int, cap bool) {
	pn.k *= 2.0 / 3
	pn.ox, pn.oy = pn.ox*3/2, pn.oy*3/2
	smoothToad(pn, toadPoses[f], toadletColour, cap)
}

// heroStamp is frame f of the hero's sprite name in the smooth look, w×h.
func (p *Play) heroStamp(name string, f, k, w, h int) *ebiten.Image {
	return p.artStamp(name, f, k, w, h)
}

// drawSmoothToad draws pl as drawToad does, in the smooth look.
func (p *Play) drawSmoothToad(ctx *scene.Context, dst *ebiten.Image, k int, pl *world.Player, name string, dx float64) {
	sheet := ctx.Sheet(Sheet)
	sp := sheet.Sprite(name)
	splash := sheet.Sprite(p.hero.Splash)
	frame, lift, sf := p.toadState(pl, splash.Frames())
	if sf >= 0 {
		lay(dst, p.heroStamp(p.hero.Splash, sf, k, splash.W, splash.H), k, splashX(pl), cellY(pl.LaneRow()), 1)
		return
	}
	lift *= float64(sp.W) / config.Cell
	x := float64(config.FieldX) + float64(pl.X)/config.SubPx + float64(config.Cell-sp.W)/2 + dx
	y := float64(config.FieldY) + float64(pl.Y)/config.SubPx - lift + float64(config.Cell-sp.H)/2
	layTurned(dst, p.heroStamp(name, frame, k, sp.W, sp.H), k, x, y, pl.Facing, sp.W, sp.H)
}

// drawSmoothNookToads draws the toads home in bank row r's nooks, as
// drawNookToads does, in the smooth look.
func (p *Play) drawSmoothNookToads(ctx *scene.Context, dst *ebiten.Image, k, r int) {
	sp := ctx.Sheet(Sheet).Sprite(p.hero.Sprite)
	img := p.heroStamp(p.hero.Sprite, frameRest, k, sp.W, sp.H)
	for _, x := range p.homeToads(sp.W) {
		lay(dst, img, k, x, cellY(r), 1)
	}
}

// layTurned lays stamp img, a w×h sprite's art facing up, turned to face d,
// with its box's top-left at x, y in the game's coordinates.
func layTurned(dst, img *ebiten.Image, k int, x, y float64, d world.Dir, w, h int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-stampMargin*float64(k), -stampMargin*float64(k))
	op.GeoM.Concat(facing(d, w*k, h*k))
	op.GeoM.Translate(math.Round(x*float64(k)), math.Round(y*float64(k)))
	dst.DrawImage(img, op)
}
