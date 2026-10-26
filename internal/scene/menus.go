package scene

import (
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/score"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// menuPad is the margin inside menu panels: one line of the small font, so
// it grows with the font (8 pixels for pixel art).
func menuPad(f gfx.Fonts) int { return f.Small.LineHeight() }

// drawMenuScreen draws a full-screen menu: a heading in the large font and
// the menu in a panel below it, with an optional line of help under that.
func drawMenuScreen(ctx *Context, dst *ebiten.Image, heading string, m *ui.Menu, help string) {
	sc, f := ctx.Scheme, ctx.Fonts
	dst.Fill(gfx.RGBA(sc.Backdrop))
	ui.TextCentred(dst, f.Large, heading, f.Large.LineHeight(), sc.Title, sc.Shadow)
	drawMenuPanel(ctx, dst, m, menuTop(ctx))
	if help != "" {
		ui.TextCentred(dst, f.Small, help, ctx.Preset.ScreenHeight-2*f.Small.LineHeight(), sc.Dim, sc.Shadow)
	}
}

// menuTop is the top of a menu screen's menu: the panel's border sits a
// line gap below the heading, so long menus still fit on a 200-pixel
// screen.
func menuTop(ctx *Context) int {
	f := ctx.Fonts
	return 2*f.Large.LineHeight() + ui.LineGap + menuPad(f)
}

// updateMenu runs m for one tick, placed as drawMenuPanel draws it with its
// top at y, so the mouse works. Headless tests have no fonts and no mouse.
func updateMenu(ctx *Context, m *ui.Menu, y func(*Context) int) {
	if f := ctx.Fonts; f.Small != nil && f.Large != nil {
		m.PlaceCentred(f.Small, ctx.Preset.ScreenWidth, y(ctx))
	}
	m.Update(ctx.Input)
}

// menuRows is how many lines of a menu fit on a menu screen: from the
// panel's top under the heading down to the panel's bottom, a line gap
// above the help line. It is 0, no limit, without fonts to measure, as in
// headless tests that never draw.
func menuRows(ctx *Context) int {
	f := ctx.Fonts
	if f.Small == nil || f.Large == nil {
		return 0
	}
	top := menuTop(ctx)
	bottom := ctx.Preset.ScreenHeight - 2*f.Small.LineHeight() - ui.LineGap - menuPad(f)
	return (bottom - top + ui.LineGap) / (f.Small.LineHeight() + ui.LineGap)
}

// drawMenuPanel draws m in a panel centred across the screen, its top at y.
func drawMenuPanel(ctx *Context, dst *ebiten.Image, m *ui.Menu, y int) {
	sc, f := ctx.Scheme, ctx.Fonts
	w, h := m.Size(f.Small)
	ui.Panel(dst, ui.PanelAround(ctx.Preset.ScreenWidth, w, h, y, menuPad(f)), sc.Panel, sc.Border)
	m.DrawCentred(dst, f.Small, y, sc)
}

// MainMenu offers Play, Options, High scores and Quit.
type MainMenu struct {
	menu ui.Menu
}

// The shared scenes draw at the screen's own size in either look.
func (*Title) Native()          {}
func (*MainMenu) Native()       {}
func (*ModeSelect) Native()     {}
func (*Intro) Native()          {}
func (*Options) Native()        {}
func (*Pause) Native()          {}
func (*SoundTest) Native()      {}
func (*Tally) Native()          {}
func (*HighScoreEntry) Native() {}
func (*HighScoreTable) Native() {}

// NewMainMenu returns the main menu.
func NewMainMenu() *MainMenu { return &MainMenu{} }

// Update implements Scene.
func (s *MainMenu) Update(ctx *Context) error {
	playTitleTune(ctx)
	if s.menu.Items == nil {
		s.menu.Items = []ui.Item{
			{Label: "Play", Disabled: ctx.NewPlay == nil, Confirm: func() { ctx.Switch(NewModeSelect()) }},
			{Label: "Options", Confirm: func() { ctx.Switch(NewOptions(func() Scene { return NewMainMenu() })) }},
			{Label: "High scores", Confirm: func() { ctx.Switch(NewHighScoreTable(ctx.ModeList()[0], -1)) }},
			{Label: "Quit", Confirm: ctx.Quit},
		}
		menuSounds(ctx, &s.menu)
	}
	if ctx.Input.JustPressed(input.Back) {
		ctx.Switch(NewTitle())
		return nil
	}
	updateMenu(ctx, &s.menu, menuTop)
	return nil
}

// Draw implements Scene.
func (s *MainMenu) Draw(ctx *Context, dst *ebiten.Image) {
	drawMenuScreen(ctx, dst, config.Title, &s.menu, "Enter: choose   Esc: back")
}

// ModeSelect chooses the mode, then shows its intro. Mastered modes are
// marked; each later mode is locked, dimmed with what opens it, until the
// one before it is mastered; and the newest one opened is suggested, the
// cursor starting on it (D64). Test games (Context.Testing) can choose a
// locked mode.
type ModeSelect struct {
	menu  ui.Menu
	start int // the mode the cursor starts on; -1 for the suggested one
}

// NewModeSelect returns the mode menu with the cursor on the suggested mode.
func NewModeSelect() *ModeSelect { return &ModeSelect{start: -1} }

// NewModeSelectAt returns the mode menu with the cursor on mode i, e.g. when
// coming back from that mode's intro.
func NewModeSelectAt(i int) *ModeSelect { return &ModeSelect{start: i} }

// SuggestedMode is the one of modes to play next: the newest one opened,
// while it is not yet mastered. It is -1 before any is mastered, or once
// all are (D64).
func SuggestedMode(modes []config.Mode, t *score.Tables) int {
	newest := 0
	for i := range modes {
		if t.Open(modes, i) {
			newest = i
		}
	}
	if newest == 0 || t.IsMastered(modes[newest].ID) {
		return -1
	}
	return newest
}

// Update implements Scene.
func (s *ModeSelect) Update(ctx *Context) error {
	if s.menu.Items == nil {
		modes := ctx.ModeList()
		next := SuggestedMode(modes, ctx.Scores)
		for i, m := range modes {
			label := m.Name
			// A mode still being built, added with -mode, is open, for testing.
			open := ctx.Scores.Open(modes, i) || !slices.ContainsFunc(config.Modes, func(c config.Mode) bool { return c.ID == m.ID })
			switch {
			case ctx.Scores.IsMastered(m.ID):
				label += "  " + Tick + " MASTERED"
			case !open:
				label += "  " + Lock + " MASTER " + strings.ToUpper(modes[i-1].Name) + " TO OPEN"
			case i == next:
				label += "  NEXT"
			}
			s.menu.Items = append(s.menu.Items, ui.Item{Label: label, Disabled: !open && !ctx.Testing(),
				Confirm: func() { ctx.Switch(NewIntro(i)) }})
		}
		s.menu.Cursor = max(next, 0)
		if s.start >= 0 {
			s.menu.Cursor = s.start
		}
		menuSounds(ctx, &s.menu)
	}
	if ctx.Input.JustPressed(input.Back) {
		ctx.Switch(NewMainMenu())
		return nil
	}
	updateMenu(ctx, &s.menu, menuTop)
	return nil
}

// Draw implements Scene.
func (s *ModeSelect) Draw(ctx *Context, dst *ebiten.Image) {
	drawMenuScreen(ctx, dst, "MODE", &s.menu, ctx.ModeList()[s.menu.Cursor].Desc)
}

// Options changes the saved settings. Changes apply at
// once. Back returns to the scene back makes.
type Options struct {
	back func() Scene
	menu ui.Menu
}

// NewOptions returns the options menu; leaving it switches to back().
func NewOptions(back func() Scene) *Options { return &Options{back: back} }

// VolumeSteps is how many steps the volume settings have.
const VolumeSteps = 10

// Update implements Scene.
func (s *Options) Update(ctx *Context) error {
	if s.menu.Items == nil {
		s.menu.Items = s.items(ctx)
		menuSounds(ctx, &s.menu)
	}
	if ctx.Input.JustPressed(input.Back) {
		ctx.Switch(s.back())
		return nil
	}
	updateMenu(ctx, &s.menu, menuTop)
	return nil
}

func (s *Options) items(ctx *Context) []ui.Item {
	set := ctx.Settings
	toggle := func(v *bool) func(int) {
		return func(int) { *v = !*v; ctx.SettingsChanged() }
	}
	// volume is a slider for v, dimmed and skipped while its sound is off.
	// The mouse sets it to the nearest step.
	volume := func(label string, on *bool, v *float64) ui.Item {
		return ui.Item{
			Label:      label,
			Slider:     func() float64 { return *v },
			DisabledIf: func() bool { return !*on },
			SetSlider: func(x float64) {
				if n := float64(steps(x)) / VolumeSteps; n != *v {
					*v = n
					ctx.SettingsChanged()
				}
			},
			Adjust: func(d int) {
				*v = float64(min(max(steps(*v)+d, 0), VolumeSteps)) / VolumeSteps
				ctx.SettingsChanged()
			},
		}
	}
	// shaded is an on/off item for an effect that needs shaders. Without
	// them it is dimmed and shows what the game does instead, off.
	shaded := func(label string, value func() string, without string, v *bool) ui.Item {
		if ctx.NoShaders {
			return ui.Item{Label: label, Value: func() string { return without }, Disabled: true}
		}
		return ui.Item{Label: label, Value: value, Adjust: toggle(v)}
	}
	looks := ctx.Looks
	// pixelOnly is it, an option of the pixel look's alone: dimmed under the
	// smooth look, where it shows what that look does instead (D80).
	pixelOnly := func(it ui.Item, smooth string) ui.Item {
		if it.Disabled {
			return it
		}
		value := it.Value
		pixel := func() bool { return set.Shown(looks).Art == config.ArtPixel }
		it.Value = func() string {
			if pixel() {
				return value()
			}
			return smooth
		}
		it.DisabledIf = func() bool { return !pixel() }
		return it
	}
	// Scale is the shown look's, dimmed while it has one mode, as F9 is.
	modes := func() []config.ScaleMode { return set.Shown(looks).ScaleModes }
	var items []ui.Item
	if looks.Two() {
		items = append(items, ui.Item{Label: "Look", Value: func() string { return set.Shown(looks).Name },
			Adjust: func(int) {
				l := looks.Second.Name
				if set.Shown(looks).Name == l {
					l = looks.First.Name
				}
				set.Look = l
				ctx.SettingsChanged()
			}})
	}
	return append(items, []ui.Item{
		{Label: "Scale", Value: func() string { return ScaleModeName(set.Mode(looks)) },
			DisabledIf: func() bool { return len(modes()) < 2 },
			Adjust: func(d int) {
				ms, i := modes(), 0
				for j, m := range ms {
					if m == set.ScaleMode {
						i = j
					}
				}
				set.ScaleMode = ms[(i+d+len(ms))%len(ms)]
				ctx.SettingsChanged()
			}},
		{Label: "Fullscreen", Value: ui.OnOff(&set.Fullscreen), Adjust: toggle(&set.Fullscreen)},
		{Label: "Sound", Value: ui.OnOff(&set.SoundOn), Adjust: toggle(&set.SoundOn)},
		volume("Volume", &set.SoundOn, &set.MasterVolume),
		{Label: "Music", Value: ui.OnOff(&set.MusicOn), Adjust: toggle(&set.MusicOn)},
		volume("Music volume", &set.MusicOn, &set.MusicVolume),
		{Label: "Effects", Value: ui.OnOff(&set.SFXOn), Adjust: toggle(&set.SFXOn)},
		volume("Effects volume", &set.SFXOn, &set.SFXVolume),
		{Label: "Sound test", Confirm: func() {
			ctx.Switch(NewSoundTest(func() Scene { return NewOptions(s.back) }))
		}},
		{Label: "Reduced motion", Value: ui.OnOff(&set.ReducedMotion), Adjust: toggle(&set.ReducedMotion)},
		pixelOnly(shaded("Light", func() string { return LightName(set.SmoothLight) }, LightName(true), &set.SmoothLight), LightName(true)),
		shaded("Glow", ui.OnOff(&set.Glow), "Off", &set.Glow),
		pixelOnly(shaded("CRT", ui.OnOff(&set.CRT), "Off", &set.CRT), "Off"),
		{Label: "High contrast", Value: ui.OnOff(&set.HighContrast), Adjust: toggle(&set.HighContrast)},
		{Label: "Back", Confirm: func() { ctx.Switch(s.back()) }},
	}...)
}

// LightName is the light setting as the options menu shows it: Pixel
// light falls off in steps; Smooth fades evenly.
func LightName(smooth bool) string {
	if smooth {
		return "Smooth"
	}
	return "Pixel"
}

// steps is a 0–1 volume as a whole number of steps.
func steps(v float64) int { return int(math.Round(v * VolumeSteps)) }

// ScaleModeName is a scale mode as the options menu shows it.
func ScaleModeName(m config.ScaleMode) string {
	switch m {
	case config.ScaleCRT:
		return "CRT 4:3"
	case config.ScaleSmooth:
		return "Smooth"
	default:
		return "Whole pixels"
	}
}

// Draw implements Scene.
func (s *Options) Draw(ctx *Context, dst *ebiten.Image) {
	drawMenuScreen(ctx, dst, "OPTIONS", &s.menu, "Left/Right: change   Esc: back")
}

// Pause is drawn over the paused scene, which stands still underneath. Back
// or Pause resumes it.
type Pause struct {
	under Scene
	menu  ui.Menu
}

// NewPause pauses under.
func NewPause(under Scene) *Pause { return &Pause{under: under} }

// Update implements Scene.
func (p *Pause) Update(ctx *Context) error {
	if p.menu.Items == nil {
		p.menu.Items = []ui.Item{
			{Label: "Resume", Confirm: func() { ctx.Cut(p.under) }},
			{Label: "Options", Confirm: func() { ctx.Switch(NewOptions(func() Scene { return p })) }},
			{Label: "Quit to menu", Confirm: func() { ctx.Switch(NewMainMenu()) }},
		}
		menuSounds(ctx, &p.menu)
	}
	if ctx.Input.JustPressed(input.Back) || ctx.Input.JustPressed(input.Pause) {
		ctx.Cut(p.under)
		return nil
	}
	updateMenu(ctx, &p.menu, pauseMenuTop)
	return nil
}

// pauseTop is the top of the pause screen's heading, and pauseMenuTop the
// top of its menu, under the heading: the two sit around the middle.
func pauseTop(ctx *Context) int {
	return ctx.Preset.ScreenHeight/2 - 3*ctx.Fonts.Large.LineHeight()/2
}
func pauseMenuTop(ctx *Context) int { return pauseTop(ctx) + 2*ctx.Fonts.Large.LineHeight() }

// Draw implements Scene.
func (p *Pause) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	drawScene(ctx, p.under, dst)
	Dim(dst, sc.Shadow, 0.6)
	ui.TextCentred(dst, f.Large, "PAUSED", pauseTop(ctx), sc.Title, sc.Shadow)
	drawMenuPanel(ctx, dst, &p.menu, pauseMenuTop(ctx))
}
