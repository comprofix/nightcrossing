package play

import (
	"math"

	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
)

// The river's platforms in the smooth look (DESIGN §3.6), drawn from shapes
// in their sprites' places, sizes and colours: seen from above, facing
// right. Places are the sprites' columns and rows; a middle piece runs to
// its box's edges, so the pieces meet. A ship that sinks has its pieces
// in the sprites' recoloured forms too: flashing, low and deep (D33).

func init() {
	smoothArts["log_back"] = func(pn pen, _ int) { smoothLog(pn, logBack) }
	smoothArts["log_mid"] = func(pn pen, _ int) { smoothLog(pn, logMid) }
	smoothArts["log_front"] = func(pn pen, _ int) { smoothLog(pn, logFront) }
	smoothArts["pad"] = smoothPad
	ships := map[string]func(pn pen, t tint){
		"barge_back":  smoothBargeBack,
		"barge_mid":   smoothBargeMid,
		"barge_front": smoothBargeFront,
		"ferry_back":  smoothFerryBack,
		"ferry_deck":  smoothFerryDeck,
		"ferry_cabin": smoothFerryCabin,
		"ferry_front": smoothFerryFront,
	}
	for suffix, t := range shipTints {
		for name, draw := range ships {
			smoothArts[name+suffix] = func(pn pen, _ int) { draw(pn, t) }
		}
	}
}

// A tint swaps palette colours for others, as the sprites' recolour lines
// do; colours it does not name stay as they are.
type tint map[int]int

// of is colour c in the tint.
func (t tint) of(c int) int {
	if to, ok := t[c]; ok {
		return to
	}
	return c
}

// shipTints are a sinking ship's tints by the suffix of their sprites'
// names, as the sprite file recolours them: as it is, the warning's flash,
// low in the water as it breaks up, and deep as it goes under.
var shipTints = map[string]tint{
	"": nil,
	"_flash": {
		resurrect64.Dusk: resurrect64.Fog, resurrect64.Sand: resurrect64.Honey,
		resurrect64.Charcoal: resurrect64.Dusk, resurrect64.Fog: resurrect64.Mist,
		resurrect64.Crimson: resurrect64.Red, resurrect64.Mist: resurrect64.White,
	},
	"_low": {
		resurrect64.Dusk: resurrect64.Shadow, resurrect64.Sand: resurrect64.Olive,
		resurrect64.Honey: resurrect64.Sand, resurrect64.Caramel: resurrect64.Bark,
		resurrect64.Charcoal: resurrect64.Ink, resurrect64.Fog: resurrect64.Dusk,
		resurrect64.Mist: resurrect64.Fog, resurrect64.Crimson: resurrect64.Rust,
		resurrect64.Gold: resurrect64.Amber, resurrect64.Orange: resurrect64.Rust,
	},
	"_deep": {
		resurrect64.Ink: resurrect64.Shadow, resurrect64.Dusk: resurrect64.Indigo,
		resurrect64.Sand: resurrect64.Indigo, resurrect64.Honey: resurrect64.Indigo,
		resurrect64.Caramel: resurrect64.Indigo, resurrect64.Charcoal: resurrect64.Shadow,
		resurrect64.Fog: resurrect64.Indigo, resurrect64.Mist: resurrect64.Indigo,
		resurrect64.Crimson: resurrect64.Shadow, resurrect64.Gold: resurrect64.Indigo,
		resurrect64.Orange: resurrect64.Shadow,
	},
}

// A log's pieces.
const (
	logBack = iota
	logMid
	logFront
)

// logStreaks are the bark's dark streaks along each log piece, as its
// sprite has them: column, row and length.
var logStreaks = [...][][3]float64{
	logBack:  {{9, 8, 4}, {20, 9, 3}, {5, 11, 4}, {15, 12, 5}, {10, 14, 4}, {21, 15, 2}},
	logMid:   {{5, 8, 4}, {17, 9, 5}, {2, 11, 4}, {12, 12, 6}, {7, 14, 4}, {19, 15, 3}},
	logFront: {{11, 8, 4}, {1, 9, 3}, {15, 11, 4}, {4, 12, 5}, {10, 14, 4}, {1, 15, 2}},
}

// logFlecks are the columns of the rust flecks in each log piece's copper
// top and in its bark bottom.
var logFlecks = [...][2][]float64{
	logBack:  {{7, 19}, {12}},
	logMid:   {{3, 15}, {5, 19}},
	logFront: {{4, 16}, {11}},
}

// smoothLog draws piece part of a log: rust brown in an ink rim, lit copper
// along its top and dark bark along its bottom, streaked along its length;
// the back and front pieces end in the log's cut ends, caramel, with a ring
// of the grain.
func smoothLog(pn pen, part int) {
	x0, x1 := 0.0, 24.0
	switch part {
	case logBack:
		x0 = 3.5
	case logFront:
		x1 = 20.5
	}
	pn.rect(x0, 4, x1-x0, 16, resurrect64.Ink)
	pn.rect(x0, 5, x1-x0, 2, resurrect64.Copper)
	pn.rect(x0, 7, x1-x0, 10, resurrect64.Rust)
	pn.rect(x0, 17, x1-x0, 2, resurrect64.Bark)
	for _, s := range logStreaks[part] {
		pn.round(s[0], s[1], s[2], 1, 0.5, resurrect64.Bark)
	}
	for _, x := range logFlecks[part][0] {
		pn.round(x, 6.1, 1, 0.8, 0.4, resurrect64.Rust)
	}
	for _, x := range logFlecks[part][1] {
		pn.round(x, 18.1, 1, 0.8, 0.4, resurrect64.Rust)
	}
	switch part {
	case logBack:
		logEnd(pn, x0)
	case logFront:
		logEnd(pn, x1)
	}
}

// logEnd draws a log's cut end about column x.
func logEnd(pn pen, x float64) {
	pn.ellipse(x, 12, 3.5, 8, resurrect64.Ink)
	pn.ellipse(x, 12, 2.6, 7, resurrect64.Caramel)
	ring(pn, x, 12, 1.4, 4.2, 0.6, resurrect64.Rust)
	pn.ellipse(x, 12, 0.5, 1.2, resurrect64.Rust)
}

// padColours are a lily pad's colours in each of its frames: its rim lit
// at the top and dark below, its leaf and its veins.
var padColours = [...][4]int{
	frameAfloat: {resurrect64.Fern, resurrect64.Pine, resurrect64.Sage, resurrect64.Pine},
	frameFlash:  {resurrect64.Lime, resurrect64.Olive, resurrect64.Moss, resurrect64.Olive},
	frameHalf:   {resurrect64.Teal, resurrect64.DeepTeal, resurrect64.Pine, resurrect64.DeepTeal},
}

// padVeins are where the veins of a lily pad run to from its middle.
var padVeins = [...][2]float64{{7.5, 6.5}, {17.5, 6.5}, {4.5, 10.5}, {19.5, 10.5}, {6.5, 15.5}, {16.5, 15.5}}

// smoothPad draws frame f of a lily pad: round, in an ink rim, its veins
// spreading from the middle. It flashes paler before it dives, is teal
// half under, and is only a ring on the water while it is under.
func smoothPad(pn pen, f int) {
	const cx, cy = 12.0, 12.0
	if f == frameRing {
		ring(pn, cx, cy, 9.5, 9.5, 1, resurrect64.Teal)
		return
	}
	c := padColours[f]
	pn.circle(cx, cy, 10, resurrect64.Ink)
	pn.circle(cx, cy, 9, c[1])
	pn.circle(cx, cy-0.4, 8.6, c[0])
	pn.circle(cx, cy+0.2, 7.6, c[2])
	const mx, my = 12, 11.5
	for _, v := range padVeins {
		pn.stroke(c[3], 0.9, mx, my, v[0], v[1])
	}
}

// sandHeap draws a heap of sand in a barge's hold, three columns wide from
// column x with its top in row y.
func sandHeap(pn pen, x, y float64, c int) {
	pn.ellipse(x+1.5, y+1.4, 1.7, 1, c)
	pn.ellipse(x+1.5, y+0.9, 0.8, 0.7, c)
}

// heaps are the heaps of sand in a barge piece's hold, as its sprite has
// them, honey and caramel: column and row of each heap's top.
type heaps struct{ honey, caramel [][2]float64 }

// bargeHold draws a barge piece's sand in tint t, from column x0 to x1
// inside its charcoal rim, and its heaps.
func bargeHold(pn pen, t tint, x0, x1 float64, h heaps) {
	pn.rect(x0, 6, x1-x0, 12, t.of(resurrect64.Sand))
	for _, p := range h.honey {
		sandHeap(pn, p[0]-1, p[1], t.of(resurrect64.Honey))
	}
	for _, p := range h.caramel {
		sandHeap(pn, p[0]-1, p[1], t.of(resurrect64.Caramel))
	}
}

// smoothBargeBack draws the barge's stern in tint t: the dusk hull with the
// wheelhouse, indigo with lit windows, and the hold behind it.
func smoothBargeBack(pn pen, t tint) {
	pn.round(0, 3, 8, 18, 1.6, t.of(resurrect64.Ink))
	pn.rect(4, 3, 20, 18, t.of(resurrect64.Ink))
	pn.round(1, 4, 6, 16, 0.9, t.of(resurrect64.Dusk))
	pn.rect(3, 4, 21, 16, t.of(resurrect64.Dusk))
	pn.round(3, 7, 8, 10, 0.8, t.of(resurrect64.Ink))
	pn.round(4, 8, 6, 8, 0.5, t.of(resurrect64.Indigo))
	for _, y := range []float64{9, 14} {
		pn.round(5, y, 4, 1, 0.4, t.of(resurrect64.Gold))
	}
	pn.rect(13, 5, 11, 14, t.of(resurrect64.Charcoal))
	bargeHold(pn, t, 14, 24, heaps{
		honey:   [][2]float64{{17, 7}, {18, 12}},
		caramel: [][2]float64{{21, 9}, {21, 15}},
	})
}

// smoothBargeMid draws a middle piece of the barge in tint t: its hold,
// split by a bulkhead.
func smoothBargeMid(pn pen, t tint) {
	pn.rect(0, 3, 24, 18, t.of(resurrect64.Ink))
	pn.rect(0, 4, 24, 16, t.of(resurrect64.Dusk))
	pn.rect(0, 5, 24, 14, t.of(resurrect64.Charcoal))
	bargeHold(pn, t, 0, 24, heaps{
		honey:   [][2]float64{{3, 7}, {18, 7}, {4, 12}, {16, 12}},
		caramel: [][2]float64{{7, 9}, {21, 9}, {7, 15}, {19, 15}},
	})
	pn.rect(11, 6, 2, 12, t.of(resurrect64.Charcoal))
}

// smoothBargeFront draws the barge's bow in tint t: the end of the hold
// and a rounded deck.
func smoothBargeFront(pn pen, t tint) {
	pn.rect(0, 3, 12, 18, t.of(resurrect64.Ink))
	pn.round(0, 3, 24, 18, 7, t.of(resurrect64.Ink))
	pn.rect(0, 4, 12, 16, t.of(resurrect64.Dusk))
	pn.round(0, 4, 23, 16, 6.2, t.of(resurrect64.Dusk))
	pn.rect(0, 5, 13, 14, t.of(resurrect64.Charcoal))
	bargeHold(pn, t, 0, 12, heaps{
		honey:   [][2]float64{{5, 7}, {3, 12}},
		caramel: [][2]float64{{9, 9}, {7, 15}},
	})
}

// planks draws the ferry's deck planks at columns xs.
func planks(pn pen, t tint, xs ...float64) {
	for _, x := range xs {
		pn.rect(x, 5, 1, 14, t.of(resurrect64.Dusk))
	}
}

// smoothFerryBack draws the ferry's stern in tint t: a rounded end in a
// crimson rim, the deck's planks and a life ring.
func smoothFerryBack(pn pen, t tint) {
	pn.round(0, 2, 8, 20, 3.4, t.of(resurrect64.Ink))
	pn.rect(4, 2, 20, 20, t.of(resurrect64.Ink))
	pn.round(1, 3, 7, 18, 2.6, t.of(resurrect64.Crimson))
	pn.rect(4, 3, 20, 18, t.of(resurrect64.Crimson))
	pn.round(2, 4, 6, 16, 1.8, t.of(resurrect64.Fog))
	pn.rect(4, 4, 20, 16, t.of(resurrect64.Fog))
	planks(pn, t, 13, 19)
	ring(pn, 6, 12, 1.5, 1.5, 1, t.of(resurrect64.Orange))
}

// smoothFerryDeck draws a piece of the ferry's open deck in tint t.
func smoothFerryDeck(pn pen, t tint) {
	pn.rect(0, 2, 24, 20, t.of(resurrect64.Ink))
	pn.rect(0, 3, 24, 18, t.of(resurrect64.Crimson))
	pn.rect(0, 4, 24, 16, t.of(resurrect64.Fog))
	planks(pn, t, 5, 11, 17, 23)
}

// smoothFerryCabin draws a piece of the ferry's cabin in tint t: a pale
// roof with lit windows along both sides and vents on top.
func smoothFerryCabin(pn pen, t tint) {
	pn.rect(0, 2, 24, 20, t.of(resurrect64.Ink))
	pn.rect(0, 3, 24, 18, t.of(resurrect64.Crimson))
	pn.rect(0, 4, 24, 16, t.of(resurrect64.Fog))
	pn.rect(0, 5, 24, 14, t.of(resurrect64.Ink))
	pn.rect(0, 6, 24, 12, t.of(resurrect64.Mist))
	for _, y := range []float64{6, 17} {
		for _, x := range []float64{2, 8, 14, 20} {
			pn.round(x, y, 2, 1, 0.4, t.of(resurrect64.Gold))
		}
	}
	for _, x := range []float64{5, 19} {
		pn.round(x, 11, 2, 2, 0.6, t.of(resurrect64.Fog))
	}
}

// bow fills the ferry's bow from column 0, inset by in: straight sides to
// column 12, then a curve round to its point.
func bow(pn pen, in float64, c int) {
	const cx, cy, rx, ry = 12, 12, 12, 10
	xy := []float64{0, cy - ry + in}
	const n = 16
	for i := range n + 1 {
		a := math.Pi * (float64(i)/n - 0.5)
		xy = append(xy, cx+(rx-in)*math.Cos(a), cy+(ry-in)*math.Sin(a))
	}
	xy = append(xy, 0, cy+ry-in)
	pn.poly(c, xy...)
}

// smoothFerryFront draws the ferry's bow in tint t, its deck planked to
// the point.
func smoothFerryFront(pn pen, t tint) {
	bow(pn, 0, t.of(resurrect64.Ink))
	bow(pn, 1, t.of(resurrect64.Crimson))
	bow(pn, 2, t.of(resurrect64.Fog))
	planks(pn, t, 5, 11)
}
