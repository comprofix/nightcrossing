package play

import (
	"math"

	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
)

// The bonus levels' own things in smooth art (DESIGN §3.6, §6.7): the
// street sweeper and the school sign. The parked cars and the goods yard's
// rakes are the road's cars and the train's.

func init() {
	smoothArts["sweeper"] = smoothSweeper
	smoothArts["schoolsign"] = func(pn pen, _ int) { smoothSchoolSign(pn) }
}

// smoothSweeper is the street sweeper facing right, frame f of its brushes
// turning, its wheels rolling and its beacon flashing: a long grey body
// with an orange band, its cab at the front and a round brush either side
// of it.
func smoothSweeper(pn pen, f int) {
	const (
		ink, body, edge, band = resurrect64.Ink, resurrect64.Fog, resurrect64.Lilac, resurrect64.Orange
	)
	for _, y := range []float64{1.8, 22.2} { // the brushes, turning
		pn.circle(42.5, y, 2.7, ink)
		pn.circle(42.5, y, 2.1, resurrect64.Bark)
		for i := range 6 {
			a := float64(i)*math.Pi/3 + float64(f)*math.Pi/6
			pn.stroke(resurrect64.Caramel, 0.6, 42.5+0.5*math.Cos(a), y+0.5*math.Sin(a), 42.5+2*math.Cos(a), y+2*math.Sin(a))
		}
	}
	for _, x := range []float64{5.5, 25.5} { // the wheels, their treads rolling
		for _, y := range []float64{2.6, 19.4} {
			pn.round(x, y, 6, 2, 0.8, resurrect64.Charcoal)
			for t := 0.0; t < 6; t += 2 {
				pn.rect(x+0.6+t+float64(f), y+0.3, 0.7, 1.4, resurrect64.Dusk)
			}
		}
	}
	pn.round(1, 4, 45, 16, 2, ink)
	pn.round(2, 5, 35, 14, 1.4, body)
	pn.round(2, 5, 1.2, 14, 0.6, edge)
	for _, y := range []float64{8, 15} {
		pn.rect(3, y, 34, 1, edge)
	}
	pn.rect(3, 11, 34, 2, band)
	pn.round(38, 5, 7, 14, 1.4, resurrect64.White) // the cab
	pn.round(41, 5.4, 3, 13.2, 1, resurrect64.Sky)
	beacon := resurrect64.Amber
	if f%2 == 1 {
		beacon = resurrect64.Lemon
	}
	pn.circle(39.4, 12, 1.3, beacon)
	for _, y := range []float64{5.8, 18.2} {
		pn.circle(44.4, y, 0.7, resurrect64.Gold)
	}
}

// smoothSchoolSign is a gold board with a toad on it, on a post.
func smoothSchoolSign(pn pen) {
	const ink = resurrect64.Ink
	pn.rect(4, 9.5, 4, 7, ink) // the post and its foot
	pn.rect(5, 9.5, 2, 6.5, resurrect64.Bark)
	pn.round(3, 15.6, 6, 1.6, 0.6, ink)
	pn.round(2, 16.4, 8, 1.6, 0.8, ink)
	pn.round(0, 0, 12, 10, 2, ink) // the board
	pn.round(1, 1, 10, 8, 1.3, resurrect64.Gold)
	pn.ellipse(6, 5.4, 3.1, 2.4, ink) // the toad
	pn.ellipse(6, 5.5, 2.3, 1.7, resurrect64.Forest)
	for _, x := range []float64{4.3, 7.7} {
		pn.circle(x, 3.2, 1.1, ink)
		pn.circle(x, 3.3, 0.5, resurrect64.Forest)
	}
	pn.stroke(ink, 0.9, 3.4, 7.5, 8.6, 7.5)
}
