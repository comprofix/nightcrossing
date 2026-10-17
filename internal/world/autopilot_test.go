package world

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/comprofix/nightcrossing/internal/config"
	"github.com/comprofix/nightcrossing/internal/session"
)

func TestCloneRunsOnItsOwn(t *testing.T) {
	w := New(session.New(config.Stroll, config.StartLives), rich, rand.New(rand.NewPCG(1, 1)))
	tick, u, lives := w.Tick, w.Lanes[1].Entities[0].U, w.Session.Lives
	c := w.Clone(rand.New(rand.NewPCG(2, 2)))
	c.Session.Lives--
	for range 100 {
		c.Update(Up)
	}
	if w.Tick != tick || w.Lanes[1].Entities[0].U != u || w.Session.Lives != lives {
		t.Errorf("running the clone changed the world")
	}
}

// TestAutopilotClearsTheDemoLevels plays the title's demo levels from the
// demo's seed: the autopilot must clear each one without losing a life.
func TestAutopilotClearsTheDemoLevels(t *testing.T) {
	for n := 1; n <= config.DemoLevels; n++ {
		s := session.New(config.Stroll, config.StartLives)
		w := New(s, config.StrollLevels.Level(n), rand.New(rand.NewPCG(config.DemoSeed, uint64(n))))
		var a Autopilot
		for range 60 * config.TPS {
			if w.Update(a.Next(w)); w.Cleared || w.Over {
				break
			}
		}
		if !w.Cleared || s.Lives != config.StartLives {
			t.Errorf("level %d: cleared %v after %d s, %d lives lost", n, w.Cleared, w.Tick/config.TPS, config.StartLives-s.Lives)
		}
	}
}

// TestAutopilotCrossesTheTrains plays Last Train's first levels: the
// autopilot must read the trains and clear them without being struck.
func TestAutopilotCrossesTheTrains(t *testing.T) {
	for n := 1; n <= 3; n++ {
		s := session.New(config.LastTrain, config.StartLives)
		w := New(s, config.LastTrain.Levels.Level(n), rand.New(rand.NewPCG(7, uint64(n))))
		var a Autopilot
		struck := 0
		for range 90 * config.TPS {
			for _, e := range w.Update(a.Next(w)) {
				if e == Struck {
					struck++
				}
			}
			if w.Cleared || w.Over {
				break
			}
		}
		if !w.Cleared || struck > 0 {
			t.Errorf("level %d: cleared %v after %d s, struck %d times", n, w.Cleared, w.Tick/config.TPS, struck)
		}
	}
}

// TestAutopilotEscorts is School Run's bot check (D55): the autopilot,
// escorting its line, plays each level through one round of the loops
// with its lives topped up, and must clear it within config.EscortCheckMins
// minutes.
func TestAutopilotEscorts(t *testing.T) {
	m := config.SchoolRun.Levels
	for n := 1; n <= len(m.Levels)+config.LoopLevels; n++ {
		t.Run(fmt.Sprintf("level %d", n), func(t *testing.T) {
			t.Parallel()
			w := New(session.New(config.SchoolRun, config.StartLives), m.Level(n), rand.New(rand.NewPCG(config.DemoSeed, uint64(n))))
			var a Autopilot
			deaths := 0
			for range config.EscortCheckMins * 60 * config.TPS {
				if w.Update(a.Next(w)); w.Cleared {
					return
				}
				if w.Dying == config.DeathTicks {
					deaths++
				}
				w.Session.Lives = config.StartLives
			}
			t.Errorf("not cleared in %d minutes: %d of 5 nooks, %d deaths", config.EscortCheckMins, filled(w), deaths)
		})
	}
}

func filled(w *World) int {
	n := 0
	for _, f := range w.Filled {
		if f {
			n++
		}
	}
	return n
}
