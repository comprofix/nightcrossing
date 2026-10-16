// Command nightcrossing starts the game: it loads the saved files, opens the
// window and runs the scenes.
package main

import (
	"flag"
	"log"
	"path/filepath"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/audio"
	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/debug"
	"github.com/comprofix/nightcrossing/internal/game"
	"github.com/comprofix/nightcrossing/internal/score"
)

func main() {
	seed := flag.Uint64("seed", 0, "random seed, for repeatable runs (0 picks one from the clock)")
	level := flag.Int("level", 1, "start new games on this level, for testing; such games keep no high scores")
	noShaders, extra := false, ""
	if debug.Enabled {
		flag.StringVar(&extra, "mode", "", "add a mode still being built to the MODE menu, by its ID, e.g. rushhour (debug builds only)")
		flag.BoolVar(&noShaders, "noshaders", false, "draw without shaders, as on a machine that can't run them (debug builds only)")
	}
	flag.Parse()
	modes := config.Modes
	if extra != "" {
		m, ok := modeInProgress(extra)
		if !ok {
			log.Fatalf("no mode in progress called %q", extra)
		}
		modes = append(slices.Clone(modes), m)
	}
	if *seed == 0 {
		*seed = uint64(time.Now().UnixNano())
	}

	// The settings and scores live beside the game.
	p := config.Project
	dir := config.SaveDir()
	saved := game.Saved{
		Settings: config.LoadSettings(filepath.Join(dir, config.SettingsFile), p),
		Scores:   score.Load(filepath.Join(dir, config.ScoresFile), modes),
		Dir:      dir,
	}

	ebiten.SetWindowTitle(config.Title)
	ebiten.SetWindowSize(saved.Settings.WindowW, saved.Settings.WindowH)
	minW, minH := p.MinWindow()
	ebiten.SetWindowSizeLimits(minW, minH, config.MaxWindowW, config.MaxWindowH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(saved.Settings.Fullscreen)
	ebiten.SetWindowClosingHandled(true) // to save the window size on close
	ebiten.SetTPS(config.TPS)

	// Sound is made in code. With no sound device the game
	// still runs, silently.
	synth := audio.NewSynth(p.Audio)
	player, err := audio.Start(synth)
	if err != nil {
		log.Printf("sound: %v", err)
		synth = nil
	}
	defer func() {
		if player != nil {
			player.Close()
		}
	}()

	g, err := game.New(*seed, saved, synth)
	if err != nil {
		log.Fatal(err)
	}
	g.StartAt(*level)
	g.UseModes(modes)
	if noShaders {
		g.WithoutShaders()
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// modeInProgress is the mode being built with the given ID.
func modeInProgress(id string) (config.Mode, bool) {
	for _, m := range config.ModesInProgress {
		if m.ID == id {
			return m, true
		}
	}
	return config.Mode{}, false
}
