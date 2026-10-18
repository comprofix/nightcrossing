package play

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/scene"
	"github.com/comprofix/nightcrossing/internal/scene/scenetest"
	"github.com/comprofix/nightcrossing/internal/session"
	"github.com/comprofix/nightcrossing/internal/world"
)

// TestPredatorsReplay plays Last Train's level 6 from a fixed seed with
// the autopilot at the keys, and logs the predators as they come and go:
// each heron that cocks its head to strike, each fox's eyes, the fox in
// its nook and gone, each water rat's ripple, swim, grab and dive, the
// snake's rustle, time out and going, and each death and nook filled
// (D67–D70).
// A change to their rules or timings shows up here; if it is intended,
// run the test with -update.
func TestPredatorsReplay(t *testing.T) {
	h := scenetest.New(t, 11)
	h.Ctx.NewPlay = New
	h.Ctx.Session = session.New(config.LastTrain, config.StartLives)
	m := config.LastTrain.Levels
	p := newPlay(h.Ctx, m, m.Level(6))
	h.Start(p)
	h.Ticks(scene.FadeTicks)

	keys := map[world.Dir]ebiten.Key{world.Up: ebiten.KeyArrowUp, world.Down: ebiten.KeyArrowDown,
		world.Left: ebiten.KeyArrowLeft, world.Right: ebiten.KeyArrowRight}
	var log strings.Builder
	var a world.Autopilot
	w := p.World()
	fox, goals, lives := w.Fox.Phase, 0, h.Ctx.Session.Lives
	var rats [len(w.Rats)]world.RatPhase
	snake := w.Snake.Phase
	for range 150 * config.TPS {
		if key, ok := keys[a.Next(w)]; ok {
			h.Run(scenetest.Hold(1, key))
		} else {
			h.Ticks(1)
		}
		if w = p.World(); w.Over || w.Level.Number != 6 {
			break
		}
		for r, l := range w.Lanes {
			for _, e := range l.Entities {
				if e.Heron && e.Cock == config.HeronCockTicks {
					fmt.Fprintf(&log, "tick %d: a heron in row %d cocks its head on cell %d, facing %d\n", w.Tick, r, e.HeronCell, e.HeronDir)
				}
			}
		}
		if f := w.Fox; f.Phase != fox {
			switch f.Phase {
			case world.FoxEyes:
				fmt.Fprintf(&log, "tick %d: fox eyes in nook %d\n", w.Tick, f.Nook)
			case world.FoxThere:
				fmt.Fprintf(&log, "tick %d: the fox in nook %d\n", w.Tick, f.Nook)
			default:
				fmt.Fprintf(&log, "tick %d: the fox gone\n", w.Tick)
			}
			fox = f.Phase
		}
		for i, r := range w.Rats {
			if r.Phase == rats[i] {
				continue
			}
			switch r.Phase {
			case world.RatRipple:
				fmt.Fprintf(&log, "tick %d: a water rat comes up in row %d\n", w.Tick, r.Row)
			case world.RatSwim:
				fmt.Fprintf(&log, "tick %d: the water rat in row %d swims\n", w.Tick, r.Row)
			case world.RatGrab:
				fmt.Fprintf(&log, "tick %d: the water rat in row %d grabs its platform\n", w.Tick, r.Row)
			default:
				fmt.Fprintf(&log, "tick %d: a water rat dives\n", w.Tick)
			}
			rats[i] = r.Phase
		}
		if s := w.Snake; s.Phase != snake {
			switch s.Phase {
			case world.SnakeRustle:
				fmt.Fprintf(&log, "tick %d: the grass rustles at column %d\n", w.Tick, s.Col)
			case world.SnakeOut:
				fmt.Fprintf(&log, "tick %d: the snake out\n", w.Tick)
			default:
				fmt.Fprintf(&log, "tick %d: the snake gone\n", w.Tick)
			}
			snake = s.Phase
		}
		if w.Stats.Goals != goals {
			goals = w.Stats.Goals
			fmt.Fprintf(&log, "tick %d: nook filled, %d of 5\n", w.Tick, goals)
		}
		if l := h.Ctx.Session.Lives; l < lives {
			fmt.Fprintf(&log, "tick %d: died (%v) in row %d\n", w.Tick, w.Player.Death, w.Player.LaneRow())
		}
		lives = h.Ctx.Session.Lives
	}
	if s := log.String(); !strings.Contains(s, "cocks its head") || !strings.Contains(s, "fox eyes") ||
		!strings.Contains(s, "water rat comes up") || !strings.Contains(s, "the snake out") {
		t.Errorf("a predator never came:\n%s", s)
	}
	scenetest.Golden(t, "predators", log.String()+summary(p, h.Ctx))
}
