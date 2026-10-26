package play

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// The field's ground in the smooth look: the verge, the track, the road,
// the river, the far bank and its nooks, the street lamps, the crossing
// signals and the traffic lights, with the water's crests and glints, the
// chop, currents, wakes, the flood and the road's sheen. Each keeps its
// sprite's or pixel drawing's places, sizes and colours (DESIGN §3.6).

// groundCache is the rows' still parts, drawn once for a level at a scale,
// and the stamps the moving parts are laid from.
type groundCache struct {
	img    *ebiten.Image
	k      int
	kinds  []config.LaneKind
	stamps stamps
}

// drawSmoothGround draws the ground at scale k on field, the screen's
// image clipped to the field, in the pixel drawing's order.
func (p *Play) drawSmoothGround(ctx *scene.Context, field *ebiten.Image, k int) {
	w := p.world
	p.bakeGround(k)
	field.DrawImage(p.ground.img, nil)
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.River {
			p.smoothWater(field, k, r)
			p.smoothChop(field, k, r)
		}
	}
	p.smoothFlood(field, k)
	p.smoothSheen(newPen(field, k))
	if p.rainbow() { // enlarged until it has its smooth art
		bounds := image.Rect(config.FieldX, config.FieldY, config.FieldX+w.Cols*config.Cell, config.FieldY+w.Rows*config.Cell)
		p.inPixels(field, k, func(art *ebiten.Image) { p.drawRainbow(art.SubImage(bounds).(*ebiten.Image)) })
	}
	lampImg := p.stamp("lamp", k, config.Cell, config.Cell, func(pn pen) { smoothLamp(pn, 0, 0) })
	for _, lp := range p.streetLamps() {
		lay(field, lampImg, k, cellX(lp.col), cellY(lp.row), 1)
	}
	if row, ok := p.signalRow(); ok {
		for side, c := range signalCols {
			f := p.signalFrame(side)
			img := p.stamp(fmt.Sprint("signal", f), k, config.Cell, config.Cell, func(pn pen) { smoothSignal(pn, 0, 0, f) })
			lay(field, img, k, cellX(c), cellY(row-1), 1)
		}
	}
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.River {
			p.smoothCurrent(field, k, r)
			p.smoothWakes(field, k, r)
		}
	}
}

// stamp is the ground's stamp called name: see stamps.get.
func (p *Play) stamp(name string, k int, w, h float64, draw func(pen)) *ebiten.Image {
	return p.ground.stamps.get(name, k, w, h, draw)
}

// bakeGround draws the rows' still parts into the cache, unless it already
// holds them at scale k.
func (p *Play) bakeGround(k int) {
	w := p.world
	kinds := make([]config.LaneKind, len(w.Lanes))
	for r := range w.Lanes {
		kinds[r] = w.Lanes[r].Kind
	}
	g := &p.ground
	if g.img != nil && g.k == k && equalKinds(g.kinds, kinds) {
		return
	}
	if g.img != nil {
		g.img.Deallocate()
	}
	g.img = ebiten.NewImage(p.size.X*k, p.size.Y*k)
	g.k, g.kinds = k, kinds
	pn := newPen(g.img, k)
	for r := range w.Lanes {
		switch kinds[r] {
		case config.Safe:
			for c := range w.Cols {
				smoothVerge(pn, cellX(c), cellY(r))
			}
		case config.Bank:
			p.smoothBank(pn, r)
		case config.Rail:
			p.smoothTrack(pn, r)
		case config.Road:
			p.smoothRoad(pn, r)
		case config.River:
			pn.rect(cellX(0), cellY(r), float64(w.Cols*config.Cell), config.Cell, resurrect64.Navy)
		}
	}
}

func equalKinds(a, b []config.LaneKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// vergeTufts are the verge sprite's tufts of grass, each by the bottom of
// its middle blade and where its tip leans to: three blades, the middle
// one lighter.
var vergeTufts = [][3]float64{
	{2, 2, 2}, {17, 2, 17}, {9, 5, 9}, {20, 6, 20}, {3, 9, 3}, {14, 10, 15},
	{10, 13, 10}, {22, 14, 22}, {4, 17, 4}, {17, 18, 16}, {12, 21, 12}, {21, 22, 21},
}

// smoothVerge draws one cell of the verge, night grass with its tufts, at
// x, y.
func smoothVerge(pn pen, x, y float64) {
	pn.rect(x, y, config.Cell, config.Cell, resurrect64.Sage)
	for _, t := range vergeTufts {
		tuft(pn, x+t[0]+0.5, y+t[1]+1, x+t[2]+0.5)
	}
}

// tuft draws a tuft of grass growing from bx, by, its middle blade leaning
// to tx.
func tuft(pn pen, bx, by, tx float64) {
	pn.stroke(resurrect64.Pine, 0.6, bx, by, bx-1.3, by-1.1)
	pn.stroke(resurrect64.Pine, 0.6, bx, by, bx+1.3, by-1.1)
	pn.stroke(resurrect64.Pine, 0.6, bx, by, tx, by-2)
	pn.stroke(resurrect64.Moss, 0.5, bx, by-0.2, (bx+tx)/2, by-1.1)
}

// trackStones are the track sprite's darker stones in its ballast.
var trackStones = [][2]float64{
	{2, 0}, {10, 0}, {19, 0}, {6, 1}, {15, 1}, {1, 2}, {13, 2}, {22, 2}, {8, 4}, {21, 4},
	{10, 9}, {19, 10}, {7, 12}, {22, 14}, {7, 19}, {4, 21}, {18, 21}, {10, 22}, {0, 23}, {16, 23},
}

// smoothTrack draws row r as the track: ballast with its stones, the
// sleepers, and two rails along the row.
func (p *Play) smoothTrack(pn pen, r int) {
	w := p.world
	y, width := cellY(r), float64(w.Cols*config.Cell)
	pn.rect(cellX(0), y, width, config.Cell, resurrect64.Dusk)
	for c := range w.Cols {
		x := cellX(c)
		for _, s := range trackStones {
			pn.ellipse(x+s[0]+0.5, y+s[1]+0.5, 0.8, 0.6, resurrect64.Shadow)
		}
		for _, sx := range []float64{2, 14} {
			pn.round(x+sx, y+3, 4, 18, 0.8, resurrect64.Bark)
		}
	}
	for _, ry := range []float64{6, 16} {
		pn.rect(cellX(0), y+ry, width, 2, resurrect64.Fog)
		pn.rect(cellX(0), y+ry, width, 0.9, resurrect64.Mist)
	}
}

// smoothRoad draws row r as asphalt, with rounded dashes where it meets
// another road lane and a solid edge where it meets anything else, as
// drawRoad does.
func (p *Play) smoothRoad(pn pen, r int) {
	w := p.world
	x, y := cellX(0), cellY(r)
	width := float64(w.Cols * config.Cell)
	pn.rect(x, y, width, config.Cell, resurrect64.Shadow)
	edge := func(other int, ly float64) {
		if other >= 0 && other < w.Rows && w.Lanes[other].Kind == config.Road {
			for dx := 0.0; dx < width; dx += dashLen + dashGap {
				pn.stroke(resurrect64.Dusk, 1, x+dx+0.5, ly+0.5, x+dx+min(dashLen, width-dx)-0.5, ly+0.5)
			}
			return
		}
		pn.rect(x, ly, width, 1, resurrect64.Dusk)
	}
	edge(r-1, y)
	if r+1 >= w.Rows || w.Lanes[r+1].Kind != config.Road {
		edge(r+1, y+config.Cell-1)
	}
}

// smoothBank draws row r as the far bank: courses of stone blocks, each
// course a half block along from the one below, and the nooks in it.
func (p *Play) smoothBank(pn pen, r int) {
	w := p.world
	y, width := cellY(r), float64(w.Cols*config.Cell)
	pn.rect(cellX(0), y, width, config.Cell, resurrect64.Shadow)
	for course := range 4 {
		cy := y + float64(course*6)
		for sx := -6.0 + float64(course%2*6); sx < width; sx += 12 {
			x := cellX(0) + sx
			pn.round(x+0.2, cy, 10.6, 5, 1, resurrect64.Charcoal)
			pn.round(x+0.2, cy, 10.6, 4.1, 1, resurrect64.Dusk)
			pn.circle(x+1.1, cy+0.9, 0.55, resurrect64.Fog)
		}
	}
	for _, c := range config.Nooks {
		smoothNook(pn, cellX(c), y)
	}
}

// nookSpecks are the nook sprite's specks in its grass: pine, then moss.
var nookSpecks = [][3]float64{
	{8, 6, 0}, {19, 6, 1}, {6, 11, 0}, {10, 11, 1}, {4, 16, 0}, {38, 16, 1}, {28, 21, 1}, {38, 21, 0},
}

// smoothNook draws a nook at x, y, two cells wide: grass inside stone
// sides, shaded under the wall at its top.
func smoothNook(pn pen, x, y float64) {
	const wd = config.NookWidth * config.Cell
	pn.rect(x, y, wd, config.Cell, resurrect64.Dusk)
	pn.rect(x+2, y, wd-4, config.Cell, resurrect64.Shadow)
	pn.round(x+2.6, y-2, wd-5.2, config.Cell+2, 1.5, resurrect64.Sage)
	pn.rect(x+2.6, y, wd-5.2, 2.4, resurrect64.Pine)
	pn.faded(0.5).rect(x+2.6, y+2.4, wd-5.2, 0.8, resurrect64.Pine)
	for _, s := range nookSpecks {
		c := resurrect64.Pine
		if s[2] == 1 {
			c = resurrect64.Moss
		}
		bx, by := x+s[0]+0.5, y+s[1]+1
		pn.stroke(c, 0.55, bx, by, bx-0.7, by-1.1)
		pn.stroke(c, 0.55, bx, by, bx+0.7, by-1.1)
	}
}

// smoothLamp draws a street lamp seen from above with its cell at x, y:
// the glowing head, the arm, the post and its foot, as the lamp sprite.
func smoothLamp(pn pen, x, y float64) {
	pn.round(x+8.2, y+1.2, 7.6, 5.6, 2.2, resurrect64.Ink)
	pn.round(x+9, y+2, 6, 4, 1.6, resurrect64.Honey)
	pn.ellipse(x+12, y+4, 2.4, 1.3, resurrect64.Gold)
	pn.ellipse(x+12, y+4, 1.5, 0.7, resurrect64.White)
	post(pn, x, y+6)
}

// post draws a lamp's or signal's arm, post and foot, from x, y down.
func post(pn pen, x, y float64) {
	pn.round(x+11, y, 2, 1.4, 0.3, resurrect64.Ink)
	pn.round(x+11, y+1, 1, 4, 0.2, resurrect64.Charcoal)
	pn.round(x+12, y+1, 1, 4, 0.2, resurrect64.Dusk)
	pn.ellipse(x+12, y+5, 3, 2.2, resurrect64.Ink)
	pn.ellipse(x+12, y+4.8, 2.1, 1.4, resurrect64.Charcoal)
	pn.ellipse(x+12.6, y+4.6, 1, 0.8, resurrect64.Dusk)
	pn.circle(x+11.2, y+4.6, 0.5, resurrect64.Fog)
}

// smoothSignal draws a crossing signal seen from above with its cell at x,
// y, showing frame: two lamps on a crossbar over a white cross, on a post,
// a lamp lit as its frame is.
func smoothSignal(pn pen, x, y float64, frame int) {
	pn.stroke(resurrect64.Ink, 1, x+5, y+6.5, x+19, y+6.5)
	for i, lx := range []float64{5.5, 18.5} {
		lit := frame == signalLeftLamp && i == 0 || frame == signalRightLamp && i == 1
		pn.circle(x+lx, y+4.5, 2.6, resurrect64.Ink)
		if lit {
			pn.circle(x+lx, y+4.5, 1.9, resurrect64.Red)
			pn.circle(x+lx, y+4.5, 0.9, resurrect64.White)
		} else {
			pn.circle(x+lx, y+4.5, 1.9, resurrect64.Brick)
			pn.circle(x+lx, y+4.5, 1, resurrect64.Crimson)
		}
	}
	for _, l := range [][4]float64{{8, 8, 16, 15}, {16, 8, 8, 15}} {
		pn.stroke(resurrect64.Ink, 2.4, x+l[0], y+l[1], x+l[2], y+l[3])
	}
	for _, l := range [][4]float64{{8, 8, 16, 15}, {16, 8, 8, 15}} {
		pn.stroke(resurrect64.White, 1.1, x+l[0], y+l[1], x+l[2], y+l[3])
	}
	post(pn, x, y+15)
}

// trafficLamps are a traffic light's lamps from the top, each lit in its
// frame: red, amber, green, with their colour and darker rim.
var trafficLamps = [3]struct{ frame, colour, rim int }{
	{lightRed, resurrect64.Red, resurrect64.Crimson},
	{lightAmber, resurrect64.Gold, resurrect64.Amber},
	{lightGreen, resurrect64.Leaf, resurrect64.Emerald},
}

// The traffic light sprite's size.
const trafficLightW, trafficLightH = 8, 21

// smoothTrafficLight draws a traffic light with its top-left at x, y,
// showing frame, as the traffic_light sprite: three lamps filling a light
// grey frame, on a post.
func smoothTrafficLight(pn pen, x, y float64, frame int) {
	pn.round(x+3, y+16, 2, 4, 0.3, resurrect64.Fog)
	pn.round(x+4, y+16, 1, 4, 0.3, resurrect64.Dusk)
	pn.ellipse(x+4, y+20.4, 2.2, 0.9, resurrect64.Ink)
	pn.round(x, y, trafficLightW, 17, 1.5, resurrect64.Fog)
	for i, l := range trafficLamps {
		cy := y + 3.5 + float64(5*i)
		pn.circle(x+4, cy, 2.4, resurrect64.Ink)
		if l.frame == frame {
			pn.circle(x+4, cy, 2, l.rim)
			pn.circle(x+4, cy, 1.5, l.colour)
			pn.circle(x+3.3, cy-0.7, 0.6, resurrect64.White)
		} else {
			pn.circle(x+4, cy, 1.9, resurrect64.Shadow)
			pn.circle(x+3.3, cy-0.7, 0.5, resurrect64.Dusk)
		}
	}
}

// drawSmoothTrafficLights draws the traffic lights as drawTrafficLights
// does, in the smooth look.
func (p *Play) drawSmoothTrafficLights(dst *ebiten.Image, k int) {
	if p.world.Spell(config.Wave) == nil {
		return
	}
	for r := range p.world.Lanes {
		if p.world.Lanes[r].Kind != config.Road {
			continue
		}
		x := float64(config.FieldX + config.Cols*config.Cell)
		if p.fromLeft(r) {
			x = float64(config.FieldX - trafficLightW)
		}
		f := p.trafficFrame(r)
		img := p.stamp(fmt.Sprint("traffic", f), k, trafficLightW, trafficLightH, func(pn pen) { smoothTrafficLight(pn, 0, 0, f) })
		lay(dst, img, k, x, cellY(r)+float64(config.Cell-trafficLightH)/2, 1)
	}
}

// waterCrest is a crest on the water, its top at 0 and its wave at y 1.5:
// an arch with a lighter top.
func waterCrest(pn pen) {
	const cx, cy = waveLen / 2.0, 1.5
	pn.stroke(resurrect64.Indigo, 1.3, cx-waveLen/2+0.6, cy, cx, cy-0.5, cx+waveLen/2-0.6, cy)
	pn.stroke(resurrect64.Cobalt, 1, cx-1.1, cy-1, cx+1.1, cy-1)
}

// smoothWater draws river row r's moving surface over its still water: the
// crests, drifting and bobbing smoothly, and the moon's glints.
func (p *Play) smoothWater(dst *ebiten.Image, k, r int) {
	w := p.world
	lane := &w.Lanes[r]
	x0, y0 := cellX(0), cellY(r)
	width := float64(w.Cols * config.Cell)
	crest := p.stamp("crest", k, waveLen, 3, waterCrest)
	const period = waveLen + waveGap
	drift := float64(w.Tick) * float64(lane.Step()) / (config.SubPx * waveSlowBy)
	for i, wr := range waveRows {
		bob := 0.5 + 0.5*math.Sin(math.Pi*(float64(w.Tick)/waveBob+float64(i)))
		off := math.Mod(math.Mod(drift+float64(wr.phase+r*7), period)+period, period) - period
		for dx := off; dx < width; dx += period {
			lay(dst, crest, k, x0+dx, y0+float64(wr.y)+bob-1, 1)
		}
	}
	glint := []int{resurrect64.Sky, resurrect64.Mist, resurrect64.Blue}
	for i := range moonGlints {
		h := noise(w.Tick/glintTicks, int64(r), int64(i))
		gx := moonX - moonHalf + int(h%(2*moonHalf))
		if d := math.Abs(float64(gx-moonX)) / moonHalf; float64(h>>8%100)/100 < d {
			continue
		}
		gy := 2 + int(h>>16%(config.Cell-4))
		gw := float64(2 + int(h>>24%4))
		c := glint[i%len(glint)]
		img := p.stamp(fmt.Sprint("glint", c, gw), k, gw, 1, func(pn pen) { pn.stroke(c, 1, 0.5, 0.5, gw-0.5, 0.5) })
		lay(dst, img, k, x0+float64(gx), y0+float64(gy), 1)
	}
}

// smoothChop draws the wind's chop on river row r, as drawChop does.
func (p *Play) smoothChop(dst *ebiten.Image, k, r int) {
	w := p.world
	if w.Spell(config.StormFront) == nil {
		return
	}
	fleck := p.stamp("chop", k, 2, 1, func(pn pen) { pn.stroke(resurrect64.Mist, 0.7, 0.4, 0.5, 1.6, 0.5) })
	for i := range chopFlecks {
		h := noise(w.Tick/chopTicks, int64(r+50), int64(i))
		x := cellX(0) + float64(h%uint32(w.Cols*config.Cell))
		lay(dst, fleck, k, x, cellY(r)+float64(3+h>>12%(config.Cell-6)), 1)
	}
}

// smoothCurrent draws river row r's current as drawCurrent does: double
// chevrons on the water, drifting smoothly the way it slides the hero.
func (p *Play) smoothCurrent(dst *ebiten.Image, k, r int) {
	w := p.world
	slide := w.Lanes[r].Slide()
	if slide == 0 {
		return
	}
	s := math.Min(1, math.Max(0, float64(abs(slide)-arrowWeakest)/float64(arrowStrong-arrowWeakest)))
	gap := float64(arrowGapMax - int(s*float64(arrowGapMax-arrowGapMin)))
	dir, tipX := 1.0, float64(arrowArm+arrowPair) // where the first chevron's tip is in the stamp
	if slide < 0 {
		dir, tipX = -1, 0
	}
	img := p.stamp(fmt.Sprint("current", dir), k, arrowArm+arrowPair, 2*arrowArm+1, func(pn pen) {
		const y = arrowArm + 0.5
		for _, back := range []float64{0, arrowPair} {
			tip := tipX - dir*back
			pn.stroke(resurrect64.Seafoam, 1.6, tip-dir*arrowArm, y-arrowArm, tip, y, tip-dir*arrowArm, y+arrowArm)
		}
	})
	drift := float64(w.Tick) * float64(slide) / config.SubPx
	off := math.Mod(math.Mod(drift+float64(r*13), gap)+gap, gap) - gap
	for dx := off; dx < float64(w.Cols*config.Cell)+gap; dx += gap {
		lay(dst, img, k, cellX(0)+dx-tipX, cellY(r)+arrowY-arrowArm, arrowDimAlpha+(1-arrowDimAlpha)*s)
	}
}

// smoothWakes draws the water stirred up by river row r's platforms, as
// drawWakes does: foam at the front of each and lines trailing behind it.
func (p *Play) smoothWakes(dst *ebiten.Image, k, r int) {
	w := p.world
	lane := &w.Lanes[r]
	y := cellY(r)
	foam := []int{resurrect64.Mist, resurrect64.Fog, resurrect64.Sky}
	for i, e := range lane.Entities {
		if st := e.Stage(); st == world.Under || st == world.Sunk {
			continue
		}
		left := cellX(0) + math.Floor(float64(e.X())/config.SubPx)
		right := left + float64(e.Kind.Length)
		front, back, dir := right, left, 1.0
		if lane.Dir < 0 {
			front, back, dir = left, right, -1
		}
		for py := 6; py < config.Cell-5; py += 2 {
			h := noise(w.Tick/foamTicks, int64(r*100+i), int64(py))
			if h%3 == 0 {
				continue
			}
			fx := front + dir*float64(h>>4%2)
			if dir < 0 {
				fx--
			}
			c := foam[h>>8%3]
			img := p.stamp(fmt.Sprint("foam", c), k, 1, 1, func(pn pen) { pn.circle(0.5, 0.5, 0.6, c) })
			lay(dst, img, k, fx, y+float64(py), 1)
		}
		for j := range wakeLines {
			n := float64(wakeLen - 2*j)
			gap := float64(2 + j*5)
			wx := back - dir*gap
			if dir > 0 {
				wx -= n
			}
			img := p.stamp(fmt.Sprint("wake", n), k, n, 1, func(pn pen) { pn.stroke(resurrect64.Cobalt, 0.8, 0.4, 0.5, n-0.4, 0.5) })
			for _, wy := range []float64{7, config.Cell - 8} {
				lay(dst, img, k, wx, y+wy+float64(j%2), 1)
			}
		}
	}
}

// smoothSheen gives the road rows their wet sheen, as drawRoadSheen does.
func (p *Play) smoothSheen(pn pen) {
	k := max(p.rainK(), p.wetK())
	if k <= 0 {
		return
	}
	for r := range p.world.Lanes {
		if p.world.Lanes[r].Kind == config.Road {
			pn.faded(rainSheen*k).rect(cellX(0), cellY(r), config.Cols*config.Cell, config.Cell, sheenColor)
		}
	}
}

// smoothFlood draws a wet storm's flood on the verge as drawFlood does: a
// rippling edge of water lapping up from the river, puddles and open
// water.
func (p *Play) smoothFlood(dst *ebiten.Image, k int) {
	w := p.world
	s := w.Spell(config.StormFront)
	if s == nil || s.Phase == world.SpellQuiet || !s.Floods() || w.Lanes[config.VergeRow].Kind != config.Safe {
		return
	}
	pn := newPen(dst, k)
	y := cellY(config.VergeRow)
	width := w.Cols * config.Cell
	edge := []float64{cellX(0), y}
	for x := 0; x <= width; x += lapStep {
		d := float64(1 + noise(w.Tick/lapTicks, int64(min(x, width-lapStep)), 9)%lapDepth)
		edge = append(edge, cellX(0)+float64(x), y+d)
	}
	edge = append(edge, cellX(0)+float64(width), y)
	pn.poly(resurrect64.Navy, edge...)
	crest := p.stamp("floodcrest", k, waveLen, 2, func(pn pen) {
		pn.stroke(resurrect64.Cobalt, 0.8, 0.5, 1, waveLen/2.0, 0.4, waveLen-0.5, 1)
	})
	puddle := p.stamp("puddle", k, puddleW, puddleH, func(pn pen) {
		pn.ellipse(puddleW/2.0, puddleH/2.0, puddleW/2.0, puddleH/2.0, resurrect64.Navy)
		pn.stroke(resurrect64.Cobalt, 0.6, 1.6, 0.8, puddleW-2.6, 0.8)
	})
	for c := range w.Cols {
		x := cellX(c)
		switch {
		case w.Flooded(c):
			pn.rect(x, y, config.Cell, config.Cell, resurrect64.Navy)
			for i, wr := range waveRows {
				off := float64((int(w.Tick)/(waveBob/2) + i*7 + c*3) % (config.Cell - waveLen))
				lay(dst, crest, k, x+off, y+float64(wr.y)-0.5, 1)
			}
		case w.Puddled(c):
			for _, pd := range [][2]float64{{3, 5}, {12, 14}} {
				lay(dst, puddle, k, x+pd[0], y+pd[1], 1)
			}
		}
	}
}
