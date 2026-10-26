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
	"github.com/comprofix/nightcrossing/internal/icon"
	"github.com/comprofix/nightcrossing/internal/score"
)

func main() {
	seed := flag.Uint64("seed", 0, "random seed, for repeatable runs (0 picks one from the clock)")
	level := flag.Int("level", 1, "start new games on this level, for testing; such games keep no high scores")
	extra := flag.String("mode", "", "add a mode still being built to the MODE menu, by its ID, for testing")
	noShaders, bonus, storms := false, false, ""
	if debug.Enabled {
		flag.BoolVar(&noShaders, "noshaders", false, "draw without shaders, as on a machine that can't run them (debug builds only)")
		flag.StringVar(&storms, "storm", "", "make every storm wet or dry, for testing; such games keep no high scores (debug builds only)")
		flag.BoolVar(&bonus, "bonus", false, "start games on the mode's bonus level, for tuning it; such games keep no high scores (debug builds only)")
	}
	flag.Parse()
	kinds := map[string]config.StormKind{"": config.AnyStorm, "wet": config.WetStorm, "dry": config.DryStorm}
	stormKind, ok := kinds[storms]
	if !ok {
		log.Fatalf("-storm %q: want wet or dry", storms)
	}
	modes := config.Modes
	if *extra != "" {
		m, ok := modeInProgress(*extra)
		if !ok {
			log.Fatalf("no mode in progress called %q", *extra)
		}
		modes = append(slices.Clone(modes), m)
	}
	if *seed == 0 {
		*seed = uint64(time.Now().UnixNano())
	}

	looks := config.ProjectLooks()
	if err := looks.Check(); err != nil {
		log.Fatal(err)
	}

	// The settings and scores live beside the game.
	p := config.Project
	dir := config.SaveDir()
	saved := game.Saved{
		Settings: config.LoadSettings(filepath.Join(dir, config.SettingsFile), looks),
		Scores:   score.Load(filepath.Join(dir, config.ScoresFile), modes),
		Dir:      dir,
	}

	ebiten.SetWindowTitle(config.Title)
	icons, err := icon.Images()
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowIcon(icons)
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
	g.StormsOf(stormKind)
	g.UseModes(modes)
	if bonus {
		g.BonusFirst()
	}
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
