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

	breakGapPx = 3 // how far apart a breaking ship's pieces drift
)

// pieces are the sprites an entity is built from, left to right as it
// faces right. Entities not listed are one sprite named by their ID
// (DESIGN §3.3: long vehicles and platforms are built from end, middle and
// end pieces).
var pieces = map[string][]string{
	config.Truck.ID:    {"truck_back", "truck_mid", "truck_front"},
	config.LogShort.ID: {"log_back", "log_mid", "log_front"},
	config.LogLong.ID:  {"log_back", "log_mid", "log_mid", "log_mid", "log_front"},
	config.Barge.ID:    {"barge_back", "barge_mid", "barge_mid", "barge_front"},
	config.Ferry.ID:    {"ferry_back", "ferry_deck", "ferry_cabin", "ferry_cabin", "ferry_deck", "ferry_front"},
	config.Pads2.ID:    {"pad", "pad"},
	config.Pads3.ID:    {"pad", "pad", "pad"},
}

// entitySprites is the sprites for entity e, left to right facing right.
func entitySprites(e config.Entity) []string {
	if ps, ok := pieces[e.ID]; ok {
		return ps
	}
	return []string{e.ID}
}

// A diving platform's frames (DESIGN §3.3: they flash before going under).
const (
	frameAfloat = 0
	frameFlash  = 1
	frameHalf   = 2 // sinking or rising
	frameRing   = 3 // under: only a ring on the water
)

// stageFrame is the frame for a platform at stage s; the warning flashes
// between afloat and the flash frame.
func stageFrame(s world.Stage, phase int) int {
	switch s {
	case world.Warning:
		if (phase/config.DiveFlashTicks)%2 == 0 {
			return frameFlash
		}
	case world.Sinking, world.Rising:
		return frameHalf
	case world.Under:
		return frameRing
	}
	return frameAfloat
}

// drawField draws the rows (grass where it is safe, the track, road and
// water), the lamps, what moves along them, and the hero, then lights it
// all for the night and draws the particles on top, so sparks and drops
// stay bright (DESIGN §3.3). Everything on the field is clipped to it, so
// vehicles and platforms slide in and out at its edges. In the smooth look
// the field's pixel art is enlarged, and lit, rained on and sparked at the
// screen's own size (DESIGN §3.6).
func (p *Play) drawField(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	k := p.scale(ctx)
	bounds := image.Rect(config.FieldX, config.FieldY,
		config.FieldX+w.Cols*config.Cell, config.FieldY+w.Rows*config.Cell)
	look := p.look() // Stroll's night, or Last Train's day and night (D39)
	if ctx.Settings.SmoothLight || ctx.Settings.Shown(ctx.Looks).Art == config.ArtSmooth {
		look.Bands = 0 // the Light option (Options: Light: Smooth); the smooth look's light is always smooth (D80)
	}
	if !ctx.Settings.Glow {
		look.Glow = 0
	}
	look = p.lightning(look, ctx.Settings.ReducedMotion) // the storm sky (D49)
	if ctx.Settings.HighContrast {
		look.Level = max(look.Level, highContrastLevel)
	}
	p.light.SetScale(k)
	lit := p.light.Begin(look)
	if k == 1 {
		field := lit.SubImage(bounds).(*ebiten.Image)
		p.drawGround(ctx, field)
		p.drawMoving(ctx, field)
		p.drawHero(ctx, field)
	} else {
		field := lit.SubImage(image.Rectangle{Min: bounds.Min.Mul(k), Max: bounds.Max.Mul(k)}).(*ebiten.Image)
		p.drawSmoothGround(ctx, field, k)
		for r := range w.Lanes {
			if w.Lanes[r].Kind == config.Bank {
				p.drawSmoothNookToads(ctx, field, k, r)
			}
		}
		p.drawSmoothMoving(ctx, lit, field, bounds, k)
		p.eachToad(func(pl *world.Player, name string, dx float64) { p.drawSmoothToad(ctx, field, k, pl, name, dx) })
	}

	p.addLights(ctx, p.light)
	p.light.Draw(dst, 0, 0)
	out := dst.SubImage(image.Rectangle{Min: bounds.Min.Mul(k), Max: bounds.Max.Mul(k)}).(*ebiten.Image)
	if k == 1 {
		p.eachFlyer(ctx, func(pc piece) { drawPiece(ctx, out, pc) })
		if ctx.Settings.HighContrast {
			p.drawOutlines(ctx, out)
		}
	} else {
		p.eachFlyer(ctx, func(pc piece) { p.drawSmoothPiece(ctx, out, pc, k) })
		if ctx.Settings.HighContrast {
			p.drawSmoothOutlines(ctx, out, k)
		}
	}
	p.drawRain(out, k)
	p.drawWind(out, k)
	if k == 1 {
		p.parts.Draw(out, 0, 0, 1)
	} else {
		p.parts.DrawSmooth(out, 0, 0, float64(k))
	}
	if debug.Enabled {
		p.inPixels(out, k, func(img *ebiten.Image) { p.drawHitBoxes(ctx, img.SubImage(bounds).(*ebiten.Image)) })
	}
}

// drawGround draws the field's ground in the pixel look, before it is lit:
// the rows, the lamps and signals, and the water's currents and wakes.
func (p *Play) drawGround(ctx *scene.Context, field *ebiten.Image) {
	w := p.world
	sheet := ctx.Sheet(Sheet)
	for r := range w.Lanes {
		switch w.Lanes[r].Kind {
		case config.Safe:
			p.drawTiles(field, sheet.Sprite("verge"), r)
		case config.Bank:
			p.drawBank(ctx, field, r)
		case config.Rail:
			p.drawTiles(field, sheet.Sprite("track"), r)
		case config.Road:
			p.drawRoad(field, r)
		case config.River:
			p.drawWater(field, r)
			p.drawChop(field, r)
		}
	}
	p.drawFlood(field)
	p.drawRoadSheen(field)
	p.drawRainbow(field, 1) // after the storm, over the water (DESIGN §6.7)
	lamp := sheet.Sprite("lamp")
	for _, lp := range p.streetLamps() {
		lamp.Draw(field, 0, cellX(lp.col), cellY(lp.row), false)
	}
	p.drawSignals(ctx, field)
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.River {
			p.drawCurrent(field, r)
			p.drawWakes(field, r)
		}
	}
}

// drawSmoothMoving draws what drawMoving does, and the nooks' other things,
// in the smooth look on field, lit's field at scale k: the vehicles,
// platforms, predators, fruit, the emergency vehicle and the train in their
// smooth art, and between them, in the same order, what is still pixel
// art, enlarged.
func (p *Play) drawSmoothMoving(ctx *scene.Context, lit, field *ebiten.Image, bounds image.Rectangle, k int) {
	w := p.world
	draw := func(pc piece) { p.drawSmoothPiece(ctx, field, pc, k) }
	for r := range w.Lanes {
		p.eachEntityPiece(ctx, r, draw)
	}
	for r := range w.Lanes {
		if w.Lanes[r].Kind == config.Bank {
			p.inPixels(lit, k, func(art *ebiten.Image) { p.drawSchoolSigns(ctx, art.SubImage(bounds).(*ebiten.Image), r) })
			p.eachNookThing(ctx, draw)
		}
		p.eachHeron(r, draw)
	}
	p.drawSmoothProwl(ctx, field, k)
	p.inPixels(lit, k, func(art *ebiten.Image) {
		field := art.SubImage(bounds).(*ebiten.Image)
		p.drawParked(ctx, field)
		p.drawYard(ctx, field)
	})
	p.eachPick(ctx, draw)
	p.inPixels(lit, k, func(art *ebiten.Image) { p.drawSweepers(ctx, art.SubImage(bounds).(*ebiten.Image)) })
	p.eachSiren(ctx, draw)
	p.eachTrainPiece(ctx, draw)
	p.drawFog(field, k)
	p.drawMist(field, k)
}

// drawMoving draws what is on the ground but the hero and its young, in the
// pixel art, before it is lit: what moves along the rows, and the fog and
// mist.
func (p *Play) drawMoving(ctx *scene.Context, field *ebiten.Image) {
	w := p.world
	draw := func(pc piece) { drawPiece(ctx, field, pc) }
	for r := range w.Lanes {
		p.drawEntities(ctx, field, r)
		p.eachHeron(r, draw)
	}
	p.drawProwl(ctx, field) // the water rats and the snake (D69, D70)
	p.drawParked(ctx, field)
	p.drawYard(ctx, field)
	p.eachPick(ctx, draw)
	p.drawSweepers(ctx, field)
	p.drawSiren(ctx, field) // an emergency vehicle coming through (DESIGN §6.9)
	p.drawTrain(ctx, field)
	p.drawFog(field, 1)
	p.drawMist(field, 1)
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

// drawTiles fills row r with a one-cell tile.
func (p *Play) drawTiles(dst *ebiten.Image, tile *gfx.Sprite, r int) {
	for c := range p.world.Cols {
		tile.Draw(dst, 0, cellX(c), cellY(r), false)
	}
}

// A piece is a frame of a sprite placed on the field, its top-left at x, y,
// mirrored if flip. Its img is set in place of a name for the things that
// have no smooth art yet but are drawn with the pieces.
type piece struct {
	name  string
	frame int
	x, y  float64
	flip  bool
	img   *ebiten.Image
}

// image is the piece's frame in the pixel art.
func (pc piece) image(ctx *scene.Context) *ebiten.Image {
	if pc.img != nil {
		return pc.img
	}
	return ctx.Sheet(Sheet).Sprite(pc.name).Frame(pc.frame)
}

// drawPiece draws pc in the pixel art.
func drawPiece(ctx *scene.Context, dst *ebiten.Image, pc piece) {
	img := pc.image(ctx)
	dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: spriteGeoM(img, pc.x, pc.y, pc.flip)})
}

// drawEntities draws row r's vehicles or platforms.
func (p *Play) drawEntities(ctx *scene.Context, dst *ebiten.Image, r int) {
	p.eachEntityPiece(ctx, r, func(pc piece) { drawPiece(ctx, dst, pc) })
}

// eachEntityPiece calls draw with each piece of row r's vehicles or
// platforms, mirrored when the lane moves left. Diving platforms show where
// they are in their dive.
func (p *Play) eachEntityPiece(ctx *scene.Context, r int, draw func(pc piece)) {
	lane := &p.world.Lanes[r]
	sheet := ctx.Sheet(Sheet)
	flip := lane.Dir < 0
	y := cellY(r)
	for _, e := range lane.Entities {
		if e.Hidden() {
			continue // a wave vehicle between waves, or away for an emergency vehicle
		}
		x := float64(config.FieldX) + math.Floor(float64(e.X())/config.SubPx)
		frame := stageFrame(e.Stage(), e.FlashPhase())
		names := entitySprites(e.Kind)
		n := len(names)
		for i := range names {
			name := names[i]
			if flip {
				name = names[n-1-i]
			}
			sp := sheet.Sprite(name)
			if e.Sinks {
				fromBow := n - 1 - i // the bow leads, on the right facing right
				if flip {
					fromBow = i
				}
				if pc, ok := shipPiece(e, name, fromBow, n, x+float64(i*config.Cell), y, flip); ok {
					draw(pc)
				}
				continue
			}
			f := frame
			if lane.Kind == config.Road {
				f = wheelFrame(e) // the wheels turn as it goes (DESIGN §3.3)
			}
			draw(piece{name: name, frame: min(f, sp.Frames()-1), x: x, y: y, flip: flip})
			x += float64(sp.W)
		}
	}
}

// shipPiece is piece fromBow of a ship that sinks, at x, y, as it sinks
// (DESIGN §5.2, D33): flashing as a warning, then low in the water and
// drifting apart from the other pieces as it breaks up, then darker as each
// piece goes under, bow first. A piece that has gone is not drawn: ok is
// false.
func shipPiece(e world.Entity, name string, fromBow, n int, x, y float64, flip bool) (pc piece, ok bool) {
	switch e.Stage() {
	case world.Breaking, world.Sinking:
		// Spread the pieces about the middle of the ship.
		pos := n - 1 - fromBow // from the stern
		if flip {
			pos = fromBow
		}
		x += float64((2*pos - (n - 1)) * breakGapPx / 2)
	}
	switch e.PieceStage(fromBow) {
	case world.Warning:
		if stageFrame(world.Warning, e.FlashPhase()) == frameFlash {
			name += "_flash"
		}
	case world.Breaking:
		name += "_low"
	case world.Sinking:
		name += "_deep"
	case world.Under, world.Sunk:
		return piece{}, false
	}
	return piece{name: name, x: x, y: y, flip: flip}, true
}

// drawHero draws the hero, its line of toadlets and any stragglers, as
// eachToad gives them.
func (p *Play) drawHero(ctx *scene.Context, dst *ebiten.Image) {
	p.eachToad(func(pl *world.Player, name string, dx float64) { p.drawToad(ctx, dst, pl, name, dx) })
}

// eachToad calls draw, in the order they are drawn, for the stragglers
// waiting at the gates, the hero, and its line of toadlets (D53), each with
// its sprite and how far across its cell it sits. The line on land is
// drawn under the hero; riders on its back over it, side by side, and they
// go with a dead hero (D56).
func (p *Play) eachToad(draw func(pl *world.Player, name string, dx float64)) {
	w := p.world
	p.eachStraggler(func(pl *world.Player) { draw(pl, p.young(), 0) })
	if w.Home > 0 {
		return // it is in its nook, drawn with the bank
	}
	young := p.young()
	for i := range w.Line {
		if !p.onBack(i) {
			draw(&w.Line[i].Player, young, 0)
		}
	}
	draw(&w.Player, p.hero.Sprite, 0)
	if w.Player.Death != 0 {
		return
	}
	n := float64(len(w.Line))
	for i := range w.Line {
		if p.onBack(i) {
			draw(&w.Line[i].Player, young, (float64(i)-(n-1)/2)*riderGapPx)
		}
	}
}

// onBack reports whether toadlet i is drawn on the hero's back: riding, or
// gathered in its cell, so a gathered line shows (D53).
func (p *Play) onBack(i int) bool {
	t, h := &p.world.Line[i], &p.world.Player
	return t.Riding || !t.Hopping() && !h.Hopping() && t.Row == h.Row && t.X == h.X
}

// riderGapPx is how far apart riders sit on the hero's back.
const riderGapPx = 6

// drawToad draws the hero or a toadlet, pl, as the sprite name, in the
// middle of its cell, dx pixels across from there, as toadState gives it.
// A smaller sprite leaps lower, in step with its size.
func (p *Play) drawToad(ctx *scene.Context, dst *ebiten.Image, pl *world.Player, name string, dx float64) {
	sheet := ctx.Sheet(Sheet)
	sp := sheet.Sprite(name)
	splash := sheet.Sprite(p.hero.Splash)
	frame, lift, sf := p.toadState(pl, splash.Frames())
	if sf >= 0 {
		splash.Draw(dst, sf, splashX(pl), cellY(pl.LaneRow()), false)
		return
	}
	lift *= float64(sp.W) / config.Cell

	x := float64(config.FieldX) + float64(pl.X)/config.SubPx + float64(config.Cell-sp.W)/2 + dx
	y := float64(config.FieldY) + float64(pl.Y)/config.SubPx - lift + float64(config.Cell-sp.H)/2
	var op ebiten.DrawImageOptions
	op.GeoM = facing(pl.Facing, sp.W, sp.H)
	op.GeoM.Translate(x, y)
	dst.DrawImage(sp.Frame(frame), &op)
}

// toadState is how pl, the hero or a toadlet, is drawn: its frame, turned
// to face its way, on its leap frames while hopping, lifted by lift at the
// hero's size over the hop, then squashed for a moment on landing (DESIGN
// §3.3). A dead one lies flat where it was hit. One that went in the water
// is shown by frame sf of the splash's frames instead; sf is -1 otherwise.
func (p *Play) toadState(pl *world.Player, frames int) (frame int, lift float64, sf int) {
	switch pl.Death {
	case world.Splashed, world.Swept:
		total := config.DeathTicks
		if p.world.Bonus() {
			total = config.SlipTicks // a slip plays the same splash (D61)
		}
		done := total - p.world.Dying
		return 0, 0, min(done*frames/total, frames-1)
	}
	switch {
	case pl.Death != 0:
		return frameSquashed, 0, -1
	case pl.Sliding, pl.Pushed:
		return frameLand, 0, -1 // a skid or a gust's push, feet down: no leap (D31, D47)
	case pl.Hopping():
		return hopFrames[max(pl.HopProgress()-1, 0)], liftOf(pl), -1 // a queued hop can show at 0
	case pl.Still < config.LandTicks:
		return frameLand, 0, -1
	}
	return frameRest, 0, -1
}

// splashX is where the splash of pl going in the water is drawn from.
func splashX(pl *world.Player) float64 {
	return float64(config.FieldX) + float64(pl.X)/config.SubPx
}

// drawHitBoxes outlines what the rules see, in debug builds: the hero's
// collision box in the row it counts as being in, each vehicle's, and
// the stretch of each platform that holds the hero's middle.
func (p *Play) drawHitBoxes(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	pl := &w.Player
	c := ctx.Scheme.Highlight
	m := config.HeroMargin
	hx := float64(config.FieldX) + float64(pl.X)/config.SubPx
	debug.Box(dst, hx+float64(m), cellY(pl.LaneRow()), float64(config.Cell-2*m), config.Cell, c)
	debug.Box(dst, hx+config.Cell/2, cellY(pl.LaneRow()), 1, config.Cell, c) // its middle
	for r := range w.Lanes {
		lane := &w.Lanes[r]
		hm := config.HitMargin
		if lane.Kind == config.River {
			hm = -config.StandGrace
		}
		for _, e := range lane.Entities {
			if e.Hidden() {
				continue
			}
			if st := e.Stage(); st == world.Under || st == world.Sunk {
				continue
			}
			x := float64(config.FieldX) + float64(e.X())/config.SubPx + float64(hm)
			debug.Box(dst, x, cellY(r), float64(e.Kind.Length-2*hm), config.Cell, c)
		}
	}
	p.drawProwlBoxes(dst, c)
	if s := w.Siren; s.Phase == world.SirenRun { // the emergency vehicle (D72)
		hm := config.HitMargin
		x := float64(config.FieldX) + float64(s.X)/config.SubPx + float64(hm)
		debug.Box(dst, x, cellY(s.Row), float64(s.Responder().Length-2*hm), config.Cell, c)
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

// drawBank draws row r as the far bank: the stone quay with its nooks, and
// a toad sitting in each nook that has been filled (DESIGN §5.3).
func (p *Play) drawBank(ctx *scene.Context, dst *ebiten.Image, r int) {
	sheet := ctx.Sheet(Sheet)
	p.drawTiles(dst, sheet.Sprite("bank"), r)
	nook := sheet.Sprite("nook")
	for _, c := range config.Nooks {
		nook.Draw(dst, 0, cellX(c), cellY(r), false)
	}
	p.drawNookToads(ctx, dst, r)
	p.drawSchoolSigns(ctx, dst, r)
	p.eachNookThing(ctx, func(pc piece) { drawPiece(ctx, dst, pc) })
}

// drawNookToads draws a toad sitting in each nook of bank row r that has
// been filled.
func (p *Play) drawNookToads(ctx *scene.Context, dst *ebiten.Image, r int) {
	toad := ctx.Sheet(Sheet).Sprite(p.hero.Sprite)
	for _, x := range p.homeToads(toad.W) {
		toad.Draw(dst, frameRest, x, cellY(r), false)
	}
}

// homeToads is where the left edge of each toad home in a nook is, for a
// toad w wide.
func (p *Play) homeToads(w int) []float64 {
	var xs []float64
	for i, c := range config.Nooks {
		if p.world.Filled[i] || p.world.Bonus() && p.world.HomeNook() == i { // a bonus level's nooks never fill (D61)
			xs = append(xs, cellX(c)+float64(config.NookWidth*config.Cell-w)/2)
		}
	}
	return xs
}

// drawSchoolSigns draws a school sign by each nook of bank row r on the
// school gates' bonus level (DESIGN §6.7).
func (p *Play) drawSchoolSigns(ctx *scene.Context, dst *ebiten.Image, r int) {
	if b := p.world.Level.Bonus; b == nil || !b.Gates {
		return
	}
	sign := ctx.Sheet(Sheet).Sprite("schoolsign")
	for _, c := range config.Nooks {
		sign.Draw(dst, 0, cellX(c)-float64(sign.W+1), cellY(r)+float64(config.Cell-sign.H), false)
	}
}

// eachNookThing calls draw with what else is in the nooks: the fruit and
// the fox.
func (p *Play) eachNookThing(ctx *scene.Context, draw func(pc piece)) {
	if n := p.world.FruitNook; n >= 0 {
		fruit := ctx.Sheet(Sheet).Sprite("fruit")
		x := cellX(config.Nooks[n]) + float64(config.NookWidth*config.Cell-fruit.W)/2
		draw(piece{name: "fruit", frame: min(p.world.Level.Fruit, fruit.Frames()-1), x: x,
			y: cellY(config.GoalRow) + float64(config.Cell-fruit.H)/2})
	}
	p.eachFox(ctx, draw)
}

// How the firefly or dragonfly moves its wings and body (D58): a fast
// beat in flight, a slower flutter and a bob where it has landed.
const (
	flyBeatTicks  = 2  // ticks for each frame of the wingbeat in flight
	perchBeatTick = 5  // and landed
	bobTicks      = 20 // ticks for each step of the bob, up or down
	bobPx         = 1
)

// beat is the wingbeat's frames, round and round: spread, half, folded,
// half.
var beat = [...]int{0, 1, 2, 1}

// flyers are the firefly or dragonfly visiting, and a bonus level's
// fireflies (D58, D61).
func (p *Play) flyers() []*world.Flyer {
	out := []*world.Flyer{&p.world.Flyer}
	for i := range p.world.Swarm {
		out = append(out, &p.world.Swarm[i])
	}
	return out
}

// flyer is the sprite of firefly or dragonfly f, its frame now,
// where its top-left is drawn, and whether it is mirrored; nil if it is not
// visiting.
func (p *Play) flyer(ctx *scene.Context, f *world.Flyer) (sp *gfx.Sprite, frame int, x, y float64, flip bool) {
	if !f.Visiting {
		return nil, 0, 0, 0, false
	}
	name := p.hero.Pickup
	if f.Day {
		name = p.hero.DayPickup
	}
	sp = ctx.Sheet(Sheet).Sprite(name)
	t, step := int(p.world.Tick), flyBeatTicks
	if f.Landed {
		step = perchBeatTick
	}
	frame = min(beat[t/step%len(beat)], sp.Frames()-1)
	x = float64(config.FieldX) + math.Floor(float64(f.X)/config.SubPx) + float64(config.Cell-sp.W)/2
	y = float64(config.FieldY) + math.Floor(float64(f.Y)/config.SubPx) + float64(config.Cell-sp.H)/2
	if f.Landed && t/bobTicks%2 == 0 {
		y -= bobPx
	}
	return sp, frame, x, y, f.Facing < 0
}

// eachFlyer calls draw with the firefly or dragonfly, if one is visiting,
// and a bonus level's fireflies (D61). They are drawn above everything on
// the river: one flies over the hero too (D58). They are drawn after the
// lighting, in their own colours: a firefly makes its own light, and the
// night's glow would blur it into a blob.
func (p *Play) eachFlyer(ctx *scene.Context, draw func(pc piece)) {
	for _, fl := range p.flyers() {
		if sp, frame, x, y, flip := p.flyer(ctx, fl); sp != nil {
			name := p.hero.Pickup
			if fl.Day {
				name = p.hero.DayPickup
			}
			draw(piece{name: name, frame: frame, x: x, y: y, flip: flip})
		}
	}
}

// lampHeadY is how far down a lamp sprite its glowing head is, in pixels.
const lampHeadY = 3

// heroLift is how high the hero is off the ground mid-hop, in pixels.
func (p *Play) heroLift() float64 { return liftOf(&p.world.Player) }

// liftOf is how high pl, the hero or a toadlet, is off the ground mid-hop,
// in pixels, at the hero's size.
func liftOf(pl *world.Player) float64 {
	if !pl.Hopping() || pl.Sliding || pl.Pushed || pl.Death != 0 {
		return 0
	}
	t := float64(pl.HopProgress()) / config.HopTicks
	return math.Round(config.HopLiftPx * math.Sin(math.Pi*t))
}

// heroCentre is the middle of the hero on the screen, as drawn.
func (p *Play) heroCentre() (x, y float64) { return centreOf(&p.world.Player) }

// centreOf is the middle of pl, the hero or a toadlet, on the screen.
func centreOf(pl *world.Player) (x, y float64) {
	return float64(config.FieldX) + float64(pl.X)/config.SubPx + config.Cell/2,
		float64(config.FieldY) + float64(pl.Y)/config.SubPx + config.Cell/2 - liftOf(pl)
}

// wheelStepPx is how far a vehicle goes between its wheel frames.
const wheelStepPx = 3

// wheelFrame is the frame of a vehicle's wheels, which turn as it moves.
func wheelFrame(e world.Entity) int {
	return e.U / (wheelStepPx * config.SubPx) % 2
}

// drawParked draws a bonus level's parked cars, each in the middle of the
// cells it takes, facing the way its lane would run, its wheels still
// (D61).
func (p *Play) drawParked(ctx *scene.Context, dst *ebiten.Image) {
	b := p.world.Level.Bonus
	if b == nil {
		return
	}
	sheet := ctx.Sheet(Sheet)
	for r := range p.world.Lanes {
		flip := p.world.Lanes[r].Dir < 0
		for _, car := range b.Cars(r) {
			names := entitySprites(car.Entity)
			x := cellX(car.Col) + float64(config.ParkedCells(car.Entity)*config.Cell-car.Entity.Length)/2
			for i := range names {
				name := names[i]
				if flip {
					name = names[len(names)-1-i]
				}
				sp := sheet.Sprite(name)
				sp.Draw(dst, 0, x, cellY(r), flip)
				x += float64(sp.W)
			}
		}
	}
}
