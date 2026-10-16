// Package game implements ebiten.Game: it runs the scenes, handles the
// window keys, applies and saves the settings, and presents the internal
// screen scaled to the window.
package game

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/comprofix/nightcrossing/assets"
	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/debug"
	"github.com/comprofix/nightcrossing/internal/gfx"
	"github.com/comprofix/nightcrossing/internal/input"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/score"
	"github.com/comprofix/nightcrossing/internal/ui"
)

// Saved is what was loaded from the saved files, and where they live.
// An empty Dir saves nothing.
type Saved struct {
	Settings config.Settings
	Scores   *score.Tables // nil for empty tables
	Dir      string
}

// Content is the game's own part, which the shared scenes don't know
// about: how to start a game, its sounds, and the sprite sheets to load
// from assets/gen.
type Content struct {
	NewPlay func(ctx *scene.Context, skill config.Skill) scene.Scene
	Sounds  audio.Bank
	Sheets  []string
	// The game's title and high score tunes, if it has its own (see
	// scene.Context).
	TitleTune, ScoresTune *audio.Tune
}

// content is set once, at start-up, by the file that wires in the game
// (demo.go in the skeleton). With none, Play is disabled in the menu.
var content Content

// Game is the top-level ebiten.Game.
type Game struct {
	preset   config.Preset
	screen   *ebiten.Image // the internal screen
	prescale *ebiten.Image // whole-number upscale used for CRT presentation
	outW     int
	outH     int

	ctx    *scene.Context
	scenes *scene.Manager

	// In debug builds, Tab shows the display test pattern instead of the
	// scenes.
	showPattern bool
	pixels      *image.RGBA // CPU-side screen for the test pattern

	dir      string
	settings config.Settings
}

// New loads the assets and starts at the title screen with the given
// random seed and saved files, playing sound through synth (nil for none).
func New(seed uint64, saved Saved, synth *audio.Synth) (*Game, error) {
	p := config.Project
	fonts, err := gfx.LoadFonts(assets.FS, "font/font8x8.txt")
	if err != nil {
		return nil, fmt.Errorf("fonts: %w", err)
	}
	sheets := map[string]*gfx.Sheet{}
	for _, name := range content.Sheets {
		sh, err := gfx.LoadSheet(assets.FS, "gen/"+name)
		if err != nil {
			return nil, fmt.Errorf("sprite sheet %s: %w", name, err)
		}
		sheets[name] = sh
	}
	g := &Game{
		preset:   p,
		screen:   ebiten.NewImage(p.ScreenWidth, p.ScreenHeight),
		pixels:   image.NewRGBA(image.Rect(0, 0, p.ScreenWidth, p.ScreenHeight)),
		dir:      saved.Dir,
		settings: saved.Settings,
	}
	g.ctx = &scene.Context{
		Input:      input.New(&input.Ebiten{}),
		RNG:        scene.NewRNG(seed),
		Fonts:      fonts,
		Scheme:     gfx.SchemeFor(p.Palette),
		Sheets:     sheets,
		NewPlay:    content.NewPlay,
		Audio:      synth,
		Sounds:     audio.Join(scene.Sounds, content.Sounds),
		TitleTune:  content.TitleTune,
		ScoresTune: content.ScoresTune,
		Scores:     saved.Scores,
	}
	g.ctx.Settings = &g.settings
	if g.ctx.Scores == nil {
		g.ctx.Scores = score.New()
	}
	g.applySound()
	g.scenes = scene.NewManager(scene.NewTitle())
	return g, nil
}

// Update handles the window keys and runs the scenes for one tick.
func (g *Game) Update() error {
	in := g.ctx.Input
	in.Update()
	g.ctx.Audio.Now(g.ctx.Tick)
	s := &g.settings

	if in.JustPressed(input.Quit) || ebiten.IsWindowBeingClosed() {
		g.saveSettings() // keeps the window size
		return ebiten.Termination
	}
	if in.JustPressed(input.Fullscreen) {
		s.Fullscreen = !s.Fullscreen
		g.ctx.SettingsChanged()
	}
	if in.JustPressed(input.ScaleMode) {
		s.ScaleMode = g.preset.NextMode(s.ScaleMode)
		g.ctx.SettingsChanged()
	}
	if in.JustPressed(input.Sound) {
		s.SoundOn = !s.SoundOn
		g.ctx.SettingsChanged()
	}
	if debug.Enabled && inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.showPattern = !g.showPattern
	}

	if err := g.scenes.Update(g.ctx); err != nil {
		return err
	}
	g.ctx.Tick++
	if g.ctx.Quitting() {
		g.saveSettings()
		return ebiten.Termination
	}
	if g.ctx.TakeSettingsChanged() {
		g.applySettings()
		g.saveSettings()
	}
	if g.ctx.TakeScoresChanged() {
		g.saveScores()
	}
	return nil
}

// applySettings makes the window and sound match the settings.
func (g *Game) applySettings() {
	g.applySound()
	if ebiten.IsFullscreen() != g.settings.Fullscreen {
		ebiten.SetFullscreen(g.settings.Fullscreen)
	}
}

// applySound sets the synth from the sound settings.
func (g *Game) applySound() {
	s := &g.settings
	g.ctx.Audio.SetEnabled(s.SoundOn)
	g.ctx.Audio.SetVolumes(s.AudioLevels()) // a silenced tune keeps time
}

// saveSettings writes the current settings. A failure is
// logged, never fatal.
func (g *Game) saveSettings() {
	s := &g.settings
	if !ebiten.IsFullscreen() {
		if w, h := ebiten.WindowSize(); w > 0 && h > 0 {
			s.WindowW, s.WindowH = w, h
		}
	}
	if g.dir == "" {
		return
	}
	if err := config.SaveSettings(filepath.Join(g.dir, config.SettingsFile), *s); err != nil {
		log.Printf("saving settings: %v", err)
	}
}

// saveScores writes the high score tables. A failure is logged,
// never fatal.
func (g *Game) saveScores() {
	if g.dir == "" {
		return
	}
	if err := g.ctx.Scores.Save(filepath.Join(g.dir, config.ScoresFile)); err != nil {
		log.Printf("saving scores: %v", err)
	}
}

// Draw renders the internal screen and presents it.
func (g *Game) Draw(out *ebiten.Image) {
	if g.showPattern {
		drawTestPattern(g.pixels, g.preset, g.ctx.Tick)
		g.screen.WritePixels(g.pixels.Pix)
	} else {
		g.scenes.Draw(g.ctx, g.screen)
	}
	if debug.Enabled {
		g.drawOverlay()
	}
	g.present(out)
}

// drawOverlay shows the debug readout in the bottom-left corner: ticks and
// frames per second, the tick count and the current scene.
func (g *Game) drawOverlay() {
	sc, f := g.ctx.Scheme, g.ctx.Fonts.Small
	name := strings.TrimPrefix(fmt.Sprintf("%T", g.scenes.Current()), "*")
	if g.showPattern {
		name = "test pattern"
	}
	lines := []string{
		fmt.Sprintf("%.0f TPS %.0f FPS", ebiten.ActualTPS(), ebiten.ActualFPS()),
		fmt.Sprintf("T%d %s", g.ctx.Tick, name),
	}
	w := 0
	for _, l := range lines {
		w = max(w, f.Width(l))
	}
	lh := f.Size + 2
	h := len(lines) * lh
	y := g.preset.ScreenHeight - h - 2
	ui.Panel(g.screen, image.Rect(0, y-2, w+4, g.preset.ScreenHeight), sc.Panel, sc.Border)
	for i, l := range lines {
		f.Draw(g.screen, l, 2, y+i*lh, sc.Highlight)
	}
}

// present scales the internal screen onto the output surface in the
// current scale mode.
func (g *Game) present(out *ebiten.Image) {
	sw, sh := g.preset.ScreenWidth, g.preset.ScreenHeight
	out.Fill(color.Black)
	r := gfx.Fit(g.settings.ScaleMode, sw, sh, g.outW, g.outH)
	if g.settings.ScaleMode != config.ScaleCRT {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(r.W/float64(sw), r.H/float64(sh))
		op.GeoM.Translate(r.X, r.Y)
		op.Filter = ebiten.FilterNearest
		out.DrawImage(g.screen, op)
		return
	}

	// Upscale by a whole number with nearest-neighbour, then resample
	// smoothly to the 4:3 size so rows stay even and sharp.
	k := gfx.PrescaleFactor(r, sw, sh)
	w, h := sw*k, sh*k
	if g.prescale == nil || g.prescale.Bounds().Dx() != w || g.prescale.Bounds().Dy() != h {
		if g.prescale != nil {
			g.prescale.Deallocate()
		}
		g.prescale = ebiten.NewImage(w, h)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(k), float64(k))
	op.Filter = ebiten.FilterNearest
	g.prescale.DrawImage(g.screen, op)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(r.W/float64(w), r.H/float64(h))
	op.GeoM.Translate(r.X, r.Y)
	op.Filter = ebiten.FilterLinear
	out.DrawImage(g.prescale, op)
}

// LayoutF makes the output surface match the window in device pixels, so
// scaling is done by present rather than by Ebitengine.
func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	w, h := outsideWidth*s, outsideHeight*s
	g.outW, g.outH = int(w), int(h)
	return w, h
}

// Layout is required by ebiten.Game; LayoutF takes precedence.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("unreachable: LayoutF is used")
}
