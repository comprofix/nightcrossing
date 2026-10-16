package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// BlinkTicks is the blink period of "press any key" prompts.
const BlinkTicks = config.TPS

// Title is the title screen: the game's name and a blinking prompt. Any key
// or button goes to the main menu.
type Title struct {
	shown int64 // ticks since the scene started, for the blink
}

// NewTitle returns the title scene.
func NewTitle() *Title { return &Title{} }

// Update implements Scene.
func (t *Title) Update(ctx *Context) error {
	t.shown++
	playTitleTune(ctx)
	if ctx.Input.AnyJustPressed() {
		ctx.Audio.PlayEffect(MenuConfirm)
		ctx.Switch(NewMainMenu())
	}
	return nil
}

// Draw implements Scene.
func (t *Title) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	h := config.Project.ScreenHeight
	dst.Fill(gfx.RGBA(sc.Backdrop))

	y := h/3 - f.Large.Size/2
	ui.TextCentred(dst, f.Large, config.Title, y, sc.Title, sc.Shadow)
	if ui.Blink(t.shown, BlinkTicks) {
		ui.TextCentred(dst, f.Small, "PRESS ANY KEY", h*2/3, sc.Text, sc.Shadow)
	}
	ui.TextCentred(dst, f.Small, "F11 fullscreen   F10 quit", h-2*f.Small.Size, sc.Dim, sc.Shadow)
}
