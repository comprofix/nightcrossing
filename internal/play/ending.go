package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/gfx/palettes/resurrect64"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// Ending is the game's ending and credits, after the school gates' tally
// (D62, DESIGN §6.7): the toad and its toadlets home in their nook under
// the night sky, a heading and a line of thanks, then the credits rolling
// slowly up, to the title tune. Confirm goes on once the roll has begun,
// and the roll's end goes on by itself, to the MASTERED screen.
type Ending struct {
	ticks  int
	all    bool // every straggler brought home
	next   func() scene.Scene
	stamps stamps // the smooth look's art, drawn once
}

// Native implements scene.Native: the ending is drawn in either look, in
// the game's coordinates shown k times the size (DESIGN §3.6).
func (*Ending) Native() {}

// newEnding is the ending, then next; all is whether every straggler
// was brought home.
func newEnding(all bool, next func() scene.Scene) *Ending { return &Ending{all: all, next: next} }

// The ending's timing and layout, in ticks and pixels.
const (
	endingRollFrom = 4 * config.TPS // the credits start rolling
	endingRollPx   = 0.4            // how far they roll a tick
	endingBankY    = 72             // the bank's top
	endingClipY    = 200            // the credits show below this, under the heading and the thanks
	endingLinePx   = 16             // a line of the credits
	endingStars    = 60
	endingTwinkle  = 40  // ticks a star takes to twinkle
	endingStarR    = 0.6 // a star's radius in the smooth look
)

// Update implements scene.Scene.
func (e *Ending) Update(ctx *scene.Context) error {
	if e.ticks == 0 {
		ctx.Audio.PlayTune(NightTitle)
	}
	e.ticks++
	last := e.rollTop(ctx) + float64((len(config.Credits)-1)*endingLinePx)
	done := last < endingClipY-endingLinePx // the last line has rolled away
	if done || e.ticks > endingRollFrom && (ctx.Input.JustPressed(input.Confirm) || ctx.Input.Clicked()) {
		ctx.Audio.PlayEffect(scene.MenuConfirm)
		ctx.Switch(e.next())
	}
	return nil
}

// rollTop is where the credits' first line is, in the game's coordinates:
// below the screen until the roll begins, then rising.
func (e *Ending) rollTop(ctx *scene.Context) float64 {
	h := float64(gamePreset(ctx).ScreenHeight)
	return h - math.Max(0, float64(e.ticks-endingRollFrom))*endingRollPx
}

// Draw implements scene.Scene.
func (e *Ending) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	pr := gamePreset(ctx)
	k := max(ctx.Preset.ScreenWidth/pr.ScreenWidth, 1)
	kf := float64(k)
	dst.Fill(gfx.RGBA(resurrect64.Ink))
	for i := range endingStars { // stars, each in its own place, twinkling in its own time
		x := float64(noise(int64(i), 1, 7) % uint32(pr.ScreenWidth))
		y := float64(noise(int64(i), 2, 7) % uint32(endingBankY))
		c := resurrect64.Lavender
		if (e.ticks+int(noise(int64(i), 3, 7)%endingTwinkle))/endingTwinkle%3 == 0 {
			c = resurrect64.White
		}
		if k == 1 {
			dst.Set(int(x), int(y), gfx.RGBA(c))
		} else {
			vector.FillCircle(dst, float32(kf*(x+0.5)), float32(kf*(y+0.5)), float32(kf*endingStarR), gfx.RGBA(c), true)
		}
	}

	sheet := ctx.Sheet(Sheet)
	home := config.Nooks[len(config.Nooks)/2]
	toad, young := sheet.Sprite(Toad.Sprite), sheet.Sprite(Toad.SchoolYoung)
	nx := float64(config.FieldX + home*config.Cell)
	toadX := nx + float64(config.NookWidth*config.Cell-toad.W)/2
	youngY := endingBankY + float64(config.Cell-young.H)
	youngXs := []float64{nx - float64(young.W)/2, nx + float64(config.NookWidth*config.Cell) - float64(young.W)/2}
	if k == 1 {
		bank, nook := sheet.Sprite("bank"), sheet.Sprite("nook")
		for c := range config.Cols {
			bank.Draw(dst, 0, float64(config.FieldX+c*config.Cell), endingBankY, false)
		}
		for _, c := range config.Nooks {
			nook.Draw(dst, 0, float64(config.FieldX+c*config.Cell), endingBankY, false)
		}
		toad.Draw(dst, frameRest, toadX, endingBankY, false)
		for _, x := range youngXs {
			young.Draw(dst, frameRest, x, youngY, false)
		}
	} else {
		w := float64(config.FieldX + config.Cols*config.Cell)
		lay(dst, e.stamps.get("bank", k, w, config.Cell, func(pn pen) {
			smoothBankRow(pn, 0, config.Cols)
		}), k, 0, endingBankY, 1)
		art := func(sp *gfx.Sprite, name string) *ebiten.Image {
			return e.stamps.get(name, k, float64(sp.W), float64(sp.H), func(pn pen) { smoothArts[name](pn, frameRest) })
		}
		lay(dst, art(toad, Toad.Sprite), k, toadX, endingBankY, 1)
		for _, x := range youngXs {
			lay(dst, art(young, Toad.SchoolYoung), k, x, youngY, 1)
		}
	}

	y := k*(endingBankY+config.Cell) + 2*f.Small.LineHeight()
	heading := config.EndingHeading
	if e.all {
		heading = config.EndingHeadingAll
	}
	ui.TextCentred(dst, f.Large, heading, y, sc.Title, sc.Shadow)
	y += f.Large.LineHeight() + f.Small.LineHeight()
	ui.TextCentred(dst, f.Small, config.EndingThanks, y, sc.Text, sc.Shadow)

	ry := e.rollTop(ctx)
	for i, line := range config.Credits {
		ly := k * (int(ry) + i*endingLinePx)
		if line == "" || ly < k*endingClipY || ly > ctx.Preset.ScreenHeight {
			continue
		}
		c := sc.Text
		if i == 0 || i == len(config.Credits)-1 {
			c = sc.Highlight
		}
		ui.TextCentred(dst, f.Small, line, ly, c, sc.Shadow)
	}
}
