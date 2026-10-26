package play

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

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
	ticks int
	all   bool // every straggler brought home
	next  func() scene.Scene
}

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
	endingTwinkle  = 40 // ticks a star takes to twinkle
)

// Update implements scene.Scene.
func (e *Ending) Update(ctx *scene.Context) error {
	if e.ticks == 0 {
		ctx.Audio.PlayTune(NightTitle)
	}
	e.ticks++
	last := e.rollTop() + float64((len(config.Credits)-1)*endingLinePx)
	done := last < endingClipY-endingLinePx // the last line has rolled away
	if done || e.ticks > endingRollFrom && ctx.Input.JustPressed(input.Confirm) {
		ctx.Audio.PlayEffect(scene.MenuConfirm)
		ctx.Switch(e.next())
	}
	return nil
}

// rollTop is where the credits' first line is: below the screen until the
// roll begins, then rising.
func (e *Ending) rollTop() float64 {
	h := float64(config.Project.ScreenHeight)
	return h - math.Max(0, float64(e.ticks-endingRollFrom))*endingRollPx
}

// Draw implements scene.Scene.
func (e *Ending) Draw(ctx *scene.Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	pr := config.Project
	dst.Fill(gfx.RGBA(resurrect64.Ink))
	for i := range endingStars { // stars, each in its own place, twinkling in its own time
		x := float64(noise(int64(i), 1, 7) % uint32(pr.ScreenWidth))
		y := float64(noise(int64(i), 2, 7) % uint32(endingBankY))
		c := resurrect64.Lavender
		if (e.ticks+int(noise(int64(i), 3, 7)%endingTwinkle))/endingTwinkle%3 == 0 {
			c = resurrect64.White
		}
		dst.Set(int(x), int(y), gfx.RGBA(c))
	}

	sheet := ctx.Sheet(Sheet)
	bank, nook := sheet.Sprite("bank"), sheet.Sprite("nook")
	for c := range config.Cols {
		bank.Draw(dst, 0, float64(config.FieldX+c*config.Cell), endingBankY, false)
	}
	home := config.Nooks[len(config.Nooks)/2]
	for _, c := range config.Nooks {
		nook.Draw(dst, 0, float64(config.FieldX+c*config.Cell), endingBankY, false)
	}
	toad, young := sheet.Sprite(Toad.Sprite), sheet.Sprite(Toad.SchoolYoung)
	nx := float64(config.FieldX + home*config.Cell)
	toad.Draw(dst, frameRest, nx+float64(config.NookWidth*config.Cell-toad.W)/2, endingBankY, false)
	young.Draw(dst, frameRest, nx-float64(young.W)/2, endingBankY+float64(config.Cell-young.H), false)
	young.Draw(dst, frameRest, nx+float64(config.NookWidth*config.Cell)-float64(young.W)/2, endingBankY+float64(config.Cell-young.H), false)

	y := endingBankY + config.Cell + 2*f.Small.LineHeight()
	heading := config.EndingHeading
	if e.all {
		heading = config.EndingHeadingAll
	}
	ui.TextCentred(dst, f.Large, heading, y, sc.Title, sc.Shadow)
	y += f.Large.LineHeight() + f.Small.LineHeight()
	ui.TextCentred(dst, f.Small, config.EndingThanks, y, sc.Text, sc.Shadow)

	ry := e.rollTop()
	for i, line := range config.Credits {
		ly := int(ry) + i*endingLinePx
		if line == "" || ly < endingClipY || ly > pr.ScreenHeight {
			continue
		}
		c := sc.Text
		if i == 0 || i == len(config.Credits)-1 {
			c = sc.Highlight
		}
		ui.TextCentred(dst, f.Small, line, ly, c, sc.Shadow)
	}
}
