package scene

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// BlinkTicks is the blink period of "press any key" prompts.
const BlinkTicks = config.TPS

// Title is the title screen: the game's name and a blinking prompt, over
// the game playing itself if it has a demo (Context.NewDemo). Any key or
// button goes to the main menu.
type Title struct {
	shown int64 // ticks since the scene started, for the blink
	demo  Scene // nil until the first update, or if the game has none
}

// TitleDim is how far the demo is dimmed under the title, 0–1.
const TitleDim = 0.45

// Demo is the demo running behind the title, or nil.
func (t *Title) Demo() Scene { return t.demo }

// NewTitle returns the title scene.
func NewTitle() *Title { return &Title{} }

// Update implements Scene.
func (t *Title) Update(ctx *Context) error {
	t.shown++
	playTitleTune(ctx)
	if t.demo == nil && ctx.NewDemo != nil {
		t.demo = ctx.NewDemo(ctx)
	}
	if t.demo != nil {
		if err := t.demo.Update(ctx); err != nil {
			return err
		}
	}
	if ctx.Input.AnyJustPressed() {
		ctx.Audio.PlayEffect(MenuConfirm)
		ctx.Switch(NewMainMenu())
	}
	return nil
}

// Draw implements Scene.
func (t *Title) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	w, h := ctx.Preset.ScreenWidth, ctx.Preset.ScreenHeight
	if t.demo != nil {
		t.demo.Draw(ctx, dst)
		Dim(dst, sc.Shadow, TitleDim)
	} else {
		dst.Fill(gfx.RGBA(sc.Backdrop))
	}

	// The name and the prompt sit on a panel, so they read over the demo.
	pad := 2 * f.Small.LineHeight()
	body := f.Large.LineHeight() + pad + f.Small.LineHeight()
	y := (h - body) / 2
	if t.demo != nil {
		ui.Panel(dst, ui.PanelAround(w, f.Large.Width(config.Title)+2*pad, body, y, pad), sc.Panel, sc.Border)
	}
	ui.TextCentred(dst, f.Large, config.Title, y, sc.Title, sc.Shadow)
	if ui.Blink(t.shown, BlinkTicks) {
		ui.TextCentred(dst, f.Small, "PRESS ANY KEY", y+f.Large.LineHeight()+pad, sc.Text, sc.Shadow)
	}
	ui.TextCentred(dst, f.Small, "F11 fullscreen   F10 quit", h-2*f.Small.LineHeight(), sc.Dim, sc.Shadow)
}
