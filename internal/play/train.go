package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/fx"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/world"
)

// trainCars is how many carriages the train pulls, and carMids how many
// middle pieces each has (DESIGN §6.2, D37): with their ends and the
// engine's three pieces, config.Train.Length.
const (
	trainCars = 4
	carMids   = 1
)

// trainPieces are the train's sprites left to right as it faces right:
// the carriages, then the engine.
var trainPieces = func() []string {
	var ps []string
	for range trainCars {
		ps = append(ps, "car_back")
		for range carMids {
			ps = append(ps, "car_mid")
		}
		ps = append(ps, "car_front")
	}
	return append(ps, "engine_back", "engine_mid", "engine_nose")
}()

// The crossing signals stand on the verge above the track, one at each
// end; the one on the side a train comes from flashes its lamps in turn
// (DESIGN §6.2, D37).
const (
	signalFlashTicks = 15 // each lamp is lit this long
	signalOff        = 0  // the signal's frames
	signalLeftLamp   = 1
	signalRightLamp  = 2
)

// Where a signal's lamps are in its sprite, and their light.
const (
	signalLampY      = 4
	signalLeftLampX  = 5
	signalRightLampX = 18
	signalLampR      = 44
	signalLampLight  = 1.0
	signalWashR      = 48 // and a red wash over the track beside it
	signalWash       = 0.6
)

// The train's lights: a long headlight beam, a glow from each carriage's
// windows on both sides, and a red tail lamp.
const (
	trainBeamR      = 120
	trainBeam       = 0.9
	trainWindowR    = 28
	trainWindow     = 0.35
	trainTailR      = 12
	trainTailLight  = 0.8
	trainWindowEdge = 4 // how far in from each side of the row the windows are, in pixels
)

// Train sound timing, in ticks: the bell rings again this often during a
// warning, and the rumble repeats this often while a train passes.
const (
	bellTicks   = 20
	rumbleTicks = 12
)

// trainShakeTicks and trainShakePx shake the screen as a train comes
// (DESIGN §3.3); being struck shakes it harder.
const (
	trainShakeTicks = 30
	trainShakePx    = 2
	struckShakePx   = 6
)

// signalCols are the columns of the two signals: left, then right.
var signalCols = [2]int{0, config.Cols - 1}

// signalRow is the row the crossing signals guard: the track with trains,
// or the main line of a bonus level's yard, with its last train (DESIGN
// §6.7); ok is false with neither.
func (p *Play) signalRow() (row int, ok bool) {
	if rl := &p.world.Rail; rl.On {
		return rl.Row, true
	}
	if k := p.world.LastTrain(); k != nil {
		return k.Row, true
	}
	return 0, false
}

// signalFrame is the frame of the signal on side (0 left, 1 right) now.
// Both flash as the last train is about to leave, and as it goes.
func (p *Play) signalFrame(side int) int {
	rl := &p.world.Rail
	leaving := p.world.Departing() || p.departing()
	if !leaving && (rl.Phase == world.RailQuiet || (side == 0) != (rl.Dir > 0)) {
		return signalOff
	}
	if (p.world.Tick/signalFlashTicks)%2 == 0 {
		return signalLeftLamp
	}
	return signalRightLamp
}

// drawSignals draws the two crossing signals.
func (p *Play) drawSignals(ctx *scene.Context, dst *ebiten.Image) {
	row, ok := p.signalRow()
	if !ok {
		return
	}
	sp := ctx.Sheet(Sheet).Sprite("signal")
	for side, c := range signalCols {
		sp.Draw(dst, p.signalFrame(side), cellX(c), cellY(row-1), false)
	}
}

// trainX is the train's left end on the screen, in pixels.
func (p *Play) trainX() float64 {
	return float64(config.FieldX) + math.Floor(float64(p.world.Rail.X)/config.SubPx)
}

// eachTrainPiece calls draw with each of the passing train's pieces, left
// to right on the screen, mirrored when it runs left.
func (p *Play) eachTrainPiece(ctx *scene.Context, draw func(img *ebiten.Image, x, y float64, flip bool)) {
	rl := &p.world.Rail
	if rl.Phase != world.RailPassing {
		return
	}
	sheet := ctx.Sheet(Sheet)
	flip := rl.Dir < 0
	x, y := p.trainX(), cellY(rl.Row)
	n := len(trainPieces)
	for i := range trainPieces {
		name := trainPieces[i]
		if flip {
			name = trainPieces[n-1-i]
		}
		draw(sheet.Sprite(name).Frame(0), x+float64(i*config.Cell), y, flip)
	}
}

// drawTrain draws the passing train, if there is one.
func (p *Play) drawTrain(ctx *scene.Context, dst *ebiten.Image) {
	p.eachTrainPiece(ctx, func(img *ebiten.Image, x, y float64, flip bool) {
		dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: spriteGeoM(img, x, y, flip)})
	})
}

// addRailLights adds the signals' flashing lamps, and the train's
// headlight, windows and tail lamp.
func (p *Play) addRailLights(l *fx.Lighting) {
	rl := &p.world.Rail
	row, ok := p.signalRow()
	if !ok {
		return
	}
	red := gfx.RGBA(resurrect64.Red)
	for side, c := range signalCols {
		x := cellX(c)
		switch p.signalFrame(side) {
		case signalLeftLamp:
			x += signalLeftLampX
		case signalRightLamp:
			x += signalRightLampX
		default:
			continue
		}
		l.Add(fx.Light{X: x, Y: cellY(row-1) + signalLampY, Radius: signalLampR, Colour: red, Strength: signalLampLight})
		l.Add(fx.Light{X: cellX(c) + float64(config.Cell)/2, Y: cellY(row) + float64(config.Cell)/2,
			Radius: signalWashR, Colour: red, Strength: signalWash})
	}
	if !rl.On || rl.Phase != world.RailPassing {
		return
	}
	cell := float64(config.Cell)
	left, length := p.trainX(), float64(config.Train.Length)
	y := cellY(rl.Row)
	front, back := left+length, left
	if rl.Dir < 0 {
		front, back = left, left+length
	}
	l.Add(fx.Light{Shape: fx.Beam, X: front, Y: y + cell/2, Radius: trainBeamR, Flip: rl.Dir < 0,
		Colour: gfx.RGBA(resurrect64.Lemon), Strength: trainBeam * p.lampK()})
	l.Add(fx.Light{X: front, Y: y + cell/2, Radius: lensR, Colour: gfx.RGBA(resurrect64.Lemon), Strength: p.lampK()})
	l.Add(fx.Light{X: back, Y: y + cell/2, Radius: trainTailR, Colour: red, Strength: trainTailLight * p.lampK()})
	carLen := float64((carMids + 2) * config.Cell)
	for i := range trainCars {
		mid := left + carLen*(float64(i)+0.5)
		if rl.Dir < 0 {
			mid = left + length - carLen*(float64(i)+0.5)
		}
		for _, wy := range []float64{y + trainWindowEdge, y + cell - trainWindowEdge} {
			l.Add(fx.Light{X: mid, Y: wy, Radius: trainWindowR, Colour: gfx.RGBA(resurrect64.Honey), Strength: trainWindow * p.lampK()})
		}
	}
}

// railEffect is the track's sound this tick, or nil: the bell rings
// through a warning, and as the last train is about to leave (DESIGN
// §6.7), and the rumble keeps going while a train passes. The horn and
// the crash come with their events (soundsFor).
func (p *Play) railEffect() *audio.Effect {
	rl := &p.world.Rail
	switch {
	case p.world.Departing() && p.world.TicksLeft%bellTicks == 0:
		return Bell
	case rl.Phase == world.RailWarning && (config.TrainWarnTicks-rl.Left)%bellTicks == 0:
		return Bell
	case rl.Phase == world.RailPassing && p.world.Tick%rumbleTicks == 0:
		return Rumble
	}
	return nil
}
