// Command gameskeleton starts the game: it loads the saved files, opens the
// window and runs the scenes.
package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/gameskeleton/internal/audio"
	"github.com/comprofix/gameskeleton/internal/config"
	"github.com/comprofix/gameskeleton/internal/game"
	"github.com/comprofix/gameskeleton/internal/score"
)

func main() {
	seed := flag.Uint64("seed", 0, "random seed, for repeatable runs (0 picks one from the clock)")
	flag.Parse()
	if *seed == 0 {
		*seed = uint64(time.Now().UnixNano())
	}

	// The settings and scores live beside the game.
	p := config.Project
	dir := config.SaveDir()
	saved := game.Saved{
		Settings: config.LoadSettings(filepath.Join(dir, config.SettingsFile), p),
		Scores:   score.Load(filepath.Join(dir, config.ScoresFile), config.Skills),
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
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
