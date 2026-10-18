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
// vehicles and platforms slide in and out at its edges.
func (p *Play) drawField(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	bounds := image.Rect(config.FieldX, config.FieldY,
		config.FieldX+w.Cols*config.Cell, config.FieldY+w.Rows*config.Cell)
	look := p.look() // Stroll's night, or Last Train's day and night (D39)
	if ctx.Settings.SmoothLight {
		look.Bands = 0 // the Light option (Options: Light: Smooth)
	}
	if !ctx.Settings.Glow {
		look.Glow = 0
	}
	look = p.lightning(look, ctx.Settings.ReducedMotion) // the storm sky (D49)
	if ctx.Settings.HighContrast {
		look.Level = max(look.Level, highContrastLevel)
	}
	field := p.light.Begin(look).SubImage(bounds).(*ebiten.Image)

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
	p.drawRainbow(field) // after the storm, over the water (DESIGN §6.7)
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
	for r := range w.Lanes {
		p.drawEntities(ctx, field, r)
		p.drawHerons(ctx, field, r)
	}
	p.drawParked(ctx, field)
	p.drawYard(ctx, field)
	p.drawPicks(ctx, field)
	p.drawSweepers(ctx, field)
	p.drawTrain(ctx, field)
	p.drawFog(field)
	p.drawMist(field)
	p.drawHero(ctx, field)

	p.addLights(ctx, p.light)
	p.light.Draw(dst, 0, 0)
	out := dst.SubImage(bounds).(*ebiten.Image)
	p.drawFlyer(ctx, out)
	if ctx.Settings.HighContrast {
		p.drawOutlines(ctx, out)
	}
	p.drawRain(out)
	p.drawWind(out)
	p.parts.Draw(out, 0, 0)
	if debug.Enabled {
		p.drawHitBoxes(ctx, out)
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

// drawTiles fills row r with a one-cell tile.
func (p *Play) drawTiles(dst *ebiten.Image, tile *gfx.Sprite, r int) {
	for c := range p.world.Cols {
		tile.Draw(dst, 0, cellX(c), cellY(r), false)
	}
}

// drawEntities draws row r's vehicles or platforms, mirrored when the lane
// moves left. Diving platforms show where they are in their dive.
func (p *Play) drawEntities(ctx *scene.Context, dst *ebiten.Image, r int) {
	lane := &p.world.Lanes[r]
	sheet := ctx.Sheet(Sheet)
	flip := lane.Dir < 0
	y := cellY(r)
	for _, e := range lane.Entities {
		if e.Out {
			continue // a wave vehicle between waves
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
				p.drawShipPiece(ctx, dst, e, name, fromBow, n, x+float64(i*config.Cell), y, flip)
				continue
			}
			f := frame
			if lane.Kind == config.Road {
				f = wheelFrame(e) // the wheels turn as it goes (DESIGN §3.3)
			}
			sp.Draw(dst, min(f, sp.Frames()-1), x, y, flip)
			x += float64(sp.W)
		}
	}
}

// drawShipPiece draws piece fromBow of a ship that sinks, at x, y, as it
// sinks (DESIGN §5.2, D33): flashing as a warning, then low in the water
// and drifting apart from the other pieces as it breaks up, then darker as
// each piece goes under, bow first. A piece that has gone is not drawn.
func (p *Play) drawShipPiece(ctx *scene.Context, dst *ebiten.Image, e world.Entity, name string, fromBow, n int, x, y float64, flip bool) {
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
		return
	}
	ctx.Sheet(Sheet).Sprite(name).Draw(dst, 0, x, y, flip)
}

// drawHero draws the hero, and its line of toadlets (D53), each turned to
// face its way, on its leap frame while hopping, rising and falling over
// the hop, then squashed for a moment on landing (DESIGN §3.3). A dead one
// lies flat where it was hit, or splashes where it went in the water. The
// line on land is drawn under the hero; riders on its back over it, side
// by side, and they go with a dead hero (D56).
func (p *Play) drawHero(ctx *scene.Context, dst *ebiten.Image) {
	w := p.world
	p.drawStragglers(ctx, dst)
	if w.Home > 0 {
		return // it is in its nook, drawn with the bank
	}
	young := p.young()
	for i := range w.Line {
		if !p.onBack(i) {
			p.drawToad(ctx, dst, &w.Line[i].Player, young, 0)
		}
	}
	p.drawToad(ctx, dst, &w.Player, p.hero.Sprite, 0)
	if w.Player.Death != 0 {
		return
	}
	n := float64(len(w.Line))
	for i := range w.Line {
		if p.onBack(i) {
			p.drawToad(ctx, dst, &w.Line[i].Player, young, (float64(i)-(n-1)/2)*riderGapPx)
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
// middle of its cell, dx pixels across from there. A smaller sprite leaps
// lower, in step with its size.
func (p *Play) drawToad(ctx *scene.Context, dst *ebiten.Image, pl *world.Player, name string, dx float64) {
	sheet := ctx.Sheet(Sheet)
	sp := sheet.Sprite(name)
	scale := float64(sp.W) / config.Cell

	frame, lift := frameRest, 0.0
	switch pl.Death {
	case world.Splashed, world.Swept:
		splash := sheet.Sprite(p.hero.Splash)
		total := config.DeathTicks
		if p.world.Bonus() {
			total = config.SlipTicks // a slip plays the same splash (D61)
		}
		done := total - p.world.Dying
		f := min(done*splash.Frames()/total, splash.Frames()-1)
		splash.Draw(dst, f, float64(config.FieldX)+float64(pl.X)/config.SubPx, cellY(pl.LaneRow()), false)
		return
	}
	switch {
	case pl.Death != 0:
		frame = frameSquashed
	case pl.Sliding, pl.Pushed:
		frame = frameLand // a skid or a gust's push, feet down: no leap (D31, D47)
	case pl.Hopping():
		frame = hopFrames[max(pl.HopProgress()-1, 0)] // a queued hop can show at 0
		lift = liftOf(pl) * scale
	case pl.Still < config.LandTicks:
		frame = frameLand
	}

	x := float64(config.FieldX) + float64(pl.X)/config.SubPx + float64(config.Cell-sp.W)/2 + dx
	y := float64(config.FieldY) + float64(pl.Y)/config.SubPx - lift + float64(config.Cell-sp.H)/2
	var op ebiten.DrawImageOptions
	op.GeoM = facing(pl.Facing, sp.W, sp.H)
	op.GeoM.Translate(x, y)
	dst.DrawImage(sp.Frame(frame), &op)
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
			if e.Out {
				continue
			}
			if st := e.Stage(); st == world.Under || st == world.Sunk {
				continue
			}
			x := float64(config.FieldX) + float64(e.X())/config.SubPx + float64(hm)
			debug.Box(dst, x, cellY(r), float64(e.Kind.Length-2*hm), config.Cell, c)
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

// drawBank draws row r as the far bank: the stone quay with its nooks, and
// a toad sitting in each nook that has been filled (DESIGN §5.3).
func (p *Play) drawBank(ctx *scene.Context, dst *ebiten.Image, r int) {
	sheet := ctx.Sheet(Sheet)
	p.drawTiles(dst, sheet.Sprite("bank"), r)
	nook, toad := sheet.Sprite("nook"), sheet.Sprite(p.hero.Sprite)
	sign := sheet.Sprite("schoolsign")
	for i, c := range config.Nooks {
		nook.Draw(dst, 0, cellX(c), cellY(r), false)
		if b := p.world.Level.Bonus; b != nil && b.Gates { // a school sign by each nook (DESIGN §6.7)
			sign.Draw(dst, 0, cellX(c)-float64(sign.W+1), cellY(r)+float64(config.Cell-sign.H), false)
		}
		if p.world.Filled[i] || p.world.Bonus() && p.world.HomeNook() == i { // a bonus level's nooks never fill (D61)
			x := cellX(c) + float64(config.NookWidth*config.Cell-toad.W)/2
			toad.Draw(dst, frameRest, x, cellY(r), false)
		}
	}
	if n := p.world.FruitNook; n >= 0 {
		fruit := sheet.Sprite("fruit")
		x := cellX(config.Nooks[n]) + float64(config.NookWidth*config.Cell-fruit.W)/2
		fruit.Draw(dst, min(p.world.Level.Fruit, fruit.Frames()-1), x, cellY(r)+float64(config.Cell-fruit.H)/2, false)
	}
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

// drawFlyer draws the firefly or dragonfly, if one is visiting, and a
// bonus level's fireflies (D61), above
// everything on the river: it flies over the hero too (D58). It is drawn
// after the lighting, in its own colours: a firefly makes its own light,
// and the night's glow would blur it into a blob.
func (p *Play) drawFlyer(ctx *scene.Context, dst *ebiten.Image) {
	for _, fl := range p.flyers() {
		if sp, frame, x, y, flip := p.flyer(ctx, fl); sp != nil {
			sp.Draw(dst, frame, x, y, flip)
		}
	}
}

// drawHerons draws the herons standing on row r's logs, facing the way
// the logs move (DESIGN §5.2, D34).
func (p *Play) drawHerons(ctx *scene.Context, dst *ebiten.Image, r int) {
	lane := &p.world.Lanes[r]
	heron := ctx.Sheet(Sheet).Sprite("heron")
	for _, e := range lane.Entities {
		if !e.Heron {
			continue
		}
		x := float64(config.FieldX) + math.Floor(float64(e.X())/config.SubPx) + float64(e.HeronCell*config.Cell)
		heron.Draw(dst, 0, x, cellY(r), lane.Dir < 0)
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
