package play

import (
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
)

// The road's vehicles, the train and the emergency vehicles in the smooth
// look (DESIGN §3.6), drawn from shapes in their sprites' places, sizes
// and colours: seen from above, facing right. Places are the sprites'
// columns and rows; a piece of a truck or the train runs to its box's edge
// with square ends, so the pieces meet.

func init() {
	for name, c := range map[string][2]int{
		"car_red":    {resurrect64.Red, resurrect64.Brick},
		"car_blue":   {resurrect64.Blue, resurrect64.Cobalt},
		"car_yellow": {resurrect64.Amber, resurrect64.Rust},
	} {
		smoothArts[name] = func(pn pen, f int) { smoothCar(pn, f, c[0], c[1]) }
	}
	smoothArts["van"] = smoothVan
	smoothArts["bike"] = smoothBike
	smoothArts["truck_back"] = func(pn pen, _ int) { smoothTruckBack(pn) }
	smoothArts["truck_mid"] = func(pn pen, _ int) { smoothTruckMid(pn) }
	smoothArts["truck_front"] = func(pn pen, _ int) { smoothTruckFront(pn) }
	smoothArts["car_back"] = func(pn pen, _ int) { smoothCarriage(pn, carriageBack) }
	smoothArts["car_mid"] = func(pn pen, _ int) { smoothCarriage(pn, carriageMid) }
	smoothArts["car_front"] = func(pn pen, _ int) { smoothCarriage(pn, carriageFront) }
	smoothArts["engine_back"] = func(pn pen, _ int) { smoothEngineBack(pn) }
	smoothArts["engine_mid"] = func(pn pen, _ int) { smoothEngineMid(pn) }
	smoothArts["engine_nose"] = func(pn pen, _ int) { smoothEngineNose(pn) }
	smoothArts["police"] = smoothPolice
	smoothArts["ambulance"] = smoothAmbulance
	smoothArts["fire_truck"] = smoothFireTruck
}

// wheel draws a wheel n long from column x, showing at row y beside the
// body, above it if top, its tread turned by frame.
func wheel(pn pen, x, y float64, n, frame int, top bool) {
	wy := y - 1.3
	if top {
		wy = y - 0.3
	}
	pn.round(x, wy, float64(n), 2.6, 0.8, resurrect64.Charcoal)
	for i := range n {
		if (i+frame)%2 == 1 {
			pn.rect(x+float64(i)+0.2, y, 0.6, 1, resurrect64.Dusk)
		}
	}
}

// wheels draws a vehicle's wheels at each of columns xs, n long, above its
// body at row 3 and below it at row 20.
func wheels(pn pen, n, frame int, xs ...float64) {
	for _, x := range xs {
		wheel(pn, x, 3, n, frame, true)
		wheel(pn, x, 20, n, frame, false)
	}
}

// vehicleBody draws a body w long in rows 4 to 19, in colour c in an ink
// rim, with tail lights at its back and headlamps at its front.
func vehicleBody(pn pen, w float64, c int) {
	pn.round(1, 4, w-2, 16, 2.2, resurrect64.Ink)
	pn.round(2, 5, w-4, 14, 1.4, c)
	for _, y := range []float64{5, 18} {
		pn.round(2, y, 1.2, 1, 0.45, resurrect64.Crimson)
		pn.round(w-3.2, y, 1.2, 1, 0.45, resurrect64.Gold)
	}
}

// stripes draws a stripe a row in from each long side of a body, from
// column x0 to x1.
func stripes(pn pen, x0, x1 float64, c int) {
	pn.rect(x0, 6, x1-x0, 1, c)
	pn.rect(x0, 17, x1-x0, 1, c)
}

// window draws a pane from column x0 to x1, rows 8 to 15.
func window(pn pen, x0, x1 float64, c int) { pn.round(x0, 8, x1-x0, 8, 0.8, c) }

// smoothCar draws frame f of a car, in colour c with its roof in roof.
func smoothCar(pn pen, f, c, roof int) {
	wheels(pn, 4, f, 4, 22)
	vehicleBody(pn, 32, c)
	pn.round(7, 7, 20, 10, 1.6, resurrect64.Ink)
	window(pn, 8, 11, resurrect64.Sky)
	pn.round(12, 8, 8, 8, 0.6, roof)
	window(pn, 21, 26, resurrect64.Sky)
}

// smoothVan draws frame f of the van: a long grey body ribbed along its
// roof, and the cab's windscreen at the front.
func smoothVan(pn pen, f int) {
	wheels(pn, 5, f, 5, 28)
	vehicleBody(pn, 40, resurrect64.Fog)
	pn.rect(2, 6, 1, 12, resurrect64.Lilac)
	pn.rect(2, 8, 29, 1, resurrect64.Lilac)
	pn.rect(2, 15, 29, 1, resurrect64.Lilac)
	pn.rect(31, 5, 1, 14, resurrect64.Ink)
	pn.round(32, 5, 3, 14, 0.8, resurrect64.Sky)
}

// smoothBike draws frame f of the motorbike and its rider: a thin bike
// with a handlebar across it, the rider in an orange helmet.
func smoothBike(pn pen, f int) {
	for _, x := range []float64{2, 19} {
		wheel(pn, x, 9, 4, f, true)
		wheel(pn, x, 14, 4, f, false)
	}
	pn.stroke(resurrect64.Ink, 1.8, 15.8, 8.5, 15.8, 15.5)
	pn.round(1, 10, 23, 4, 1.4, resurrect64.Ink)
	pn.round(2, 11, 20, 2, 0.8, resurrect64.Fog)
	pn.rect(2, 11, 1, 2, resurrect64.Crimson)
	pn.round(22, 10, 1, 4, 0.4, resurrect64.Gold)
	pn.ellipse(12, 12, 3.1, 4.1, resurrect64.Ink)
	pn.ellipse(12, 12, 2.3, 3.3, resurrect64.Indigo)
	pn.circle(12.5, 12, 2.3, resurrect64.Orange)
}

// truckRibs draws the trailer's ribs across it at columns xs.
func truckRibs(pn pen, xs ...float64) {
	for _, x := range xs {
		pn.rect(x, 3, 1, 18, resurrect64.Lilac)
	}
}

// smoothTruckBack draws the back of the truck's trailer, with its tail
// lights.
func smoothTruckBack(pn pen) {
	pn.round(1, 2, 6, 20, 1.6, resurrect64.Ink)
	pn.rect(3, 2, 21, 20, resurrect64.Ink)
	pn.round(2, 3, 5, 18, 0.9, resurrect64.Fog)
	pn.rect(4, 3, 20, 18, resurrect64.Fog)
	pn.rect(2, 5, 1, 14, resurrect64.Lilac)
	truckRibs(pn, 7, 13, 19)
	pn.round(2, 3, 1.1, 2, 0.4, resurrect64.Crimson)
	pn.round(2, 19, 1.1, 2, 0.4, resurrect64.Crimson)
}

// smoothTruckMid draws a middle piece of the truck's trailer.
func smoothTruckMid(pn pen) {
	pn.rect(0, 2, 24, 20, resurrect64.Ink)
	pn.rect(0, 3, 24, 18, resurrect64.Fog)
	truckRibs(pn, 5, 11, 17, 23)
}

// smoothTruckFront draws the trailer's front end, the hitch and the cab.
func smoothTruckFront(pn pen) {
	pn.rect(0, 2, 2, 20, resurrect64.Ink)
	pn.rect(0, 3, 1, 18, resurrect64.Fog)
	pn.rect(2, 9, 3, 2, resurrect64.Charcoal)
	pn.round(5, 4, 17, 16, 2, resurrect64.Ink)
	pn.round(6, 5, 15, 14, 1.2, resurrect64.Amber)
	pn.rect(6, 6, 1, 12, resurrect64.Rust)
	pn.rect(16, 5, 1, 14, resurrect64.Ink)
	pn.round(17, 5, 3, 14, 0.6, resurrect64.Sky)
	pn.round(19.9, 5, 1.1, 1, 0.4, resurrect64.Gold)
	pn.round(19.9, 18, 1.1, 1, 0.4, resurrect64.Gold)
}

// A carriage's pieces.
const (
	carriageBack = iota
	carriageMid
	carriageFront
)

// carriages are a carriage's pieces as their sprites have them: the
// columns the body runs between, and its windows along each side.
var carriages = [...]struct {
	x0, x1  float64
	windows [][2]float64
}{
	carriageBack:  {3, 24, [][2]float64{{3, 7}, {8, 13}, {14, 19}, {20, 24}}},
	carriageMid:   {0, 24, [][2]float64{{0, 1}, {2, 7}, {8, 13}, {14, 19}, {20, 24}}},
	carriageFront: {0, 21, [][2]float64{{0, 1}, {2, 7}, {8, 13}, {14, 21}}},
}

// smoothCarriage draws piece part of a carriage: dark blue with a stripe
// along its roof and lit windows along both sides; the back and front
// pieces end short of their boxes, so the track shows between carriages.
func smoothCarriage(pn pen, part int) {
	cr := carriages[part]
	switch part {
	case carriageBack:
		pn.round(2, 2, 6, 20, 1.2, resurrect64.Ink)
		pn.rect(4, 2, 20, 20, resurrect64.Ink)
	case carriageFront:
		pn.round(16, 2, 6, 20, 1.2, resurrect64.Ink)
		pn.rect(0, 2, 18, 20, resurrect64.Ink)
	default:
		pn.rect(0, 2, 24, 20, resurrect64.Ink)
	}
	pn.rect(cr.x0, 4, cr.x1-cr.x0, 16, resurrect64.Indigo)
	stripe0, stripe1 := cr.x0, cr.x1
	switch part {
	case carriageBack:
		stripe0 = 4
	case carriageFront:
		stripe1 = 20
	}
	pn.rect(stripe0, 10, stripe1-stripe0, 4, resurrect64.Cobalt)
	for _, y := range []float64{3, 20} {
		for _, w := range cr.windows {
			pn.rect(w[0], y, w[1]-w[0], 1, resurrect64.Honey)
		}
	}
	if part == carriageMid {
		pn.round(5, 6, 2, 2, 0.5, resurrect64.Fog)
		pn.round(16, 16, 2, 2, 0.5, resurrect64.Fog)
	}
}

// smoothEngineBack draws the back of the engine, red, with a grille on
// its roof.
func smoothEngineBack(pn pen) {
	pn.round(2, 2, 6, 20, 1.2, resurrect64.Ink)
	pn.rect(4, 2, 20, 20, resurrect64.Ink)
	pn.rect(3, 3, 21, 18, resurrect64.Brick)
	pn.rect(4, 4, 20, 16, resurrect64.Red)
	pn.round(7, 6, 14, 12, 1, resurrect64.Ink)
	for i := range 12 {
		c := resurrect64.Charcoal
		if i%2 == 1 {
			c = resurrect64.Fog
		}
		pn.rect(8+float64(i), 7, 1, 10, c)
	}
}

// smoothEngineMid draws the middle of the engine: a hatch and a vent on
// its roof.
func smoothEngineMid(pn pen) {
	pn.rect(0, 2, 24, 20, resurrect64.Ink)
	pn.rect(0, 3, 24, 18, resurrect64.Brick)
	pn.rect(0, 4, 24, 16, resurrect64.Red)
	pn.round(2, 5, 10, 12, 1, resurrect64.Ink)
	pn.round(3, 6, 8, 10, 0.6, resurrect64.Fog)
	pn.rect(4, 7, 6, 8, resurrect64.Mist)
	pn.round(14, 5, 8, 12, 1, resurrect64.Ink)
	pn.rect(15, 6, 6, 10, resurrect64.Charcoal)
	for _, y := range []float64{7, 10, 13} {
		pn.rect(16, y, 4, 2, resurrect64.Fog)
	}
}

// smoothEngineNose draws the engine's rounded nose: the cab's window, an
// amber panel and the headlamps.
func smoothEngineNose(pn pen) {
	pn.rect(0, 2, 12, 20, resurrect64.Ink)
	pn.round(0, 2, 23, 20, 6, resurrect64.Ink)
	pn.rect(0, 3, 12, 18, resurrect64.Brick)
	pn.round(0, 3, 22, 18, 5.2, resurrect64.Brick)
	pn.rect(0, 4, 12, 16, resurrect64.Red)
	pn.round(0, 4, 21, 16, 4.4, resurrect64.Red)
	pn.round(17, 5, 5, 14, 2.6, resurrect64.Amber)
	pn.round(20.8, 6, 1, 2, 0.5, resurrect64.White)
	pn.round(20.8, 16, 1, 2, 0.5, resurrect64.White)
	pn.round(4, 8, 12, 8, 1.8, resurrect64.Ink)
	pn.round(5, 9, 10, 6, 1.2, resurrect64.Sky)
}

// lightBar draws an emergency vehicle's light bar from column x, its halves
// in top and bottom, in an ink frame.
func lightBar(pn pen, x float64, top, bottom int) {
	pn.round(x, 7, 6, 10, 0.8, resurrect64.Ink)
	pn.round(x+1, 8, 4, 4, 0.5, top)
	pn.round(x+1, 12, 4, 4, 0.5, bottom)
}

// smoothPolice draws frame f of the police car: white with navy stripes,
// its light bar's red half lit in frame 0 and its blue half in frame 1.
func smoothPolice(pn pen, f int) {
	wheels(pn, 4, 0, 5, 36)
	vehicleBody(pn, 48, resurrect64.White)
	stripes(pn, 3, 45, resurrect64.Navy)
	pn.round(10, 7, 28, 10, 1.6, resurrect64.Ink)
	window(pn, 11, 14, resurrect64.Sky)
	pn.rect(15, 8, 5, 8, resurrect64.White)
	pn.rect(26, 8, 5, 8, resurrect64.White)
	window(pn, 32, 37, resurrect64.Sky)
	top, bottom := resurrect64.Red, resurrect64.Indigo
	if f == 1 {
		top, bottom = resurrect64.Brick, resurrect64.Blue
	}
	lightBar(pn, 20, top, bottom)
}

// smoothAmbulance draws frame f of the ambulance: white with red stripes
// and a red cross, its light bar's red half lit in frame 0 and its white
// half in frame 1.
func smoothAmbulance(pn pen, f int) {
	wheels(pn, 4, 0, 6, 30, 58)
	vehicleBody(pn, 72, resurrect64.White)
	stripes(pn, 3, 69, resurrect64.Red)
	pn.rect(23, 8, 4, 8, resurrect64.Red)
	pn.rect(19, 10, 12, 4, resurrect64.Red)
	top, bottom := resurrect64.Red, resurrect64.Fog
	if f == 1 {
		top, bottom = resurrect64.Brick, resurrect64.White
	}
	lightBar(pn, 43, top, bottom)
	pn.round(50, 7, 7, 10, 0.8, resurrect64.Ink)
	window(pn, 51, 56, resurrect64.Sky)
}

// smoothFireTruck draws frame f of the fire truck: red, a ladder along
// its back, the cab behind a dark line, its light bar's red half lit in
// frame 0 and its amber half in frame 1.
func smoothFireTruck(pn pen, f int) {
	wheels(pn, 4, 0, 6, 22, 66, 84)
	vehicleBody(pn, 96, resurrect64.Red)
	stripes(pn, 3, 62, resurrect64.Brick)
	stripes(pn, 63, 93, resurrect64.Brick)
	pn.rect(62, 5, 1, 14, resurrect64.Ink)
	pn.rect(4, 9, 57, 1, resurrect64.Fog)
	pn.rect(4, 14, 57, 1, resurrect64.Fog)
	for x := 6.0; x <= 58; x += 4 {
		pn.rect(x, 10, 1, 4, resurrect64.Charcoal)
	}
	top, bottom := resurrect64.Flamingo, resurrect64.Rust
	if f == 1 {
		top, bottom = resurrect64.Brick, resurrect64.Amber
	}
	lightBar(pn, 64, top, bottom)
	pn.round(75, 7, 7, 10, 0.8, resurrect64.Ink)
	window(pn, 76, 81, resurrect64.Sky)
}
