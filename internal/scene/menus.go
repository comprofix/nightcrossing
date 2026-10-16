package scene

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/score"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// menuPad is the margin inside menu panels.
const menuPad = 8

// drawMenuScreen draws a full-screen menu: a heading in the large font and
// the menu in a panel below it, with an optional line of help under that.
func drawMenuScreen(ctx *Context, dst *ebiten.Image, heading string, m *ui.Menu, help string) {
	sc, f := ctx.Scheme, ctx.Fonts
	dst.Fill(gfx.RGBA(sc.Backdrop))
	y := f.Large.Size
	ui.TextCentred(dst, f.Large, heading, y, sc.Title, sc.Shadow)
	// The panel's border sits a line gap below the heading, so long menus
	// still fit on a 200-pixel screen.
	drawMenuPanel(ctx, dst, m, y+f.Large.Size+ui.LineGap+menuPad)
	if help != "" {
		ui.TextCentred(dst, f.Small, help, config.Project.ScreenHeight-2*f.Small.Size, sc.Dim, sc.Shadow)
	}
}

// drawMenuPanel draws m in a panel centred across the screen, its top at y.
func drawMenuPanel(ctx *Context, dst *ebiten.Image, m *ui.Menu, y int) {
	sc, f := ctx.Scheme, ctx.Fonts
	w, h := m.Size(f.Small)
	ui.Panel(dst, ui.PanelAround(w, h, y, menuPad), sc.Panel, sc.Border)
	m.DrawCentred(dst, f.Small, y, sc)
}

// MainMenu offers Play, Options, High scores and Quit.
type MainMenu struct {
	menu ui.Menu
}

// NewMainMenu returns the main menu.
func NewMainMenu() *MainMenu { return &MainMenu{} }

// Update implements Scene.
func (s *MainMenu) Update(ctx *Context) error {
	playTitleTune(ctx)
	if s.menu.Items == nil {
		s.menu.Items = []ui.Item{
			{Label: "Play", Disabled: ctx.NewPlay == nil, Confirm: func() { ctx.Switch(NewModeSelect()) }},
			{Label: "Options", Confirm: func() { ctx.Switch(NewOptions(func() Scene { return NewMainMenu() })) }},
			{Label: "High scores", Confirm: func() { ctx.Switch(NewHighScoreTable(config.Modes[0], -1)) }},
			{Label: "Quit", Confirm: ctx.Quit},
		}
		menuSounds(ctx, &s.menu)
	}
	if ctx.Input.JustPressed(input.Back) {
		ctx.Switch(NewTitle())
		return nil
	}
	s.menu.Update(ctx.Input)
	return nil
}

// Draw implements Scene.
func (s *MainMenu) Draw(ctx *Context, dst *ebiten.Image) {
	drawMenuScreen(ctx, dst, config.Title, &s.menu, "Enter: choose   Esc: back")
}

// ModeSelect chooses the mode, then shows its intro. Cleared modes are
// marked, and once any mode is cleared the first one not yet cleared is
// suggested and the cursor starts on it.
type ModeSelect struct {
	menu  ui.Menu
	start int // the mode the cursor starts on; -1 for the suggested one
}

// NewModeSelect returns the mode menu with the cursor on the suggested mode.
func NewModeSelect() *ModeSelect { return &ModeSelect{start: -1} }

// NewModeSelectAt returns the mode menu with the cursor on mode i, e.g. when
// coming back from that mode's intro.
func NewModeSelectAt(i int) *ModeSelect { return &ModeSelect{start: i} }

// SuggestedMode is the mode to play next: the first one not yet cleared,
// once any has been. It is -1 if none is cleared, or all are.
func SuggestedMode(t *score.Tables) int {
	next, any := -1, false
	for i, m := range config.Modes {
		switch {
		case t.IsCleared(m.ID):
			any = true
		case next < 0:
			next = i
		}
	}
	if !any {
		return -1
	}
	return next
}

// Update implements Scene.
func (s *ModeSelect) Update(ctx *Context) error {
	if s.menu.Items == nil {
		next := SuggestedMode(ctx.Scores)
		for i, m := range config.Modes {
			label := m.Name
			switch {
			case ctx.Scores.IsCleared(m.ID):
				label += "  CLEARED"
			case i == next:
				label += "  NEXT"
			}
			s.menu.Items = append(s.menu.Items, ui.Item{Label: label, Confirm: func() { ctx.Switch(NewIntro(i)) }})
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
	s.menu.Update(ctx.Input)
	return nil
}

// Draw implements Scene.
func (s *ModeSelect) Draw(ctx *Context, dst *ebiten.Image) {
	drawMenuScreen(ctx, dst, "MODE", &s.menu, config.Modes[s.menu.Cursor].Desc)
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
	s.menu.Update(ctx.Input)
	return nil
}

func (s *Options) items(ctx *Context) []ui.Item {
	set := ctx.Settings
	toggle := func(v *bool) func(int) {
		return func(int) { *v = !*v; ctx.SettingsChanged() }
	}
	// volume is a slider for v, dimmed and skipped while its sound is off.
	volume := func(label string, on *bool, v *float64) ui.Item {
		return ui.Item{
			Label:      label,
			Slider:     func() float64 { return *v },
			DisabledIf: func() bool { return !*on },
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
	modes := config.Project.ScaleModes
	return []ui.Item{
		{Label: "Scale", Value: func() string { return ScaleModeName(set.ScaleMode) }, Disabled: len(modes) < 2,
			Adjust: func(d int) {
				i := 0
				for j, m := range modes {
					if m == set.ScaleMode {
						i = j
					}
				}
				set.ScaleMode = modes[(i+d+len(modes))%len(modes)]
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
		shaded("Light", func() string { return LightName(set.SmoothLight) }, LightName(true), &set.SmoothLight),
		shaded("Glow", ui.OnOff(&set.Glow), "Off", &set.Glow),
		shaded("CRT", ui.OnOff(&set.CRT), "Off", &set.CRT),
		{Label: "High contrast", Value: ui.OnOff(&set.HighContrast), Adjust: toggle(&set.HighContrast)},
		{Label: "Back", Confirm: func() { ctx.Switch(s.back()) }},
	}
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
	p.menu.Update(ctx.Input)
	return nil
}

// Draw implements Scene.
func (p *Pause) Draw(ctx *Context, dst *ebiten.Image) {
	sc, f := ctx.Scheme, ctx.Fonts
	p.under.Draw(ctx, dst)
	Dim(dst, sc.Shadow, 0.6)
	y := config.Project.ScreenHeight/2 - 3*f.Large.Size/2
	ui.TextCentred(dst, f.Large, "PAUSED", y, sc.Title, sc.Shadow)
	drawMenuPanel(ctx, dst, &p.menu, y+2*f.Large.Size)
}
