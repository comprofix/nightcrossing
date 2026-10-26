package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
)

// The predators, the pickups and the fruit in the smooth look (DESIGN
// §3.6), drawn from shapes in their sprites' places, sizes and colours:
// seen from above, facing right. Places are the sprites' columns and rows.

func init() {
	smoothArts["heron"] = func(pn pen, f int) { smoothHeron(pn, f, nil) }
	smoothArts["heron_flash"] = func(pn pen, f int) { smoothHeron(pn, f, heronFlashTint) }
	smoothArts["heron_jab"] = func(pn pen, _ int) { smoothHeronJab(pn) }
	smoothArts["fox"] = func(pn pen, _ int) { smoothFox(pn) }
	smoothArts["fox_eyes"] = func(pn pen, _ int) { foxEyes(pn) }
	smoothArts["rat"] = smoothRat
	smoothArts["rat_grab"] = smoothRatGrab
	smoothArts["snake"] = smoothSnake
	smoothArts["rustle"] = smoothRustle
	smoothArts["firefly"] = smoothFirefly
	smoothArts["dragonfly"] = smoothDragonfly
	smoothArts["fruit"] = smoothFruit
}

// tilted is the outline of an ellipse with radii rx and ry about x, y,
// turned by a radians, as x, y pairs for poly.
func tilted(x, y, rx, ry, a float64) []float64 {
	const n = 24
	xy := make([]float64, 0, 2*n)
	sin, cos := math.Sincos(a)
	for i := range n {
		t := 2 * math.Pi * float64(i) / n
		ex, ey := rx*math.Cos(t), ry*math.Sin(t)
		xy = append(xy, x+ex*cos-ey*sin, y+ex*sin+ey*cos)
	}
	return xy
}

// heronFlashTint is the heron lit up as it cocks its head, as the sprite
// file recolours it (D67).
var heronFlashTint = tint{
	resurrect64.Fog: resurrect64.White, resurrect64.Dusk: resurrect64.Mist, resurrect64.Mist: resurrect64.White,
}

// smoothHeron draws frame f of the heron in tint t: a grey body, its wing
// tips dark at the back, and its neck and head out in front, the beak
// amber; cocked back and glinting before it strikes, or reaching out into
// the next cell, where heron_jab draws its head.
func smoothHeron(pn pen, f int, t tint) {
	pn.ellipse(7.5, 12, 7.6, 7.1, t.of(resurrect64.Ink))
	pn.ellipse(7.6, 12, 6.7, 6.2, t.of(resurrect64.Fog))
	pn.poly(t.of(resurrect64.Dusk), 1.1, 8.4, 5, 11.4, 5, 12.6, 1.1, 15.6)
	if f == heronStep {
		pn.round(4, 8, 1, 1, 0.4, t.of(resurrect64.Dusk))
		pn.round(6, 17, 2, 1, 0.4, t.of(resurrect64.Dusk))
	} else {
		pn.round(4, 17, 2, 1, 0.4, t.of(resurrect64.Dusk))
	}
	switch f {
	case heronCock:
		pn.circle(17.5, 12, 3, t.of(resurrect64.Ink))
		pn.circle(17.5, 12, 2.1, t.of(resurrect64.Mist))
		pn.poly(t.of(resurrect64.Lemon), 19.6, 10.9, 23, 11.8, 23, 12.2, 19.6, 13.1)
		pn.circle(17.6, 11.5, 0.55, t.of(resurrect64.Ink))
	case heronReach:
		pn.rect(14.5, 10, 9.5, 4, t.of(resurrect64.Ink))
		pn.rect(14.5, 11, 9.5, 2, t.of(resurrect64.Mist))
	default:
		pn.round(14, 10, 6, 4, 1.5, t.of(resurrect64.Ink))
		pn.circle(19.5, 12, 3, t.of(resurrect64.Ink))
		pn.round(14.5, 11, 5, 2, 1, t.of(resurrect64.Mist))
		pn.circle(19.5, 12, 2.1, t.of(resurrect64.Mist))
		pn.poly(t.of(resurrect64.Amber), 21.6, 10.9, 24, 11.8, 24, 12.2, 21.6, 13.1)
		pn.circle(20, 11.4, 0.55, t.of(resurrect64.Ink))
	}
}

// smoothHeronJab draws a heron's jab: its neck, head and beak reaching in
// from the left edge.
func smoothHeronJab(pn pen) {
	pn.rect(0, 10, 11, 4, resurrect64.Ink)
	pn.circle(12, 12, 3, resurrect64.Ink)
	pn.rect(0, 11, 11, 2, resurrect64.Mist)
	pn.circle(12, 12, 2.1, resurrect64.Mist)
	pn.poly(resurrect64.Amber, 14, 10.9, 20, 11.8, 20, 12.2, 14, 13.1)
	pn.circle(12.6, 11.4, 0.55, resurrect64.Ink)
}

// foxHead is the outline of the fox's head inset by in: a broad brow
// narrowing to its chin.
func foxHead(in float64) []float64 {
	return []float64{
		3 + in, 5.6 + in, 21 - in, 5.6 + in, 22 - in, 8, 22 - in, 13, 20 - in*0.7, 16 - in*0.3,
		15.5, 19 - in, 12, 20 - in, 8.5, 19 - in, 4 + in*0.7, 16 - in*0.3, 2 + in, 13, 2 + in, 8,
	}
}

// smoothFox draws the fox's head peeking over its nook's edge: ears up,
// orange in a rust rim, a white muzzle and glowing eyes.
func smoothFox(pn pen) {
	for _, s := range []float64{1, -1} {
		x := func(v float64) float64 { return 12 + s*(v-12) }
		pn.poly(resurrect64.Ink, x(3.1), 0.8, x(9.4), 6.4, x(3.1), 8)
		pn.poly(resurrect64.Rust, x(4), 2.3, x(7.8), 6.2, x(4), 7.4)
		pn.poly(resurrect64.Shadow, x(5), 4.4, x(6.6), 6.2, x(5), 6.8)
	}
	pn.poly(resurrect64.Ink, foxHead(0)...)
	pn.poly(resurrect64.Rust, foxHead(1)...)
	pn.poly(resurrect64.Orange, foxHead(1.9)...)
	pn.poly(resurrect64.White, 2.8, 12, 7, 13.6, 10, 15, 14, 15, 17, 13.6, 21.2, 12,
		20.4, 14.6, 16, 18, 12, 19.3, 8, 18, 3.6, 14.6)
	pn.round(10, 15, 4, 2, 1, resurrect64.Ink)
	foxEyes(pn)
}

// foxEyes draws the fox's eyes glinting, lemon over gold.
func foxEyes(pn pen) {
	for _, x := range []float64{8, 16} {
		pn.ellipse(x, 10.1, 1.1, 1.1, resurrect64.Gold)
		pn.ellipse(x, 9.6, 1, 0.75, resurrect64.Lemon)
	}
}

// ratHead draws a water rat's head about x, y: ear, snout, pink nose and
// glinting eye.
func ratHead(pn pen, x, y float64) {
	pn.circle(x-1.2, y-3.2, 1.5, resurrect64.Ink)
	pn.circle(x-1.1, y-3.1, 0.9, resurrect64.Flamingo)
	pn.ellipse(x, y, 4, 3.2, resurrect64.Ink)
	pn.ellipse(x, y, 3.2, 2.5, resurrect64.Bark)
	pn.round(x-0.4, y+0.6, 4.6, 2.2, 1.1, resurrect64.Caramel)
	pn.circle(x+4.6, y+1.2, 0.8, resurrect64.Flamingo)
	pn.circle(x+4.3, y+2.4, 0.6, resurrect64.Mist)
	pn.ellipse(x+1, y-0.7, 1, 0.6, resurrect64.Gold)
	pn.ellipse(x+0.6, y-0.7, 0.6, 0.6, resurrect64.Lemon)
}

// waterline draws the water lapping along a rat from column x0 to x1 at
// row y, its ripples moved along by frame.
func waterline(pn pen, x0, x1, y float64, frame int) {
	pn.round(x0-1, y, x1-x0+2, 1, 0.5, resurrect64.Sky)
	pn.rect(x0, y, x1-x0, 1, resurrect64.Cobalt)
	for x := x0 + float64(frame); x < x1; x += 3 {
		pn.circle(x+0.5, y+1.5, 0.45, resurrect64.Sky)
	}
}

// smoothRat draws frame f of the water rat swimming: its brown back and
// head above the water, its tail trailing behind, paddling.
func smoothRat(pn pen, f int) {
	tail := 14.4
	if f == 1 {
		tail = 15.5
	}
	pn.stroke(resurrect64.Ink, 1.8, 5, 14.6, 0.6, tail)
	pn.stroke(resurrect64.Bark, 0.9, 5, 14.6, 0.6, tail)
	pn.ellipse(12, 13, 8.4, 4.6, resurrect64.Ink)
	pn.ellipse(12, 13, 7.5, 3.8, resurrect64.Bark)
	ratHead(pn, 18, 12.2)
	waterline(pn, 4-float64(f), 22-float64(f), 16, f)
}

// smoothRatGrab draws frame f of the water rat grabbing a platform's
// end: reared up out of the water, head and front paws over the platform,
// scrabbling.
func smoothRatGrab(pn pen, f int) {
	pn.stroke(resurrect64.Ink, 1.8, 5, 15.6, 1.2, 16.4)
	pn.stroke(resurrect64.Bark, 0.9, 5, 15.6, 1.2, 16.4)
	pn.poly(resurrect64.Ink, tilted(15.5, 12.4, 11, 4.8, -0.33)...)
	pn.poly(resurrect64.Bark, tilted(15.5, 12.4, 10.1, 4, -0.33)...)
	dx := -float64(f)
	pn.circle(29.5+dx, 14.2, 1.9, resurrect64.Ink)
	pn.circle(29.5+dx, 14.2, 1.2, resurrect64.Caramel)
	for _, x := range []float64{28.5, 30.5} {
		pn.circle(x+dx, 15.6, 0.55, resurrect64.Caramel)
	}
	ratHead(pn, 26.5, 8.6)
	waterline(pn, 4-float64(f), 24-float64(f), 17, f)
}

// snakeY is the middle of the snake's body at column x in frame f, waving
// as it slides, still at its head.
func snakeY(x float64, f int) float64 {
	a := 1.8 * min(max((34-x)/5, 0), 1)
	if f == 1 {
		a = -a
	}
	return 13 + a*math.Sin(2*math.Pi*(x-1)/16)
}

// smoothSnake draws frame f of the snake: olive, thin at its tail, a lime
// stripe and chartreuse marks along it, its head at the front with its
// forked tongue flicking out and in.
func smoothSnake(pn pen, f int) {
	width := func(x float64) float64 { return min(1.4+x*0.5, 5.4) }
	for _, c := range []struct {
		col int
		in  float64
	}{{resurrect64.Ink, 0}, {resurrect64.Olive, 1.4}} {
		for x := 1.0; x < 34; x++ {
			pn.stroke(c.col, max(width(x)-c.in, 0.6), x, snakeY(x, f), x+1, snakeY(x+1, f))
		}
	}
	for x := 3.0; x < 34; x++ {
		pn.stroke(resurrect64.Lime, 0.9, x, snakeY(x, f), x+1, snakeY(x+1, f))
	}
	for x := 7.0; x < 34; x += 6 {
		pn.circle(x, snakeY(x, f), 0.6, resurrect64.Chartreuse)
	}
	if f == 0 {
		pn.stroke(resurrect64.Red, 0.7, 43, 13, 45.4, 13, 47.4, 11.4)
		pn.stroke(resurrect64.Red, 0.7, 45.4, 13, 47.4, 14.6)
	} else {
		pn.stroke(resurrect64.Red, 0.7, 42.5, 13, 44.6, 13)
	}
	pn.ellipse(38.5, 13, 5, 4.3, resurrect64.Ink)
	pn.ellipse(38.5, 13, 4.2, 3.5, resurrect64.Olive)
	pn.ellipse(37.5, 13, 2.4, 0.5, resurrect64.Lime)
	for _, y := range []float64{11.1, 14.9} {
		pn.ellipse(40.2, y, 1, 0.75, resurrect64.Lemon)
	}
}

// rustleBlades are the tall verge grass where the snake will come out:
// each blade's foot, its tip's row and how far it sways.
var rustleBlades = [...][3]float64{
	{2, 15, 2}, {7, 12, 2.5}, {12, 10, 3}, {14, 13, 2}, {19, 12, 2.5}, {24, 10, 3},
	{26, 13, 2}, {31, 12, 2.5}, {36, 11, 3}, {38, 13, 2}, {43, 10, 3},
}

// smoothRustle draws frame f of the rustling grass: tall blades, dark at
// their feet and bright at their tips, swaying one way, then the other.
func smoothRustle(pn pen, f int) {
	cols := [...]int{resurrect64.Pine, resurrect64.Fern, resurrect64.Leaf, resurrect64.Chartreuse}
	for i, b := range rustleBlades {
		sway := b[2]
		if (f+i)%2 == 1 {
			sway = -sway
		}
		x0, tip := b[0]+0.5, b[1]
		for s := range len(cols) {
			t0, t1 := float64(s)/4, float64(s+1)/4
			pt := func(t float64) (float64, float64) { return x0 + sway*t*t, 19.5 - (19.5-tip)*t }
			ax, ay := pt(t0)
			bx, by := pt(t1)
			pn.stroke(cols[s], 1.1-0.2*float64(s), ax, ay, bx, by)
		}
	}
}

// smoothFirefly draws frame f of the firefly: pale wings beating, spread,
// half and folded, a dark head, and its tail glowing lemon.
func smoothFirefly(pn pen, f int) {
	wings := func(y, ry float64) {
		for _, s := range []float64{1, -1} {
			pn.ellipse(8.3, 8+s*y, 2, ry, resurrect64.Fog)
			pn.ellipse(7.9, 8+s*y, 1.7, ry-0.3, resurrect64.Mist)
		}
	}
	switch f {
	case 0:
		wings(4.6, 3)
	case 1:
		wings(2.8, 1.3)
	}
	pn.ellipse(3.2, 8, 2.4, 2.9, resurrect64.Lemon)
	pn.round(5, 6, 1, 4, 0.4, resurrect64.Chartreuse)
	pn.ellipse(7, 8, 1.3, 2, resurrect64.Shadow)
	pn.ellipse(9.9, 8, 2.1, 1.9, resurrect64.Ink)
	for _, y := range []float64{7.5, 8.5} {
		pn.circle(12.4, y, 0.5, resurrect64.White)
	}
	if f == 2 {
		pn.round(6, 6, 4, 4, 1.6, resurrect64.Fog)
		pn.round(6, 6, 3.4, 4, 1.6, resurrect64.Mist)
	}
}

// smoothDragonfly draws frame f of the dragonfly: two pairs of clear
// wings beating, a long banded tail, a teal thorax and a navy head with
// big aqua eyes.
func smoothDragonfly(pn pen, f int) {
	wing := func(x, y, rx, ry float64) {
		pn.ellipse(x, y, rx, ry, resurrect64.Seafoam)
		pn.stroke(resurrect64.Sky, 0.5, x-rx*0.5, y, x+rx*0.5, y)
	}
	for _, s := range []float64{1, -1} {
		switch f {
		case 0:
			wing(6.8, 8-s*4.3, 1.4, 2.4)
			wing(11.6, 8-s*4.3, 1.4, 2.4)
		case 1:
			wing(7.2, 8-s*3, 1.9, 1.1)
			wing(11.4, 8-s*3, 1.5, 1.1)
		default:
			wing(7, 8-s*1.5, 2.2, 0.6)
			wing(11.6, 8-s*1.5, 1.6, 0.6)
		}
	}
	pn.round(0, 7, 9.4, 2, 1, resurrect64.Jade)
	for x := 1.0; x < 9; x += 2 {
		pn.rect(x, 7, 1, 2, resurrect64.Aqua)
	}
	pn.ellipse(10, 8, 1.2, 2, resurrect64.DeepTeal)
	pn.ellipse(10.9, 8, 1, 1, resurrect64.Teal)
	pn.ellipse(12.5, 8, 1, 1.2, resurrect64.Navy)
	for _, y := range []float64{6.6, 9.4} {
		pn.circle(13.2, y, 0.9, resurrect64.Aqua)
	}
}

// stalk draws a fruit's stalk from x, y up to row y0, with a leaf
// beside it.
func stalk(pn pen, x, y, y0, leafX, leafY float64) {
	pn.stroke(resurrect64.Bark, 0.9, x, y, x, y0)
	pn.ellipse(leafX, leafY, 1.4, 0.8, resurrect64.Emerald)
	pn.ellipse(leafX, leafY, 0.6, 0.35, resurrect64.Leaf)
}

// smoothFruit draws fruit f in its 16px box, in the sprite's order:
// strawberry, apple, pear, banana, grapes, orange, pineapple, watermelon.
func smoothFruit(pn pen, f int) {
	switch f {
	case 0:
		berry := func(in float64) []float64 {
			return []float64{4 + in, 3.5 + in, 12 - in, 3.5 + in, 14 - in, 6, 13.6 - in, 9, 11 - in*0.7, 12.6 - in*0.5,
				8, 15 - in*1.2, 5 + in*0.7, 12.6 - in*0.5, 2.4 + in, 9, 2 + in, 6}
		}
		pn.poly(resurrect64.Ink, berry(0)...)
		pn.poly(resurrect64.Crimson, berry(1)...)
		lit := berry(1.4) // the crimson left as shade down its right side
		for i := 0; i < len(lit); i += 2 {
			lit[i] -= 0.6
		}
		pn.poly(resurrect64.Red, lit...)
		pn.round(3.2, 5, 1, 2, 0.5, resurrect64.White)
		for _, s := range [][2]float64{{6.5, 5.5}, {10.5, 5.5}, {4.8, 7.5}, {8.5, 7.5}, {6.5, 9.5}, {10.5, 9.5}, {7, 11.5}} {
			pn.ellipse(s[0], s[1], 0.45, 0.6, resurrect64.Gold)
		}
		for _, x := range []float64{5.6, 8, 10.4} {
			pn.ellipse(x, 2.8, 1.5, 1, resurrect64.Emerald)
		}
		pn.ellipse(8, 2, 0.8, 0.9, resurrect64.Leaf)
	case 1:
		stalk(pn, 7.5, 4, 0.4, 10.6, 1.8)
		pn.ellipse(8.5, 8.7, 6.2, 5.6, resurrect64.Ink)
		pn.ellipse(8.5, 8.7, 5.3, 4.7, resurrect64.Crimson)
		pn.ellipse(8.1, 8.4, 4.9, 4.3, resurrect64.Red)
		pn.ellipse(7.8, 3.6, 1.4, 0.7, resurrect64.Ink)
		pn.round(3.4, 5.4, 2, 1, 0.5, resurrect64.White)
		pn.round(3.4, 6.4, 1, 1, 0.5, resurrect64.White)
	case 2:
		stalk(pn, 7.5, 3.4, 0.2, 9.8, 1.8)
		for _, c := range []struct {
			col int
			in  float64
		}{{resurrect64.Ink, 0}, {resurrect64.Olive, 1}, {resurrect64.Lime, 1.6}} {
			pn.ellipse(8, 6.4, 2.8-c.in, 3.4-c.in, c.col)
			pn.ellipse(8, 10.6, 5.6-c.in, 4.6-c.in, c.col)
		}
		pn.ellipse(5.4, 8.4, 0.8, 1, resurrect64.Chartreuse)
		pn.ellipse(4.6, 6.4, 0.5, 0.6, resurrect64.Lemon)
	case 3:
		band := func(r0, r1, a0, a1 float64, c int) {
			const n = 14
			var xy []float64
			for i := range n + 1 {
				a := a0 + (a1-a0)*float64(i)/n
				xy = append(xy, 4+r1*math.Cos(a), r1*math.Sin(a))
			}
			for i := n; i >= 0; i-- {
				a := a0 + (a1-a0)*float64(i)/n
				xy = append(xy, 4+r0*math.Cos(a), r0*math.Sin(a))
			}
			pn.poly(c, xy...)
		}
		band(7.4, 12.4, 0.26, 1.78, resurrect64.Ink)
		band(8.2, 11.6, 0.3, 1.74, resurrect64.Gold)
		band(10.6, 11.6, 0.34, 1.7, resurrect64.Amber)
		band(9, 9.8, 0.4, 1.6, resurrect64.Lemon)
		pn.round(12.6, 1.6, 2, 1.6, 0.5, resurrect64.Bark)
		pn.circle(2.6, 10, 0.7, resurrect64.Bark)
	case 4:
		stalk(pn, 7.5, 3.2, 0.2, 9, 1.8)
		for _, g := range [][2]float64{{5, 4.4}, {9, 4.4}, {4.6, 6.6}, {8.6, 6.6}, {12.4, 6.6}, {6.4, 8.6}, {10.4, 8.6}, {7, 10.6}, {11, 10.6}, {9, 12.6}} {
			pn.circle(g[0], g[1], 2.1, resurrect64.Ink)
		}
		for _, g := range [][2]float64{{5, 4.4}, {9, 4.4}, {4.6, 6.6}, {8.6, 6.6}, {12.4, 6.6}, {6.4, 8.6}, {10.4, 8.6}, {7, 10.6}, {11, 10.6}, {9, 12.6}} {
			pn.circle(g[0], g[1], 1.5, resurrect64.Grape)
			pn.circle(g[0]-0.2, g[1]-0.2, 1.2, resurrect64.Violet)
			pn.circle(g[0]-0.6, g[1]-0.6, 0.45, resurrect64.Lavender)
		}
	case 5:
		stalk(pn, 7.5, 3.4, 0.6, 9.6, 2.2)
		pn.circle(8, 8.6, 6, resurrect64.Ink)
		pn.circle(8, 8.6, 5.2, resurrect64.Amber)
		pn.circle(7.5, 8.1, 4.7, resurrect64.Orange)
		pn.ellipse(4.8, 5.8, 1, 0.7, resurrect64.Caramel)
		pn.circle(4, 7, 0.5, resurrect64.Caramel)
	case 6:
		for _, l := range [][2]float64{{4, 0.4}, {6.2, 0}, {8, 0}, {9.8, 0}, {12, 0.4}} {
			pn.stroke(resurrect64.Emerald, 1.2, 8, 4.4, l[0], l[1])
		}
		pn.stroke(resurrect64.Leaf, 0.6, 8, 3.8, 8, 1)
		pn.ellipse(8, 9.2, 5, 5, resurrect64.Ink)
		pn.ellipse(8, 9.2, 4.2, 4.2, resurrect64.Honey)
		for _, d := range [][2]float64{{6, 6}, {10, 6}, {4.5, 8}, {8, 8}, {11.5, 8}, {6, 10}, {10, 10}, {4.5, 12}, {8, 12}, {11.5, 12}} {
			pn.circle(d[0], d[1], 0.55, resurrect64.Amber)
			pn.circle(d[0]+1, d[1]+1, 0.4, resurrect64.Bark)
		}
	default:
		slice := func(in float64, c int) {
			const n = 16
			var xy []float64
			for i := range n + 1 {
				a := math.Pi * float64(i) / n
				xy = append(xy, 8-(6.5-in)*math.Cos(a), 4.5+in*0.5+(10.4-in*1.4)*math.Sin(a))
			}
			pn.poly(c, xy...)
		}
		slice(0, resurrect64.Ink)
		slice(0.8, resurrect64.Forest)
		slice(1.4, resurrect64.Emerald)
		slice(2.1, resurrect64.White)
		slice(2.7, resurrect64.Flamingo)
		slice(3.3, resurrect64.Red)
		for _, s := range [][2]float64{{5.5, 6.5}, {10.5, 6.5}, {8.5, 7.5}, {6.5, 8.6}, {9.5, 9.6}} {
			pn.ellipse(s[0], s[1], 0.45, 0.7, resurrect64.Ink)
		}
	}
}

// drawSmoothProwl draws the water rats and the snake, with their marks on
// the water, in the smooth look on dst at scale k.
func (p *Play) drawSmoothProwl(ctx *scene.Context, dst *ebiten.Image, k int) {
	pn := newPen(dst, k)
	p.eachProwl(ctx, func(cx, cy float64, i int) {
		p.eachRipple(i, func(rx, ry float64, c int) { ring(pn, cx+0.5, cy+0.5, rx, ry, 0.7, c) })
	}, func(x, y, back float64) {
		eachWakeDot(func(dx, dy float64, c int) { pn.circle(x+back*dx+0.5, y+dy+0.5, 0.55, c) })
	}, func(pc piece) { p.drawSmoothPiece(ctx, dst, pc, k) })
}
