package play

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/scene"
)

// The smooth look's art (DESIGN §3.6): drawn in code from shapes, in the
// game's coordinates on a screen k times their size, so a thing keeps its
// sprite's place, size and colours and only the drawing is finer.

// pen draws anti-aliased shapes in palette colours. Points and sizes are
// in the game's pixels; the pen moves them by ox, oy and scales them by k.
// Drawing a shape is slow, so what is drawn every frame is drawn once as a
// stamp and laid where it goes (see stamps).
type pen struct {
	dst    *ebiten.Image
	k      float64
	ox, oy float64
	alpha  float64 // how opaque, 0–1
}

// newPen is a pen on dst at scale k, drawing opaque.
func newPen(dst *ebiten.Image, k int) pen { return pen{dst: dst, k: float64(k), alpha: 1} }

// faded is the pen drawing a times as opaque.
func (pn pen) faded(a float64) pen {
	pn.alpha *= a
	return pn
}

// options are the drawing options for palette colour c.
func (pn pen) options(c int) *vector.DrawPathOptions {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(gfx.RGBA(c))
	a := float32(pn.alpha)
	op.ColorScale.Scale(a, a, a, a)
	return op
}

// fill fills path p, in the game's pixels, with palette colour c.
func (pn pen) fill(p *vector.Path, c int) {
	var s vector.Path
	s.AddPath(p, &vector.AddPathOptions{GeoM: pn.geoM()})
	vector.FillPath(pn.dst, &s, nil, pn.options(c))
}

// geoM moves and scales the game's pixels to the screen's.
func (pn pen) geoM() ebiten.GeoM {
	var g ebiten.GeoM
	g.Translate(pn.ox, pn.oy)
	g.Scale(pn.k, pn.k)
	return g
}

// rect fills the rectangle at x, y, w×h. Its edges are not smoothed, so it
// is quick enough to draw every frame.
func (pn pen) rect(x, y, w, h float64, c int) {
	k := pn.k
	vector.FillRect(pn.dst, float32((x+pn.ox)*k), float32((y+pn.oy)*k), float32(w*k), float32(h*k), faded(c, pn.alpha), false)
}

// round fills the rectangle at x, y, w×h with its corners rounded by rad.
func (pn pen) round(x, y, w, h, rad float64, c int) {
	rad = min(max(rad, 0), min(w, h)/2)
	x0, y0, x1, y1 := float32(x), float32(y), float32(x+w), float32(y+h)
	r := float32(rad)
	var p vector.Path
	p.MoveTo(x0+r, y0)
	p.LineTo(x1-r, y0)
	p.ArcTo(x1, y0, x1, y0+r, r)
	p.LineTo(x1, y1-r)
	p.ArcTo(x1, y1, x1-r, y1, r)
	p.LineTo(x0+r, y1)
	p.ArcTo(x0, y1, x0, y1-r, r)
	p.LineTo(x0, y0+r)
	p.ArcTo(x0, y0, x0+r, y0, r)
	p.Close()
	pn.fill(&p, c)
}

// circle fills the circle of radius r about x, y.
func (pn pen) circle(x, y, r float64, c int) { pn.ellipse(x, y, r, r, c) }

// ellipse fills the ellipse with radii rx and ry about x, y.
func (pn pen) ellipse(x, y, rx, ry float64, c int) {
	const n = 24
	xy := make([]float64, 0, 2*n)
	for i := range n {
		t := 2 * math.Pi * float64(i) / n
		xy = append(xy, x+rx*math.Cos(t), y+ry*math.Sin(t))
	}
	pn.poly(c, xy...)
}

// poly fills the polygon through the x, y pairs.
func (pn pen) poly(c int, xy ...float64) {
	if len(xy) < 6 {
		return
	}
	var p vector.Path
	p.MoveTo(float32(xy[0]), float32(xy[1]))
	for i := 2; i+1 < len(xy); i += 2 {
		p.LineTo(float32(xy[i]), float32(xy[i+1]))
	}
	p.Close()
	pn.fill(&p, c)
}

// stroke draws a line w wide through the x, y pairs, rounded at its ends
// and corners.
func (pn pen) stroke(c int, w float64, xy ...float64) {
	if len(xy) < 4 {
		return
	}
	k := pn.k
	var p vector.Path
	p.MoveTo(float32((xy[0]+pn.ox)*k), float32((xy[1]+pn.oy)*k))
	for i := 2; i+1 < len(xy); i += 2 {
		p.LineTo(float32((xy[i]+pn.ox)*k), float32((xy[i+1]+pn.oy)*k))
	}
	vector.StrokePath(pn.dst, &p, &vector.StrokeOptions{
		Width: float32(w * k), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound,
	}, pn.options(c))
}

// stamps are small pieces of the smooth art drawn once at a scale and laid
// as often as needed, which is far quicker than drawing them each time.
type stamps struct {
	k    int
	imgs map[string]*ebiten.Image
}

// stampMargin is the room round a stamp's box, in the game's pixels, for
// anything that reaches past it.
const stampMargin = 2

// get is the stamp called name at scale k, w×h in the game's pixels with
// its drawing's origin at its box's top-left, drawn by draw the first time
// it is asked for. A new scale draws them all afresh.
func (s *stamps) get(name string, k int, w, h float64, draw func(pen)) *ebiten.Image {
	if s.k != k {
		for _, img := range s.imgs {
			img.Deallocate()
		}
		s.k, s.imgs = k, map[string]*ebiten.Image{}
	}
	if img, ok := s.imgs[name]; ok {
		return img
	}
	img := ebiten.NewImage(int(math.Ceil((w+2*stampMargin)*float64(k))), int(math.Ceil((h+2*stampMargin)*float64(k))))
	pn := newPen(img, k)
	pn.ox, pn.oy = stampMargin, stampMargin
	draw(pn)
	s.imgs[name] = img
	return img
}

// smoothArts draw the sprites that have smooth art, by name: frame f in a
// box the sprite's size at the pen's origin, facing as the sprite does.
var smoothArts = map[string]func(pn pen, f int){}

// artStamp is frame f of sprite name's smooth art at scale k, for a sprite
// w×h, or nil if it has none yet.
func (p *Play) artStamp(name string, f, k, w, h int) *ebiten.Image {
	art, ok := smoothArts[name]
	if !ok {
		return nil
	}
	return p.stamp(fmt.Sprint("art:", name, "/", f), k, float64(w), float64(h), func(pn pen) { art(pn, f) })
}

// smoothPiece is the image pc is drawn with in the smooth look and where:
// its smooth art, or its sprite enlarged until it has some.
func (p *Play) smoothPiece(ctx *scene.Context, pc piece, k int) (*ebiten.Image, ebiten.GeoM) {
	kf := float64(k)
	var g ebiten.GeoM
	if pc.img == nil {
		sp := ctx.Sheet(Sheet).Sprite(pc.name)
		if img := p.artStamp(pc.name, pc.frame, k, sp.W, sp.H); img != nil {
			g.Translate(-stampMargin*kf, -stampMargin*kf)
			if pc.flip {
				g.Scale(-1, 1)
				g.Translate(float64(sp.W*k), 0)
			}
			g.Translate(math.Round(pc.x*kf), math.Round(pc.y*kf))
			return img, g
		}
	}
	img := pc.image(ctx)
	g = spriteGeoM(img, pc.x, pc.y, pc.flip)
	g.Scale(kf, kf)
	return img, g
}

// drawSmoothPiece draws pc in the smooth look.
func (p *Play) drawSmoothPiece(ctx *scene.Context, dst *ebiten.Image, pc piece, k int) {
	img, g := p.smoothPiece(ctx, pc, k)
	dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: g})
}

// lay draws stamp img on dst with its origin at x, y in the game's
// coordinates, on whole screen pixels, a times as opaque.
func lay(dst, img *ebiten.Image, k int, x, y, a float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round((x-stampMargin)*float64(k)), math.Round((y-stampMargin)*float64(k)))
	if a < 1 {
		op.ColorScale.ScaleAlpha(float32(a))
	}
	dst.DrawImage(img, op)
}
